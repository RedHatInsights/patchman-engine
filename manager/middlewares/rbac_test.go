package middlewares

import (
	"app/base/rbac"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
	granularPerms = map[string]struct {
		Permission string
		Read       bool
		Write      bool
	}{"SingleRead": {"patch:single:read", true, false}}
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
	granularPerms = map[string]struct {
		Permission string
		Read       bool
		Write      bool
	}{"SingleReadWrite": {"patch:single:*", true, true}}
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
	granularPerms = map[string]struct {
		Permission string
		Read       bool
		Write      bool
	}{"Read": {"patch:*:read", true, false}}
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
	group1 := "df57820e-965c-49a6-b0bc-797b7dd60581"
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:   "group.id",
					Value: []*string{&group1},
				},
			}},
		}},
	}
	groups := findInventoryGroups(access)
	assert.Equal(t,
		`{"[{\"id\":\"df57820e-965c-49a6-b0bc-797b7dd60581\"}]"}`,
		groups[rbac.KeyGrouped],
	)
	val, ok := groups[rbac.KeyUngrouped]
	assert.Equal(t, "", val)
	assert.Equal(t, false, ok)
}

func TestFindInventoryGroupsUnrouped(t *testing.T) {
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:   "group.id",
					Value: []*string{nil},
				},
			}},
		}},
	}
	groups := findInventoryGroups(access)
	val, ok := groups[rbac.KeyGrouped]
	assert.Equal(t, "", val)
	assert.Equal(t, false, ok)
	assert.Equal(t, "[]", groups[rbac.KeyUngrouped])
}

// nolint:lll
func TestFindInventoryGroups(t *testing.T) {
	group1 := "df57820e-965c-49a6-b0bc-797b7dd60581"
	group2 := "df3f0efd-c853-41b5-80a1-86881d5343d1"
	access := &rbac.AccessPagination{
		Data: []rbac.Access{{
			Permission: "inventory:hosts:read",
			ResourceDefinitions: []rbac.ResourceDefinition{{
				AttributeFilter: rbac.AttributeFilter{
					Key:   "group.id",
					Value: []*string{&group1, &group2, nil},
				},
			}},
		}},
	}
	groups := findInventoryGroups(access)
	assert.Equal(t,
		`{"[{\"id\":\"df57820e-965c-49a6-b0bc-797b7dd60581\"}]","[{\"id\":\"df3f0efd-c853-41b5-80a1-86881d5343d1\"}]"}`,
		groups[rbac.KeyGrouped],
	)
	assert.Equal(t, "[]", groups[rbac.KeyUngrouped])
}
