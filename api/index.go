package api

import (
	"net/http"
)

var router = func() http.Handler {
	r, err := SetupRouter()
	if err != nil {
		panic(err)
	}
	return r
}()

func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}
