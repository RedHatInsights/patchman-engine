package controllers

type Links struct {
	First    string  `json:"first" example:"/link/to/the/first"`
	Last     string  `json:"last" example:"/link/to/the/last"`
	Next     *string `json:"next" example:"/link/to/the/next"`
	Previous *string `json:"previous" example:"/link/to/the/previous"`
}

type ListMeta struct {
	// Used response limit (page size) - pagination
	Limit int `json:"limit" example:"20"`

	// Used response offset - pagination
	Offset int `json:"offset" example:"0"`

	// Used sorting fields
	Sort []string `json:"sort,omitempty" example:"name"`

	// Used search terms
	Search string `json:"search,omitempty" example:"kernel"`

	// Used filters
	Filter map[string]FilterData `json:"filter"`

	// Total items count to return
	TotalItems int `json:"total_items" example:"1000"`

	// Some subtotals used by some endpoints
	SubTotals map[string]int `json:"subtotals,omitempty"`

	// Show whether customer has some registered systems
	HasSystems *bool `json:"has_systems,omitempty"`
}

type IDsResponse struct {
	IDs []string `json:"ids"`
}

type IDStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type IDsStatusResponse struct {
	Data []IDStatus `json:"data"`
	// backward compatibility
	// TODO: delete later once UI is using only the new `data` field
	IDsResponse
}

type SystemGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
