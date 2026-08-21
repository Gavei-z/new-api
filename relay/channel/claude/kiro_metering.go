package claude

import (
	"math"

	"github.com/QuantumNous/new-api/dto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const kiroMeteringExtensionPath = "x_kiro_metering"

// stripKiroMeteringExtension removes the reserved internal Kiro billing
// extension before a response is sent to the client. When accept is true, it
// also validates and returns the v1 actual-credit observation. Callers pass
// accept=true only for a terminal response (non-stream response or stream
// message_delta).
func stripKiroMeteringExtension(data string, accept bool) (string, *dto.KiroMeteringUsage) {
	extension := gjson.Get(data, kiroMeteringExtensionPath)
	if !extension.Exists() {
		return data, nil
	}

	cleaned, err := sjson.Delete(data, kiroMeteringExtensionPath)
	if err != nil {
		// Invalid JSON will be rejected by the normal Claude decoder and therefore
		// never forwarded. Preserve it here so that decoder returns the canonical
		// bad-response error.
		cleaned = data
	}
	if !accept {
		return cleaned, nil
	}

	observation := &dto.KiroMeteringUsage{EventCount: 1}
	invalid := func(reason string) (string, *dto.KiroMeteringUsage) {
		observation.InvalidReason = reason
		observation.InvalidEventCount = 1
		return cleaned, observation
	}
	if !extension.IsObject() {
		return invalid("extension_not_object")
	}

	schemaVersion := extension.Get("schema_version")
	if !schemaVersion.Exists() {
		return invalid("schema_version_missing")
	}
	if schemaVersion.Type != gjson.Number || schemaVersion.Float() != 1 {
		return invalid("schema_version_invalid")
	}

	credits := extension.Get("credits_used")
	if !credits.Exists() {
		return invalid("credits_used_missing")
	}
	if credits.Type != gjson.Number {
		return invalid("credits_used_not_number")
	}
	creditsUsed := credits.Float()
	if math.IsNaN(creditsUsed) || math.IsInf(creditsUsed, 0) || creditsUsed < 0 {
		return invalid("credits_used_invalid")
	}

	observation.Valid = true
	observation.CreditsUsed = creditsUsed
	return cleaned, observation
}

// mergeKiroMeteringObservation records, but never sums, terminal Kiro credit
// observations. The first valid value wins. Repeated or conflicting terminal
// events remain visible to billing audit without amplifying the charge.
func mergeKiroMeteringObservation(usage *dto.Usage, candidate *dto.KiroMeteringUsage) {
	if usage == nil || candidate == nil || candidate.EventCount == 0 {
		return
	}
	if usage.KiroMetering == nil {
		clone := *candidate
		usage.KiroMetering = &clone
		return
	}

	current := usage.KiroMetering
	current.EventCount += candidate.EventCount
	current.InvalidEventCount += candidate.InvalidEventCount
	current.Duplicate = true

	if !candidate.Valid {
		if current.InvalidReason == "" {
			current.InvalidReason = candidate.InvalidReason
		}
		return
	}
	if !current.Valid {
		current.Valid = true
		current.CreditsUsed = candidate.CreditsUsed
		return
	}
	if current.CreditsUsed != candidate.CreditsUsed {
		current.Conflict = true
	}
}

// overwriteClaudeUsageData writes every calibrated billing bucket, including
// explicit zeroes, while preserving unrelated upstream/vendor fields.
func overwriteClaudeUsageData(data string, usage *dto.Usage) string {
	if data == "" || usage == nil {
		return data
	}
	cacheCreationTotal := usage.PromptTokensDetails.CacheCreationTokensTotal()
	values := []struct {
		path  string
		value int
	}{
		{path: "usage.input_tokens", value: usage.PromptTokens},
		{path: "usage.output_tokens", value: usage.CompletionTokens},
		{path: "usage.cache_read_input_tokens", value: usage.PromptTokensDetails.CachedTokens},
		{path: "usage.cache_creation_input_tokens", value: cacheCreationTotal},
		{path: "usage.cache_creation.ephemeral_5m_input_tokens", value: usage.ClaudeCacheCreation5mTokens},
		{path: "usage.cache_creation.ephemeral_1h_input_tokens", value: usage.ClaudeCacheCreation1hTokens},
	}
	result := data
	for _, item := range values {
		updated, err := sjson.Set(result, item.path, item.value)
		if err != nil {
			return data
		}
		result = updated
	}
	return result
}
