package datahub

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	datahubcontrollers "github.com/nearby-eats/datahub/controllers"
)

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "POST", "DELETE", "GET"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowWildcard:    true,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	health := new(datahubcontrollers.HealthController)

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello from nearby-eats datahub API"})
	})
	router.GET("/health", health.Status)
	// router.Use(middlewares.AuthMiddleware())

	SetUpV1(router)
	return router

}
