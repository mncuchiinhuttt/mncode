package agent

import (
	"mncode/pkg/config"
	"reflect"
	"testing"
)

func TestResolveAuxiliaryRoutePurposeOverridesGlobalAndPrimary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderAnthropic
	cfg.Model = "primary-model"
	cfg.Settings = map[string]string{
		"auxiliary_provider":         "openai",
		"auxiliary_model":            "global-model",
		"auxiliary_summary_provider": "gemini",
		"auxiliary_summary_model":    "purpose-model",
		"auxiliary_vision_provider":  "openrouter",
		"auxiliary_vision_model":     "vision-model",
		"auxiliary_search_provider":  "anthropic",
		"auxiliary_search_model":     "search-model",
	}
	before := *cfg
	beforeSettings := map[string]string{}
	for key, value := range cfg.Settings {
		beforeSettings[key] = value
	}

	for _, test := range []struct {
		purpose  AuxiliaryPurpose
		provider config.ProviderType
		model    string
	}{
		{AuxiliarySummary, config.ProviderGemini, "purpose-model"},
		{AuxiliaryVision, config.ProviderOpenRouter, "vision-model"},
		{AuxiliarySearch, config.ProviderAnthropic, "search-model"},
	} {
		route, err := ResolveAuxiliaryRoute(cfg, test.purpose)
		if err != nil {
			t.Fatalf("ResolveAuxiliaryRoute(%q): %v", test.purpose, err)
		}
		if route.Purpose != test.purpose || route.Provider != test.provider || route.Model != test.model {
			t.Fatalf("route = %#v, want %q/%q", route, test.provider, test.model)
		}
		if route.Source != AuxiliaryPurposeOverride || route.Fallback {
			t.Fatalf("route metadata = %#v, want purpose override", route)
		}
	}
	if !reflect.DeepEqual(cfg.Settings, beforeSettings) || cfg.Provider != before.Provider || cfg.Model != before.Model {
		t.Fatal("routing mutated primary configuration")
	}
}

func TestResolveAuxiliaryRouteFallsBackThroughGlobalToPrimary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderAnthropic
	cfg.Model = "primary-model"
	cfg.Settings = map[string]string{
		"auxiliary_summary_provider": "not-a-provider",
		"auxiliary_summary_model":    "unavailable-purpose",
		"auxiliary_provider":         "openai",
		"auxiliary_model":            "global-model",
	}
	available := func(provider config.ProviderType, model string) bool {
		return provider == config.ProviderOpenAI && model == "global-model" ||
			provider == config.ProviderAnthropic && model == "primary-model"
	}

	route, err := ResolveAuxiliaryRoute(cfg, AuxiliarySummary, available)
	if err != nil {
		t.Fatalf("ResolveAuxiliaryRoute: %v", err)
	}
	if route.Provider != config.ProviderOpenAI || route.Model != "global-model" || route.Source != AuxiliaryGlobalOverride {
		t.Fatalf("route = %#v, want available global override", route)
	}

	cfg.Settings["auxiliary_model"] = "also-unavailable"
	route, err = ResolveAuxiliaryRoute(cfg, AuxiliarySummary, available)
	if err != nil {
		t.Fatalf("ResolveAuxiliaryRoute primary fallback: %v", err)
	}
	if route.Provider != config.ProviderAnthropic || route.Model != "primary-model" || route.Source != AuxiliaryPrimary || !route.Fallback {
		t.Fatalf("route = %#v, want primary fallback", route)
	}
}

func TestResolveAuxiliaryRouteRejectsUnavailablePrimary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderOpenAI
	cfg.Model = "primary-model"
	available := func(config.ProviderType, string) bool { return false }
	if _, err := ResolveAuxiliaryRoute(cfg, AuxiliaryVision, available); err == nil {
		t.Fatal("expected error when all configured models are unavailable")
	}
}

func TestResolveAuxiliaryRouteSupportsDottedSettingAliases(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderAnthropic
	cfg.Model = "primary-model"
	cfg.Settings = map[string]string{
		"auxiliary.vision.provider": "gemini",
		"auxiliary.vision.model":    "vision-model",
	}
	route, err := ResolveAuxiliaryRoute(cfg, AuxiliaryVision)
	if err != nil {
		t.Fatalf("ResolveAuxiliaryRoute: %v", err)
	}
	if route.Provider != config.ProviderGemini || route.Model != "vision-model" {
		t.Fatalf("route = %#v, want gemini/vision-model", route)
	}
}
