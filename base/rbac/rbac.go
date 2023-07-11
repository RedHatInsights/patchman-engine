package rbac

const KeyGrouped = "grouped"
const KeyUngrouped = "ungrouped"

type AccessPagination struct {
	Data []Access `json:"data"`
}

type Access struct {
	Permission          string               `json:"permission"`
	ResourceDefinitions []ResourceDefinition `json:"resourceDefinitions"`
}

type ResourceDefinition struct {
	AttributeFilter AttributeFilter `json:"attributeFilter"`
}

type AttributeFilter struct {
	Key   string    `json:"key"`
	Value []*string `json:"value"`
}

type InventoryGroupID struct {
	ID string `json:"id"`
}

type InventoryGroup []InventoryGroupID
