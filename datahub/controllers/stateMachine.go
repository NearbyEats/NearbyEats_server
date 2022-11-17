package datahubcontrollers

import (
	"log"
)

func (h *DataHubController) handleCases(c ClientPayload) (DataHubPayload, bool, bool) {

	datahubPayload := DataHubPayload{}
	errorVal := false
	closeSession := false

	datahubPayload.ClientID = c.ClientID

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			closeSession = true
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

			h.startRatingCounter = 0

			datahubPayload.ClientID = "allClients"
			datahubPayload.PlaceApiData = h.getNewRestaurants()
		}

	case "finishRating":
		if h.currentUserIDs[c.ClientID] != FinishRating {
			h.finishRatingCounter += 1
			h.currentUserIDs[c.ClientID] = FinishRating
		}

		if h.finishRatingCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = Results
			}

			h.finishRatingCounter = 0

			datahubPayload.ClientID = "allClients"
			datahubPayload.ResultsData.SearchResult = append(datahubPayload.ResultsData.SearchResult, h.getRatingResult())
		}

	case "sendResult":
		datahubPayload.ClientID = ""
		h.updateScore(c.RestaurantID)

	default:
		errorVal = true
	}

	if status, found := h.currentUserIDs[c.ClientID]; found {
		datahubPayload.State = status.String()
	} else {
		datahubPayload.State = "closeConnection"
	}

	return datahubPayload, closeSession, errorVal
}
