package clientcontrollers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	dh "github.com/nearby-eats/datahub/controllers"
)

func (h SessionController) handleDataHub(ctx context.Context) {
	pubsub := h.redisClient.Subscribe(ctx, "datahub"+h.sessionID)
	defer pubsub.Close()

	// write back the token we recieved
	message := []byte(h.sessionID)
	mt := websocket.TextMessage
	err := h.conn.WriteMessage(mt, message)
	if err != nil {
		log.Println("write token:", err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		payload := dh.DataHubPayload{}
		err := json.Unmarshal([]byte(msg.Payload), &payload)
		if err != nil {
			log.Println(err)
		}

		if payload.ClientID != h.clientID && payload.ClientID != "allClients" {
			continue
		}

		if payload.State == "closeConnection" {
			closeNormalClosure := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			err := h.conn.WriteControl(websocket.CloseMessage, closeNormalClosure, time.Now().Add(time.Second))
			if err != nil {
				log.Println("close write:", err)
			}
			break
		}

		err = h.conn.WriteMessage(mt, []byte(msg.Payload))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

	h.wg.Done()
}

func (h SessionController) handleClient(ctx context.Context) {
	isConnOpen := true

	for isConnOpen {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			isConnOpen = false
			message = []byte("{\"requestType\" : \"leaveSession\",\"clientID\" : \"" + h.clientID + "\"}")
		}

		err = h.redisClient.Publish(ctx, "client"+h.sessionID, string(message)).Err()
		if err != nil {
			panic(err)
		}
	}

	h.wg.Done()
}
