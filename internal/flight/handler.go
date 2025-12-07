package flight

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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

func (h *FlightHandler) SearchFlightsHandler(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON body",
			"code":  ErrorCodeValidation,
		})
		return
	}

	response, err := h.service.SearchFlights(c.Request.Context(), req)
	if err != nil {
		sendError(c, err)
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
		sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func sendError(c *gin.Context, err error) {
	var appErr *AppError

	if errors.As(err, &appErr) {
		c.JSON(appErr.Status, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return
	}

	// Default to 500 for unknown errors
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"code":    ErrorCodeInternalFailure,
		"details": err.Error(),
	})
}

func (s *Service) FilterFlights(ctx context.Context, req FilterRequest) (*FlightSearchResponse, error) {
	startTime := time.Now()
	if err := req.SearchRequest.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	flights, metadata, err := s.getOrFetchFlights(ctx, req.SearchRequest)
	if err != nil {
		return nil, err
	}
	if req.Filters != nil {
		flights = s.applyFilters(flights, *req.Filters)
	}
	if req.Sort != nil {
		flights = s.applySorting(flights, *req.Sort)
	}
	metadata.TotalResults = uint32(len(flights))
	metadata.SearchTimeMs = uint32(time.Since(startTime).Milliseconds())

	return &FlightSearchResponse{
		SearchCriteria: req.SearchRequest,
		Metadata:       metadata,
		Flights:        flights,
	}, nil
}

func (s *Service) SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	flights, metadata, err := s.getOrFetchFlights(ctx, req)
	if err != nil {
		return nil, err
	}

	return &FlightSearchResponse{
		SearchCriteria: req,
		Metadata:       metadata,
		Flights:        flights,
	}, nil
}
