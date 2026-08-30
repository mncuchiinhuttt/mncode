package ui

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewCommandRouterAndRename(t *testing.T) {
	shadow := captureCommandOutput(func() {
		if !HandleSlashCommand("/shadow", nil) {
			t.Fatal("shadow command not handled")
		}
	})
	if !strings.Contains(shadow, "renamed to /sandbox") {
		t.Fatalf("unexpected shadow output: %q", shadow)
	}
	arena := captureCommandOutput(func() { HandleSlashCommand("/arena", nil) })
	debate := captureCommandOutput(func() { HandleSlashCommand("/debate", nil) })
	if !strings.Contains(arena, "[Arena]") || strings.Contains(arena, "Debate Arena") {
		t.Fatalf("arena routed incorrectly: %q", arena)
	}
	if !strings.Contains(debate, "/debate") {
		t.Fatalf("debate route changed: %q", debate)
	}
	if ResolveNumericSlashCommand("/53") != "/drift" || ResolveNumericSlashCommand("/59") != "/spec" {
		t.Fatal("new palette entries not resolvable")
	}
}

func TestAdvancedFlagParsing(t *testing.T) {
	sub, query, kind, pathGlob, limit, asJSON, err := parseIndexArgs([]string{"query", "token", "validation", "--kind", "func", "--path", "pkg/**", "--limit", "5", "--json"})
	if err != nil || sub != "query" || query != "token validation" || kind != "func" || pathGlob != "pkg/**" || limit != 5 || !asJSON {
		t.Fatalf("index parse: %q %q %q %q %d %t %v", sub, query, kind, pathGlob, limit, asJSON, err)
	}
	fixture, extra, keep, err := parseSandboxRun([]string{"default", "--keep", "--", "--verbose"})
	if err != nil || fixture != "default" || len(extra) != 1 || extra[0] != "--verbose" || !keep {
		t.Fatalf("sandbox parse: %q %#v %t %v", fixture, extra, keep, err)
	}
	if _, _, _, _, _, _, err := parseArenaArgs([]string{"--rounds", "4"}); err == nil {
		t.Fatal("expected arena rounds limit")
	}
	trace, at, name, noTools, err := parseForkArgs([]string{"trace-id", "--at", "4", "--name", "alternate", "--no-tools"})
	if err != nil || trace != "trace-id" || at != 4 || name != "alternate" || !noTools {
		t.Fatalf("fork parse: %q %d %q %t %v", trace, at, name, noTools, err)
	}
	if validateReplayArgs("show", []string{"/replay", "show", "trace-id", "extra"}) == nil {
		t.Fatal("expected replay extra argument rejection")
	}
	if validateSpecArgs("check", []string{"check", "spec", "--bogus"}) == nil {
		t.Fatal("expected spec extra argument rejection")
	}
}

func captureCommandOutput(fn func()) string {
	original := os.Stdout
	reader, writer, _ := os.Pipe()
	os.Stdout = writer
	fn()
	_ = writer.Close()
	os.Stdout = original
	data, _ := io.ReadAll(reader)
	_ = reader.Close()
	return string(data)
}
