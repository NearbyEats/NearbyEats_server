package datahubcontrollers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nearby-eats/utils"
	"googlemaps.github.io/maps"
)

type DataHubController struct {
	mapsClient               *maps.Client
	redisClient              *redis.Client
	sessionID                uuid.UUID
	updateRestaurantsCounter int
	startRatingCounter       int
	finishRatingCounter      int
	placeApiData             maps.PlacesSearchResponse
	currentUserIDs           map[string]UserStatus
}

type UserStatus int

const (
	Idle = iota //ranges from 0-4
	StartRating
	CurrRating
	FinishRating
	UpdateRestaurants
	Results
)

func (us UserStatus) String() string {
	return []string{"Idle", "StartRating", "CurrRating",
		"FinishRating", "UpdateRestaurants", "Results"}[us]
}

type ClientPayload struct {
	RequestType    string `json:"requestType"`
	ClientID       string `json:"clientID"`
	RestaurantID   string `json:"restaurantID"`
	RestaurantVote string `json:"restaurantVote"`
}

func (c ClientPayload) fillDefaults() {
	v := reflect.Indirect(reflect.ValueOf(&c))
	for i := 0; i < v.NumField(); i++ {
		v.Field(i).SetString("")
	}
}

type DataHubPayload struct {
	ClientID     string
	State        string
	PlaceApiData maps.PlacesSearchResponse
	ResultsData  ResultsDataPayload
}

type ResultsDataPayload struct {
	SearchResult []maps.PlacesSearchResult
}

func (h DataHubController) Create(c *gin.Context) {
	h.sessionID = uuid.New()

	defer c.JSON(http.StatusOK, map[string]string{"token": h.sessionID.String()})

	err := error(nil)
	h.mapsClient, err = maps.NewClient(maps.WithAPIKey(utils.Config.PLACE_API_KEY))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}
	h.currentUserIDs = make(map[string]UserStatus)

	go h.handleSession()
}

func (h DataHubController) handleSession() { //sub to channel, continuously re publish anything we recieve from the channel
	ctx := context.Background()

	h.redisClient = redis.NewClient(&redis.Options{
		Addr: utils.Config.REDIS_URI,
	})

	pubsub := h.redisClient.Subscribe(ctx, "client"+h.sessionID.String())

	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		clientPayload := ClientPayload{}
		clientPayload.fillDefaults()
		err := json.Unmarshal([]byte(msg.Payload), &clientPayload)
		if err != nil {
			log.Println(err)
		}

		datahubPayload, closeSession, errorVal := h.handleCases(clientPayload)

		if errorVal {
			log.Println("ERROR ------------------")
			err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), "ERROR").Err()
			if err != nil {
				panic(err)
			}
		}

		marshaledDatahubPayload, err := json.Marshal(datahubPayload)
		if err != nil {
			panic(err)
		}

		err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), marshaledDatahubPayload).Err()
		if err != nil {
			panic(err)
		}

		if closeSession {
			break
		}

	}

}
