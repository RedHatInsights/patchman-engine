package controllers

import "time"

type Links struct {
	First    string  `json:"first"`
	Last     string  `json:"last"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

type AdvisoryItem struct {
	Attributes AdvisoryItemAttributes `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type AdvisoryItemAttributes struct {
	Description       string    `json:"description"`
	Severity          string    `json:"severity"`
	PublicDate        time.Time `json:"public_date"`
	Synopsis          string    `json:"synopsis"`
	AdvisoryType      int       `json:"advisory_type"`
	ApplicableSystems int       `json:"applicable_systems"`
}

type AdvisoryMeta struct {
	DataFormat string  `json:"data_format"`
	Filter     *string `json:"filter"`
	Severity   *string `json:"severity"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Pages      int     `json:"pages"`
	PublicFrom *int    `json:"public_from"`
	PublicTo   *int    `json:"public_to"`
	ShowAll    bool    `json:"show_all"`
	Sort       *bool   `json:"sort"`
	TotalItems int     `json:"total_items"`
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
