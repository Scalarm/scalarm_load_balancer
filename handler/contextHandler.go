package handler

import (
	"net/http"

	"github.com/scalarm/scalarm_load_balancer/services"
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
	verbose            bool
}

func AppContext(redirectionsList, servicesTypesList services.TypesMap, loadBalancerScheme string, verbose bool) *appContext {
	return &appContext{
		redirectionsList:   redirectionsList,
		servicesTypesList:  servicesTypesList,
		loadBalancerScheme: loadBalancerScheme,
		verbose:            verbose,
	}
}

type contextHandlerFunction func(*appContext, http.ResponseWriter, *http.Request) error

type contextHandler struct {
	context        *appContext
	f              contextHandlerFunction
	disableLogging bool
}

func (ch contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := http.StatusOK
	message := ""
	err, _ := ch.f(ch.context, w, r).(*httpError)
	if err != nil {
		code = err.Code()
		message = err.Error()
		jsonStatusResponseWriter(w, message, code)
	}
	if !ch.disableLogging {
		logRequest(r.Method, r.URL.String(), code, message)
	}
}

func Context(context *appContext, f contextHandlerFunction) http.Handler {
	return contextHandler{context, f, false}
}

func ContextWithoutLogging(context *appContext, f contextHandlerFunction) http.Handler {
	return contextHandler{context, f, true}
}
