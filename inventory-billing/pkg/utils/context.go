package utils

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UserIDFromCtx(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user not authenticated")
	}
	id, err := uuid.Parse(val.(string))
	if err != nil {
		return uuid.Nil, errors.New("invalid user id in context")
	}
	return id, nil
}

func UserRoleFromCtx(c *gin.Context) string {
	val, _ := c.Get("user_role")
	role, _ := val.(string)
	return role
}
