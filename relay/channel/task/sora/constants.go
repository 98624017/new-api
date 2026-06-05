package sora

import (
	"os"
	"strings"
	"sync"
)

var ModelList = []string{
	"sora-2",
	"sora-2-pro",
}

var ChannelName = "sora"

var (
	referenceVideoDoublePriceModelsMu sync.RWMutex
	ReferenceVideoDoublePriceModels   = map[string]bool{}
	referenceVideoDurationBilling     bool
)

func ReloadReferenceVideoDoublePriceModelsFromEnv() {
	models := parseReferenceVideoDoublePriceModels(os.Getenv("SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS"))
	for model := range parseReferenceVideoDoublePriceModels(os.Getenv("SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS")) {
		models[model] = true
	}
	durationBilling := parseReferenceVideoDurationBillingEnabled(os.Getenv("SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED"))
	referenceVideoDoublePriceModelsMu.Lock()
	defer referenceVideoDoublePriceModelsMu.Unlock()
	ReferenceVideoDoublePriceModels = models
	referenceVideoDurationBilling = durationBilling
}

func IsReferenceVideoDoublePriceModel(modelName string) bool {
	referenceVideoDoublePriceModelsMu.RLock()
	defer referenceVideoDoublePriceModelsMu.RUnlock()
	return ReferenceVideoDoublePriceModels[modelName]
}

func ReferenceVideoDurationBillingEnabled() bool {
	referenceVideoDoublePriceModelsMu.RLock()
	defer referenceVideoDoublePriceModelsMu.RUnlock()
	return referenceVideoDurationBilling
}

func parseReferenceVideoDoublePriceModels(raw string) map[string]bool {
	models := map[string]bool{}
	for _, item := range strings.Split(raw, ",") {
		model := strings.TrimSpace(item)
		if model == "" {
			continue
		}
		models[model] = true
	}
	return models
}

func parseReferenceVideoDurationBillingEnabled(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
