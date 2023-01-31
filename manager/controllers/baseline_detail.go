package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BaselineDetailResponse struct {
	Data BaselineDetailItem `json:"data"`
}

type BaselineDetailItem struct {
	Attributes BaselineDetailAttributes `json:"attributes"`              // Additional baseline attributes
	ID         int64                    `json:"id" example:"1"`          // Baseline ID
	Type       string                   `json:"type" example:"baseline"` // Document type name
}

type BaselineDetailAttributes struct {
	Name        string          `json:"name" example:"my_baseline"` // Baseline name
	Config      *BaselineConfig `json:"config"`                     // Baseline config
	Description string          `json:"description"`
}

// @Summary Show baseline detail by given baseline ID
// @Description Show baseline detail by given baseline ID
// @ID detailBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id    path    string   true "Baseline ID"
// @Success 200 {object} BaselineDetailResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines/{baseline_id} [get]
func BaselineDetailHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.Atoi(baselineIDstr)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid baseline_id: "+baselineIDstr)
		return
	}

	db := middlewares.DBFromContext(c)
	respItem, err := getBaseline(db, account, baselineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			LogAndRespNotFound(c, err, "baseline not found")
		} else {
			LogAndRespError(c, err, "baseline detail error")
		}
		return
	}
	resp := BaselineDetailResponse{Data: *respItem}

	c.JSON(http.StatusOK, &resp)
}

func getBaseline(db *gorm.DB, accountID, baselineID int) (*BaselineDetailItem, error) {
	var baseline models.Baseline
	err := db.Model(&models.Baseline{}).
		Where("rh_account_id = ? AND id = ?", accountID, baselineID).
		First(&baseline).Error
	if err != nil {
		return nil, err
	}

	var description string
	if d := baseline.Description; d != nil {
		description = *d
	}

	config := tryParseBaselineConfig(baseline.Config)
	resp := BaselineDetailItem{
		ID: baseline.ID,
		Attributes: BaselineDetailAttributes{
			Name:        baseline.Name,
			Config:      config,
			Description: description,
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
		utils.LogWarn("err", err.Error(), "Unable to parse baseline config json")
		return nil
	}
	return &baselineConfig
}
