package utils

import (
	"net/http"

	//nolint:misspell
	"github.com/unknwon/com"
)

// PostJSON is a simple wrapper to post and get JSON from a given url.
func PostJSON(url string, data, resp interface{}) error {
	return com.HttpPostJSON(&http.Client{}, url, data, resp)
}

// GetJSON is a simple wrapper to get a json object from a given URL.
func GetJSON(url string, resp interface{}) error {
	return com.HttpGetJSON(&http.Client{}, url, resp)
}
