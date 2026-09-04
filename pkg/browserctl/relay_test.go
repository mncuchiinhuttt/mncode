package browserctl

import (
	"strings"
	"testing"
)

func TestMaskSensitiveDOM(t *testing.T) {
	html := `
<form>
  <input type="text" name="user" value="alice" />
  <input type="password" name="pass" value="SuperSecretPassword123" />
  <span>Credit card: 4111 2222 3333 4444</span>
  <p>Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xyz.abc</p>
</form>
`
	masked := MaskSensitiveDOM(html)

	if strings.Contains(masked, "SuperSecretPassword123") {
		t.Fatalf("failed to mask password in DOM:\n%s", masked)
	}
	if !strings.Contains(masked, "[REDACTED_PASSWORD]") {
		t.Errorf("missing password redaction marker")
	}
	if strings.Contains(masked, "4111 2222 3333 4444") {
		t.Fatalf("failed to mask credit card in DOM")
	}
	if !strings.Contains(masked, "[REDACTED_CREDIT_CARD]") {
		t.Errorf("missing credit card redaction marker")
	}
	if strings.Contains(masked, "eyJhbGciOi") {
		t.Fatalf("failed to mask bearer token in DOM")
	}
}

func TestIsConsequentialAction(t *testing.T) {
	cases := []struct {
		action   string
		target   string
		expected bool
	}{
		{"click", "button#submit-payment", true},
		{"click", "a.checkout-btn", true},
		{"click", "button.delete-database", true},
		{"click", "button.nav-home", false},
		{"type", "input.search", false},
	}

	for _, c := range cases {
		got := IsConsequentialAction(c.action, c.target)
		if got != c.expected {
			t.Errorf("IsConsequentialAction(%q, %q) = %v, want %v", c.action, c.target, got, c.expected)
		}
	}
}

func TestFindTargetTab(t *testing.T) {
	tabs := []RelayTab{
		{ID: "1", Title: "Local Dev Dashboard", URL: "http://localhost:3000"},
		{ID: "2", Title: "GitHub Pull Requests", URL: "https://github.com/org/repo/pulls"},
	}

	match1, err := FindTargetTab(tabs, "localhost")
	if err != nil || match1.ID != "1" {
		t.Fatalf("expected tab 1, got %v, err=%v", match1, err)
	}

	match2, err := FindTargetTab(tabs, "GitHub")
	if err != nil || match2.ID != "2" {
		t.Fatalf("expected tab 2, got %v, err=%v", match2, err)
	}

	_, err = FindTargetTab(tabs, "nonexistent-domain.xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent target, got nil")
	}
}
