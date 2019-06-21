package servicehandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type TestUser struct {
	Name  string `valid:"alphanum,required"`
	Email string `valid:"email,required"`
	Bio   string `valid:"ascii,required"`
	Date  string `valid:"-"`
}

type TestUserService struct {
	Foo string
}

func (s *TestUserService) Save(u *TestUser) error {
	fmt.Printf("Called Save with %v from %v\n", u, s)
	return nil
}

func (s *TestUserService) Get(id int) (*TestUser, error) {
	fmt.Printf("Called Get with %v from %v\n", id, s)
	return nil, errors.New("User not found")
}

// type sample struct {
// }
//
// // https://gist.github.com/tonyhb/5819315
// func structToMap(i interface{}) (values url.Values) {
// 	values = url.Values{}
// 	iVal := reflect.ValueOf(i).Elem()
// 	typ := iVal.Type()
// 	for i := 0; i < iVal.NumField(); i++ {
// 		values.Set(typ.Field(i).Name, fmt.Sprint(iVal.Field(i)))
// 	}
// 	return
// }

func TestValidation(t *testing.T) {

	scenarios := []struct {
		Name       string
		Object     interface{}
		JSON       interface{}
		Form       url.Values
		URL        string // URL Params & query string
		StatusCode int
		Response   string
	}{
		{
			Name:       "Valid JSON",
			URL:        "/Save",
			JSON:       map[string]string{"name": "john", "email": "a@b"},
			StatusCode: http.StatusOK,
		},
	}

	var err error
	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {

			var req *http.Request

			if s.JSON != nil {
				var b []byte
				b, err = json.Marshal(s.JSON)
				if err != nil {
					log.Fatal(err)
				}

				req, err = http.NewRequest("POST", s.URL, bytes.NewReader(b))
				if err != nil {
					t.Fatal(err)
				}

				req.Header.Add("Content-Type", "application/json")
			} else if s.Form != nil {

				f := s.Form
				req, err = http.NewRequest("POST", s.URL, strings.NewReader(f.Encode()))
				if err != nil {
					t.Fatal(err)
				}

				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req, err = http.NewRequest("POST", s.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
			}

			rr := httptest.NewRecorder()

			// Create HTTP mux/router
			mux := Wrap(&TestUserService{Foo: "foo"})
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != s.StatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, s.StatusCode)
				t.Log(rr.Body.String())
			}

			if s.Response != "" {
				response := strings.TrimSpace(rr.Body.String())
				if response != s.Response {
					t.Errorf("handler returned wrong response:\ngot %qwant %q", response, s.Response)
				}
			}

		})
	}

}
