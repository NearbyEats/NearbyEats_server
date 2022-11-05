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
	pageToken                string
	updateRestaurantsCounter int
	startRatingCounter       int
	finishRatingCounter      int
	currentUserIDs           map[string]UserStatus
}

type UserStatus int

const (
	Idle = iota //ranges from 0-4
	StartRating
	CurrRating
	FinishRating
	UpdateRestaurants
)

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
	State        string
	PlaceApiData maps.PlacesSearchResponse
	ResultsData  ResultsDataPayload
}

type ResultsDataPayload struct {
	SearchResult maps.PlacesSearchResult
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

	closeConnection := false

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		clientPayload := ClientPayload{}
		clientPayload.fillDefaults()
		err := json.Unmarshal([]byte(msg.Payload), &clientPayload)
		if err != nil {
			log.Println(err)
		}

		switch clientPayload.RequestType {
		case "leaveSession":
			delete(h.currentUserIDs, clientPayload.ClientID)
			if len(h.currentUserIDs) == 0 {
				closeConnection = true
			}
			log.Println(h.currentUserIDs)

		case "joinSession":
			h.currentUserIDs[clientPayload.ClientID] = Idle
			log.Println(h.currentUserIDs)

		case "updateRestaurants":
			if h.currentUserIDs[clientPayload.ClientID] != UpdateRestaurants {
				h.updateRestaurantsCounter += 1
				h.currentUserIDs[clientPayload.ClientID] = UpdateRestaurants
			}

			if h.updateRestaurantsCounter == len(h.currentUserIDs) {
				for key := range h.currentUserIDs {
					h.currentUserIDs[key] = CurrRating
				}

				h.updateRestaurantsCounter = 0

				h.sendNewRestaurants()
			}

		case "startRating":
			if h.currentUserIDs[clientPayload.ClientID] != StartRating {
				h.startRatingCounter += 1
				h.currentUserIDs[clientPayload.ClientID] = StartRating
			}

			if h.startRatingCounter == len(h.currentUserIDs) {
				for key := range h.currentUserIDs {
					h.currentUserIDs[key] = CurrRating
				}

				h.startRatingCounter = 0

				h.sendNewRestaurants()
			}

		case "finishRating":
			if h.currentUserIDs[clientPayload.ClientID] != FinishRating {
				h.finishRatingCounter += 1
				h.currentUserIDs[clientPayload.ClientID] = FinishRating
			}

			if h.finishRatingCounter == len(h.currentUserIDs) {
				// h.sendResults()
				log.Println("FINISHED RATING -----------------------")
			}

		case "sendRating":
			log.Println(h.currentUserIDs)
		}

		if closeConnection {
			log.Println("CLOSING CONNECTION ------------------")
			break
		}
	}

}

func (h *DataHubController) getPlaceAPIData() maps.PlacesSearchResponse {
	r := &maps.NearbySearchRequest{
		Location: &maps.LatLng{
			Lat: 43.475074,
			Lng: -80.543213,
		},
		Radius:  10000,
		OpenNow: true,
		Type:    maps.PlaceTypeRestaurant,
		RankBy:  maps.RankByProminence,
	}

	if h.pageToken != "" {
		r.PageToken = h.pageToken
	}

	response, err := h.mapsClient.NearbySearch(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
		return maps.PlacesSearchResponse{}
	}

	h.pageToken = response.NextPageToken

	return response
}

func (h *DataHubController) sendNewRestaurants() {
	ctx := context.Background()

	searchResponse := h.getPlaceAPIData()

	marshaledResponse, err := json.Marshal(searchResponse)
	if err != nil {
		panic(err)
	}

	err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), marshaledResponse).Err()
	if err != nil {
		panic(err)
	}
}
