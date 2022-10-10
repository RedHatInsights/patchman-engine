package middlewares

import (
	"app/base"
	"app/base/types/entitlements"
	"app/base/utils"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var EntitlementCacheValidity = utils.GetIntEnvOrDefault("ENTITLEMENT_CACHE_VALIDITY", 30*60)

type EntitlementCache struct {
	IsEntitled map[string]bool
	Lock       sync.Mutex
}

func (c *EntitlementCache) invalidate() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.IsEntitled = map[string]bool{}
}

var cache = EntitlementCache{IsEntitled: map[string]bool{}, Lock: sync.Mutex{}}
var entitlementsURL = fmt.Sprintf(
	"%s%s/services",
	utils.FailIfEmpty(utils.Cfg.EntitlementsAddress, "ENTITLEMENTS_ADDRESS"),
	base.EntitlementsAPIPrefix,
)

func Entitlements() gin.HandlerFunc {
	return func(c *gin.Context) {
		cache.Lock.Lock()
		defer cache.Lock.Unlock()

		ident, identStr := (*ginContext)(c).GetIdentity()
		if ident == nil {
			abortMissingEntitlement(c)
			return
		}

		entitled, has := cache.IsEntitled[ident.OrgID]
		if !has {
			client := makeClient(identStr)
			out := entitlements.Response{}
			err := makeRequest(client, &base.Context, entitlementsURL, "Entitlements", &out)
			if err != nil {
				out.SmartManagement.IsEntitled = false
			}
			entitled = out.SmartManagement.IsEntitled
			cache.IsEntitled[ident.OrgID] = entitled
		}
		if !entitled {
			abortMissingEntitlement(c)
		}
	}
}

func abortMissingEntitlement(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized,
		utils.ErrorResponse{Error: "You don't have smart management entitlement"})
}

func InvalidateEntitlementCache() {
	ticker := time.NewTicker(time.Second * time.Duration(EntitlementCacheValidity))
	go func() {
		for {
			<-ticker.C
			cache.invalidate()
			utils.Log("cache.IsEntitled", cache.IsEntitled).Debug("Entitlements cache invalidated")
		}
	}()
}
