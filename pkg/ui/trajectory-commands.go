package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"strings"
)

// HandleTrajectoryExportCommand exports the current session as ShareGPT JSON.
func HandleTrajectoryExportCommand(parts []string, s *agent.Session) {
	destination := ""
	if len(parts) > 1 {
		if !strings.EqualFold(parts[1], "sharegpt") && !strings.EqualFold(parts[1], "trajectory") {
			destination = parts[1]
		} else if len(parts) > 2 {
			destination = parts[2]
		}
	}
	path, err := agent.ExportShareGPTFile(s, destination)
	if err != nil {
		fmt.Printf("\n%s %v\n\n", BoldRed("[Export Error]"), err)
		return
	}
	fmt.Printf("\n%s ShareGPT trajectory exported.\n", BoldGreen("[OK]"))
	fmt.Printf(" %s %s\n\n", GrayText("File:"), Bold(path))
}
