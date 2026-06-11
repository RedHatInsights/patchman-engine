package middlewares

import (
	"app/base/rbac"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var (
	group1 = "df57820e-965c-49a6-b0bc-797b7dd60581"
	group2 = "df3f0efd-c853-41b5-80a1-86881d5343d1"
)

func okHandler(c *gin.Context) {
	c.JSON(http.StatusOK, nil)
}

func testRBAC(t *testing.T, method string, expectedStatus int) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, "/", nil)
	router := gin.Default()
	router.Use(RBAC())
	router.Handle(method, "/", okHandler)
	router.ServeHTTP(w, req)
	assert.Equal(t, expectedStatus, w.Code)
}

func TestRBACGet(t *testing.T) {
	testRBAC(t, "GET", http.StatusOK)
}

func TestRBACPost(t *testing.T) {
	testRBAC(t, "POST", http.StatusOK)
}

func TestRBACDelete(t *testing.T) {
	testRBAC(t, "DELETE", http.StatusUnauthorized)
}

func TestRBACPut(t *testing.T) {
	testRBAC(t, "PUT", http.StatusUnauthorized)
}

func TestPermissionsSingleWrite(t *testing.T) {
	// handler needs `content-sources:templates:write`
	handler := "TemplateSystemsUpdateHandler"
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:*:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:*:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:templates:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:asdf:write"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:asdf:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "content-sources:*:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))
}

func TestPermissionsSingleRead(t *testing.T) {
	// handler needs `patch:single:read`
	handler := "SingleRead"
	granularPerms = map[string]string{"SingleRead": "patch:single:read"}
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:single:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:asdf:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:asdf:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "GET"))
}

// nolint:funlen
func TestPermissionsSingleReadWrite(t *testing.T) {
	// handler needs `patch:single:read`
	handler := "SingleReadWrite"
	granularPerms = map[string]string{"SingleReadWrite": "patch:single:*"}
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:single:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:single:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:asdf:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:asdf:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "PUT"))
}

func TestPermissionsRead(t *testing.T) {
	// handler needs `patch:single:read`
	handler := "Read"
	granularPerms = map[string]string{"Read": "patch:*:read"}
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:read"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.False(t, checkPermissions(&access, handler, "GET"))
}

func TestFindInventoryWorkspacesGrouped(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:       "group.id",
					Value:     []*string{&group1},
					Operation: "in",
				},
			}},
		}},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{group1}, workspaces)
	}
}

func TestFindInventoryWorkspacesUngrouped(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:       "group.id",
					Value:     []*string{nil},
					Operation: "in",
				},
			}},
		}},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{rootWorkspaceID}, workspaces)
	}
}

func TestFindInventoryWorkspaces(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:       "group.id",
					Value:     []*string{&group1, &group2, nil},
					Operation: "in",
				},
			}},
		}},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{group1, group2, rootWorkspaceID}, workspaces)
	}
}

func TestFindInventoryWorkspacesAllAccess(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:       "group.id",
						Value:     []*string{&group1, nil},
						Operation: "in",
					},
				}},
			},
			{
				Permission:          "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{},
			},
		},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Nil(t, workspaces)
	}
}

func TestFindInventoryWorkspacesAllAccessFirst(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{
			{
				Permission:          "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{},
			},
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:       "group.id",
						Value:     []*string{&group1, nil},
						Operation: "in",
					},
				}},
			},
		},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Nil(t, workspaces)
	}
}

func TestFindInventoryWorkspacesInvalidOp(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:       "group.id",
						Value:     []*string{},
						Operation: "unsupported",
					},
				}},
			},
		},
	}
	workspaces, err := findInventoryWorkspaces(access)
	assert.Error(t, err)
	assert.Nil(t, workspaces)
}

func TestFindInventoryWorkspacesSkipsInvalidUUID(t *testing.T) {
	invalid := "not-a-uuid"
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:       "group.id",
					Value:     []*string{&invalid, &group1},
					Operation: "in",
				},
			}},
		}},
	}
	workspaces, err := findInventoryWorkspaces(access)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{group1}, workspaces)
	}
}

func TestMultiplePermissions(t *testing.T) {
	handler := "MultiplePermissions"
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "inventory:*:read"},
			{Permission: "inventory:hosts:write"},
			{Permission: "inventory:hosts:read"},
			{Permission: "inventory:groups:write"},
			{Permission: "inventory:groups:read"},
			{Permission: "patch:*:*"},
			{Permission: "patch:*:read"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))
	assert.True(t, checkPermissions(&access, handler, "DELETE"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "inventory:*:read"},
			{Permission: "inventory:hosts:write"},
			{Permission: "inventory:groups:write"},
			{Permission: "patch:*:read"},
			{Permission: "inventory:hosts:read"},
			{Permission: "inventory:groups:read"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "GET"))
	assert.False(t, checkPermissions(&access, handler, "DELETE"))
}

var allowedOperations = `{"data": [
    {
      "resourceDefinitions": [],
      "permission": "patch:*:read"
    },
    {
      "resourceDefinitions": [
        {
          "attributeFilter": {
            "key": "group.id",
            "value": "00000000-f688-49d4-a8e2-87394f1ac1b1",
            "operation": "equal"
          }
        }
      ],
      "permission": "inventory:hosts:read"
    },
    {
      "resourceDefinitions": [
        {
          "attributeFilter": {
            "key": "group.id",
            "value": [ "00000000-f7a6-45a1-b5a8-410f20052fb1", "00000000-78e0-4cad-bf01-63cf1e4b1dca" ],
            "operation": "in"
          }
        }
      ],
      "permission": "inventory:hosts:read"
    },
    {
      "resourceDefinitions": [
        {
          "attributeFilter": {
            "key": "group.id",
            "value": [ "00000000-f688-49d4-a8e2-ee394f1ac1b1" ],
            "operation": "in"
          }
        }
      ],
      "permission": "inventory:hosts:read"
    },
    {
      "resourceDefinitions": [
        {
          "attributeFilter": {
            "key": "group.id",
            "value": null,
            "operation": "equal"
          }
        }
      ],
      "permission": "inventory:hosts:read"
    }
  ]
}
`

func TestPermissionsAllowedOperations(t *testing.T) {
	handler := "SystemsListHandler"
	access := rbac.AccessPagination{}
	err := json.Unmarshal([]byte(allowedOperations), &access)
	assert.NoError(t, err)
	assert.True(t, checkPermissions(&access, handler, "GET"))
	workspaces, err := findInventoryWorkspaces(&access)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"00000000-f688-49d4-a8e2-87394f1ac1b1",
		"00000000-f7a6-45a1-b5a8-410f20052fb1",
		"00000000-78e0-4cad-bf01-63cf1e4b1dca",
		"00000000-f688-49d4-a8e2-ee394f1ac1b1",
		rootWorkspaceID,
	}, workspaces)
}
