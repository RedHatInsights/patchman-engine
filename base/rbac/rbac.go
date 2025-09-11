package rbac

import (
	"github.com/bytedance/sonic"
)

type AccessPagination struct {
	Data []Access `json:"data"`
}

type Access struct {
	Permission          string               `json:"permission"`
	ResourceDefinitions []ResourceDefinition `json:"resourceDefinitions"`
}

type ResourceDefinition struct {
	AttributeFilter AttributeFilter `json:"attributeFilter,omitempty"`
}

type AttributeFilterValue []*string

type AttributeFilter struct {
	Key       string               `json:"key"`
	Value     AttributeFilterValue `json:"value"`
	Operation string               `json:"operation"`
}

type inventoryGroup struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type InventoryGroup []inventoryGroup

func (a *AttributeFilterValue) UnmarshalJSON(data []byte) error {
	var (
		array []*string
		value *string
		err   error
	)

	if err = sonic.Unmarshal(data, &array); err != nil {
		// parsing of AttributeFilter Value into []*string failed
		// try to parse it as *string
		if err = sonic.Unmarshal(data, &value); err != nil {
			// fail, the value is neither []*string nor *string
			return err
		}
		if value != nil {
			// according to RBAC team, value is a single string value
			// not comma delimited strings, multiple values are always in array
			array = append(array, value)
		}
	}
	if array == nil && value == nil {
		// in this case we got `"value": null`
		// we should apply the permission to systems with no inventory groups
		array = append(array, value)
	}

	*a = array
	return nil
}

type WorkspaceResponseData struct {
	ID string `json:"id"`
}

type DefaultWorkspaceResponse struct {
	Data []WorkspaceResponseData `json:"data"`
}
