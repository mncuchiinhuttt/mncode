package spec

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContractRoundTripAndMatrix(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := store.NewContract(context.Background(), "auth-contract", "Auth behavior")
	if err != nil {
		t.Fatal(err)
	}
	contract.Cases = append(contract.Cases,
		Case{ID: "file", Name: "source exists", Kind: "file_exists", Input: json.RawMessage(`{"path":"go.mod"}`), Expected: json.RawMessage(`true`)},
		Case{ID: "missing", Name: "missing file is expected", Kind: "file_exists", Input: json.RawMessage(`{"path":"missing.go"}`), Expected: json.RawMessage(`false`)},
	)
	if err := store.Save(context.Background(), contract); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), "auth-contract")
	if err != nil {
		t.Fatal(err)
	}
	matrix, err := store.Check(context.Background(), loaded)
	if err != nil {
		t.Fatal(err)
	}
	if matrix.Passed != 3 || matrix.Invalid != 0 {
		t.Fatalf("unexpected matrix: %+v", matrix)
	}
}

func TestRejectsUnknownKindAndSecret(t *testing.T) {
	contract := Contract{SchemaVersion: 1, ID: "x", Title: "x", Version: 1, Cases: []Case{{ID: "bad", Name: "bad", Kind: "shell", Input: json.RawMessage(`{}`)}}}
	if err := Validate(contract); err == nil {
		t.Fatal("expected unknown kind rejection")
	}
	contract.Cases[0].Kind = "invariant"
	contract.Cases[0].Input = json.RawMessage(`{"operator":"non_empty","value":"AIzaSy1234567890abcdef1234567890"}`)
	if err := Validate(contract); err == nil {
		t.Fatal("expected Gemini key rejection")
	}
	contract.Cases[0].Input = json.RawMessage(`{"operator":"non_empty","value":"sk-proj-12345678901234567890123456789012"}`)
	if err := Validate(contract); err == nil {
		t.Fatal("expected secret rejection")
	}
}

func TestSourceFileUnchangedAfterCommandCase(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "go.mod")
	if err := os.WriteFile(source, []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	contract := Contract{SchemaVersion: 1, ID: "copy", Title: "copy", Version: 1, Cases: []Case{{ID: "echo", Name: "echo", Kind: "command", Command: []string{"printf", "ok"}, Expected: json.RawMessage(`{"stdout_contains":"ok"}`)}}}
	if _, err := store.Check(context.Background(), contract); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(source)
	absoluteCommand := []string{"python3", filepath.Join(root, "victim")}
	contract.Cases = append(contract.Cases, Case{ID: "escape", Name: "reject source path", Kind: "command", Command: absoluteCommand})
	matrix, err := store.Check(context.Background(), contract)
	if err != nil || matrix.Failed != 1 {
		t.Fatalf("expected source-path rejection: %+v %v", matrix, err)
	}
	if _, err := os.Stat(filepath.Join(root, "victim")); !os.IsNotExist(err) {
		t.Fatal("command reached the source workspace")
	}
	if string(data) != "module test\n" {
		t.Fatal("source changed")
	}
}
