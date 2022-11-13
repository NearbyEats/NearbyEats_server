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

type StateEventPayload struct {
	ClientID string
	State    string
}

type DataEventPayload struct {
	ClientID         string
	PlaceApiData     *maps.PlacesSearchResponse `json:",omitempty"`
	ResultsData      *ResultsDataPayload        `json:",omitempty"`
	SessionStateData *SessionStateDataPayload   `json:",omitempty"`
}

type ResultsDataPayload struct {
	SearchResult []maps.PlacesSearchResult
}

type SessionStateDataPayload struct {
	NumStartRating       int
	NumUpdateRestaurants int
	NumFinishRating      int
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

		handleCasesResult := h.handleCases(clientPayload)

		if handleCasesResult.ErrorVal {
			log.Println("ERROR ------------------")
		}

		h.checkAndReturnPayloads(ctx, handleCasesResult)

		if handleCasesResult.CloseSession {
			break
		}

	}

}

func (h *DataHubController) checkAndReturnPayloads(ctx context.Context, handleCasesResult HandleCasesResult) {
	v := reflect.Indirect(reflect.ValueOf(&handleCasesResult))
	vt := reflect.TypeOf(handleCasesResult)
	for i := 0; i < v.NumField(); i++ {
		if vt.Field(i).Tag.Get("type") == "data" {
			if v.FieldByName(vt.Field(i).Tag.Get("control")).Bool() {

				marshaledPayload, err := json.Marshal(v.Field(i).Interface())
				if err != nil {
					panic(err)
				}

				err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), marshaledPayload).Err()
				if err != nil {
					panic(err)
				}

			}
		}
	}
}
