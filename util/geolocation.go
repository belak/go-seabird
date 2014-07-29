package util

import (
	"errors"
	"encoding/json"
	"net/url"
	"net/http"
)

type LocationResult struct {
}

type LocationResponse struct {
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

type Location struct {
	Address string
	Lat float64
	Lon float64
}

func FetchLocation(where string) (*Location, error) {
	if where == ""{
		return nil, errors.New("Empty query string")
	}

	v := url.Values{}
	v.Set("address", where)
	v.Set("sensor", "false")

	u, _ := url.Parse("http://maps.googleapis.com/maps/api/geocode/json")
	u.RawQuery = v.Encode()

	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	loc := LocationResponse{}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	dec.Decode(&loc)

	if len(loc.Results) == 0 {
		return nil, errors.New("No location results found")
	} else if len(loc.Results) > 1 {
		// TODO: display results
		return nil, errors.New("More than 1 result")
	}

	ret := Location{
		Address: loc.Results[0].Address,
		Lat: loc.Results[0].Geometry.Location.Lat,
		Lon: loc.Results[0].Geometry.Location.Lon}

	return &ret, nil
}

