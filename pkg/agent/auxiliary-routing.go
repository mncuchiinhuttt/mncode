package agent

import (
	"fmt"
	"mncode/pkg/config"
	"strings"
)

// AuxiliaryPurpose identifies a model call that is not the primary conversation.
type AuxiliaryPurpose string

const (
	AuxiliarySummary AuxiliaryPurpose = "summary"
	AuxiliaryVision  AuxiliaryPurpose = "vision"
	AuxiliarySearch  AuxiliaryPurpose = "search"

	// AuxiliarySummarization is a descriptive alias for callers.
	AuxiliarySummarization = AuxiliarySummary
)

// AuxiliaryRouteSource describes the precedence level that selected a route.
type AuxiliaryRouteSource string

const (
	AuxiliaryPurposeOverride AuxiliaryRouteSource = "purpose_override"
	AuxiliaryGlobalOverride  AuxiliaryRouteSource = "global_override"
	AuxiliaryPrimary         AuxiliaryRouteSource = "primary"
)

// AuxiliaryRoute is immutable routing metadata for one auxiliary call. It is
// deliberately separate from Config: resolving a route must never rewrite the
// primary provider or model.
type AuxiliaryRoute struct {
	Purpose  AuxiliaryPurpose     `json:"purpose"`
	Provider config.ProviderType  `json:"provider"`
	Model    string               `json:"model"`
	Source   AuxiliaryRouteSource `json:"source"`
	Fallback bool                 `json:"fallback"`
}

// ModelAvailability reports whether a provider/model pair can be used. A nil
// checker applies the conservative built-in check (known provider and non-empty
// model) without making network calls.
type ModelAvailability func(provider config.ProviderType, model string) bool

// AuxiliaryRouteResolver resolves purpose-specific routes without mutating the
// supplied configuration. IsAvailable is optional and useful when the caller
// has a local capability/catalog view; it must be deterministic.
type AuxiliaryRouteResolver struct {
	IsAvailable ModelAvailability
}

// ResolveAuxiliaryRoute resolves a route from the session's primary config.
// It is a convenience boundary for auxiliary callsites and does not mutate the
// session or its config.
func (s *Session) ResolveAuxiliaryRoute(purpose AuxiliaryPurpose, availability ...ModelAvailability) (AuxiliaryRoute, error) {
	if s == nil {
		return AuxiliaryRoute{}, fmt.Errorf("cannot resolve %s route without session", purpose)
	}
	return ResolveAuxiliaryRoute(s.Config, purpose, availability...)
}

// NewAuxiliaryRouteResolver creates a resolver with an optional availability
// predicate. The predicate is only consulted locally; this type never probes a
// provider or changes credentials.
func NewAuxiliaryRouteResolver(available ModelAvailability) AuxiliaryRouteResolver {
	return AuxiliaryRouteResolver{IsAvailable: available}
}

// ResolveAuxiliaryRoute resolves an auxiliary route. The optional availability
// argument exists so callers can use the package-level helper without building a
// resolver. Precedence is purpose override, global override, then the primary
// provider/model. An unavailable override is skipped deterministically.
func ResolveAuxiliaryRoute(cfg *config.Config, purpose AuxiliaryPurpose, availability ...ModelAvailability) (AuxiliaryRoute, error) {
	var available ModelAvailability
	if len(availability) > 0 {
		available = availability[0]
	}
	return (AuxiliaryRouteResolver{IsAvailable: available}).Resolve(cfg, purpose)
}

