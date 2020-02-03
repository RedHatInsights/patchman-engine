package controllers

import "time"

type Links struct {
	First    string  `json:"first"`
	Last     string  `json:"last"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

type ListMeta struct {
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Pages    int      `json:"pages"`
	Sort     []string `json:"sort"`
	// TODO: Implement
	Filter     []string `json:"filter"`
	TotalItems int      `json:"total_items"`
}

type SystemAdvisoryItem struct {
	Attributes SystemAdvisoryItemAttributes `json:"attributes"`
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
}

type SystemAdvisoryItemAttributes struct {
	Description  string    `json:"description"`
	PublicDate   time.Time `json:"public_date"`
	Synopsis     string    `json:"synopsis"`
	AdvisoryType int       `json:"advisory_type"`
	Severity     *int      `json:"severity,omitempty"`
}

type AdvisoryItem struct {
	Attributes AdvisoryItemAttributes `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type AdvisoryItemAttributes struct {
	SystemAdvisoryItemAttributes
	ApplicableSystems int `json:"applicable_systems"`
}

type SystemItem struct {
	Attributes SystemItemAttributes `json:"attributes"`
	ID         string               `json:"id"`
	Type       string               `json:"type"`
}

type SystemItemAttributes struct {
	LastEvaluation *time.Time `json:"last_evaluation"`
	LastUpload     *time.Time `json:"last_upload"`
	RhsaCount      int        `json:"rhsa_count"`
	RhbaCount      int        `json:"rhba_count"`
	RheaCount      int        `json:"rhea_count"`
	Enabled        bool       `json:"enabled"`
}
