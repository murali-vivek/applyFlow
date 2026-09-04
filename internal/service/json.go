package service

import (
	"encoding/json"
	"io"
	"net/http"
)

func decodeJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
