package clienthandler

import (
	"github.com/gin-gonic/gin"
	clientcontrollers "github.com/nearby-eats/clienthandler/controllers"
)

func SetUpV1(router *gin.Engine) {
	session := new(clientcontrollers.SessionController)
	v1 := router.Group("v1")
	sessionRoute := v1.Group("session")
	sessionRoute.GET("/create", session.Create)
	sessionRoute.GET("/join/:token", session.Join)
}
