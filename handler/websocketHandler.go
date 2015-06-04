package handler

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func isWebsocket(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Connection"), "Upgrade") && strings.Contains(req.Header.Get("Upgrade"), "websocket")
}

func Websocket(director func(*http.Request), h http.Handler) contextHandlerFunction {
	errorMessage := "Unable to establish websocket connection"
	return func(context *appContext, w http.ResponseWriter, req *http.Request) error {
		if !isWebsocket(req) {
			h.ServeHTTP(w, req)
			return nil
		}
		director(req)
		if req.URL.Host == "/error" {
			RedirectionError(w, req)
			return nil
		}
		//weboscket
		w.Header().Set("Upgrade", "websocket")

		//connection to target
		var d net.Conn
		var err error
		host := req.URL.Host
		ok := strings.Contains(host, ":")

		if req.URL.Scheme == "http" {
			if !ok {
				host = host + ":80"
			}
			d, err = net.Dial("tcp", host)
		} else {
			if !ok {
				host = host + ":443"
			}
			d, err = tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
		}

		if err != nil {
			log.Printf("Websocket connection failed: %v", err.Error())
			return newHTTPError(errorMessage, http.StatusNotFound)
		}
		defer d.Close()

		//hijacking connection
		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("Websocket connection failed: Casting to http.Hijacker failed")
			return newHTTPError(errorMessage, http.StatusInternalServerError)
		}
		nc, _, err := hj.Hijack()
		if err != nil {
			log.Printf("Websocket connection failed: %v", err.Error())
			return newHTTPError(errorMessage, http.StatusInternalServerError)
		}
		defer nc.Close()

		//copying requst to target and response to client
		err = req.Write(d)
		if err != nil {
			log.Printf("Websocket connection failed: %v", err.Error())
			return newHTTPError(errorMessage, http.StatusNotFound)
		}

		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err := io.Copy(dst, src)
			errc <- err
		}
		go cp(d, nc)
		go cp(nc, d)
		log.Println("Websocket connection")
		<-errc
		return nil
	}
}
