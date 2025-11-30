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
	router.POST("/v1/flights/filter", h.FilterFlightsHandler)
}

// SearchFlightsHandler godoc
// @Summary      Search for flights
// @Description  Search flights based on origin, destination, and dates
// @Tags         flights
// @Accept       json
// @Produce      json
// @Param        request body SearchRequest true "Flight Search Criteria"
// @Success      200 {object} map[string]interface{} "Replace this with your actual Response Struct"
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /v1/flights/search [post]
func (h *FlightHandler) SearchFlightsHandler(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request format: %v", err),
		})
		return
	}

	response, err := h.service.SearchFlights(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Flight search failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// FilterFlightsHandler godoc
// @Summary      Filter existing flight results
// @Description  Apply filters like price range, airline, or transit
// @Tags         flights
// @Accept       json
// @Produce      json
// @Param        request body FilterRequest true "Filter Criteria"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Router       /v1/flights/filter [post]
func (h *FlightHandler) FilterFlightsHandler(c *gin.Context) {
	var req FilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request format: %v", err),
		})
		return
	}

	response, err := h.service.FilterFlights(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Flight filter failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
