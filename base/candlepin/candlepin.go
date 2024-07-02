package candlepin

var APIPrefix = "/candlepin"

type EnvironmentsConsumersRequestGuestID struct {
	ID      string
	GuestID string `json:"guestId"`
}

type EnvironmentsConsumersRequest struct {
	ID       string
	UUID     string
	GuestIds []EnvironmentsConsumersRequestGuestID `json:"guestIds"`
}

type EnvironmentsConsumersResponse struct {
	GuestIds []EnvironmentsConsumersRequestGuestID `json:"guestIds"`
}
