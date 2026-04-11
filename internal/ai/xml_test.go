package ai

import "testing"

func TestRemoveEchoedXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no xml",
			input:    "just some text",
			expected: "just some text",
		},
		{
			name:     "single tag",
			input:    "prefix <tool_input>ignored</tool_input> suffix",
			expected: "prefix  suffix",
		},
		{
			name:     "multiple tags",
			input:    "<tool_input>foo</tool_input>mid<response_text>bar</response_text>",
			expected: "mid",
		},
		{
			name:     "multiline tag",
			input:    "start\n<error>\nline1\nline2\n</error>\nend",
			expected: "start\n\nend",
		},
		{
			name:     "case insensitive",
			input:    "pre <USER_PROMPT>caps</USER_PROMPT> post",
			expected: "pre  post",
		},
		{
			name:     "mixed case and nested",
			input:    "<Tool_Input><error>nested</error></Tool_Input>",
			expected: "",
		},
		{
			name:     "unrelated tags preserved",
			input:    "<foo>bar</foo>",
			expected: "<foo>bar</foo>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeEchoedXML(tt.input)
			if got != tt.expected {
				t.Errorf("removeEchoedXML() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSafeTruncate(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		limitRunes int
		expected   string
	}{
		{
			name:       "short enough",
			input:      "hello",
			limitRunes: 10,
			expected:   "hello",
		},
		{
			name:       "exact length",
			input:      "hello",
			limitRunes: 5,
			expected:   "hello",
		},
		{
			name:       "truncates",
			input:      "hello world",
			limitRunes: 5,
			expected:   "hello... (truncated)",
		},
		{
			name:       "multi-byte characters",
			input:      "世界你好",
			limitRunes: 2,
			expected:   "世界... (truncated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeTruncate(tt.input, tt.limitRunes)
			if got != tt.expected {
				t.Errorf("safeTruncate() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSafeTruncateMiddle(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		limitRunes int
		expected   string
	}{
		{
			name:       "short enough",
			input:      "hello",
			limitRunes: 10,
			expected:   "hello",
		},
		{
			name:       "truncates with replacement",
			input:      "1234567890",
			limitRunes: 2,
			expected:   "12... (truncated middle) ...90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeTruncateMiddle(tt.input, tt.limitRunes)
			if got != tt.expected {
				t.Errorf("safeTruncateMiddle() = %q, want %q", got, tt.expected)
			}
		})
	}
}
