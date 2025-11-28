package flight

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FlightHandler struct {
	service *Service
}

func NewFlightHandler(s *Service) *FlightHandler {
	return &FlightHandler{
		service: s,
	}
}

func (h *FlightHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/v1/flights/search", h.SearchFlightsHandler)
}

func (h *FlightHandler) SearchFlightsHandler(c *gin.Context) {
	var req SearchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request format: %v", err),
		})
		return
	}

	response, err := h.service.SearchFlights(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Flight search failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
