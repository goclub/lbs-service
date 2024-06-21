package main

import (
	xjson "github.com/goclub/json"
	"net/http"
)

func writeError(resp http.ResponseWriter, msg string) {
	var body []byte
	var err error
	if body, err = xjson.Marshal(map[string]any{
		"status":  1,
		"message": msg,
	}); err != nil {
		resp.Write([]byte(err.Error()))
		return
	}
	resp.Write(body)
}