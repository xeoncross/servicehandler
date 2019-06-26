package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xeoncross/servicehandler"
)

func setup() (http.Handler, error) {
	// Our database
	memoryStore := NewMemoryStore()

	// Our business/domain logic
	userService := &UserService{memoryStore}

	// Our HTTP handlers (MVC "controllers") are created for us
	handler, err := servicehandler.Wrap(userService)

	if err != nil {
		return nil, err
	}

	return handler, nil
}

func TestCreateUser(t *testing.T) {
	var req *http.Request

	// Create HTTP mux/router
	mux, err := setup()
	if err != nil {
		t.Error(err)
	}

	b, err := json.Marshal(map[string]string{"name": "john", "email": "email@example.com"})
	if err != nil {
		t.Error(err)
	}

	// Create 9 users
	for i := 1; i < 10; i++ {
		req, err = http.NewRequest("POST", "/Create", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Wrong status code: got %v want %v", status, http.StatusOK)
		}

		response := strings.TrimSpace(rr.Body.String())
		want := fmt.Sprintf(`{"success":true,"data":%d}`, i)
		if response != want {
			t.Errorf("Wrong response:\ngot %s\nwant %s", response, want)
		}
	}

}

func TestCreateUserFailure(t *testing.T) {

	var req *http.Request

	b, err := json.Marshal(map[string]string{"name": "john", "email": "a@b"})
	if err != nil {
		t.Error(err)
	}

	req, err = http.NewRequest("POST", "/Create", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Create HTTP mux/router
	mux, err := setup()
	if err != nil {
		t.Error(err)
	}

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	response := strings.TrimSpace(rr.Body.String())
	want := `{"success":false,"error":"Invalid Request","fields":{"Email":"a@b does not validate as email"}}`
	if response != want {
		t.Errorf("Wrong response:\ngot %s\nwant %s", response, want)
	}

}
