package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/scalarm/scalarm_load_balancer/services"
)

func logRequest(method string, url string, code int, message string) {
	log.Printf("[%v] %q Response: %v %v\n", method, url, code, message)
}

func jsonResponseWriter(w http.ResponseWriter, res interface{}) {
	js, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "Internal server error, unable to parse json response", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func jsonStatusResponseWriter(w http.ResponseWriter, reason string, code int) {
	w.WriteHeader(code)
	jsonResponseWriter(w, map[string]interface{}{"status": code, "message": reason})
}

func HostFilter(allowedAddress string, h http.Handler) http.Handler {
	code := http.StatusForbidden
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "localhost" && r.Host != allowedAddress {
			message := fmt.Sprintf("Request on forbidden host [%v] rejected", r.Host)
			logRequest(r.Method, r.URL.String(), code, message)
			jsonStatusResponseWriter(w, message, code)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func ServicesManagment(f func(string, *services.List, http.ResponseWriter, *http.Request)) contextHandlerFunction {
	return func(context *appContext, w http.ResponseWriter, r *http.Request) error {
		address := r.FormValue("address")
		service_name := r.FormValue("name")
		if address == "" {
			return newHTTPError("Missing address", http.StatusPreconditionFailed)
		}
		if service_name == "" {
			return newHTTPError("Missing service name", http.StatusPreconditionFailed)
		}

		sl, ok := context.servicesTypesList[service_name]
		if ok == false {
			return newHTTPError(fmt.Sprintf("Service %s does not exist", service_name), http.StatusPreconditionFailed)
		}
		f(address, sl, w, r)
		return nil
	}
}

func Registration(address string, sl *services.List, w http.ResponseWriter, r *http.Request) {
	response := fmt.Sprintf("Registered new %s: %s", sl.Name(), address)
	if err := sl.AddService(address); err == nil {
		response = err.Error()
	}
	jsonStatusResponseWriter(w, response, http.StatusOK)
}

func Deregistration(address string, sl *services.List, w http.ResponseWriter, r *http.Request) {
	sl.UnregisterService(address)
	jsonStatusResponseWriter(w, fmt.Sprintf("Deregistered %s: %s", sl.Name(), address), http.StatusOK)
}

func List(context *appContext, w http.ResponseWriter, r *http.Request) error {
	service_name := r.FormValue("name")
	if service_name == "" {
		response := map[string][]string{}
		for _, sl := range context.servicesTypesList {
			response[sl.Name()] = sl.AddressesList()
		}
		jsonResponseWriter(w, response)
		return nil
	}

	sl, ok := context.servicesTypesList[service_name]
	if ok == false {
		return newHTTPError(fmt.Sprintf("Service %s does not exist", service_name), http.StatusPreconditionFailed)
	}
	jsonResponseWriter(w, sl.AddressesList())
	return nil
}

func RedirectionError(w http.ResponseWriter, req *http.Request) {
	message := req.FormValue("message")
	if message == "" {
		message = "Service list is empty or no service instance is responding."
	}
	jsonStatusResponseWriter(w, message, redirectionErrorCode)
}
