package listener

type HostInfo struct {
	InsightsID *string `json:"insights_id"`
}

type PlatformEvent struct {
	ID string `json:"id"`

	Type        *string `json:"type"`
	Timestamp   *string `json:"timestamp"`
	Account     *string `json:"account"`
	B64Identity *string `json:"b64_identity"`
	URL         *string `json:"url"`

	HostInfo *HostInfo `json:"host_info"`
}
