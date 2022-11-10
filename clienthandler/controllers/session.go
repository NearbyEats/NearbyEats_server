package clientcontrollers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/nearby-eats/utils"
)

type SessionController struct {
	conn        *websocket.Conn
	redisClient *redis.Client
	wg          *sync.WaitGroup
	sessionID   string
	clientID    string
}

type createDataPayload struct {
	Token string `json:"token"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header["Origin"]
		log.Println("Upgrading connection origin: ", origin)
		return true
	},
}

func (h SessionController) Create(c *gin.Context) {
	config := utils.Config

	url := "http://localhost:" + config.DATA_HUB_PORT + "/v1/datahub/create"

	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(body))

	data_obj := createDataPayload{}

	err = json.Unmarshal(body, &data_obj)
	if err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, data_obj)
}

func (h SessionController) Join(c *gin.Context) {
	// upgrade the connection to WebSocket
	w, r := c.Writer, c.Request
	var err error
	h.conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer h.conn.Close()

	h.redisClient = redis.NewClient(&redis.Options{
		Addr: utils.Config.REDIS_URI,
	})

	h.wg = &sync.WaitGroup{}

	h.sessionID = c.Param("sessionID")

	h.clientID = c.Param("clientID")

	ctx := context.Background()

	h.wg.Add(1)
	go h.handleDataHub(ctx)

	h.wg.Add(1)
	go h.handleClient(ctx)

	h.wg.Wait()
}
