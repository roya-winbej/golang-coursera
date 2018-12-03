package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func (srv *MyApi) m(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		func(w http.ResponseWriter, r *http.Request) {}(w, r)
		srv.wrapperDoSomeJob(w, r)
	default:
		// 404
	}
}

func (h *MyApi) wrapperDoSomeJob(w http.ResponseWriter, r *http.Request) {

	params, ok := r.URL.Query()["login"]

	if !ok || len(params[0]) < 1 {
		log.Println("Url Param 'login' is missing")
		return
	}

	str := ProfileParams{
		Login: params[0],
	}

	// заполнение структуры params
	// валидирование параметров
	res, err := h.Profile(r.Context(), str)
	if err != nil {
		http.Error(w, err.Error(), err.(ApiError).HTTPStatus)
	}


	encoder := json.NewEncoder(w)

	type APIRES struct {
		Error string `json:"error"`
		Response interface{} `json:"response, omitempty"`
	}

	some := &APIRES{
		Error: "",
		Response: res,
	}

	_ = encoder.Encode(some)
}
