package datahubcontrollers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/go-redis/redis/v8"
	"googlemaps.github.io/maps"
)

func (h *DataHubController) initializeRedisDB() {
	ctx := context.Background()

	h.cleanRedisDB()

	for _, restaurant := range h.placeApiData.Results {
		err := h.redisClient.ZAdd(ctx, h.sessionID.String()+"set", &redis.Z{
			Score:  0,
			Member: restaurant.PlaceID,
		}).Err()
		if err != nil {
			panic(err)
		}

		marshaledRestaurant, err := json.Marshal(restaurant)
		if err != nil {
			panic(err)
		}

		err = h.redisClient.HSet(ctx, h.sessionID.String()+"hash", restaurant.PlaceID, string(marshaledRestaurant)).Err()
		if err != nil {
			panic(err)
		}
	}
}

func (h *DataHubController) cleanRedisDB() {
	ctx := context.Background()

	err := h.redisClient.Del(ctx, h.sessionID.String()+"set", h.sessionID.String()+"hash").Err()
	if err != nil {
		panic(err)
	}
}

func (h *DataHubController) updateScore(PlaceID string) {
	ctx := context.Background()

	err := h.redisClient.ZIncrBy(ctx, h.sessionID.String()+"set", 1, PlaceID).Err()
	if err != nil {
		panic(err)
	}
}

func (h *DataHubController) getRatingResult() maps.PlacesSearchResult {
	ctx := context.Background()

	key, err := h.redisClient.ZRevRange(ctx, h.sessionID.String()+"set", 0, 0).Result()
	if err != nil {
		panic(err)
	}

	result, err := h.redisClient.HGet(ctx, h.sessionID.String()+"hash", key[0]).Result()
	if err != nil {
		log.Println(err)
	}

	marshaledResult := maps.PlacesSearchResult{}
	err = json.Unmarshal([]byte(result), &marshaledResult)
	if err != nil {
		log.Println(err)
	}

	return marshaledResult

}
