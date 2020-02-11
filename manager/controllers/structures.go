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
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	Pages      int                   `json:"pages"`
	Sort       []string              `json:"sort"`
	Filter     map[string]FilterData `json:"filter"`
	TotalItems int                   `json:"total_items"`
}
