package controllers

import (
	"app/base/utils"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
)

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
	pprofHandler(c, "evaluator-upload")
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
	pprofHandler(c, "evaluator-recalc")
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
	pprofHandler(c, "listener")
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
	pprofHandler(c, "manager")
}

var allowedParams = []string{"heap", "profile", "block", "mutex", "trace"}

func pprofHandler(c *gin.Context, serviceName string) {
	query := c.Request.URL.RawQuery
	param := c.Param("param")
	if !slices.Contains(allowedParams, param) {
		c.Status(http.StatusBadRequest)
		return
	}
	data, err := getPprof(serviceName, param, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", param))
	c.Data(http.StatusOK, "application/octet-stream", data)
}

func getPprof(serviceName, param, query string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	if len(query) > 0 {
		param = param + "?" + query
	}
	var address string
	switch serviceName {
	case "manager":
		address = utils.CoreCfg.ManagerPrivateAddress
	case "listener":
		address = utils.CoreCfg.ListenerPrivateAddress
	case "evaluator-upload":
		address = utils.CoreCfg.EvaluatorUploadPrivateAddress
	case "evaluator-recalc":
		address = utils.CoreCfg.EvaluatorRecalcPrivateAddress
	default:
		return nil, fmt.Errorf("invalid service name: %s", serviceName)
	}
	urlPath := fmt.Sprintf("%s/debug/pprof/%s", address, param)
	req, err := http.NewRequest(http.MethodGet, urlPath, nil) // #nosec G704
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req) // #nosec G704
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
