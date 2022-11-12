package datahubcontrollers

import (
	"log"
)

type HandleCasesResult struct {
	stateEventPayload StateEventPayload `type:"data" control:"returnState"`
	dataEventPayload  DataEventPayload  `type:"data" control:"returnData"`
	returnState       bool              `type:"control"`
	returnData        bool              `type:"control"`
	closeSession      bool              `type:"control"`
	errorVal          bool              `type:"control"`
}

func (h *DataHubController) handleCases(c ClientPayload) HandleCasesResult {

	res := HandleCasesResult{}

	res.stateEventPayload.ClientID = c.ClientID

	res.returnData = true
	res.returnState = true

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			res.closeSession = true
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

			res.stateEventPayload.ClientID = "allClients"

			h.updateRestaurantsCounter = 0
			res.dataEventPayload.PlaceApiData = h.getNewRestaurants()

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

			h.startRatingCounter = 0

			res.stateEventPayload.ClientID = "allClients"
			res.dataEventPayload.PlaceApiData = h.getNewRestaurants()

			h.initializeRedisDB()
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

			res.stateEventPayload.ClientID = "allClients"
			res.dataEventPayload.ResultsData.SearchResult = append(res.dataEventPayload.ResultsData.SearchResult, h.getRatingResult())
		}

	case "sendResult":
		h.updateScore(c.RestaurantID)

	default:
		res.errorVal = true
	}

	if status, found := h.currentUserIDs[c.ClientID]; found {
		res.stateEventPayload.State = status.String()
	} else {
		res.stateEventPayload.State = "closeConnection"
	}

	return res
}
