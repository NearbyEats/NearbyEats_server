package datahubcontrollers

import (
	"context"
	"log"

	"googlemaps.github.io/maps"
)

func (h *DataHubController) getPlaceAPIData() *maps.PlacesSearchResponse {
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
		return &maps.PlacesSearchResponse{}
	}

	return &response
}

func (h *DataHubController) getNewRestaurants() *maps.PlacesSearchResponse {
	if len(h.placeApiData.Results) == 0 {
		h.placeApiData = *h.getPlaceAPIData()
		log.Println("had to do new api call, len results: ", len(h.placeApiData.Results))
	}

	searchResponse := h.placeApiData
	searchResponse.Results = searchResponse.Results[:10] // get only first ten results

	h.placeApiData.Results = h.placeApiData.Results[11:] // remove first 10 results

	h.initializeRedisDB(searchResponse)

	return &searchResponse
}
