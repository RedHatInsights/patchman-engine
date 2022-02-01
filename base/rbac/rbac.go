package rbac

type AccessPagination struct {
	Data []Access `json:"data"`
}

type Access struct {
	Permission string `json:"permission"`
}
