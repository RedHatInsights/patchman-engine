package notification

import (
	"time"
)

const (
	Version     = "v1.1.0"
	Bundle      = "rhel"
	Application = "patch"
)

type Context struct {
	InventoryID string `json:"inventory_id"`
}

type Event struct {
	// Future-proof. Needs to be there and empty for now.
	Metadata interface{} `json:"metadata,omitempty"`
	// Your application payload.
	// All the data required by the app to compose the various messages (Email, webhook…​) after transformation.
	Payload interface{} `json:"payload,omitempty"`
}

type Recipient struct {
	// Only admins: Setting to true sends an email to the administrators of the account.
	// Setting to false sends an email to all users of the account. (since v1.1.0).
	OnlyAdmins bool `json:"only_admins"`
	// Ignore user preferences: Setting to true ignores all the user preferences on this Recipient setting
	// (It doesn’t affect other configuration that an Administrator sets on their Notification settings).
	// Setting to false honors the user preferences. (since v1.1.0).
	IgnoreUserPreferences bool `json:"ignore_user_preferences"`
	// List of users to direct the notification to. This won’t override notification’s administrators settings.
	// Users list will be merge with other settings. (since v1.2.0).
	Users []string `json:"users"`
}

type Notification struct {
	// Version of the notification message. Current is "v1.2.0".
	// Older versions don’t need to update unless they want to use newer features.
	// Defaults to "v1.0.0" when not specified.
	Version string `json:"version"`
	// Bundle name as used during application registration.
	Bundle string `json:"bundle"`
	// Application name as used during application registration.
	Application string `json:"application"`
	// Event type name as used during application registration.
	EventType string `json:"event_type"`
	// ISO-8601 formatted date (per platform convention when the message was sent).
	Timestamp string `json:"timestamp"`
	// Account id to address notification to.
	AccountID string `json:"account_id"`
	// Extra information that are common to all the events that are sent in this message.
	Context interface{} `json:"context,omitempty"`
	Events  []Event     `json:"events"`
	// Recipients settings - Applications can add extra email recipients by adding entries to this array.
	// This setting extends whatever the Administrators configured in their Notifications settings (since v1.1.0).
	Recipients []Recipient `json:"recipients,omitempty"`
	// org_id as future replacement of the account_id.
	OrgID string `json:"org_id,omitempty"`
}

type Advisory struct {
	AdvisoryName string `json:"advisory_name"`
	AdvisoryType string `json:"advisory_type"`
	Synopsis     string `json:"synopsis"`
}

func MakeNotification(inventoryID, accountName, orgID, eventType string, events []Event) *Notification {
	return &Notification{
		Version:     Version,
		Bundle:      Bundle,
		Application: Application,
		EventType:   eventType,
		// ISO-8601 formatted time
		Timestamp: time.Now().Format(time.RFC3339),
		AccountID: accountName,
		Context:   Context{InventoryID: inventoryID},
		Events:    events,
		OrgID:     orgID,
	}
}
