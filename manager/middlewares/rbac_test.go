package middlewares

import (
	"app/base/rbac"
	"app/base/utils"
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
	// handler needs `patch:template:write`
	handler := "CreateBaselineHandler"
	access := rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:*"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:template:write"},
			{Permission: "inventory:*:*"},
		},
	}
	assert.True(t, checkPermissions(&access, handler, "PUT"))

	access = rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:asdf:write"},
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
			{Permission: "patch:*:read"},
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

func TestFindInventoryGroupsGrouped(t *testing.T) {
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
	groups, err := findInventoryGroups(access)
	if assert.NoError(t, err) {
		assert.Equal(t,
			`{"[{\"id\":\"df57820e-965c-49a6-b0bc-797b7dd60581\"}]"}`,
			groups[utils.KeyGrouped],
		)
		val, ok := groups[utils.KeyUngrouped]
		assert.Equal(t, "", val)
		assert.Equal(t, false, ok)
	}
}

func TestFindInventoryGroupsUnrouped(t *testing.T) {
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
	groups, err := findInventoryGroups(access)
	if assert.NoError(t, err) {
		val, ok := groups[utils.KeyGrouped]
		assert.Equal(t, "", val)
		assert.Equal(t, false, ok)
		assert.Equal(t, "[]", groups[utils.KeyUngrouped])
	}
}

func TestFindInventoryGroups(t *testing.T) {
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
	groups, err := findInventoryGroups(access)
	if assert.NoError(t, err) {
		assert.Equal(t,
			`{"[{\"id\":\"df57820e-965c-49a6-b0bc-797b7dd60581\"}]","[{\"id\":\"df3f0efd-c853-41b5-80a1-86881d5343d1\"}]"}`,
			groups[utils.KeyGrouped],
		)
		assert.Equal(t, "[]", groups[utils.KeyUngrouped])
	}
}

func TestFindInventoryGroupsOverwrite(t *testing.T) {
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
	groups, err := findInventoryGroups(access)
	if assert.NoError(t, err) {
		// we expect access to all groups (empty map)
		assert.Equal(t, 0, len(groups))
	}
}

func TestFindInventoryGroupsOverwrite2(t *testing.T) {
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
	groups, err := findInventoryGroups(access)
	if assert.NoError(t, err) {
		// we expect access to all groups (empty map)
		assert.Equal(t, 0, len(groups))
	}
}

func TestFindInventoryGroupsInvalidOp(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:       "group.id",
						Value:     []*string{},
						Operation: "equal",
					},
				}},
			},
		},
	}
	groups, err := findInventoryGroups(access)
	assert.Error(t, err)
	assert.Nil(t, groups)
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
