package datahub

import (
	"github.com/gin-gonic/gin"
	datahubcontrollers "github.com/nearby-eats/datahub/controllers"
)

func SetUpV1(router *gin.Engine) {
	datahub := new(datahubcontrollers.DataHubController)
	v1 := router.Group("v1")
	datahubRoute := v1.Group("datahub")
	datahubRoute.GET("/create", datahub.Create)
}
