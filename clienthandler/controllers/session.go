package clientcontrollers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

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

	h.redisClient = redis.NewClient(&redis.Options{
		Addr: utils.Config.REDIS_URI,
	})

	h.wg = &sync.WaitGroup{}

	h.sessionID = c.Param("sessionID")

	ctx := context.Background()

	h.handleDataHub(c, ctx)
}

func (h SessionController) handleDataHub(c *gin.Context, ctx context.Context) {
	defer h.conn.Close()

	pubsub := h.redisClient.Subscribe(ctx, "datahub"+h.sessionID)

	defer pubsub.Close()

	h.wg.Add(1)

	go h.handleClient(c, ctx)

	// write back the token we recieved
	message := []byte(h.sessionID)
	mt := websocket.TextMessage
	err := h.conn.WriteMessage(mt, message)
	if err != nil {
		log.Println("write token:", err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		if msg.Payload == "closeConnection" {
			closeNormalClosure := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			err := h.conn.WriteControl(websocket.CloseMessage, closeNormalClosure, time.Now().Add(time.Second))
			if err != nil {
				log.Println("close write:", err)
			}
			h.conn.Close()
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

func (h SessionController) handleClient(c *gin.Context, ctx context.Context) {
	defer h.wg.Done()

	isConnOpen := true

	for isConnOpen {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			isConnOpen = false
			message = []byte("{\"requestType\": \"leaveSession\"}")
		}

		err = h.redisClient.Publish(ctx, "client"+h.sessionID, string(message)).Err()
		if err != nil {
			panic(err)
		}

	}
}
