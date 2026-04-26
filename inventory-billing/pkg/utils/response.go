package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginatedData struct {
	Items  interface{} `json:"items"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

func SuccessResponse(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, Response{Success: true, Message: message, Data: data})
}

func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Success: false, Message: message})
}

func PaginatedResponse(c *gin.Context, status int, message string, items interface{}, total int64, limit, offset int) {
	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    PaginatedData{Items: items, Total: total, Limit: limit, Offset: offset},
	})
}

func Pagination(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	return
}
