package ui

import (
	"fmt"
	"math/rand"
	"mncode/pkg/agent"
	"strings"
)

var (
	brainrotExitMessagesEN = []string{
		"Aura +10000 for cooking today. Stay sigma, king [GIGA][KING]",
		"Logging off with maximum rizz. No bugs survived today fr fr [MAX]",
		"Mewing streak preserved. See you next commit, chief [MEW]",
		"We cooked and we ate. Catch you on the next push boss [PEACE][COOK]",
		"Aura levels maxed out. Terminal safely disengaged [LAUNCH][ACTION]",
		"Skibidi Ohio level 10x developer detected. Terminal disengaged [GIGA][MAX]",
		"Code pushed smoother than butter with zero cap. See ya king! [KING][PEACE]",
		"Unbounded cognitive depth achieved. Take a break, champ [SHINE]",
	}

	brainrotExitMessagesVN = []string{
		"Aura +10000 cho buổi code hôm nay. Mãi đỉnh, mãi sigma ní ơi [GIGA][KING]",
		"Nghỉ ngơi thôi ní ơi, nay cook căng quá aura tăng chóng mặt fr fr [GIGA][MAX]",
		"Đã push code mượt như bơ, không một chút cap. See ya king! [KING][PEACE]",
		"Skibidi toilet cũng phải nể độ 10x dev của ní hôm nay [THINK][SHINE]",
		"Giữ vững chuỗi mewing và hẹn gặp lại ở commit sau nha ní [MEW]",
	}

	standardExitMessagesEN = []string{
		"[SHINE] Session finished. Happy coding and see you next time!",
		"[ACTION] mncode disengaged. Keep building amazing things!",
		"[LAUNCH] Work saved. Until next session, happy hacking!",
	}

	standardExitMessagesVN = []string{
		"[SHINE] Phiên làm việc đã kết thúc. Chúc bạn code vui vẻ và hẹn gặp lại!",
		"[ACTION] mncode đã ngắt kết nối an toàn. Tiếp tục tạo nên những điều tuyệt vời nhé!",
		"[LAUNCH] Đã lưu lại toàn bộ công việc. Hẹn gặp lại bạn ở phiên tiếp theo!",
	}
)

// PrintRizzGoodbye prints a stylish exit message strictly respecting the configured language
func PrintRizzGoodbye(s *agent.Session) {
	isBrainrot := false
	isVN := false

	if s != nil && s.Config != nil {
		if s.Config.GetSetting("brainrot_mode", "false") == "true" {
			isBrainrot = true
		}
		lang := strings.ToLower(s.Config.GetSetting("language", "Default (English)"))
		if strings.Contains(lang, "vietnam") || strings.Contains(lang, "vi") {
			isVN = true
		}
	}

	if isBrainrot {
		pool := brainrotExitMessagesEN
		if isVN {
			pool = brainrotExitMessagesVN
		}
		idx := rand.Intn(len(pool))
		fmt.Printf("\n\033[1;38;5;218m[mncode]\033[0m \033[1;38;5;225m%s\033[0m\n\n", pool[idx])
	} else {
		pool := standardExitMessagesEN
		if isVN {
			pool = standardExitMessagesVN
		}
		idx := rand.Intn(len(pool))
		fmt.Printf("\n\033[1;36m[mncode]\033[0m %s\n\n", BoldGreen(pool[idx]))
	}
}
