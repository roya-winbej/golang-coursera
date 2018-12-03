
		package main

		import (
			"encoding/json"
			"fmt"
			"io/ioutil"
			"net/http"
			"net/url"
		)

	

		func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
				
					case "/user/profile":
					func(w http.ResponseWriter, r *http.Request) {

						fnParams := ProfileParams{}
						var queryString string

		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			defer r.Body.Close()

			queryString = string(body)

		} else {
			queryString = r.URL.RawQuery
		}


	q, _ := url.ParseQuery(queryString)

	values := make(map[string]string)
	for key := range q {
		values[key] = q.Get(key)
	}
	
	JSON, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	
	_ = json.Unmarshal(JSON, &fnParams)

	if fnParams.Login == "" {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "login must me not empty",
						})
		
						return
				}

				


						res, err := srv.Profile(r.Context(), fnParams)
						if err != nil {
							http.Error(w, err.Error(), err.(ApiError).HTTPStatus)
						}

						encoder := json.NewEncoder(w)

						_ = encoder.Encode(&struct{
							Error string `json:"error"`
							Response interface{}`json:"response, omitempty"`
						}{
							Error: "",
							Response: res,
						})

					}(w, r)

				
					case "/user/create":
					func(w http.ResponseWriter, r *http.Request) {

						fnParams := CreateParams{}
						var queryString string

		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			defer r.Body.Close()

			queryString = string(body)

			fmt.Println(queryString)

		} else {
			queryString = r.URL.RawQuery
		}


	q, _ := url.ParseQuery(queryString)

	values := make(map[string]string)
	for key := range q {
		values[key] = q.Get(key)
	}


	JSON, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}

						fmt.Printf("%v", string(JSON))


						_ = json.Unmarshal(JSON, &fnParams)

	fmt.Printf("%v", fnParams)

	if fnParams.Login == "" {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "login must me not empty",
						})
		
						return
				}

				if len(fnParams.Login) <= 10 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "login len must be >= 10",
						})
		
						return
				}

	fmt.Println(fnParams.Age)

				if fnParams.Age <= 0 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "age must be >= 0",
						})
		
						return
				}

				if fnParams.Age >= 128 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "age must be <= 128",
						})
		
						return
				}

				


						res, err := srv.Create(r.Context(), fnParams)
						if err != nil {
							http.Error(w, err.Error(), err.(ApiError).HTTPStatus)
						}

						encoder := json.NewEncoder(w)

						_ = encoder.Encode(&struct{
							Error string `json:"error"`
							Response interface{}`json:"response, omitempty"`
						}{
							Error: "",
							Response: res,
						})

					}(w, r)

				
					default:
					// 404
			}
		}
	
		func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
				
					case "/user/create":
					func(w http.ResponseWriter, r *http.Request) {

						fnParams := OtherCreateParams{}
						var queryString string

		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			defer r.Body.Close()

			queryString = string(body)

		} else {
			queryString = r.URL.RawQuery
		}


	q, _ := url.ParseQuery(queryString)

	values := make(map[string]string)
	for key := range q {
		values[key] = q.Get(key)
	}
	
	JSON, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	
	_ = json.Unmarshal(JSON, &fnParams)

	if fnParams.Username == "" {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "login must me not empty",
						})
		
						return
				}

				if len(fnParams.Username) <= 3 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "username len must be >= 3",
						})
		
						return
				}

				if fnParams.Level <= 1 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "level must be >= 1",
						})
		
						return
				}

				if fnParams.Level <= 50 {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `json:"error"`
						}{
							Error: "level must be <= 50",
						})
		
						return
				}

				


						res, err := srv.Create(r.Context(), fnParams)
						if err != nil {
							http.Error(w, err.Error(), err.(ApiError).HTTPStatus)
						}

						encoder := json.NewEncoder(w)

						_ = encoder.Encode(&struct{
							Error string `json:"error"`
							Response interface{}`json:"response, omitempty"`
						}{
							Error: "",
							Response: res,
						})

					}(w, r)

				
					default:
					// 404
			}
		}
	