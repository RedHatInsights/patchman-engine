package controllers

type Links struct {
	First    string  `json:"first"`
	Last     string  `json:"last"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

type ListMeta struct {
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
	Sort       []string              `json:"sort,omitempty"`
	Search     string                `json:"search,omitempty"`
	Filter     map[string]FilterData `json:"filter"`
	TotalItems int                   `json:"total_items"`
}
