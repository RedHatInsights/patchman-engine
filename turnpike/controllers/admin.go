package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/tasks/caches"
	sync "app/tasks/vmaas_sync"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Session struct {
	Pid   int
	Query string
}

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
	utils.LogInfo("manual syncing called...")
	sync.Configure()
	vmaasExportedTS := sync.VmaasDBExported()
	err := sync.SyncData(nil, vmaasExportedTS)
	if err != nil {
		utils.LogError("err", err.Error(), "manual called syncing failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.LogInfo("manual syncing finished successfully")
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
	utils.LogInfo("manual re-calc messages sending called...")
	err := sync.SendReevaluationMessages()
	if err != nil {
		utils.LogError("err", err.Error(), "manual re-calc msgs sending failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.LogInfo("manual re-calc messages sent successfully")
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
		utils.LogError("error", err, "Could not check validity of caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	if !valid {
		utils.LogError("Cache mismatch found")
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
		utils.LogError("error", err, "Could not refresh package caches")
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
		utils.LogError("error", err.Error(), "Could not refresh package caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, "refreshing package caches")
}

// @Summary Get active db sessions
// @Description Get active db sessions
// @ID getSessions
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    search path string false "Search string" SchemaExample(refresh_package)
// @Success 200 {object} []Session
// @Failure 409 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /sessions/{search} [get]
func GetActiveSessionsHandler(c *gin.Context) {
	param := c.Param("search")
	data := make([]Session, 0)
	q := database.Db.Table("pg_stat_activity").Select("pid, query")
	if param != "" {
		q.Where("query like ?", fmt.Sprint("%", param, "%"))
	}
	err := q.Find(&data).Error
	if err != nil {
		utils.LogError("error", err, "DB query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, &data)
}

// @Summary Terminate db session
// @Description Terminate db session
// @ID TerminateSession
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    pid path int true "pid"
// @Success 200 {object} string
// @Failure 409 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /sessions/{pid} [delete]
func TerminateSessionHandler(c *gin.Context) {
	param := c.Param("pid")
	if param == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "pid param not found"})
		return
	}
	err := database.Db.Exec("select pg_terminate_backend(?)", param).Error
	if err != nil {
		utils.LogError("error", err, "DB query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fmt.Sprintf("pid: %s terminated", param))
}

// @Summary Get profile info
// @Description Get profile info
// @ID getEvaluatorUploadPprof
// @Security RhIdentity
// @Produce  application/octet-stream
// @Param    param path string false "What to profile" SchemaExample(profile)
// @Success 200
// @Failure 500 {object} map[string]interface{}
// @Router /pprof/evaluator_upload/{param} [get]
func GetEvaluatorUploadPprof(c *gin.Context) {
	param := c.Param("param")
	data, err := getPprof(utils.Cfg.EvaluatorUploadPrivateAddress, param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// @Summary Get profile info
// @Description Get profile info
// @ID getEvaluatorRecalcPprof
// @Security RhIdentity
// @Produce  application/octet-stream
// @Param    param path string false "What to profile" SchemaExample(profile)
// @Success 200
// @Failure 500 {object} map[string]interface{}
// @Router /pprof/evaluator_recalc/{param} [get]
func GetEvaluatorRecalcPprof(c *gin.Context) {
	param := c.Param("param")
	data, err := getPprof(utils.Cfg.EvaluatorRecalcPrivateAddress, param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// @Summary Get profile info
// @Description Get profile info
// @ID getListenerPprof
// @Security RhIdentity
// @Produce  application/octet-stream
// @Param    param path string false "What to profile" SchemaExample(profile)
// @Success 200
// @Failure 500 {object} map[string]interface{}
// @Router /pprof/listener/{param} [get]
func GetListenerPprof(c *gin.Context) {
	param := c.Param("param")
	data, err := getPprof(utils.Cfg.ListenerPrivateAddress, param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// @Summary Get profile info
// @Description Get profile info
// @ID getManagerPprof
// @Security RhIdentity
// @Produce  application/octet-stream
// @Param    param path string false "What to profile" SchemaExample(profile)
// @Success 200
// @Failure 500 {object} map[string]interface{}
// @Router /pprof/manager/{param} [get]
func GetManagerPprof(c *gin.Context) {
	param := c.Param("param")
	data, err := getPprof(utils.Cfg.ManagerPrivateAddress, param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", data)
}

func getPprof(address, param string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	urlPath := fmt.Sprintf("%s/debug/pprof/%s", address, param)
	req, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return resBody, nil
}
