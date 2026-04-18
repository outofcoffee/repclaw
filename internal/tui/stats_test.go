package tui

import (
	"strings"
	"testing"
)

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "$0.0000"},
		{0.005, "$0.0050"},
		{0.01, "$0.01"},
		{1.50, "$1.50"},
		{24.13, "$24.13"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCost(tt.input)
			if got != tt.want {
				t.Errorf("formatCost(%f) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatStatsTable(t *testing.T) {
	m := &chatModel{
		stats: &sessionStats{
			inputTokens:      728,
			outputTokens:     70857,
			cacheRead:        28538316,
			cacheWrite:       3868238,
			totalCost:        24.13,
			inputCost:        0.002,
			outputCost:       1.06,
			cacheReadCost:    8.56,
			cacheWriteCost:   14.51,
			totalMessages:    100,
			userMessages:     45,
			assistantMessages: 55,
		},
	}

	table := m.formatStatsTable()

	// Should contain key labels.
	for _, label := range []string{"Input", "Output", "Cache read", "Cache write", "Total", "User", "Assistant"} {
		if !strings.Contains(table, label) {
			t.Errorf("table should contain %q", label)
		}
	}

	// Should contain formatted token values.
	if !strings.Contains(table, "28.5M") {
		t.Error("table should contain formatted cache read tokens")
	}
	// Should contain cost.
	if !strings.Contains(table, "24.13") {
		t.Error("table should contain total cost")
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{32478, "32.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{32478139, "32.5M"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTokens(tt.input)
			if got != tt.want {
				t.Errorf("formatTokens(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
