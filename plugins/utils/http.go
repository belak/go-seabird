package utils

import (
	"net/http"

	"github.com/Unknwon/com"
)

// GetJSON is a simple wrapper to get a json object from a given URL.
func GetJSON(url string, target interface{}) error {
	return com.HttpGetJSON(&http.Client{}, url, target)
}
