package datahubcontrollers

import (
	"log"
)

func (h *DataHubController) handleCases(c ClientPayload) (DataHubPayload, bool, bool) {

	datahubPayload := DataHubPayload{}
	closeConnection := false
	errorVal := false

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			closeConnection = true
			h.cleanRedisDB()
		}
		log.Println(h.currentUserIDs)
		datahubPayload.State = UserStatus.String(3)

	case "joinSession":
		h.currentUserIDs[c.ClientID] = Idle
		log.Println(h.currentUserIDs)
		datahubPayload.State = UserStatus.String(0)

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

			h.initializeRedisDB()
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

			h.initializeRedisDB()
		}

	case "finishRating":
		datahubPayload.State = UserStatus.String(3)
		if h.currentUserIDs[c.ClientID] != FinishRating {
			h.finishRatingCounter += 1
			h.currentUserIDs[c.ClientID] = FinishRating
		}

		if h.finishRatingCounter == len(h.currentUserIDs) {
			datahubPayload.ResultsData.SearchResult = append(datahubPayload.ResultsData.SearchResult, h.getRatingResult())
			log.Println("FINISHED RATING -----------------------")
		}

	case "sendResult":
		datahubPayload.State = UserStatus.String(2)
		h.updateScore(c.RestaurantID)

	default:
		closeConnection = true
		errorVal = true
	}

	return datahubPayload, closeConnection, errorVal
}
