package datahubcontrollers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type DataHubController struct{}

func (h DataHubController) Create(c *gin.Context) {
	id := uuid.New()

	defer c.JSON(http.StatusOK, map[string]string{"token": id.String()})

	go handleSession(id)

}

func handleSession(id uuid.UUID) { //sub to channel, continuously re publish anything we recieve from the channle
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	pubsub := rdb.Subscribe(ctx, "client"+id.String())

	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		err := rdb.Publish(ctx, "datahub"+id.String(), msg.Payload).Err()
		if err != nil {
			panic(err)
		}
	}
}
