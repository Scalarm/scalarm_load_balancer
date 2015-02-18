package handler

import (
	"log"
	"net/http"

	"github.com/mwrona/scalarm_load_balancer/services"
)

type httpError struct {
	message string
	code    int
}

func newHTTPError(message string, code int) *httpError {
	return &httpError{message, code}
}

func (e *httpError) Error() string {
	return e.message
}

func (e *httpError) Code() int {
	return e.code
}

//multi thread use - do not modify entries
type appContext struct {
	redirectionsList   services.TypesMap
	servicesTypesList  services.TypesMap
	loadBalancerScheme string
}

func AppContext(redirectionsList, servicesTypesList services.TypesMap, loadBalancerScheme string) *appContext {
	return &appContext{
		redirectionsList:   redirectionsList,
		servicesTypesList:  servicesTypesList,
		loadBalancerScheme: loadBalancerScheme,
	}
}

type contextHandlerFunction func(*appContext, http.ResponseWriter, *http.Request) error

type contextHandler struct {
	context *appContext
	f       contextHandlerFunction
}

func (ch contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err, _ := ch.f(ch.context, w, r).(*httpError)
	if err != nil {
		log.Printf("%s\nResponse: %v; %s\n\n", r.URL.RequestURI(), err.Code(), err.Error())
		http.Error(w, err.Error(), err.Code())
	}
}

func Context(context *appContext, f contextHandlerFunction) http.Handler {
	return contextHandler{context, f}
}
