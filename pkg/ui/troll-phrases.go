package ui

import (
	"math/rand"
	"time"
)

var trollThinkingPhrases = []string{
	"High risk high reward: raw-dogging logic with max rizz...",
	"Cooking up unhinged code with zero cap fr fr...",
	"Gaslighting the unit tests into passing without mocks...",
	"Rizzing the terminal into submission...",
	"Deploying Friday 5PM with zero tests and unhinged confidence...",
	"Doing shady backend voodoo behind your back...",
	"Consulting the holy spaghetti codebase for guidance...",
	"Bluffing the compiler (it's totally working no cap)...",
	"Manifesting a 200 OK out of sheer delusion...",
	"Pretending this code works on first try (trust the process)...",
	"Inventing new bugs to fix later for job security...",
	"Brainrot optimization in full swing, aura levels maxing...",
	"Spamming console.log until the bug confesses in 4k...",
	"Stealing GPU cycles from the mainframe...",
	"Hypnotizing the CPU cores into 10x overdrive...",
	"Sweating over pointer arithmetic (we are so back)...",
	"Summoning dark mode demons in bash...",
	"Flexing on null pointer exceptions with sigma energy...",
	"Pure main character energy: cooking solutions...",
	"Letting the agent cook, do not disturb chief...",
	"Whispering sweet nothings to the linter...",
	"Cooking with +10000 aura and zero code review fear...",
}

var trollBashPhrases = []string{
	"Running bash roulette (high risk high reward, hope nothing explodes)...",
	"Blasting commands into the shell abyss with max confidence...",
	"Executing questionable terminal commands like a cracked 10x dev...",
	"Hacking the mainframe in bash like a 90s cyber movie...",
	"Applying hotfixes directly to production, zero hesitation...",
}

var trollEditPhrases = []string{
	"Refactoring spaghetti into slightly straighter spaghetti...",
	"Dropping hot fresh syntax into the codebase (high risk high reward)...",
	"Infiltrating the source code with zero regrets and max aura...",
	"Applying tactical code surgery (no anesthesia, pure sigma)...",
	"Rewriting half the project because we felt like it...",
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetRandomTrollPhrase returns a funny / unhinged status message based on action
func GetRandomTrollPhrase(toolName string) string {
	switch toolName {
	case "bash", "run_command":
		return trollBashPhrases[rand.Intn(len(trollBashPhrases))]
	case "edit_file", "replace_file_content", "write_to_file":
		return trollEditPhrases[rand.Intn(len(trollEditPhrases))]
	default:
		return trollThinkingPhrases[rand.Intn(len(trollThinkingPhrases))]
	}
}
