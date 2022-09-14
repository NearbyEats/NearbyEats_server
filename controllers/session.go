package controllers

import (
	"github.com/gin-gonic/gin"
)

type SessionController struct{}

func (h SessionController) Create(c *gin.Context) {
	c.JSON(200, map[string]string{"token": "test"})
}
