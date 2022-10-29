package clienthandler

import (
	"github.com/gin-gonic/gin"
	"github.com/nearby-eats/utils"
)

func Init() {
	config := utils.Config
	if config.ENVIRONMENT == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := NewRouter()
	r.Run(":" + config.CLIENT_HANDLER_PORT)
}
