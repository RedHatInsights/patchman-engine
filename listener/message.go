package listener

type HostInfo struct {
	InsightsId *string `json:"insights_id"`
}

type PlatformEvent struct {
	Id        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Account   string `json:"account"`

	// Optional fields
	Type        *string `json:"type"`
	B64Identity *string `json:"b64_identity"`
	Url         *string `json:"url"`

	HostInfo *HostInfo `json:"host_info"`
}
