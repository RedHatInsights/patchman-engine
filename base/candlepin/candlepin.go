package candlepin

type ConsumersUpdateRequest struct {
	Environments []ConsumersUpdateEnvironment `json:"environments"`
}

type ConsumersUpdateEnvironment struct {
	ID string `json:"id"`
}

type ConsumersUpdateResponse struct {
	Message string `json:"displayMessage"`
}
