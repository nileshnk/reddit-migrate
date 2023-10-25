package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func apiRouter(router chi.Router) {
	
	router.Get("/test", func(w http.ResponseWriter, _ *http.Request){
		type test_data struct {
			Hello string `json:"hello"`
		}
		var test test_data;
		test.Hello = "world!"
		t := []byte(`{"hello": "nilesh"}`)
		w.Header().Set("Content-Type", "application/json")
		w.Write(t);
	});

	router.Post("/verify-cookie", verifyTokenResponse)

	router.Post("/migrate", MigrationHandler)
}
