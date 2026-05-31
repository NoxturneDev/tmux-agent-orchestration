package tmux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEscapeShellSingleQuote(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"I'm here", "I'\\''m here"},
		{"'hello'", "'\\''hello'\\''"},
	}

	for _, tc := range cases {
		got := EscapeShellSingleQuote(tc.input)
		if got != tc.expected {
			t.Errorf("EscapeShellSingleQuote(%q) = %q; expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestExtractActiveGoal(t *testing.T) {
	tempDir := t.TempDir()

	// Case 1: File doesn't exist
	goal := extractActiveGoal(filepath.Join(tempDir, "non_existent.md"))
	if goal != "[No active plan - Idle]" {
		t.Errorf("Expected default idle goal for non-existent file, got %q", goal)
	}

	// Case 2: File is empty or contains only whitespace
	emptyFile := filepath.Join(tempDir, "empty.md")
	if err := os.WriteFile(emptyFile, []byte("   \n\n  \n"), 0644); err != nil {
		t.Fatal(err)
	}
	goal = extractActiveGoal(emptyFile)
	if goal != "[No active plan - Idle]" {
		t.Errorf("Expected default idle goal for empty file, got %q", goal)
	}

	// Case 3: File starts with text lines (no headers)
	plainFile := filepath.Join(tempDir, "plain.md")
	plainContent := "\n\n  This is a raw line of objective text  \nSecond line"
	if err := os.WriteFile(plainFile, []byte(plainContent), 0644); err != nil {
		t.Fatal(err)
	}
	goal = extractActiveGoal(plainFile)
	if goal != "This is a raw line of objective text" {
		t.Errorf("Expected raw line trimmed, got %q", goal)
	}

	// Case 4: File starts with markdown header
	mdFile := filepath.Join(tempDir, "plan.md")
	mdContent := "\n  \n#   ### Critical Task Dispatcher  \nSome detail lines"
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}
	goal = extractActiveGoal(mdFile)
	if goal != "Critical Task Dispatcher" {
		t.Errorf("Expected markdown prefix stripped and trimmed, got %q", goal)
	}
}

func TestFindAllProjectSubdirs(t *testing.T) {
	dirs, err := FindAllProjectSubdirs()
	if err != nil {
		// Skip if run in minimal container environment without find tool
		t.Skip("find command might not be available")
		return
	}
	if len(dirs) == 0 {
		t.Errorf("Expected at least one directory (root), got empty list")
	}
	if dirs[0] != "." {
		t.Errorf("Expected first directory to be '.', got %q", dirs[0])
	}
}

