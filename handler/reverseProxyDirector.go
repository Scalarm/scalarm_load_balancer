package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/scalarm/scalarm_load_balancer/services"
)

const redirectionErrorCode = http.StatusBadGateway

func filterParams(url string, verbose bool) string {
	if verbose {
		return url
	}
	return strings.Split(url, "?")[0]
}

func redirectToError(context *appContext, req *http.Request, err error) {
	logRequest(req.Method, filterParams(req.URL.RequestURI(), context.verbose), redirectionErrorCode, err.Error())

	values := url.Values{}
	values.Add("message", err.Error())

	req.URL.RawQuery = values.Encode()
	req.URL.Scheme = context.loadBalancerScheme
	req.URL.Host = req.Host
	req.URL.Path = "/error"
}

func parseURL(context *appContext, req *http.Request) (string, *services.List) {
	splitted := strings.SplitN(req.URL.Path, "/", 3)
	if len(splitted) < 3 {
		splitted = append(splitted, "")
	}

	prefix := "/" + splitted[1]
	sl := context.redirectionsList[prefix]
	path := "/" + splitted[2]

	if sl == nil {
		sl = context.redirectionsList["/"]
		path = req.URL.Path
	}

	return path, sl
}

func ReverseProxyDirector(context *appContext) func(*http.Request) {
	return func(req *http.Request) {
		oldURL := req.URL.RequestURI()

		req.Header.Add("X-Forwarded-Proto", context.loadBalancerScheme)

		path, servicesList := parseURL(context, req)
		if servicesList == nil {
			redirectToError(context, req, fmt.Errorf("Requested redirection does not exists"))
			return
		}

		host, err := servicesList.GetNext()
		if err != nil {
			redirectToError(context, req, err)
			return
		}

		req.URL.Scheme = servicesList.Scheme()
		req.URL.Host = host
		req.URL.Path = path

		logRequest(req.Method, filterParams(oldURL, context.verbose), http.StatusOK,
			fmt.Sprintf("redirected to %v", filterParams(req.URL.RequestURI(), context.verbose)))
	}
}
