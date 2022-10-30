package datahubcontrollers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	databasehandler "github.com/nearby-eats/datahub/mongodb"
)

type HealthController struct{}

func (h HealthController) Status(c *gin.Context) {
	c.String(http.StatusOK, "Server Functional!")
	dbHandler := databasehandler.NewDatabaseHandler()
	dbHandler.Connect()
	defer dbHandler.Disconnect()
	dbHandler.ListDatabaseNames()
}
