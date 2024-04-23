package candlepin

type ConsumersUpdateRequest struct {
	Environments []ConsumersUpdateEnvironment
}

type ConsumersUpdateEnvironment struct {
	ID string
}

type ConsumersUpdateResponse struct {
	Message string `json:"displayMessage"`
}
