package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/assets/service"
	"github.com/gin-gonic/gin"
)

type DatasetHandler struct {
	service *service.DatasetService
}

func NewDatasetHandler(service *service.DatasetService) *DatasetHandler {
	return &DatasetHandler{
		service: service,
	}
}

// GetGoldenDataset handles GET /api/v1/dataset/golden
// GetGoldenDataset godoc
// @Summary Get golden dataset for classifier training
// @Tags assets
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /dataset/golden [get]
func (h *DatasetHandler) GetGoldenDataset(c *gin.Context) {
	data, err := h.service.GenerateGoldenDataset(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate dataset"})
		return
	}

	c.Header("Content-Type", "application/x-jsonlines")
	c.Header("Content-Disposition", "attachment; filename=\"golden_dataset.jsonl\"")
	c.Data(http.StatusOK, "application/x-jsonlines", data)
}
