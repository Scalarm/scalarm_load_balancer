package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/mwrona/scalarm_load_balancer/services"
)

func redirectToError(context *appContext, req *http.Request, err error) {
	log.Printf("%v\nUnable to redirect: %v", req.URL.RequestURI(), err.Error())

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

		log.Printf("%v \nredirect to %v\n\n", oldURL, req.URL)
	}
}
