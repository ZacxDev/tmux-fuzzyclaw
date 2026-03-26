package dashboard

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/claude"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/config"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/state"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui"
)

// Model is the Bubble Tea model for the dashboard view.
type Model struct {
	cfg    *config.Config
	width  int
	height int

	// Data
	entries  []tui.WindowEntry
	filtered []int // indices into entries after search filter
	cursor   int
	selected map[int]bool // multi-select set (indices into filtered)

	// Search
	searchInput textinput.Model
	searchQuery string
	searching   bool

	// Deep search — stores matched CWDs (not indices) so results survive entry refreshes
	deepMatchCwds  map[string]bool
	deepQuery      string // query that produced deepMatchCwds

	// Preview
	previewPrompts []string
	previewSummary string
	previewTarget  string // window ID currently previewed
	searchResults  []claude.SearchResult

	// Scroll
	scrollOffset int // first visible row index

	// State
	ready     bool
	err       error
	lastFetch time.Time
}

// New creates a new dashboard model.
func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "/ "
	ti.CharLimit = 120
	ti.Width = 40

	return Model{
		cfg:      cfg,
		selected: make(map[int]bool),
		searchInput: ti,
	}
}

// Init returns the initial command to fetch data and start tickers.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchWindows(m.cfg),
		refreshTick(),
		dataPollTick(m.cfg.Dashboard.RefreshInterval),
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tui.WindowsRefreshedMsg:
		m.entries = msg.Entries
		m.lastFetch = time.Now()
		m.applyFilter()
		// Load preview for current selection
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tui.ConversationLoadedMsg:
		if msg.WindowID == m.previewTarget {
			m.previewPrompts = msg.Prompts
			m.previewSummary = msg.Summary
		}

	case tui.SearchResultsMsg:
		m.searchResults = msg.Results

	case tui.RefreshTickMsg:
		cmds = append(cmds, refreshTick())

	case tui.DataPollMsg:
		cmds = append(cmds, fetchWindows(m.cfg))
		cmds = append(cmds, dataPollTick(m.cfg.Dashboard.RefreshInterval))

	case tui.FileChangedMsg:
		cmds = append(cmds, fetchWindows(m.cfg))

	case deepSearchResultMsg:
		if msg.query == m.searchQuery {
			m.deepMatchCwds = msg.matchedCwds
			m.deepQuery = msg.query
			m.applyFilter()
			if cmd := m.loadPreview(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tui.ErrorMsg:
		m.err = msg.Err

	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchKey(msg, cmds)
		}
		return m.handleKey(msg, cmds)
	}

	if m.searching {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Live filter as user types
		newQuery := m.searchInput.Value()
		if newQuery != m.searchQuery {
			m.searchQuery = newQuery
			// Clear stale deep results if query changed
			if m.deepQuery != newQuery {
				m.deepMatchCwds = nil
			}
			m.applyFilter()
			// Kick off async deep search via ripgrep
			if cmd := m.deepSearch(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if cmd := m.loadPreview(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		m.moveCursor(1)
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "k", "up":
		m.moveCursor(-1)
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "g", "home":
		m.cursor = 0
		m.scrollOffset = 0
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "G", "end":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
		m.ensureCursorVisible()
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "enter":
		if entry := m.currentEntry(); entry != nil {
			return m, tea.Sequence(
				func() tea.Msg {
					_ = tmux.SwitchClient(entry.Window.Target)
					return nil
				},
				tea.Quit,
			)
		}

	case "tab":
		if len(m.filtered) > 0 {
			idx := m.filtered[m.cursor]
			if m.selected[idx] {
				delete(m.selected, idx)
			} else {
				m.selected[idx] = true
			}
			m.moveCursor(1)
		}

	case "ctrl+a":
		if len(m.selected) == len(m.filtered) {
			m.selected = make(map[int]bool)
		} else {
			for _, idx := range m.filtered {
				m.selected[idx] = true
			}
		}

	case "ctrl+x":
		targets := m.selectedTargets()
		if len(targets) == 0 {
			if entry := m.currentEntry(); entry != nil {
				targets = []string{entry.Window.Target}
			}
		}
		if len(targets) > 0 {
			return m, killWindows(targets)
		}

	case "/":
		m.searching = true
		m.searchInput.Focus()
		cmds = append(cmds, m.searchInput.Focus())
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSearchKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchInput.Blur()
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.searchResults = nil
		m.deepMatchCwds = nil
		m.deepQuery = ""
		m.applyFilter()

	case "enter":
		// Select current entry and switch
		if entry := m.currentEntry(); entry != nil {
			return m, tea.Sequence(
				func() tea.Msg {
					_ = tmux.SwitchClient(entry.Window.Target)
					return nil
				},
				tea.Quit,
			)
		}
		m.searching = false
		m.searchInput.Blur()

	case "up", "ctrl+p", "ctrl+k":
		m.moveCursor(-1)
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "down", "ctrl+n", "ctrl+j":
		m.moveCursor(1)
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case "tab":
		if len(m.filtered) > 0 {
			idx := m.filtered[m.cursor]
			if m.selected[idx] {
				delete(m.selected, idx)
			} else {
				m.selected[idx] = true
			}
			m.moveCursor(1)
		}

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		newQuery := m.searchInput.Value()
		if newQuery != m.searchQuery {
			m.searchQuery = newQuery
			if m.deepQuery != newQuery {
				m.deepMatchCwds = nil
			}
			m.applyFilter()
			if cmd := m.deepSearch(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if cmd := m.loadPreview(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

// View renders the dashboard.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Layout: header + table | preview + status bar
	tableWidth := m.width
	previewWidth := 0
	if m.cfg.Dashboard.Preview.Width > 0 && m.width > 80 {
		previewWidth = m.width * m.cfg.Dashboard.Preview.Width / 100
		tableWidth = m.width - previewWidth - 1
	}

	contentHeight := m.height - 2 // header + status bar
	if m.searching {
		contentHeight-- // search bar
	}

	// Header
	header := m.renderHeader(tableWidth)

	// Table
	table := m.renderTable(tableWidth, contentHeight)

	// Preview panel
	var content string
	if previewWidth > 0 {
		preview := m.renderPreview(previewWidth, contentHeight+1)
		content = lipgloss.JoinHorizontal(lipgloss.Top, table, " ", preview)
	} else {
		content = table
	}

	// Search bar
	search := ""
	if m.searching {
		search = m.searchInput.View() + "\n"
	}

	// Status bar
	statusBar := m.renderStatusBar()

	return header + "\n" + content + "\n" + search + statusBar
}

func (m *Model) moveCursor(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	m.ensureCursorVisible()
}

// ensureCursorVisible adjusts scrollOffset so the cursor is within the viewport.
// Called after any cursor or filter change. Uses contentHeight estimate since
// exact height isn't known until View(), but this is close enough.
func (m *Model) ensureCursorVisible() {
	// Estimate visible height (same logic as View)
	height := m.height - 2
	if m.searching {
		height--
	}
	if height < 1 {
		height = 1
	}

	// Scroll up if cursor above viewport
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	// Scroll down if cursor below viewport
	if m.cursor >= m.scrollOffset+height {
		m.scrollOffset = m.cursor - height + 1
	}
	// Clamp
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *Model) currentEntry() *tui.WindowEntry {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return &m.entries[m.filtered[m.cursor]]
	}
	return nil
}

func (m *Model) selectedTargets() []string {
	var targets []string
	for idx := range m.selected {
		if idx >= 0 && idx < len(m.entries) {
			targets = append(targets, m.entries[idx].Window.Target)
		}
	}
	return targets
}

func (m *Model) applyFilter() {
	m.filtered = m.filtered[:0]
	if m.searchQuery == "" {
		// No filter — sort by idle time (ascending = most recent first)
		indices := make([]int, len(m.entries))
		for i := range indices {
			indices[i] = i
		}
		now := time.Now()
		sort.Slice(indices, func(a, b int) bool {
			idleA := m.entries[indices[a]].IdleSeconds(now)
			idleB := m.entries[indices[b]].IdleSeconds(now)
			if m.cfg.Dashboard.SortAscending {
				return idleA < idleB
			}
			return idleA > idleB
		})
		m.filtered = indices
	} else {
		// Fast: substring match against cached fields only
		queryLower := strings.ToLower(m.searchQuery)
		for i, e := range m.entries {
			searchable := strings.ToLower(strings.Join([]string{
				e.CleanName(),
				e.Window.Dir,
				e.Window.FullCwd,
				e.Summary,
				e.Keywords,
			}, " "))
			if strings.Contains(searchable, queryLower) {
				m.filtered = append(m.filtered, i)
			}
		}
		// Also include entries whose cwd matched in the async JSONL deep search
		if len(m.deepMatchCwds) > 0 {
			for i, e := range m.entries {
				if m.deepMatchCwds[e.Window.FullCwd] && !m.inFiltered(i) {
					m.filtered = append(m.filtered, i)
				}
			}
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.ensureCursorVisible()
}

func (m *Model) inFiltered(idx int) bool {
	for _, f := range m.filtered {
		if f == idx {
			return true
		}
	}
	return false
}

// deepSearch launches a single async ripgrep scan across all cwds at once.
func (m *Model) deepSearch() tea.Cmd {
	query := m.searchQuery
	if query == "" {
		return nil
	}
	entries := m.entries
	cfg := m.cfg
	return func() tea.Msg {
		// Collect unique cwds
		cwdSet := make(map[string]bool)
		var cwds []string
		for _, e := range entries {
			cwd := e.Window.FullCwd
			if cwd != "" && !cwdSet[cwd] {
				cwdSet[cwd] = true
				cwds = append(cwds, cwd)
			}
		}
		// Single rg call across all cwds — returns matched CWDs (not indices)
		matched := claude.BatchCwdSearch(cfg.ClaudeProjectDir, query, cwds)
		return deepSearchResultMsg{query: query, matchedCwds: matched}
	}
}

func (m *Model) loadPreview() tea.Cmd {
	entry := m.currentEntry()
	if entry == nil {
		return nil
	}
	target := entry.Window.WindowID
	if target == m.previewTarget && m.searchQuery == "" {
		return nil // already loaded
	}
	m.previewTarget = target

	cfg := m.cfg
	cwd := entry.Window.FullCwd
	sessionID := ""
	if entry.Task != nil {
		sessionID = entry.Task.ClaudeSession
	}
	query := m.searchQuery

	return func() tea.Msg {
		if query != "" {
			// Search mode
			jsonlPath := findJSONL(cfg, cwd, sessionID)
			if jsonlPath != "" {
				results, _ := claude.SearchConversation(jsonlPath, query)
				return tui.SearchResultsMsg{Results: results, Query: query}
			}
			return tui.SearchResultsMsg{Query: query}
		}

		// Default: load recent prompts
		jsonlPath := findJSONL(cfg, cwd, sessionID)
		if jsonlPath == "" {
			return tui.ConversationLoadedMsg{WindowID: target}
		}
		prompts := claude.RecentPrompts(jsonlPath, 5)
		summary := claude.ExtractSummary(jsonlPath, 200)
		return tui.ConversationLoadedMsg{
			WindowID: target,
			Prompts:  prompts,
			Summary:  summary,
		}
	}
}

func findJSONL(cfg *config.Config, cwd, sessionID string) string {
	if sessionID != "" {
		if path, err := claude.SessionFile(cfg.ClaudeProjectDir, cwd, sessionID); err == nil {
			return path
		}
	}
	path, err := claude.LatestSessionFile(cfg.ClaudeProjectDir, cwd)
	if err != nil {
		return ""
	}
	return path
}

// deepSearchResultMsg carries results from async JSONL scanning.
type deepSearchResultMsg struct {
	query      string
	matchedCwds map[string]bool
}

// --- Commands ---

func fetchWindows(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		windows, err := tmux.ListAllWindows()
		if err != nil {
			return tui.ErrorMsg{Err: err}
		}

		tasks, _ := state.ReadAllTasks(cfg.StateDir)

		// Build keyword/summary cache per cwd
		type cwdCache struct {
			keywords string
			summary  string
		}
		cwdCaches := make(map[string]*cwdCache)

		entries := make([]tui.WindowEntry, 0, len(windows))
		for _, w := range windows {
			entry := tui.WindowEntry{Window: w}

			// Task state
			cleanID := w.CleanID()
			if t, ok := tasks[cleanID]; ok {
				entry.Task = &tui.TaskSnapshot{
					Task:          t.Task,
					Status:        t.Status,
					Cwd:           t.Cwd,
					ClaudeSession: t.ClaudeSession,
					Started:       t.Started,
					LastActivity:  t.LastActivity,
					Summary:       t.Summary,
				}
				entry.Summary = t.Summary
			}

			// Activity timestamp
			if act, err := state.ReadActivity(cfg.ActivityDir, w.WindowID); err == nil {
				entry.Activity = act
			}

			// Keywords/summary from JSONL (cached per cwd)
			cwd := w.FullCwd
			if cwd != "" {
				cc, ok := cwdCaches[cwd]
				if !ok {
					cc = &cwdCache{}
					jsonlPath, err := claude.LatestSessionFile(cfg.ClaudeProjectDir, cwd)
					if err == nil {
						cc.keywords = claude.ExtractKeywords(jsonlPath, 3000)
						if entry.Summary == "" {
							cc.summary = claude.ExtractSummary(jsonlPath, 80)
						}
					}
					cwdCaches[cwd] = cc
				}
				entry.Keywords = cc.keywords
				if entry.Summary == "" {
					entry.Summary = cc.summary
				}
			}

			entries = append(entries, entry)
		}

		return tui.WindowsRefreshedMsg{Entries: entries}
	}
}

func killWindows(targets []string) tea.Cmd {
	return func() tea.Msg {
		for _, t := range targets {
			_ = tmux.KillWindow(t)
		}
		return tui.DataPollMsg{}
	}
}

func refreshTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tui.RefreshTickMsg{}
	})
}

func dataPollTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tui.DataPollMsg{}
	})
}
