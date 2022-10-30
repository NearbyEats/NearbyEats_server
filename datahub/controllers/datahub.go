package datahubcontrollers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nearby-eats/utils"
	"googlemaps.github.io/maps"
)

type DataHubController struct {
	mapsClient *maps.Client
	pageToken  string
}

func (h *DataHubController) Create(c *gin.Context) {
	id := uuid.New()

	defer c.JSON(http.StatusOK, map[string]string{"token": id.String()})

	err := error(nil)
	h.mapsClient, err = maps.NewClient(maps.WithAPIKey(utils.Config.PLACE_API_KEY))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	go h.handleSession(id)

}

func (h *DataHubController) handleSession(id uuid.UUID) { //sub to channel, continuously re publish anything we recieve from the channel
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: utils.Config.REDIS_URI,
	})

	pubsub := rdb.Subscribe(ctx, "client"+id.String())

	defer pubsub.Close()

	ch := pubsub.Channel()

	closeConnection := false

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		err := rdb.Publish(ctx, "datahub"+id.String(), msg.Payload).Err()
		if err != nil {
			panic(err)
		}

		switch msg.Payload {
		case "close":
			closeConnection = true
		case "updateRestaurants":
			searchResponse := h.getPlaceAPIData()
			marshaledResponse, err := json.Marshal(searchResponse)
			if err != nil {
				panic(err)
			}
			err = rdb.Publish(ctx, "datahub"+id.String(), marshaledResponse).Err()
			if err != nil {
				panic(err)
			}
		}

		if closeConnection {
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
