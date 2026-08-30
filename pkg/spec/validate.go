package spec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"mncode/pkg/artifacts"
	"mncode/pkg/sandbox"
)

var allowedCaseKinds = map[string]bool{"command": true, "file_exists": true, "file_contains": true, "invariant": true}

const (
	maxContractBytes = 512 * 1024
	maxCases         = 256
	maxInvariants    = 128
	maxTextBytes     = 16 * 1024
	maxCommandArg    = 4096
)

// Validate enforces the versioned, deterministic v1 contract grammar.
func Validate(contract Contract) error {
	encoded, err := json.Marshal(contract)
	if err != nil || len(encoded) > maxContractBytes {
		return errors.New("contract exceeds the 512KB persistence limit")
	}
	if contract.SchemaVersion != 1 || !validID(contract.ID) || strings.TrimSpace(contract.Title) == "" {
		return errors.New("contract requires schema_version 1, safe id, and title")
	}
	if contract.Version <= 0 || len(contract.Cases) > maxCases || len(contract.Invariants) > maxInvariants {
		return errors.New("contract version or item count is invalid")
	}
	if unsafeMetadataText(contract.Title) || unsafeMetadataText(contract.Description) {
		return errors.New("contract metadata contains a secret-like value or is too large")
	}
	seen := make(map[string]bool, len(contract.Invariants)+len(contract.Cases))
	for _, invariant := range contract.Invariants {
		if !validID(invariant.ID) || seen[invariant.ID] || strings.TrimSpace(invariant.Description) == "" || unsafeMetadataText(invariant.ID) || unsafeMetadataText(invariant.Description) {
			return fmt.Errorf("invalid invariant %q", invariant.ID)
		}
		seen[invariant.ID] = true
		if err := validJSON(invariant.Value); err != nil || hasSecret(invariant.Value) {
			return fmt.Errorf("invariant %s contains invalid or secret-like data", invariant.ID)
		}
	}
	for _, testCase := range contract.Cases {
		if !validID(testCase.ID) || seen[testCase.ID] || strings.TrimSpace(testCase.Name) == "" || !allowedCaseKinds[testCase.Kind] {
			return fmt.Errorf("invalid case %q", testCase.ID)
		}
		if unsafeText(testCase.ID) || unsafeText(testCase.Name) || unsafeText(strings.Join(testCase.Tags, " ")) {
			return fmt.Errorf("case %s metadata contains a secret-like value", testCase.ID)
		}
		seen[testCase.ID] = true
		if err := validJSON(testCase.Input); err != nil {
			return fmt.Errorf("case %s input: %w", testCase.ID, err)
		}
		if err := validJSON(testCase.Expected); err != nil {
			return fmt.Errorf("case %s expected: %w", testCase.ID, err)
		}
		if hasSecret(testCase.Input) || hasSecret(testCase.Expected) {
			return fmt.Errorf("case %s contains a secret-like value", testCase.ID)
		}
		switch testCase.Kind {
		case "command":
			for _, arg := range testCase.Command {
				if len(arg) > maxCommandArg || unsafeText(arg) {
					return fmt.Errorf("case %s command argument is unsafe", testCase.ID)
				}
			}
			if err := sandbox.ValidateCommand(testCase.Command); err != nil {
				return fmt.Errorf("case %s command: %w", testCase.ID, err)
			}
			if len(testCase.Expected) > 0 {
				var expected map[string]any
				if json.Unmarshal(testCase.Expected, &expected) != nil {
					return fmt.Errorf("case %s command expected must be an object", testCase.ID)
				}
			}
		case "file_exists", "file_contains":
			if len(testCase.Command) != 0 {
				return fmt.Errorf("case %s must not include command", testCase.ID)
			}
			if err := validateFileInput(testCase); err != nil {
				return err
			}
			if _, err := expectedBool(testCase.Expected, true); err != nil {
				return fmt.Errorf("case %s: %w", testCase.ID, err)
			}
		case "invariant":
			if err := validateInvariantInput(testCase); err != nil {
				return err
			}
			if _, err := expectedBool(testCase.Expected, true); err != nil {
				return fmt.Errorf("case %s: %w", testCase.ID, err)
			}
		}
	}
	return nil
}

func validID(value string) bool {
	if len(value) == 0 || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func validJSON(data json.RawMessage) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if len(data) > 256*1024 || !json.Valid(data) {
		return errors.New("must be valid JSON under 256KB")
	}
	return nil
}
func hasSecret(data json.RawMessage) bool {
	if len(data) == 0 {
		return false
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err == nil {
		if jsonContainsSecret(generic) {
			return true
		}
	}
	return unsafeText(string(data))
}

func jsonContainsSecret(value any) bool {
	switch v := value.(type) {
	case string:
		return unsafeText(v)
	case []any:
		for _, item := range v {
			if jsonContainsSecret(item) {
				return true
			}
		}
	case map[string]any:
		for k, val := range v {
			normK := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(k))
			for _, marker := range []string{"token", "cookie", "credential", "secret", "apikey", "password", "auth", "bearer", "privatekey"} {
				if strings.Contains(normK, marker) {
					return true
				}
			}
			if jsonContainsSecret(val) {
				return true
			}
		}
	}
	return false
}

func unsafeText(value string) bool {
	if len(value) > maxTextBytes || artifacts.ScrubSecrets(value) != value {
		return true
	}
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(value))
	for _, marker := range []string{"apikey", "authorization", "password", "secretkey", "accesstoken", "privatekey", "thoughtsignature", "credentials"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
func unsafeMetadataText(value string) bool {
	if len(value) > maxTextBytes || artifacts.ScrubSecrets(value) != value {
		return true
	}
	return false
}

func validateFileInput(testCase Case) error {
	var input struct {
		Path string `json:"path"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(testCase.Input, &input); err != nil || strings.TrimSpace(input.Path) == "" {
		return fmt.Errorf("case %s requires input.path", testCase.ID)
	}
	if testCase.Kind == "file_contains" && input.Text == "" {
		return fmt.Errorf("case %s requires input.text", testCase.ID)
	}
	return nil
}

func validateInvariantInput(testCase Case) error {
	var input struct {
		Operator string `json:"operator"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(testCase.Input, &input); err != nil {
		return fmt.Errorf("case %s invariant input must be an object", testCase.ID)
	}
	if input.Operator != "non_empty" && input.Operator != "path_within_workspace" {
		return fmt.Errorf("case %s uses unsupported invariant %q", testCase.ID, input.Operator)
	}
	if input.Value == "" || unsafeText(input.Value) {
		return fmt.Errorf("case %s invariant value is invalid", testCase.ID)
	}
	return nil
}
