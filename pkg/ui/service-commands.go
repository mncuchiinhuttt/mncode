package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/hub"
)

func handleServiceCommand(args string, s *agent.Session) {
	parts := strings.Fields(strings.TrimSpace(args))
	subcmd := ""
	if len(parts) > 0 {
		subcmd = strings.ToLower(parts[0])
	}

	mgr := hub.GlobalManager()

	switch subcmd {
	case "", "ps", "list":
		list := mgr.PS()
		if len(list) == 0 {
			fmt.Println("\n\033[2mNo active background services. Use '/service start <name> <cmd>' to launch.\033[0m")
			fmt.Println()
			return
		}
		fmt.Println("\n\033[1;36m=== Background Supervised Services ===\033[0m")
		fmt.Printf("\033[2m%-14s %-8s %-10s %-8s %s\033[0m\n", "NAME", "PID", "STATUS", "PORT", "COMMAND")
		fmt.Println(strings.Repeat("-", 60))
		for _, svc := range list {
			portStr := "-"
			if svc.ReadyPort > 0 {
				portStr = strconv.Itoa(svc.ReadyPort)
			}
			statusColor := "\033[32m"
			if svc.State == hub.StateStopped {
				statusColor = "\033[31m"
			}
			fmt.Printf("%-14s %-8d %s%-10s\033[0m %-8s %s\n", svc.Name, svc.PID, statusColor, svc.State, portStr, svc.Command)
		}
		fmt.Println()

	case "start":
		if len(parts) < 3 {
			fmt.Println("\033[33mUsage: /service start <name> <command> [args...]\033[0m")
			return
		}
		name := parts[1]
		cmd := parts[2]
		cmdArgs := parts[3:]
		cwd := "."
		if s != nil && s.WorkspaceDir != "" {
			cwd = s.WorkspaceDir
		}

		fmt.Printf("Starting background service '%s'...\n", name)
		spec := hub.ServiceSpec{
			Name:        name,
			Command:     cmd,
			Args:        cmdArgs,
			Cwd:         cwd,
			TimeoutSec:  10,
		}
		info, err := mgr.Start(context.Background(), spec)
		if err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m[OK] Service '%s' running with PID %d.\033[0m\n", info.Name, info.PID)
		}

	case "logs":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /service logs <name> [limit]\033[0m")
			return
		}
		name := parts[1]
		limit := 50
		if len(parts) > 2 {
			if l, err := strconv.Atoi(parts[2]); err == nil && l > 0 {
				limit = l
			}
		}
		lines, err := mgr.Logs(name, limit, "")
		if err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
			return
		}
		fmt.Printf("\n\033[1;36m=== Recent Logs for Service '%s' (%d lines) ===\033[0m\n", name, len(lines))
		for _, l := range lines {
			fmt.Println(l)
		}
		fmt.Println()

	case "stop", "kill":
		if len(parts) < 2 {
			fmt.Println("\033[33mUsage: /service stop <name>\033[0m")
			return
		}
		name := parts[1]
		if err := mgr.Stop(name); err != nil {
			fmt.Printf("\033[31m[Error] %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m[OK] Service '%s' stopped successfully.\033[0m\n", name)
		}

	default:
		fmt.Printf("\033[33mUnknown service action '%s'. Use ps, start, logs, stop.\033[0m\n", subcmd)
	}
}
