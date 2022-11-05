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
)

func (us UserStatus) String() string {
	return []string{"Idle", "StartRating", "CurrRating",
		"FinishRating", "UpdateRestaurants"}[us]
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

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		clientPayload := ClientPayload{}
		clientPayload.fillDefaults()
		err := json.Unmarshal([]byte(msg.Payload), &clientPayload)
		if err != nil {
			log.Println(err)
		}

		datahubPayload, closeConnection, errorVal := h.handleCases(clientPayload)

		if closeConnection {
			log.Println("CLOSING CONNECTION ------------------")
			if errorVal {
				log.Println("ERROR ------------------")
				err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), "ERROR").Err()
				if err != nil {
					panic(err)
				}
			}

			err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), "closeConnection").Err()
			if err != nil {
				panic(err)
			}

			break
		}

		marshaledDatahubPayload, err := json.Marshal(datahubPayload)
		if err != nil {
			panic(err)
		}

		err = h.redisClient.Publish(ctx, "datahub"+h.sessionID.String(), marshaledDatahubPayload).Err()
		if err != nil {
			panic(err)
		}

	}

}

func (h *DataHubController) handleCases(c ClientPayload) (DataHubPayload, bool, bool) {

	datahubPayload := DataHubPayload{}
	closeConnection := false
	errorVal := false

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			closeConnection = true
		}
		log.Println(h.currentUserIDs)
		datahubPayload.State = UserStatus.String(3)

	case "joinSession":
		h.currentUserIDs[c.ClientID] = Idle
		log.Println(h.currentUserIDs)
		datahubPayload.State = UserStatus.String(1)

	case "updateRestaurants":
		datahubPayload.State = UserStatus.String(4)
		if h.currentUserIDs[c.ClientID] != UpdateRestaurants {
			h.updateRestaurantsCounter += 1
			h.currentUserIDs[c.ClientID] = UpdateRestaurants
		}

		if h.updateRestaurantsCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			h.updateRestaurantsCounter = 0
			datahubPayload.State = UserStatus.String(4)
			datahubPayload.PlaceApiData = h.getNewRestaurants()
		}

	case "startRating":
		datahubPayload.State = UserStatus.String(1)
		if h.currentUserIDs[c.ClientID] != StartRating {
			h.startRatingCounter += 1
			h.currentUserIDs[c.ClientID] = StartRating
		}

		if h.startRatingCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			h.startRatingCounter = 0
			datahubPayload.State = UserStatus.String(2)
			datahubPayload.PlaceApiData = h.getNewRestaurants()
		}

	case "finishRating":
		datahubPayload.State = UserStatus.String(3)
		if h.currentUserIDs[c.ClientID] != FinishRating {
			h.finishRatingCounter += 1
			h.currentUserIDs[c.ClientID] = FinishRating
		}

		if h.finishRatingCounter == len(h.currentUserIDs) {
			// h.sendResults()
			log.Println("FINISHED RATING -----------------------")
		}

	case "sendResult":
		log.Println(h.currentUserIDs)

	default:
		closeConnection = true
		errorVal = true
	}

	return datahubPayload, closeConnection, errorVal
}

func (h *DataHubController) getPlaceAPIData() maps.PlacesSearchResponse {
	r := &maps.NearbySearchRequest{
		Location: &maps.LatLng{
			Lat: 43.475074,
			Lng: -80.543213,
		},
		Radius:    10000,
		OpenNow:   true,
		Type:      maps.PlaceTypeRestaurant,
		RankBy:    maps.RankByProminence,
		PageToken: h.placeApiData.NextPageToken,
	}

	response, err := h.mapsClient.NearbySearch(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
		return maps.PlacesSearchResponse{}
	}

	return response
}

func (h *DataHubController) getNewRestaurants() maps.PlacesSearchResponse {
	if len(h.placeApiData.Results) == 0 {
		h.placeApiData = h.getPlaceAPIData()
		log.Println("had to do new api call, len results: ", len(h.placeApiData.Results))
	}

	searchResponse := h.placeApiData
	searchResponse.Results = searchResponse.Results[:9] // get only first ten results

	h.placeApiData.Results = h.placeApiData.Results[10:] // remove first 10 results

	return searchResponse
}
