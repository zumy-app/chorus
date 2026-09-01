package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type ReleaseGateHandler struct {
	svc *services.ReleaseGateService
}

func NewReleaseGateHandler(svc *services.ReleaseGateService) *ReleaseGateHandler {
	return &ReleaseGateHandler{svc: svc}
}

func (h *ReleaseGateHandler) GetReadiness(c *gin.Context) {
	report := h.svc.Evaluate()
	status := http.StatusOK
	if !report.Overall {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, report)
}
