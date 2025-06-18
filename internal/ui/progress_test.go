package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{name: "normal progress bar", width: 50},
		{name: "wide progress bar", width: 80},
		{name: "minimal progress bar", width: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.width)
			assert.NotNil(t, pb)
			assert.Equal(t, tt.width, pb.Width)
			assert.Equal(t, 0.0, pb.Progress)
			assert.Equal(t, "", pb.Text)
		})
	}
}

func TestProgressBarSetProgress(t *testing.T) {
	pb := NewProgressBar(50)

	tests := []struct {
		name     string
		progress float64
		text     string
		expected float64
	}{
		{name: "zero progress", progress: 0.0, text: "Starting", expected: 0.0},
		{name: "half progress", progress: 0.5, text: "Halfway", expected: 0.5},
		{name: "full progress", progress: 1.0, text: "Complete", expected: 1.0},
		{name: "over limit", progress: 1.5, text: "Over", expected: 1.0},    // should be capped
		{name: "negative", progress: -0.1, text: "Negative", expected: 0.0}, // should be capped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb.SetProgress(tt.progress, tt.text)
			assert.Equal(t, tt.expected, pb.Progress)
			assert.Equal(t, tt.text, pb.Text)
		})
	}
}

func TestProgressBarRender(t *testing.T) {
	tests := []struct {
		name          string
		width         int
		progress      float64
		text          string
		shouldContain []string
	}{
		{
			name:          "empty progress bar",
			width:         20,
			progress:      0.0,
			text:          "Loading",
			shouldContain: []string{"Loading", "[0%]"},
		},
		{
			name:          "half progress",
			width:         20,
			progress:      0.5,
			text:          "Processing",
			shouldContain: []string{"Processing", "[50%]"},
		},
		{
			name:          "full progress",
			width:         20,
			progress:      1.0,
			text:          "Complete",
			shouldContain: []string{"Complete", "[100%]"},
		},
		{
			name:          "no text",
			width:         20,
			progress:      0.25,
			text:          "",
			shouldContain: []string{"[25%]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.width)
			pb.SetProgress(tt.progress, tt.text)

			result := pb.Render()
			assert.NotEmpty(t, result)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		rows     [][]string
		expected []string
	}{
		{
			name:     "simple table",
			headers:  []string{"Name", "Size"},
			rows:     [][]string{{"db1", "1MB"}, {"db2", "2MB"}},
			expected: []string{"Name", "Size", "db1", "1MB", "db2", "2MB"},
		},
		{
			name:     "empty table",
			headers:  []string{"Col1", "Col2"},
			rows:     [][]string{},
			expected: []string{}, // пустая таблица возвращает пустую строку
		},
		{
			name:     "single row",
			headers:  []string{"Database", "Tables", "Size"},
			rows:     [][]string{{"prod_db", "25", "100MB"}},
			expected: []string{"Database", "Tables", "Size", "prod_db", "25", "100MB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTable(tt.headers, tt.rows)

			if len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				for _, expected := range tt.expected {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestRenderBox(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		content  string
		expected []string
	}{
		{
			name:     "simple box",
			title:    "Status",
			content:  "All systems operational",
			expected: []string{"Status", "All systems operational"},
		},
		{
			name:     "empty content",
			title:    "Empty",
			content:  "",
			expected: []string{"Empty"},
		},
		{
			name:     "multiline content",
			title:    "Summary",
			content:  "Line 1\nLine 2",
			expected: []string{"Summary", "Line 1", "Line 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderBox(tt.title, tt.content)
			assert.NotEmpty(t, result)

			for _, expected := range tt.expected {
				assert.Contains(t, result, expected)
			}
		})
	}
}
