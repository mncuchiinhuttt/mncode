package codex

import (
	"encoding/json"
	"testing"
)

func TestProtocolVersionPinned(t *testing.T) {
	if ProtocolVersion != "2024-11-05" {
		t.Fatalf("unexpected ProtocolVersion %s", ProtocolVersion)
	}
}

func TestRequestSerialization(t *testing.T) {
	params, _ := json.Marshal(LoginStartParams{Type: "chatgpt"})
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "account/login/start",
		Params:  params,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Request
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Method != "account/login/start" || parsed.ID != 1 {
		t.Fatalf("unexpected request: %+v", parsed)
	}
}

func TestResponseParsing(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":1,"result":{"type":"chatgpt","authUrl":"https://auth.openai.com/test"}}`
	var resp Response
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.ID != 1 {
		t.Fatalf("expected ID 1, got %d", resp.ID)
	}

	var res LoginStartResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		t.Fatal(err)
	}

	if res.AuthURL != "https://auth.openai.com/test" {
		t.Fatalf("unexpected authUrl: %s", res.AuthURL)
	}
}
