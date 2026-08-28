package ui

import (
	"testing"

	"mncode/pkg/agent"
	"mncode/pkg/config"
)

func TestPermissionSettingsIncludePlanMode(t *testing.T) {
	var found bool
	for _, definition := range GetAllSettingsDefinitions() {
		if definition.Key != "permission_mode" {
			continue
		}
		found = true
		planFound := false
		for _, choice := range definition.Choices {
			if choice == "Plan mode" {
				planFound = true
			}
		}
		if !planFound {
			t.Fatal("permission_mode choices do not include Plan mode")
		}
	}
	if !found {
		t.Fatal("permission_mode setting definition is missing")
	}

	session := &agent.Session{Config: &config.Config{PermissionMode: config.PermissionModePlan}}
	if got, want := GetSettingValue(session, SettingDef{Key: "permission_mode", DefaultVal: "Manual (Ask)"}), "Plan mode"; got != want {
		t.Fatalf("GetSettingValue() = %q, want %q", got, want)
	}
}
