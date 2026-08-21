package ui

import (
	"fmt"
	"math/rand"
	"mncode/pkg/agent"
)

var (
	brainrotExitMessages = []string{
		"Aura +1000 for cooking today. Stay sigma, king 🗿👑",
		"Logging off with maximum rizz. No bugs survived today fr fr 🔥",
		"Mewing streak preserved. See you next commit, chief 🤫🧏",
		"We cooked and we ate. Catch you on the next push boss ✌️🍳",
		"Aura levels maxed out. Terminal safely disengaged 🚀⚡",
	}

	standardExitMessages = []string{
		"✨ Session finished. Happy coding and see you next time!",
		"⚡ mncode disengaged. Keep building amazing things!",
		"🚀 Work saved. Until next session, happy hacking!",
	}
)

// PrintRizzGoodbye prints a stylish, rizz-filled exit message based on mode
func PrintRizzGoodbye(s *agent.Session) {
	isBrainrot := false
	if s != nil && s.Config != nil && s.Config.GetSetting("brainrot_mode", "false") == "true" {
		isBrainrot = true
	}

	if isBrainrot {
		idx := rand.Intn(len(brainrotExitMessages))
		fmt.Printf("\n\033[1;38;5;218m[mncode]\033[0m \033[1;38;5;225m%s\033[0m\n\n", brainrotExitMessages[idx])
	} else {
		idx := rand.Intn(len(standardExitMessages))
		fmt.Printf("\n\033[1;36m[mncode]\033[0m %s\n\n", BoldGreen(standardExitMessages[idx]))
	}
}
