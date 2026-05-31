package prompt

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	cases := []struct {
		macro    Macro
		contains string
	}{
		{NoMacro, ""},
		{WrapItUp, "Wrap up"},
		{CookIt, "Tech Lead"},
		{Implement, "implementing the next step"},
		{Recon, "surface-level reconnaissance"},
	}

	for _, tc := range cases {
		p := BuildPrompt(tc.macro)
		if !strings.Contains(p, string(tc.macro)) {
			t.Errorf("Expected prompt to contain macro %q, got %q", tc.macro, p)
		}
		if tc.macro == NoMacro {
			if p != NoMacroReminder {
				t.Errorf("Expected NoMacro prompt to equal NoMacroReminder exactly, got %q", p)
			}
		} else {
			if !strings.Contains(p, BaseReminder) {
				t.Errorf("Expected prompt to contain base reminder, got %q", p)
			}
		}
	}
}
