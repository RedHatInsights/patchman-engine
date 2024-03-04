package mqueue

import (
	"time"
)

// nolint:lll
// copied from https://github.com/content-services/content-sources-backend/blob/6fe3dd9409cfc048eefb07d60d31574da2a47217/pkg/api/templates.go#L20-L30
// importing "github.com/content-services/content-sources-backend/pkg/api"
// adds too many dependencies and some are incompatible
type TemplateResponse struct {
	UUID            string    `json:"uuid" readonly:"true"`
	Name            string    `json:"name"`             // Name of the template
	OrgID           string    `json:"org_id"`           // Organization ID of the owner
	Description     string    `json:"description"`      // Description of the template
	Arch            string    `json:"arch"`             // Architecture of the template
	Version         string    `json:"version"`          // Version of the template
	Date            time.Time `json:"date"`             // Latest date to include snapshots for
	RepositoryUUIDS []string  `json:"repository_uuids"` // Repositories added to the template
}

type TemplateEvent struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Source  string             `json:"source"`
	Subject string             `json:"subject"`
	Time    time.Time          `json:"time"`
	OrgID   string             `json:"redhatorgid"`
	Data    []TemplateResponse `json:"data"`
}
