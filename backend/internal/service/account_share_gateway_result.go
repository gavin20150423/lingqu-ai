// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package service

func newAccountShareModeSelectionResult(account *Account, acquired bool, release func(), waitPlan *AccountWaitPlan) *AccountSelectionResult {
	return &AccountSelectionResult{
		Account:     account,
		Acquired:    acquired,
		ReleaseFunc: release,
		WaitPlan:    waitPlan,
	}
}

func (s *OpenAIGatewayService) SetAccountShareModeService(accountShareModeService *AccountShareModeService) {
	if s != nil {
		s.accountShareModeService = accountShareModeService
	}
}
