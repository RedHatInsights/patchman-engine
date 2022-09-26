package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/tasks/caches"
	sync "app/tasks/vmaas_sync"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @Summary Sync data from VMaaS
// @Description Sync data from VMaaS
// @ID sync
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /sync [get]
func Syncapi(c *gin.Context) {
	utils.Log().Info("manual syncing called...")
	err := sync.SyncData(nil, nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("manual called syncing failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.Log().Info("manual syncing finished successfully")
	c.JSON(http.StatusOK, "OK")
}

// @Summary Re-evaluate systems
// @Description Re-evaluate systems
// @ID recalc
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /re-calc [get]
func Recalc(c *gin.Context) {
	utils.Log().Info("manual re-calc messages sending called...")
	err := sync.SendReevaluationMessages()
	if err != nil {
		utils.Log("err", err.Error()).Error("manual re-calc msgs sending failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.Log().Info("manual re-calc messages sent successfully")
	c.JSON(http.StatusOK, "OK")
}

// @Summary Check cached counts
// @Description Check cached counts
// @ID checkCaches
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 409 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /check-caches [get]
func CheckCaches(c *gin.Context) {
	valid, err := database.CheckCachesValidRet()
	if err != nil {
		utils.Log("error", err).Error("Could not check validity of caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	if !valid {
		utils.Log().Error("Cache mismatch found")
		c.JSON(http.StatusConflict, "conflict")
		return
	}

	c.JSON(http.StatusOK, "caches counts OK")
}

// @Summary Refresh package caches
// @Description Refresh package caches for all accounts with invalidated cache
// @ID refreshPackagesCaches
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 409 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /refresh-packages [put]
func RefreshPackagesHandler(c *gin.Context) {
	err := caches.RefreshPackagesCaches(nil)
	if err != nil {
		utils.Log("error", err).Error("Could not refresh package caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, "refreshing package caches")
}

// @Summary Refresh package caches per account
// @Description Refresh package caches for specified account by internal account id
// @ID refreshPackagesAccountCaches
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    account    path    string   true "Internal account ID"
// @Success 200 {object} string
// @Failure 409 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /refresh-packages/{account} [put]
func RefreshPackagesAccountHandler(c *gin.Context) {
	param := c.Param("account")
	if param == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "account_param not found"})
		return
	}
	accID, err := strconv.Atoi(param)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "invalid account_param"})
		return
	}
	err = caches.RefreshPackagesCaches(&accID)
	if err != nil {
		utils.Log("error", err.Error()).Error("Could not refresh package caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, "refreshing package caches")
}
