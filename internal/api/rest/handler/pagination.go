package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page     int
	PageSize int
}

// PaginationMeta holds pagination metadata returned in responses.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse is the standardized paginated response envelope.
// Used for OpenAPI/Swagger schema generation.
type PaginatedResponse struct {
	Data       any            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// Offset returns the database offset
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ParsePagination parses pagination parameters from context
func ParsePagination(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// NewPaginationResponse creates a standardized pagination response
func NewPaginationResponse(data any, params PaginationParams, total int64) gin.H {
	totalPages := (int(total) + params.PageSize - 1) / params.PageSize

	return gin.H{
		"data": data,
		"pagination": gin.H{
			"page":        params.Page,
			"page_size":   params.PageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}

// NewTypedPaginationResponse creates a PaginatedResponse struct value
// (schema-compatible; currently reserved for typed responses).
func NewTypedPaginationResponse(data any, params PaginationParams, total int64) PaginatedResponse {
	totalPages := (int(total) + params.PageSize - 1) / params.PageSize
	return PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
