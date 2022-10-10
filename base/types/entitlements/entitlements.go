package entitlements

type Response struct {
	SmartManagement EntitlementFields `json:"smart_management"`
}
type EntitlementFields struct {
	IsEntitled bool `json:"is_entitled"`
	IsTrial    bool `json:"is_trial"`
}
