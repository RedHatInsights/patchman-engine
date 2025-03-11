package controllers

import (
	"app/base/utils"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
	pprofHandler(c, utils.CoreCfg.EvaluatorUploadPrivateAddress)
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
	pprofHandler(c, utils.CoreCfg.EvaluatorRecalcPrivateAddress)
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
	pprofHandler(c, utils.CoreCfg.ListenerPrivateAddress)
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
	pprofHandler(c, utils.CoreCfg.ManagerPrivateAddress)
}

var paramRegexp = regexp.MustCompile("^(heap|profile|block|mutex|trace)$")

func pprofHandler(c *gin.Context, address string) {
	query := c.Request.URL.RawQuery
	param := c.Param("param")
	match := paramRegexp.FindStringSubmatch(param)
	if len(match) < 1 {
		c.Status(http.StatusBadRequest)
		return
	}
	data, err := getPprof(address, match[0], query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", param))
	c.Data(http.StatusOK, "application/octet-stream", data)
}

func getPprof(address, param, query string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	if len(query) > 0 {
		param = param + "?" + query
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
