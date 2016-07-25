package plugins

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/Unknwon/com"
)

type locationResponse struct {
	Results []struct {
		Address  string `json:"formatted_address"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

// Location represents basic location data as queried from the Google
// geocoding api
type Location struct {
	Address string
	Lat     float64
	Lon     float64
}

// FetchLocation takes a string and attempts to find a single location
// using Google's geocoder
func FetchLocation(where string) (*Location, error) {
	if where == "" {
		return nil, errors.New("Empty query string")
	}

	v := url.Values{}
	v.Set("address", where)
	v.Set("sensor", "false")

	u, _ := url.Parse("http://maps.googleapis.com/maps/api/geocode/json")
	u.RawQuery = v.Encode()

	loc := locationResponse{}
	err := com.HttpGetJSON(&http.Client{}, u.String(), loc)
	if err != nil {
		return nil, err
	} else if len(loc.Results) == 0 {
		return nil, errors.New("No location results found")
	} else if len(loc.Results) > 1 {
		// TODO: display results
		return nil, errors.New("More than 1 result")
	}

	ret := Location{
		Address: loc.Results[0].Address,
		Lat:     loc.Results[0].Geometry.Location.Lat,
		Lon:     loc.Results[0].Geometry.Location.Lon,
	}

	return &ret, nil
}
