package server

import (
	"github.com/gin-gonic/gin"
	"github.com/nearby-eats/controllers"
)

func SetUpV1(router *gin.Engine) {
	session := new(controllers.SessionController)
	v1 := router.Group("v1")
	sessionRoute := v1.Group("session")
	sessionRoute.GET("/", session.Create)
}
