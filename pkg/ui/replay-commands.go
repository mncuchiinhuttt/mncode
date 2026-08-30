package ui

import (
	"context"
	"fmt"
	"strings"

	"mncode/pkg/agent"
	"mncode/pkg/replay"
)

// HandleReplayCommand controls explicit flight recording and trace inspection.
func HandleReplayCommand(parts []string, session *agent.Session) {
	if session == nil || strings.TrimSpace(session.WorkspaceDir) == "" {
		fmt.Println(BoldRed("[Replay] Workspace is required."))
		return
	}
	store, err := replay.NewStore(session.WorkspaceDir)
	if err != nil {
		fmt.Println(BoldRed("[Replay] " + err.Error()))
		return
	}
	sub := ""
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}
	if err := validateReplayArgs(sub, parts); err != nil {
		fmt.Println(BoldRed("[Replay] " + err.Error()))
		return
	}
	switch sub {
	case "start", "record":
		if session.RecorderSnapshot() != nil {
			fmt.Println(BoldYellow("[Replay] Recording is already active."))
			return
		}
		session.EnsureIdentity()
		meta := replay.Trace{}
		if session.Config != nil {
			meta.Model, meta.Provider = session.Config.Model, string(session.Config.Provider)
		}
		recorder, startErr := store.Start(context.Background(), session.ID, meta)
		if startErr != nil {
			fmt.Println(BoldRed("[Replay] " + startErr.Error()))
			return
		}
		session.SetRecorder(recorder)
		fmt.Printf("\n%s Recording trace %s. Stop with /replay stop.\n\n", BoldGreen("[Replay OK]"), recorder.ID())
	case "stop":
		recorder := session.RecorderSnapshot()
		closer, ok := recorder.(interface{ Close(bool) error })
		if !ok {
			fmt.Println(BoldYellow("[Replay] No active recorder."))
			return
		}
		if closeErr := closer.Close(true); closeErr != nil {
			fmt.Println(BoldRed("[Replay] " + closeErr.Error()))
			return
		}
		session.DetachRecorder()
		fmt.Println(BoldGreen("[Replay OK] Recording finalized."))
	case "list", "":
		traces, listErr := store.List(context.Background())
		if listErr != nil {
			fmt.Println(BoldRed("[Replay] " + listErr.Error()))
			return
		}
		if len(traces) == 0 {
			fmt.Println(GrayText("\n[Replay] No traces. Use /replay start.\n"))
			return
		}
		fmt.Println("\n" + BoldCyan("REPLAY TRACES:"))
		for _, trace := range traces {
			fmt.Printf("  %-34s events=%-4d complete=%t %s\n", trace.ID, trace.Events, trace.Complete, trace.StartedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	case "show":
		id := replayIDArg(parts)
		trace, events, loadErr := store.Load(context.Background(), id)
		if loadErr != nil {
			fmt.Println(BoldRed("[Replay] " + loadErr.Error()))
			return
		}
		fmt.Printf("\n%s %s events=%d complete=%t\n", BoldCyan("[Replay Trace]"), trace.ID, len(events), trace.Complete)
		for _, event := range events {
			fmt.Printf("  #%d turn=%d %-18s payload=%d bytes\n", event.Seq, event.Turn, event.Kind, len(event.Data))
		}
		fmt.Println()
	case "export":
		id := replayIDArg(parts)
		destination := ""
		if len(parts) > 3 {
			destination = parts[3]
		}
		path, exportErr := store.Export(context.Background(), id, destination)
		if exportErr != nil {
			fmt.Println(BoldRed("[Replay] " + exportErr.Error()))
			return
		}
		fmt.Printf("%s Exported: %s\n", BoldGreen("[Replay OK]"), path)
	case "delete", "rm":
		id := replayIDArg(parts)
		if !confirmCommand(session, "delete_replay_trace", id) {
			return
		}
		if deleteErr := store.Delete(context.Background(), id, true); deleteErr != nil {
			fmt.Println(BoldRed("[Replay] " + deleteErr.Error()))
			return
		}
		fmt.Println(BoldGreen("[Replay OK] Trace deleted."))
	case "fork":
		forkParts := append([]string{"/fork"}, parts[2:]...)
		HandleForkCommand(forkParts, session)
	default:
		fmt.Println("\n" + BoldCyan("REPLAY COMMANDS:"))
		fmt.Println("  /replay start|stop               - start or finalize recorder")
		fmt.Println("  /replay list|show <trace>        - inspect bounded timeline")
		fmt.Println("  /replay export <trace> [path]    - export redacted trace")
		fmt.Println("  /replay delete <trace>           - delete trace (approval)")
		fmt.Println("  /fork <trace> [--at N]           - fork conversation context")
	}
}

// HandleForkCommand reconstructs a trace prefix and switches active session after approval.
func HandleForkCommand(parts []string, session *agent.Session) {
	if session == nil || len(parts) < 2 {
		fmt.Println(BoldYellow("Usage: /fork <trace-id> [--at N] [--name NAME] [--no-tools]"))
		return
	}
	traceID, at, name, _, err := parseForkArgs(parts[1:])
	if err != nil {
		fmt.Println(BoldRed("[Fork] " + err.Error()))
		return
	}
	store, err := replay.NewStore(session.WorkspaceDir)
	if err != nil {
		fmt.Println(BoldRed("[Fork] " + err.Error()))
		return
	}
	result, err := store.Fork(context.Background(), replay.ForkRequest{TraceID: traceID, At: at, Name: name, ReplayTools: false})
	if err != nil {
		fmt.Println(BoldRed("[Fork] " + err.Error()))
		return
	}
	if !confirmCommand(session, "activate_replay_fork", result.SessionID) {
		return
	}
	if err := session.ActivateFork(result.History, result.SessionID); err != nil {
		fmt.Println(BoldRed("[Fork] " + err.Error()))
		return
	}
	if name != "" {
		fmt.Printf("%s Active session forked as %q from %s at event %d (%d messages).\n", BoldGreen("[Fork OK]"), name, result.ParentTraceID, result.At, len(result.History))
	} else {
		fmt.Printf("%s Active session forked from %s at event %d (%d messages).\n", BoldGreen("[Fork OK]"), result.ParentTraceID, result.At, len(result.History))
	}
}
