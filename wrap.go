package servicehandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"

	"github.com/asaskevich/govalidator"
)

const (
	// TagQuery is the field tag to define a query parameter's key
	TagQuery = "q"
)

// JSONResponse for validation errors or service responses
type JSONResponse struct {
	Success bool              `json:"success"`
	Data    interface{}       `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Wrapper for a service method
type serviceMethod struct {
	in        []reflect.Type
	method    reflect.Value
	anonymous bool
}

func Wrap(service interface{}) (http.Handler, error) {

	// Improve performance (and clarity) by pre-computing needed variables
	serviceType := reflect.TypeOf(service)

	// For error logs
	var serviceName string
	if serviceType.Kind() == reflect.Ptr {
		serviceName = serviceType.Elem().Name()
	} else {
		serviceName = serviceType.Name()
	}

	// The method Call() needs this as the first value
	serviceValue := reflect.ValueOf(service)

	var methods = make(map[string]*serviceMethod)

	for i := 0; i < serviceType.NumMethod(); i++ {
		methodType := serviceType.Method(i)
		method := methodType.Func

		if methodType.Type.NumIn() != 2 {
			return nil, fmt.Errorf("%s.%s() can only take 1 struct parameter. Wrap existing parameters in a struct.", serviceName, methodType.Name)
		}

		if methodType.Type.NumOut() > 2 {
			return nil, fmt.Errorf("%s.%s() should return ([]slice/struct{}, error) or (error).", serviceName, methodType.Name)
		}

		// TODO we've basically decided on only a single parameter
		// Time to remove all this code for handling multiple in
		in := make([]reflect.Type, methodType.Type.NumIn())

		// Marker for anonymous structs as parameters
		var anonymous bool

		for j := 0; j < methodType.Type.NumIn(); j++ {
			paramType := methodType.Type.In(j)
			in[j] = paramType

			// First param is method receiver
			if j == 0 {
				continue
			}

			if paramType.Kind() != reflect.Struct && paramType.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("%s.%s() can only take 1 struct parameter. Wrap existing parameters in a struct.", serviceName, methodType.Name)
			}

			// Is this check needed? Is there ever a time when a struct/struct ptr
			// can't be used as an interface?
			// var object reflect.Value
			// switch paramType.Kind() {
			// case reflect.Struct:
			// 	object = newReflectType(paramType).Elem()
			// case reflect.Ptr:
			// 	object = newReflectType(paramType)
			// }
			//
			// if !object.CanInterface() {
			// 	log.Fatalf("%s.%s() should only take 1 struct parameter. Wrap existing parameters in a struct.", serviceName, methodType.Name)
			// }

			// Is this an anonymous struct?
			if paramType.Kind() == reflect.Struct {
				if paramType.Name() == "" {
					anonymous = true
				}
			}

		}

		name := methodType.Name
		methods[name] = &serviceMethod{
			in:        in,
			anonymous: anonymous,
			method:    method,
		}
	}

	// Cache setup finished, now get ready to process requests
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		method, ok := methods[name]

		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// fmt.Printf("HTTP %s %s(%v)\n", r.Method, name, methods[name].in[1])
		in := make([]reflect.Value, len(method.in))

		for i, paramType := range method.in {

			// The first item should be the method receiver instance
			// This also enables access to struct fields from inside the method
			if i == 0 {
				in[i] = serviceValue
				continue
			}

			// Create a new instance for each goroutine
			var object reflect.Value

			switch paramType.Kind() {
			case reflect.Struct:
				object = newReflectType(paramType).Elem()
			case reflect.Ptr:
				object = newReflectType(paramType)
			}

			if r.Method == http.MethodGet {
				if !method.anonymous {
					http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
					return
				}

				numFields := paramType.NumField()
				queryValues := r.URL.Query()
				for j := 0; j < numFields; j++ {
					field := paramType.Field(j)

					var s string
					tag, ok := field.Tag.Lookup(TagQuery)
					if ok {
						s = queryValues.Get(tag)
					} else {
						s = queryValues.Get(field.Name)
					}

					// fmt.Printf("Field: %v = %q\n", field.Name, s)

					if s == "" {
						// Do not fail right now, it is the job of validator
						continue
					}

					val := object.Field(j)

					err := parseSimpleParam(s, "Query Parameter", field, &val)
					if err != nil {
						// fmt.Println(err)
						// What should we do here?
					}
				}

				// fmt.Printf("GET objectInterface = %v\n", object.Interface())

			} else if r.Method == "POST" {

				if method.anonymous {
					http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
					return
				}

				oi := object.Interface()
				_ = json.NewDecoder(r.Body).Decode(&oi)
				// if err != nil {
				// 	// We don't care about JSON type errors nor want to give app details out
				// 	// the validator will handle those messages better below
				// 	log.Println(err)
				// }
			}

			// 2. Validate the struct data rules
			isValid, err := govalidator.ValidateStruct(object.Interface())

			if !isValid {
				validationErrors := govalidator.ErrorsByField(err)

				w.WriteHeader(http.StatusBadRequest)
				JSON(w, JSONResponse{
					Success: false,
					Error:   "Invalid Request",
					Fields:  validationErrors,
				})
				return
			}

			in[i] = object
		}

		response := method.method.Call(in)

		// Expect all service methods in one of two forms:
		// func (...) error
		// func (...) (interface{}, error)
		ek := 0
		if method.method.Type().NumOut() == 2 {
			ek = 1
		}

		if err, ok := response[ek].Interface().(error); ok {
			if err != nil {
				// http.Error(w, err.Error(), http.StatusBadRequest)
				JSON(w, JSONResponse{
					Success: false,
					Error:   err.Error(),
				})
				return
			}
		}

		if ek == 0 {
			return
		}

		JSON(w, JSONResponse{
			Success: true,
			Data:    response[0].Interface(),
		})

	}), nil
}

func newReflectType(t reflect.Type) reflect.Value {
	// Dereference pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return reflect.New(t)
}

// JSON response helper
func JSON(w http.ResponseWriter, i interface{}) {
	var err error
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	// e.SetIndent("", "  ")
	err = e.Encode(i)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