// Resolve selects the first available candidate in deterministic order.
func (r AuxiliaryRouteResolver) Resolve(cfg *config.Config, purpose AuxiliaryPurpose) (AuxiliaryRoute, error) {
	if cfg == nil {
		return AuxiliaryRoute{}, fmt.Errorf("cannot resolve %s route without config", purpose)
	}
	purpose = normalizeAuxiliaryPurpose(purpose)
	if purpose == "" {
		return AuxiliaryRoute{}, fmt.Errorf("unsupported auxiliary purpose")
	}

	primary := routeCandidate{provider: strings.ToLower(strings.TrimSpace(string(cfg.Provider))), model: strings.TrimSpace(cfg.Model), source: AuxiliaryPrimary}
	if primary.provider == "" || primary.model == "" {
		return AuxiliaryRoute{}, fmt.Errorf("primary provider/model is incomplete")
	}

	for _, candidate := range auxiliaryCandidates(cfg, purpose, primary) {
		providerName := config.ProviderType(strings.ToLower(strings.TrimSpace(candidate.provider)))
		model := strings.TrimSpace(candidate.model)
		if !isAvailable(cfg, r.IsAvailable, providerName, model) {
			continue
		}
		return AuxiliaryRoute{
			Purpose:  purpose,
			Provider: providerName,
			Model:    model,
			Source:   candidate.source,
			Fallback: candidate.source == AuxiliaryPrimary,
		}, nil
	}
	return AuxiliaryRoute{}, fmt.Errorf("no available %s route (primary %s/%s)", purpose, primary.provider, primary.model)
}

type routeCandidate struct {
	provider string
	model    string
	source   AuxiliaryRouteSource
}

func auxiliaryCandidates(cfg *config.Config, purpose AuxiliaryPurpose, primary routeCandidate) []routeCandidate {
	purposeProvider := firstSetting(cfg,
		"auxiliary_"+string(purpose)+"_provider",
		"auxiliary."+string(purpose)+".provider",
	)
	purposeModel := firstSetting(cfg,
		"auxiliary_"+string(purpose)+"_model",
		"auxiliary."+string(purpose)+".model",
	)
	globalProvider := firstSetting(cfg, "auxiliary_provider", "auxiliary.provider")
	globalModel := firstSetting(cfg, "auxiliary_model", "auxiliary.model")

	candidates := make([]routeCandidate, 0, 3)
	// Provider and model overrides are independent: a caller may override only
	// one dimension while retaining the primary value for the other.
	if purposeProvider != "" || purposeModel != "" {
		candidates = append(candidates, routeCandidate{
			provider: valueOr(purposeProvider, primary.provider),
			model:    valueOr(purposeModel, primary.model),
			source:   AuxiliaryPurposeOverride,
		})
	}
	if globalProvider != "" || globalModel != "" {
		candidates = append(candidates, routeCandidate{
			provider: valueOr(globalProvider, primary.provider),
			model:    valueOr(globalModel, primary.model),
			source:   AuxiliaryGlobalOverride,
		})
	}
	candidates = append(candidates, primary)
	return candidates
}

func firstSetting(cfg *config.Config, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(cfg.GetSetting(key, "")); value != "" {
			return value
		}
	}
	return ""
}

func valueOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func normalizeAuxiliaryPurpose(purpose AuxiliaryPurpose) AuxiliaryPurpose {
	switch strings.ToLower(strings.TrimSpace(string(purpose))) {
	case "summary", "summarize", "summarisation", "summarization":
		return AuxiliarySummary
	case "vision", "image":
		return AuxiliaryVision
	case "search", "web-search", "web_search":
		return AuxiliarySearch
	default:
		return ""
	}
}

func isAvailable(cfg *config.Config, checker ModelAvailability, providerName config.ProviderType, model string) bool {
	if model == "" || !knownProvider(providerName) {
		return false
	}
	if providerName == config.ProviderCustom && cfg.CustomProviderID == "" {
		return false
	}
	if checker != nil {
		return checker(providerName, model)
	}
	return true
}

func knownProvider(providerName config.ProviderType) bool {
	switch providerName {
	case config.ProviderAnthropic, config.ProviderOpenAI, config.ProviderGemini,
		config.ProviderOpenRouter, config.ProviderOpenCode, config.ProviderAntigravity,
		config.ProviderCustom:
		return true
	default:
		return false
	}
}
