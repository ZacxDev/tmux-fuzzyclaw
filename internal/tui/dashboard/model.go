package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui/styles"
)

// projectEntry is an aggregated project row for the project-level view.
type projectEntry struct {
	cwd          string // full cwd path
	name         string // basename for display
	sessionCount int
	linesAdded   int
	running      int
	waiting      int
	paused       int
	done         int
	bestPriority int // lowest statusPriority = most urgent
}

// Model is the Bubble Tea model for the dashboard view.
type Model struct {
	cfg    *config.Config
	width  int
	height int

	// Data
	entries   []tui.WindowEntry
	filtered  []int // indices into entries after search filter
	cursor    int
	selected  map[int]bool // multi-select set (indices into filtered)

	// Two-level navigation
	viewMode        string          // "projects" or "sessions"
	selectedProject string          // cwd of drilled-in project ("" = all)
	projects        []projectEntry  // aggregated project rows

	// Search
	searchInput textinput.Model
	searchQuery string
	searching   bool

	// Status filter — "" means show all
	statusFilter string // "running", "paused", "waiting", "done", ""

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
		cfg:         cfg,
		selected:    make(map[int]bool),
		searchInput: ti,
		viewMode:    "sessions",
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
			m.cursor = 0
			m.scrollOffset = 0
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
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.viewMode == "sessions" && m.selectedProject != "" {
			// Go back to project view
			m.viewMode = "projects"
			m.selectedProject = ""
			m.cursor = 0
			m.scrollOffset = 0
			m.applyFilter()
			if cmd := m.loadPreview(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
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
		if m.viewMode == "projects" {
			// Drill into selected project
			if m.cursor >= 0 && m.cursor < len(m.projects) {
				m.selectedProject = m.projects[m.cursor].cwd
				m.viewMode = "sessions"
				m.cursor = 0
				m.scrollOffset = 0
				m.applyFilter()
				if cmd := m.loadPreview(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		} else if entry := m.currentEntry(); entry != nil {
			// Switch to selected window
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

	case "1": // Filter: running
		m.toggleStatusFilter("running")
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "2": // Filter: paused
		m.toggleStatusFilter("paused")
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "3": // Filter: waiting
		m.toggleStatusFilter("waiting")
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "4": // Filter: done
		m.toggleStatusFilter("done")
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "0": // Clear filter
		m.toggleStatusFilter("")
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "p": // Toggle between project and session view
		if m.viewMode == "projects" {
			m.viewMode = "sessions"
			m.selectedProject = ""
		} else {
			m.viewMode = "projects"
			m.selectedProject = ""
		}
		m.cursor = 0
		m.scrollOffset = 0
		m.applyFilter()
		if cmd := m.loadPreview(); cmd != nil {
			cmds = append(cmds, cmd)
		}
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
		m.cursor = 0
		m.scrollOffset = 0
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
			m.cursor = 0
			m.scrollOffset = 0
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
	if m.viewMode == "sessions" && m.selectedProject != "" {
		contentHeight-- // breadcrumb line
	}

	// Header
	header := m.renderHeader(tableWidth)

	// Breadcrumb when drilled into a project
	if m.viewMode == "sessions" && m.selectedProject != "" {
		crumb := styles.SectionStyle.Render(fmt.Sprintf("── %s ──", filepath.Base(m.selectedProject)))
		header = header + "\n" + crumb
	}

	// Table
	table := m.renderTable(tableWidth, contentHeight)

	// Preview panel
	var content string
	if previewWidth > 0 {
		preview := m.renderPreview(previewWidth, contentHeight-2) // -2 for border lines
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
	listLen := len(m.filtered)
	if m.viewMode == "projects" {
		listLen = len(m.projects)
	}
	if listLen == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= listLen {
		m.cursor = listLen - 1
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

// statusPriority returns sort priority (lower = more urgent).
func statusPriority(status string) int {
	switch status {
	case "waiting":
		return 0
	case "paused":
		return 1
	case "running":
		return 2
	case "done":
		return 3
	default:
		return 4
	}
}

// buildProjects aggregates entries into project-level rows.
func (m *Model) buildProjects() {
	seen := make(map[string]*projectEntry)
	m.projects = m.projects[:0]
	for i := range m.entries {
		e := &m.entries[i]
		cwd := e.Window.FullCwd
		if cwd == "" {
			cwd = e.Window.Dir
		}
		p, ok := seen[cwd]
		if !ok {
			p = &projectEntry{
				cwd:          cwd,
				name:         filepath.Base(cwd),
				bestPriority: 99,
			}
			seen[cwd] = p
			m.projects = append(m.projects, *p)
		}
		idx := -1
		for j := range m.projects {
			if m.projects[j].cwd == cwd {
				idx = j
				break
			}
		}
		if idx < 0 {
			continue
		}
		m.projects[idx].sessionCount++
		m.projects[idx].linesAdded += e.LinesAdded
		switch m.entryStatus(e) {
		case "running":
			m.projects[idx].running++
		case "waiting":
			m.projects[idx].waiting++
		case "paused":
			m.projects[idx].paused++
		case "done":
			m.projects[idx].done++
		}
		pri := statusPriority(m.entryStatus(e))
		if pri < m.projects[idx].bestPriority {
			m.projects[idx].bestPriority = pri
		}
	}
	// Sort by attention priority, then name
	sort.Slice(m.projects, func(a, b int) bool {
		if m.projects[a].bestPriority != m.projects[b].bestPriority {
			return m.projects[a].bestPriority < m.projects[b].bestPriority
		}
		return m.projects[a].name < m.projects[b].name
	})
}

func (m *Model) applyFilter() {
	m.filtered = m.filtered[:0]

	if m.viewMode == "projects" {
		// Project view — filtered is not used for rendering, buildProjects handles it
		m.buildProjects()
		if m.cursor >= len(m.projects) {
			m.cursor = max(0, len(m.projects)-1)
		}
		m.ensureCursorVisible()
		return
	}

	// Session view — filter entries to selectedProject (or all if empty)
	if m.searchQuery == "" {
		indices := make([]int, 0, len(m.entries))
		for i := range m.entries {
			if m.selectedProject != "" {
				cwd := m.entries[i].Window.FullCwd
				if cwd == "" {
					cwd = m.entries[i].Window.Dir
				}
				if cwd != m.selectedProject {
					continue
				}
			}
			indices = append(indices, i)
		}

		// Attention-priority sort
		now := time.Now()
		sort.Slice(indices, func(a, b int) bool {
			ea := &m.entries[indices[a]]
			eb := &m.entries[indices[b]]
			pa := statusPriority(m.entryStatus(ea))
			pb := statusPriority(m.entryStatus(eb))
			if pa != pb {
				return pa < pb
			}
			return ea.MsgAgeSeconds(now) < eb.MsgAgeSeconds(now)
		})

		// Apply status filter if active
		if m.statusFilter != "" {
			var matched []int
			for _, idx := range indices {
				if m.entryStatus(&m.entries[idx]) == m.statusFilter {
					matched = append(matched, idx)
				}
			}
			indices = matched
		}

		m.filtered = indices
	} else {
		// Search: substring match against cached fields
		queryLower := strings.ToLower(m.searchQuery)
		for i, e := range m.entries {
			if m.selectedProject != "" {
				cwd := e.Window.FullCwd
				if cwd == "" {
					cwd = e.Window.Dir
				}
				if cwd != m.selectedProject {
					continue
				}
			}
			searchable := strings.ToLower(strings.Join([]string{
				e.CleanName(),
				e.Window.Dir,
				e.Window.FullCwd,
				e.Summary,
				e.Keywords,
				e.FirstPrompt,
				e.GitBranch,
			}, " "))
			if strings.Contains(searchable, queryLower) {
				m.filtered = append(m.filtered, i)
			}
		}
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

func (m *Model) toggleStatusFilter(status string) {
	if m.statusFilter == status || status == "" {
		m.statusFilter = ""
	} else {
		m.statusFilter = status
	}
	m.cursor = 0
	m.scrollOffset = 0
	m.applyFilter()
}

// entryStatus returns the effective status for filtering.
func (m *Model) entryStatus(e *tui.WindowEntry) string {
	if e.Task != nil && e.Task.Status != "" {
		return e.Task.Status
	}
	if isClaudeRunning(e) {
		return "running"
	}
	return ""
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
	transcriptPath := ""
	if entry.Task != nil {
		sessionID = entry.Task.ClaudeSession
		transcriptPath = entry.Task.TranscriptPath
	}
	query := m.searchQuery

	return func() tea.Msg {
		if query != "" {
			// Search mode
			jsonlPath := findJSONL(cfg, cwd, sessionID, transcriptPath)
			if jsonlPath != "" {
				results, _ := claude.SearchConversation(jsonlPath, query)
				return tui.SearchResultsMsg{Results: results, Query: query}
			}
			return tui.SearchResultsMsg{Query: query}
		}

		// Default: load recent prompts
		jsonlPath := findJSONL(cfg, cwd, sessionID, transcriptPath)
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

func findJSONL(cfg *config.Config, cwd, sessionID, transcriptPath string) string {
	// Use transcript path directly when available from hooks
	if transcriptPath != "" {
		if _, err := os.Stat(transcriptPath); err == nil {
			return transcriptPath
		}
	}
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

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
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
		telemetry := state.ReadTelemetryByCwd(cfg.ActivityDir)

		// Build keyword/summary cache per cwd
		type cwdCache struct {
			keywords   string
			summary    string
			indexEntry *claude.IndexEntry
		}
		cwdCaches := make(map[string]*cwdCache)

		entries := make([]tui.WindowEntry, 0, len(windows))
		for _, w := range windows {
			entry := tui.WindowEntry{Window: w}

			// Task state
			cleanID := w.CleanID()
			if t, ok := tasks[cleanID]; ok {
				entry.Task = &tui.TaskSnapshot{
					Task:           t.Task,
					Status:         t.Status,
					Cwd:            t.Cwd,
					ClaudeSession:  t.ClaudeSession,
					Started:        t.Started,
					LastActivity:   t.LastActivity,
					Summary:        t.Summary,
					TranscriptPath: t.TranscriptPath,
				}
				entry.Summary = t.Summary
			}

			// Activity timestamp
			if act, err := state.ReadActivity(cfg.ActivityDir, w.WindowID); err == nil {
				entry.Activity = act
			}

			// Telemetry from statusline
			if t, ok := telemetry[w.FullCwd]; ok {
				entry.Model = t.Model
				entry.CostUSD = t.CostUSD
				entry.ContextPct = t.ContextPct
				entry.LinesAdded = t.LinesAdded
				entry.LinesRemoved = t.LinesRemoved
			}

			// Keywords/summary: try sessions-index.json first (fast), fall back to JSONL
			cwd := w.FullCwd
			if cwd != "" {
				cc, ok := cwdCaches[cwd]
				if !ok {
					cc = &cwdCache{}
					if ie, err := claude.LatestIndexEntry(cfg.ClaudeProjectDir, cwd); err == nil {
						// Fast path: small JSON index
						cc.keywords = ie.FirstPrompt
						if ie.Summary != "" {
							cc.summary = ie.Summary
						} else {
							cc.summary = truncateStr(ie.FirstPrompt, 80)
						}
						cc.indexEntry = ie
					} else {
						// Fallback: JSONL parsing (index doesn't exist)
						jsonlPath, err := claude.LatestSessionFile(cfg.ClaudeProjectDir, cwd)
						if err == nil {
							cc.keywords = claude.ExtractKeywords(jsonlPath, 3000)
							if entry.Summary == "" {
								cc.summary = claude.ExtractSummary(jsonlPath, 80)
							}
						}
					}
					cwdCaches[cwd] = cc
				}
				entry.Keywords = cc.keywords
				// Prefer sessions-index summary (curated) over task state summary (last_assistant_message)
				if cc.summary != "" {
					entry.Summary = cc.summary
				}
				if cc.indexEntry != nil {
					entry.FirstPrompt = cc.indexEntry.FirstPrompt
					entry.GitBranch = cc.indexEntry.GitBranch
					entry.MessageCount = cc.indexEntry.MessageCount
					entry.SessionID = cc.indexEntry.SessionID
					// Only set LastMessage when this window has an active session (hook has fired)
					if entry.Task != nil {
						if t, err := time.Parse(time.RFC3339Nano, cc.indexEntry.Modified); err == nil {
							entry.LastMessage = t
						} else if t, err := time.Parse(time.RFC3339, cc.indexEntry.Modified); err == nil {
							entry.LastMessage = t
						}
					}
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
