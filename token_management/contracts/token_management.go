package contracts

type ITokenManagement interface {
	UsedTokens(inputToken int, outputToken int)
	CalculateCost(providerName string, modelName string, inputToken int, outputToken int) float64
	DisplayTokens(chatProviderName string, chatModel string)
	DisplayLiveTokens(chatProviderName string, chatModel string)
	GetCurrentTokenUsage() (total int, input int, output int)
	ClearToken()
}
