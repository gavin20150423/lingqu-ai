// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package service

// isOpenAIAccountEligibleForRequest centralises the schedulable /
// OpenAI-compatible / model / compact-support checks used during account selection.
func isOpenAIAccountEligibleForRequest(account *Account, requestedModel string, requireCompact bool) bool {
	if account == nil || !account.IsSchedulable() || !account.IsOpenAICompatible() {
		return false
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return false
	}
	if requireCompact && openAICompactSupportTier(account) == 0 {
		return false
	}
	return true
}

func accountSupportsRequestedOpenAIImageCapability(account *Account, capability OpenAIImagesCapability) bool {
	if account == nil {
		return false
	}
	if capability == "" {
		return true
	}
	return account.SupportsOpenAIImageCapability(capability)
}
