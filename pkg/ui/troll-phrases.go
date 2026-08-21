package ui

import (
	"math/rand"
	"time"
)

var trollThinkingPhrases = []string{
	"Straight up yapping to the LLM fr fr...",
	"Cooking up unhinged code with zero cap...",
	"Gaslighting the unit tests into passing...",
	"Rizzing the terminal into submission...",
	"Doing shady backend voodoo behind your back...",
	"Consulting the holy spaghetti codebase...",
	"Bluffing the compiler (it's totally working)...",
	"Manifesting a 200 OK out of thin air...",
	"Pretending this code works on first try...",
	"Inventing bugs to fix later for job security...",
	"Brainrot optimization in full swing no cap...",
	"Blatantly faking the performance benchmarks...",
	"Stealing GPU cycles from the mainframe...",
	"Hypnotizing the CPU cores into overdrive...",
	"Sweating over pointer arithmetic (trust the process)...",
	"Summoning dark mode demons in bash...",
	"Flexing on null pointer exceptions...",
	"Pure main character energy: cooking solutions...",
	"Letting the agent cook, do not disturb...",
	"Whispering sweet nothings to the linter...",
}

var trollBashPhrases = []string{
	"Running bash roulette (hope nothing explodes)...",
	"Blasting commands into the shell abyss...",
	"Executing questionable terminal commands with confidence...",
	"Hacking the mainframe in bash like a 90s cyber movie...",
}

var trollEditPhrases = []string{
	"Refactoring spaghetti into slightly straighter spaghetti...",
	"Dropping hot fresh syntax into the codebase...",
	"Infiltrating the source code with zero regrets...",
	"Applying tactical code surgery (no anesthesia)...",
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
