package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mwrona/scalarm_load_balancer/services"
)

func messageWriter(query, message string, w http.ResponseWriter) {
	log.Printf("%s\nResponse: %s\n\n", query, message)
	fmt.Fprintf(w, message)
}

func Authentication(allowedAddress string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "localhost" && r.Host != allowedAddress {
			http.Error(w, "Registration from remote client is forbidden", 403)
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
			return newHTTPError("Missing address", 412)
		}
		if service_name == "" {
			return newHTTPError("Missing service name", 412)
		}

		sl, ok := context.servicesTypesList[service_name]
		if ok == false {
			return newHTTPError(fmt.Sprintf("Service %s does not exist", service_name), 412)
		}
		f(address, sl, w, r)
		return nil
	}
}

func Registration(address string, sl *services.List, w http.ResponseWriter, r *http.Request) {
	if err := sl.AddService(address); err == nil {
		messageWriter(r.URL.String(), fmt.Sprintf("Registered new %s: %s", sl.Name(), address), w)
	} else {
		messageWriter(r.URL.String(), err.Error(), w)
	}
}

func Unregistration(address string, sl *services.List, w http.ResponseWriter, r *http.Request) {
	sl.UnregisterService(address)
	messageWriter(r.URL.String(), fmt.Sprintf("Unregistered %s: %s", sl.Name(), address), w)
}

func printServicesList(sl *services.List, w http.ResponseWriter) {
	fmt.Fprintf(w, "%s:\n", sl.Name())
	for _, val := range sl.AddressesList() {
		fmt.Fprintf(w, "\t%v\n", val)
	}
}

func printAllServicesList(slt services.TypesMap, w http.ResponseWriter) {
	for _, sl := range slt {
		printServicesList(sl, w)
		fmt.Fprintln(w)
	}
}

func List(context *appContext, w http.ResponseWriter, r *http.Request) error {
	service_name := r.FormValue("name")
	if service_name == "" {
		printAllServicesList(context.servicesTypesList, w)
		log.Printf("%s\nMessage: all services list\n\n", r.URL.String())
		return nil
	}

	sl, ok := context.servicesTypesList[service_name]
	if ok == false {
		return newHTTPError(fmt.Sprintf("Service %s does not exist", service_name), 412)
	}
	log.Printf("%s\nMessage: %s list\n\n", r.URL.String(), sl.Name())

	printServicesList(sl, w)

	return nil
}

func RedirectionError(w http.ResponseWriter, req *http.Request) {
	message := req.FormValue("message")
	if message != "" {
		http.Error(w, message, 404)
	} else {
		http.Error(w, "Service list is empty or all services are not responding.", 404)
	}
}
