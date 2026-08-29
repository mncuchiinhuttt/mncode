package tools

import (
	"encoding/json"
	"strings"
)

type groundedPart struct {
	Text string `json:"text"`
}

type groundedWeb struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
}

type groundedChunk struct {
	Web groundedWeb `json:"web"`
}

type groundedCandidate struct {
	Content struct {
		Parts []groundedPart `json:"parts"`
	} `json:"content"`
	GroundingMetadata struct {
		GroundingChunks []groundedChunk `json:"groundingChunks"`
	} `json:"groundingMetadata"`
}

type groundedPayload struct {
	Candidates []groundedCandidate `json:"candidates"`
}

type groundedEnvelope struct {
	Response   *groundedPayload    `json:"response"`
	Candidates []groundedCandidate `json:"candidates"`
}

func parseGroundingResponse(data []byte) []searchResult {
	payloads := make([]groundedPayload, 0, 2)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "" || line == "[DONE]" {
			continue
		}
		var envelope groundedEnvelope
		if json.Unmarshal([]byte(line), &envelope) != nil {
			continue
		}
		if envelope.Response != nil {
			payloads = append(payloads, *envelope.Response)
		} else if len(envelope.Candidates) > 0 {
			payloads = append(payloads, groundedPayload{Candidates: envelope.Candidates})
		}
	}
	if len(payloads) == 0 {
		var envelope groundedEnvelope
		if json.Unmarshal(data, &envelope) == nil {
			if envelope.Response != nil {
				payloads = append(payloads, *envelope.Response)
			} else if len(envelope.Candidates) > 0 {
				payloads = append(payloads, groundedPayload{Candidates: envelope.Candidates})
			}
		}
	}

	var summary strings.Builder
	results := make([]searchResult, 0)
	seen := make(map[string]struct{})
	for _, payload := range payloads {
		for _, candidate := range payload.Candidates {
			for _, part := range candidate.Content.Parts {
				if text := cleanSearchText(part.Text); text != "" {
					if summary.Len() > 0 {
						summary.WriteString("\n")
					}
					summary.WriteString(text)
				}
			}
			for _, chunk := range candidate.GroundingMetadata.GroundingChunks {
				uri := strings.TrimSpace(chunk.Web.URI)
				if uri == "" {
					continue
				}
				if _, ok := seen[uri]; ok {
					continue
				}
				seen[uri] = struct{}{}
				title := truncateSearchText(cleanSearchText(chunk.Web.Title), 500)
				if title == "" {
					title = uri
				}
				results = append(results, searchResult{
					Title:   title,
					URL:     uri,
					Snippet: "Source cited by Google Search Grounding",
					Source:  "Google Grounding",
				})
			}
		}
	}
	if summaryText := strings.TrimSpace(summary.String()); summaryText != "" {
		results = append([]searchResult{{Title: "Google grounded answer", Snippet: truncateSearchText(summaryText, maxSearchAnswerLength), Source: "Google Search Grounding"}}, results...)
	}
	return results
}
