package servicehandler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"reflect"

	"github.com/asaskevich/govalidator"
)

const (
	// TagQuery is the field tag to define a query parameter's key
	TagQuery = "q"
)

// ValidationResponse on client input error
type ValidationResponse struct {
	Success bool              `json:"success"`
	Errors  map[string]string `json:"errors"`
}

// Wrapper for a service method
type serviceMethod struct {
	in        []reflect.Type
	out       []reflect.Type
	method    reflect.Value
	anonymous bool
}

func Wrap(service interface{}) http.Handler {

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
			log.Fatalf("%s.%s() should only take 1 struct parameter. Wrap existing parameters in a struct.", serviceName, methodType.Name)
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
				log.Fatalf("%s.%s() should only take 1 struct parameter. Wrap existing parameters in a struct.", serviceName, methodType.Name)
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

		out := make([]reflect.Type, methodType.Type.NumOut())

		for j := 0; j < methodType.Type.NumOut(); j++ {
			paramType := methodType.Type.Out(j)
			out[j] = paramType
		}

		name := methodType.Name
		methods[name] = &serviceMethod{
			in:        in,
			out:       out,
			anonymous: anonymous,
			method:    method,
		}
	}

	// Cache setup finished, now get ready to process requests
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		method, ok := methods[name]

		if !ok {
			http.Error(w, fmt.Sprintf("Unknown method %s", name), http.StatusNotFound)
			return
		}

		fmt.Printf("HTTP %s %s(%v)\n", r.Method, name, methods[name].in[1])

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

			// fmt.Printf("paramType: %v = %v\n", paramType.Kind(), paramType)

			switch paramType.Kind() {
			case reflect.Struct:
				object = newReflectType(paramType).Elem()
			case reflect.Ptr:
				object = newReflectType(paramType)
				// default:
				// 	fmt.Printf("Unknown type: %s", paramType.Kind().String())
			}

			if r.Method == http.MethodGet {

				if !method.anonymous {
					log.Fatal("This isn't how you call this method")
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
						fmt.Println(err)
					}
				}

				// fmt.Printf("GET objectInterface = %v\n", object.Interface())

			} else if r.Method == "POST" {

				if method.anonymous {
					log.Fatal("this isn't how you call this method")
				}

				oi := object.Interface()
				_ = json.NewDecoder(r.Body).Decode(&oi)
				// if err != nil {
				// 	// We don't care about type errors
				// 	// the validator will handle those messages better below
				// 	log.Println(err)
				// }
			}

			// 2. Validate the struct data rules
			// var isValid bool
			isValid, err := govalidator.ValidateStruct(object.Interface())

			if !isValid {
				validationErrors := govalidator.ErrorsByField(err)

				w.WriteHeader(http.StatusBadRequest)
				JSON(w, ValidationResponse{
					false,
					validationErrors,
				})
				return
			}

			// } else if object.CanSet() {
			// 	// TODO handle each type of variable
			// 	// var b []byte
			// 	// err := json.NewDecoder(strings.NewReader(`{"a":"foo"}`)).Decode(&b)
			// 	// if err != nil {
			// 	// 	t.Error(err)
			// 	// }
			// 	// object.Set(reflect.ValueOf(b))
			// }

			in[i] = object
		}

		response := method.method.Call(in)

		var results []interface{}

		// TODO use method.out here for proper names instead of []slice

		for _, item := range response {
			if err, ok := item.Interface().(error); ok {
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			} else {
				results = append(results, item.Interface())
			}
		}

		if len(results) > 0 {
			if len(results) == 1 {
				JSON(w, results[0])
			} else {
				JSON(w, results)
			}
		}

	})
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
	err = json.NewEncoder(w).Encode(i)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
