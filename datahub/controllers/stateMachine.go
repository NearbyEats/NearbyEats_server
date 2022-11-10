package datahubcontrollers

import (
	"log"
)

func (h *DataHubController) handleCases(c ClientPayload) (DataHubPayload, bool, bool) {

	datahubPayload := DataHubPayload{}
	closeConnection := false
	errorVal := false

	datahubPayload.ClientID = c.ClientID

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			closeConnection = true
			h.cleanRedisDB()
		}
		log.Println(h.currentUserIDs)

	case "joinSession":
		h.currentUserIDs[c.ClientID] = Idle
		log.Println(h.currentUserIDs)

	case "updateRestaurants":
		if h.currentUserIDs[c.ClientID] != UpdateRestaurants {
			h.updateRestaurantsCounter += 1
			h.currentUserIDs[c.ClientID] = UpdateRestaurants
		}

		if h.updateRestaurantsCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			datahubPayload.ClientID = "allClients"

			h.updateRestaurantsCounter = 0
			datahubPayload.PlaceApiData = h.getNewRestaurants()

			h.initializeRedisDB()
		}

	case "startRating":
		if h.currentUserIDs[c.ClientID] != StartRating {
			h.startRatingCounter += 1
			h.currentUserIDs[c.ClientID] = StartRating
		}

		if h.startRatingCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			datahubPayload.ClientID = "allClients"

			h.startRatingCounter = 0
			datahubPayload.PlaceApiData = h.getNewRestaurants()

			h.initializeRedisDB()
		}

	case "finishRating":
		if h.currentUserIDs[c.ClientID] != FinishRating {
			h.finishRatingCounter += 1
			h.currentUserIDs[c.ClientID] = FinishRating
		}

		if h.finishRatingCounter == len(h.currentUserIDs) {
			datahubPayload.ClientID = "allClients"
			datahubPayload.ResultsData.SearchResult = append(datahubPayload.ResultsData.SearchResult, h.getRatingResult())
		}

	case "sendResult":
		h.updateScore(c.RestaurantID)

	default:
		closeConnection = true
		errorVal = true
	}

	datahubPayload.State = h.currentUserIDs[c.ClientID].String()

	return datahubPayload, closeConnection, errorVal
}
