package datahubcontrollers

import (
	"log"
)

type HandleCasesResult struct {
	StateEventPayload StateEventPayload `type:"data" control:"ReturnState"`
	DataEventPayload  DataEventPayload  `type:"data" control:"ReturnData"`
	ReturnState       bool              `type:"control"`
	ReturnData        bool              `type:"control"`
	CloseSession      bool              `type:"control"`
	ErrorVal          bool              `type:"control"`
}

func (h *DataHubController) handleCases(c ClientPayload) HandleCasesResult {

	res := HandleCasesResult{}

	res.StateEventPayload.MessageType = "stateEvent"
	res.DataEventPayload.MessageType = "dataEvent"

	res.StateEventPayload.ClientID = c.ClientID
	res.DataEventPayload.ClientID = "allClients"

	res.ReturnState = true
	res.ReturnData = false

	switch c.RequestType {
	case "leaveSession":
		delete(h.currentUserIDs, c.ClientID)
		if len(h.currentUserIDs) == 0 {
			res.CloseSession = true
			h.cleanRedisDB()
		}
		log.Println(h.currentUserIDs)

	case "joinSession":
		h.currentUserIDs[c.ClientID] = Idle
		log.Println(h.currentUserIDs)

	case "updateRestaurants":
		res.ReturnData = true

		if h.currentUserIDs[c.ClientID] != UpdateRestaurants {
			h.updateRestaurantsCounter += 1
			h.currentUserIDs[c.ClientID] = UpdateRestaurants
		}

		if h.updateRestaurantsCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			h.updateRestaurantsCounter = 0

			res.StateEventPayload.ClientID = "allClients"

			res.DataEventPayload.PlaceApiData = h.getNewRestaurants()

			h.initializeRedisDB()
		}

		res.DataEventPayload.SessionStateData = &SessionStateDataPayload{len(h.currentUserIDs), h.startRatingCounter, h.updateRestaurantsCounter, h.finishRatingCounter}

	case "startRating":
		res.ReturnData = true

		if h.currentUserIDs[c.ClientID] != StartRating {
			h.startRatingCounter += 1
			h.currentUserIDs[c.ClientID] = StartRating
		}

		if h.startRatingCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = CurrRating
			}

			h.startRatingCounter = 0

			res.StateEventPayload.ClientID = "allClients"

			res.DataEventPayload.PlaceApiData = h.getNewRestaurants()

			h.initializeRedisDB()
		}

		res.DataEventPayload.SessionStateData = &SessionStateDataPayload{len(h.currentUserIDs), h.startRatingCounter, h.updateRestaurantsCounter, h.finishRatingCounter}

	case "finishRating":
		res.ReturnData = true

		if h.currentUserIDs[c.ClientID] != FinishRating {
			h.finishRatingCounter += 1
			h.currentUserIDs[c.ClientID] = FinishRating
		}

		if h.finishRatingCounter == len(h.currentUserIDs) {
			for key := range h.currentUserIDs {
				h.currentUserIDs[key] = Results
			}

			h.finishRatingCounter = 0

			res.StateEventPayload.ClientID = "allClients"

			res.DataEventPayload.ResultsData = &ResultsDataPayload{}
			res.DataEventPayload.ResultsData.SearchResult = append(res.DataEventPayload.ResultsData.SearchResult, h.getRatingResult())
		}

		res.DataEventPayload.SessionStateData = &SessionStateDataPayload{len(h.currentUserIDs), h.startRatingCounter, h.updateRestaurantsCounter, h.finishRatingCounter}

	case "sendResult":
		h.updateScore(c.RestaurantID)

	default:
		res.ErrorVal = true
		res.ReturnData = false
	}

	if status, found := h.currentUserIDs[c.ClientID]; found {
		res.StateEventPayload.State = status.String()
	} else {
		res.StateEventPayload.State = "closeConnection"
	}

	return res
}
