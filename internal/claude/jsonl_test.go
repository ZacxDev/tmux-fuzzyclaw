package claude

import (
	"path/filepath"
	"strings"
	"testing"
)

var testDataDir = filepath.Join("..", "..", "testdata", "sessions")

func TestParseJSONL(t *testing.T) {
	messages, err := ParseJSONL(filepath.Join(testDataDir, "test.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 7 {
		t.Errorf("expected 7 messages, got %d", len(messages))
	}

	// First message should be external user
	if !messages[0].IsExternalUser() {
		t.Error("expected first message to be external user")
	}
	text := messages[0].ExtractText()
	if text != "implement authentication system" {
		t.Errorf("unexpected text: %s", text)
	}
}

func TestExtractTextString(t *testing.T) {
	msg := Message{
		Type: TypeUser,
		Message: MessageBody{
			Content: []byte(`"hello world"`),
		},
	}
	if got := msg.ExtractText(); got != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", got)
	}
}

func TestExtractTextBlocks(t *testing.T) {
	msg := Message{
		Type: TypeAssistant,
		Message: MessageBody{
			Content: []byte(`[{"type":"text","text":"part one"},{"type":"tool_use","name":"test"},{"type":"text","text":"part two"}]`),
		},
	}
	got := msg.ExtractText()
	if !strings.Contains(got, "part one") || !strings.Contains(got, "part two") {
		t.Errorf("expected both text parts, got '%s'", got)
	}
}

func TestIsExternalUser(t *testing.T) {
	tests := []struct {
		msg    Message
		expect bool
	}{
		{Message{Type: TypeUser, UserType: "external"}, true},
		{Message{Type: TypeUser, UserType: "internal"}, false},
		{Message{Type: TypeAssistant}, false},
	}
	for _, tt := range tests {
		if got := tt.msg.IsExternalUser(); got != tt.expect {
			t.Errorf("IsExternalUser() = %v for type=%s userType=%s", got, tt.msg.Type, tt.msg.UserType)
		}
	}
}

func TestRecentPrompts(t *testing.T) {
	prompts := RecentPrompts(filepath.Join(testDataDir, "test.jsonl"), 5)
	if len(prompts) != 3 {
		t.Errorf("expected 3 external user prompts, got %d", len(prompts))
	}
	if prompts[0] != "implement authentication system" {
		t.Errorf("unexpected first prompt: %s", prompts[0])
	}
	if prompts[2] != "fix the token expiration bug" {
		t.Errorf("unexpected last prompt: %s", prompts[2])
	}
}

func TestExtractKeywords(t *testing.T) {
	keywords := ExtractKeywords(filepath.Join(testDataDir, "test.jsonl"), 3000)
	if keywords == "" {
		t.Error("expected non-empty keywords")
	}
	// Should contain user prompts
	if !strings.Contains(keywords, "authentication") {
		t.Error("expected keywords to contain 'authentication'")
	}
	// Should contain assistant text
	if !strings.Contains(keywords, "JWT") {
		t.Error("expected keywords to contain 'JWT'")
	}
}

func TestExtractSummary(t *testing.T) {
	summary := ExtractSummary(filepath.Join(testDataDir, "test.jsonl"), 200)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	// Should be the last assistant message
	if !strings.Contains(summary, "expiration") {
		t.Error("expected summary to contain text from last assistant message")
	}
}

func TestParseJSONLTail(t *testing.T) {
	messages, err := ParseJSONLTail(filepath.Join(testDataDir, "test.jsonl"), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
}
