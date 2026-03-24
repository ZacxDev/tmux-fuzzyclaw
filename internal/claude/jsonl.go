package claude

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
)

// MessageType identifies the type of a conversation message.
type MessageType string

const (
	TypeUser      MessageType = "user"
	TypeAssistant MessageType = "assistant"
	TypeSystem    MessageType = "system"
)

// Message represents a single line from a Claude JSONL conversation file.
type Message struct {
	Type     MessageType `json:"type"`
	UserType string      `json:"userType,omitempty"`
	Message  MessageBody `json:"message"`
}

// MessageBody holds the message content, which can be a string or content blocks.
type MessageBody struct {
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content"`
}

// ContentBlock represents a typed content block in assistant messages.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ExtractText extracts all text content from a message body.
func (m *Message) ExtractText() string {
	if m.Message.Content == nil {
		return ""
	}

	// Try as string first (user messages)
	var s string
	if err := json.Unmarshal(m.Message.Content, &s); err == nil {
		return s
	}

	// Try as array of content blocks (assistant messages)
	var blocks []ContentBlock
	if err := json.Unmarshal(m.Message.Content, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, " ")
	}

	return ""
}

// IsExternalUser returns true if this is an external user message (typed by human).
func (m *Message) IsExternalUser() bool {
	return m.Type == TypeUser && m.UserType == "external"
}

// ParseJSONL parses all messages from a JSONL file.
func ParseJSONL(path string) ([]Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseJSONLReader(f)
}

// ParseJSONLReader parses messages from a reader.
func ParseJSONLReader(r io.Reader) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // skip malformed lines
		}
		messages = append(messages, msg)
	}
	return messages, scanner.Err()
}

// ParseJSONLTail parses the last N lines from a JSONL file.
func ParseJSONLTail(path string, n int) ([]Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read all lines but keep only last N
	var messages []Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(messages) > n {
		messages = messages[len(messages)-n:]
	}
	return messages, nil
}

// ExtractKeywords extracts searchable keywords from a JSONL file.
// Returns user prompts + recent assistant text, matching the bash implementation.
func ExtractKeywords(path string, maxLen int) string {
	messages, err := ParseJSONL(path)
	if err != nil {
		return ""
	}

	var b strings.Builder

	// All user prompts (full history)
	for _, m := range messages {
		if m.Type == TypeUser {
			text := m.ExtractText()
			if text != "" {
				b.WriteString(text)
				b.WriteByte(' ')
			}
		}
	}

	// Last 200 assistant messages for recent context
	assistantMsgs := filterType(messages, TypeAssistant)
	start := 0
	if len(assistantMsgs) > 200 {
		start = len(assistantMsgs) - 200
	}
	for _, m := range assistantMsgs[start:] {
		text := m.ExtractText()
		if text != "" {
			b.WriteString(text)
			b.WriteByte(' ')
		}
	}

	result := b.String()
	// Normalize whitespace
	result = strings.Join(strings.Fields(result), " ")
	if maxLen > 0 && len(result) > maxLen {
		result = result[:maxLen]
	}
	return result
}

// ExtractSummary extracts the last assistant text for summary display.
func ExtractSummary(path string, maxLen int) string {
	messages, err := ParseJSONLTail(path, 50)
	if err != nil {
		return ""
	}

	var lastText string
	for _, m := range messages {
		if m.Type == TypeAssistant {
			text := m.ExtractText()
			if text != "" {
				lastText = text
			}
		}
	}

	// Normalize and truncate
	lastText = strings.Join(strings.Fields(lastText), " ")
	if maxLen > 0 && len(lastText) > maxLen {
		lastText = lastText[:maxLen]
	}
	return lastText
}

// RecentPrompts extracts recent external user prompts from a JSONL file.
func RecentPrompts(path string, n int) []string {
	messages, err := ParseJSONLTail(path, 500)
	if err != nil {
		return nil
	}

	var prompts []string
	for _, m := range messages {
		if m.IsExternalUser() {
			text := m.ExtractText()
			if text != "" {
				prompts = append(prompts, text)
			}
		}
	}

	if len(prompts) > n {
		prompts = prompts[len(prompts)-n:]
	}
	return prompts
}

func filterType(messages []Message, t MessageType) []Message {
	var filtered []Message
	for _, m := range messages {
		if m.Type == t {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
