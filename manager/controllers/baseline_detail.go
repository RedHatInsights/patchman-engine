package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type BaselineDetailResponse struct {
	Data BaselineDetailItem `json:"data"`
}

type BaselineDetailItem struct {
	Attributes BaselineDetailAttributes
	ID         int    `json:"id"`
	Type       string `json:"type"`
}

type BaselineDetailAttributes struct {
	Name   string          `json:"name"`
	Config *BaselineConfig `json:"config"`
}

// @Summary Show baseline detail by given baseline ID
// @Description Show baseline detail by given baseline ID
// @ID detailBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id    path    string   true "Baseline ID"
// @Success 200 {object} BaselineDetailResponse
// @Router /api/patch/v1/baselines/{baseline_id} [get]
func BaselineDetailHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.Atoi(baselineIDstr)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid baseline_id: "+baselineIDstr)
		return
	}

	respItem, err := getBaseline(account, baselineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			LogAndRespNotFound(c, err, "baseline not found")
		} else {
			LogAndRespError(c, err, "baseline detail error")
		}
		return
	}
	resp := BaselineDetailResponse{
		Data: *respItem,
	}

	c.JSON(http.StatusOK, &resp)
}

func getBaseline(accountID, baselineID int) (*BaselineDetailItem, error) {
	var baseline models.Baseline
	err := database.Db.Model(&models.Baseline{}).
		Where("rh_account_id = ? AND id = ?", accountID, baselineID).
		First(&baseline).Error
	if err != nil {
		return nil, err
	}

	config := tryParseBaselineConfig(baseline.Config)
	resp := BaselineDetailItem{
		ID: baseline.ID,
		Attributes: BaselineDetailAttributes{
			Name:   baseline.Name,
			Config: config,
		},
		Type: "baseline",
	}
	return &resp, nil
}

func tryParseBaselineConfig(configBytes []byte) *BaselineConfig {
	if configBytes == nil {
		return nil
	}

	var baselineConfig BaselineConfig
	err := json.Unmarshal(configBytes, &baselineConfig)
	if err != nil {
		utils.Log("err", err.Error()).Warn("Unable to parse baseline config json")
		return nil
	}
	return &baselineConfig
}
