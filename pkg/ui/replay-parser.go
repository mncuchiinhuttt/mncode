package ui

import (
	"fmt"
	"strconv"
	"strings"
)

func parseForkArgs(args []string) (string, int64, string, bool, error) {
	if len(args) == 0 {
		return "", 0, "", false, fmt.Errorf("trace id is required")
	}
	id, at, name, noTools := args[0], int64(-1), "", false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--no-tools":
			noTools = true
		case "--at":
			if i+1 >= len(args) {
				return "", 0, "", false, fmt.Errorf("--at requires a sequence")
			}
			i++
			parsed, parseErr := strconv.ParseInt(args[i], 10, 64)
			if parseErr != nil || parsed < 1 {
				return "", 0, "", false, fmt.Errorf("invalid --at")
			}
			at = parsed
		case "--name":
			if i+1 >= len(args) {
				return "", 0, "", false, fmt.Errorf("--name requires a value")
			}
			i++
			name = strings.TrimSpace(args[i])
			if name == "" || len(name) > 80 || strings.ContainsAny(name, "\x00\n\r") {
				return "", 0, "", false, fmt.Errorf("invalid --name")
			}
		default:
			return "", 0, "", false, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return id, at, name, noTools, nil
}

func replayIDArg(parts []string) string {
	if len(parts) > 2 {
		return parts[2]
	}
	return ""
}
func validateReplayArgs(sub string, parts []string) error {
	switch sub {
	case "start", "record", "stop", "list", "":
		if len(parts) > 2 {
			return fmt.Errorf("%s does not accept extra arguments", sub)
		}
	case "show", "delete", "rm":
		if len(parts) != 3 {
			return fmt.Errorf("%s requires exactly one trace id", sub)
		}
	case "export":
		if len(parts) < 3 || len(parts) > 4 {
			return fmt.Errorf("export requires a trace id and optional destination")
		}
	}
	return nil
}
