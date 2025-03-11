package controllers

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Session struct {
	Pid   int
	Query string
}

// @Summary Recreate pg_repack database extension
// @Description Recreate pg_repack database extension
// @ID repackRecreate
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /database/pg_repack/recreate [put]
func RepackRecreateHandler(c *gin.Context) {
	if err := database.DB.Exec("DROP EXTENSION IF EXISTS pg_repack CASCADE").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	if err := database.DB.Exec("CREATE EXTENSION pg_repack").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
	}
	c.JSON(http.StatusOK, "pg_repack extension re-created")
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
// @Router /database/sessions/{search} [get]
func GetActiveSessionsHandler(c *gin.Context) {
	param := c.Param("search")
	data := make([]Session, 0)
	q := database.DB.Table("pg_stat_activity").Select("pid, query")
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
// @Router /database/sessions/{pid} [delete]
func TerminateSessionHandler(c *gin.Context) {
	param := c.Param("pid")
	if param == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "pid param not found"})
		return
	}
	err := database.DB.Exec("select pg_terminate_backend(?)", param).Error
	if err != nil {
		utils.LogError("error", err, "DB query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fmt.Sprintf("pid: %s terminated", param))
}
