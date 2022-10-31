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
}

type createData struct {
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

	data_obj := createData{}

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

	h.redisClient = redis.NewClient(&redis.Options{
		Addr: utils.Config.REDIS_URI,
	})

	h.wg = &sync.WaitGroup{}

	h.handleDataHub(c)
}

func (h SessionController) handleDataHub(c *gin.Context) {
	defer h.conn.Close()

	token := c.Param("token")

	ctx := context.Background()

	pubsub := h.redisClient.Subscribe(ctx, "datahub"+token)

	defer pubsub.Close()

	h.wg.Add(1)

	go h.handleClient(c)

	// write back the token we recieved
	message := []byte(token)
	mt := websocket.TextMessage
	err := h.conn.WriteMessage(mt, message)
	if err != nil {
		log.Println("write token:", err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		if msg.Payload == "close" {
			break
		}
		err = h.conn.WriteMessage(mt, []byte(msg.Payload))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

	h.wg.Wait()
}

func (h SessionController) handleClient(c *gin.Context) {
	defer h.wg.Done()

	ctx := context.Background()

	token := c.Param("token")
	closeConnection := false

	for {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			closeConnection = true
			message = []byte("{requestType: close}")
		}

		err = h.redisClient.Publish(ctx, "client"+token, string(message)).Err()
		if err != nil {
			panic(err)
		}

		if closeConnection {
			break
		}
	}
}
