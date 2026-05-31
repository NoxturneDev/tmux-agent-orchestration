package prompt

import "fmt"

const BaseReminder = "CRITICAL: Enforce all rules in ~/agents/AGENTS.md. Read .agents/plan/active_plan.md for current state."
const NoMacroReminder = "Read ~/agents/AGENTS.md"

type Macro string

const (
	NoMacro   Macro = ""
	Implement Macro = "Continue implementing the next step in the active plan."
	CookIt    Macro = "Act as Tech Lead. Critique the active plan pragmatically."
	WrapItUp  Macro = "Wrap up the current task, update docs/, and prepare a commit plan."
	Recon     Macro = "Perform a surface-level reconnaissance of this repository. Map stack, infra, and boot instructions."
)

// BuildPrompt constructs the prompt string. If macro is empty (NoMacro), only return the NoMacroReminder.
func BuildPrompt(macro Macro) string {
	if macro == NoMacro {
		return NoMacroReminder
	}
	return fmt.Sprintf("%s %s", BaseReminder, string(macro))
}
