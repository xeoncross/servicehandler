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

// ValidationErrors occurs whenever one or more fields fail the validation by govalidator
// type ValidationErrors map[string]string

type serviceMethod struct {
	params  []reflect.Type
	method  reflect.Value
	unnamed bool
	// receiver reflect.Value // Struct instance
}

func Wrap(service interface{}) http.Handler {

	// Improve performance (and clarity) by pre-computing needed variables
	serviceType := reflect.TypeOf(service)

	// The method Call() needs this as the first value
	serviceValue := reflect.ValueOf(service)

	var methods = make(map[string]*serviceMethod)

	for i := 0; i < serviceType.NumMethod(); i++ {
		methodType := serviceType.Method(i)
		method := methodType.Func

		// fmt.Printf("%v has %d params\n", methodType.Name, method.Type().NumIn())

		params := make([]reflect.Type, methodType.Type.NumIn())

		for j := 0; j < methodType.Type.NumIn(); j++ {
			// fmt.Printf("\t%d: %v\n", j, method.Type().In(j).Kind())
			// params[j] = method.Type().In(j)
			paramType := methodType.Type.In(j)
			params[j] = paramType

			// if paramType...CanInterface() {
			// 	log.Fatalf("cantinterface %s", paramType.Name())
			// }
		}

		name := methodType.Name
		methods[name] = &serviceMethod{params: params, method: method}
	}

	// fmt.Printf("methods: %#v\n", methods)

	// Cache setup finished, now get ready to process requests
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.RequestURI())
		fmt.Printf("HTTP %s %s(%v)\n", r.Method, name, methods[name].params)

		method, ok := methods[name]

		if !ok {
			http.Error(w, fmt.Sprintf("Unknown method %s", name), http.StatusNotFound)
			return
		}

		in := make([]reflect.Value, len(method.params))

		type ValidationError struct {
			Parameter string            `json:"parameter"`
			Errors    map[string]string `json:"errors"`
		}

		// Collect them all so we can tell the client everything wrong at once
		// Nah, why waste resources? Fix each problem as you have it.
		// var validationErrors []*ValidationError

		for i, paramType := range method.params {

			// The first item should be the method receiver instance
			// This also enables access to struct fields from inside the method
			if i == 0 {
				in[i] = serviceValue
				continue
			}

			fmt.Printf("%d = %s (%v)\n", i, paramType.Name(), paramType.Kind())

			// If the variable fails validation
			// var validation ValidationErrors

			// Create a new instance of each param
			var object reflect.Value

			// fmt.Printf("paramType: %v = %v\n", paramType.Kind(), paramType)

			switch paramType.Kind() {
			case reflect.Struct:
				object = newReflectType(paramType).Elem()
			case reflect.Ptr:
				object = newReflectType(paramType)
			// case reflect.String:
			// 	object = reflect.New(paramType).Elem()
			default:
				fmt.Printf("Unknown type: %s", paramType.Kind().String())
			}

			// TODO check this sooner before clients start connecting!
			if !object.CanInterface() {
				err := fmt.Errorf("Cannot interface %s", paramType.Name())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Fatal(err)
				return
			}

			// Do we need this var?
			objectInterface := object.Interface()

			if r.Method == http.MethodGet {

				fmt.Println(r.Method, "running")

				numFields := paramType.NumField()
				queryValues := r.URL.Query()
				for i := 0; i < numFields; i++ {
					field := paramType.Field(i)

					var s string
					tag, ok := field.Tag.Lookup(TagQuery)
					if ok {
						s = queryValues.Get(tag)
					} else {
						s = queryValues.Get(field.Name)
					}

					if s == "" {
						// Do not fail right now, it is the job of validator
						continue
					}

					val := object.Field(i)

					fmt.Printf("%v = %s\n", field, s)

					err := parseSimpleParam(s, "Query Parameter", field, &val)
					if err != nil {
						fmt.Println(err)
					}
				}

			} else {
				err := json.NewDecoder(r.Body).Decode(object.Interface())
				if err != nil {
					// We don't care about type errors
					// the validator will handle those messages better below
					log.Println(err)
				}
			}

			// 2. Validate the struct data rules
			var isValid bool
			isValid, err := govalidator.ValidateStruct(objectInterface)

			if !isValid {
				validationErrors := govalidator.ErrorsByField(err)

				// type ValidationResponse struct {
				// 	Success bool             `json:"success"`
				// 	Errors  ValidationErrors `json:"errors"`
				// }

				w.WriteHeader(http.StatusBadRequest)
				JSON(w, ValidationError{
					"WIP",
					validationErrors,
				})
				return
			}

			// if len(validation) > 0 {
			// 	log.Fatalf("%v\n", validation)
			// 	return
			// }

			// fmt.Printf("%#v\n", object.Interface())
			// fmt.Printf("%#v\n", i.(*ProviderA))

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

		// in = append([]reflect.Type{method.Method})

		response := method.method.Call(in)

		var results []interface{}

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
