package candlepin

type ConsumersEnvironmentsRequest struct {
	ConsumerUuids  []string `json:"consumerUuids"`
	EnvironmentIDs []string `json:"environmentIds"`
}

type ConsumersUpdateResponse struct {
	Message string `json:"displayMessage"`
}
