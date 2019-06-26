package main

import (
	"log"
	"net/http"

	"github.com/Xeoncross/servicehandler"
)

func main() {

	// Our database
	memoryStore := NewMemoryStore()

	// Our business/domain logic
	userService := &UserService{memoryStore}

	// Our HTTP handlers (MVC "controllers") are created for us
	handler, err := servicehandler.Wrap(userService)

	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(":8080", handler))
}
