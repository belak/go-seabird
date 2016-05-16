package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func JSONRequest(i interface{}, format string, v ...interface{}) error {
	resp, err := http.Get(fmt.Sprintf(format, v...))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(i)
	if err != nil {
		return err
	}

	return nil
}
