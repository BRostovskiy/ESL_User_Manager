package common

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"
)

type Request interface {
	Decode(r *http.Request) error
}

type Response interface {
	WriteTo(w http.ResponseWriter) error
}

func RespondObject(w http.ResponseWriter, status int, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set(HeaderContentLength, strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}
