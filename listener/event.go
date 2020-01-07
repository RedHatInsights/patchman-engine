package listener

type HostInfo struct {
	InsightsId *string `json:"insights_id"`
}

type PlatformEvent struct {
	Id string `json:"id"`

	Type        *string `json:"type"`
	Timestamp   *string `json:"timestamp"`
	Account     *string `json:"account"`
	B64Identity *string `json:"b64_identity"`
	Url         *string `json:"url"`

	HostInfo *HostInfo `json:"host_info"`
}
