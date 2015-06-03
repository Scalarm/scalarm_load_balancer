package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"

	"github.com/natefinch/lumberjack"
	"github.com/rcrowley/go-tigertonic"
	"github.com/scalarm/scalarm_load_balancer/handler"
	"github.com/scalarm/scalarm_load_balancer/services"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	//loading config
	var configFile string
	if len(os.Args) == 2 {
		configFile = os.Args[1]
	} else {
		configFile = "config.json"
	}
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("An error occurred while loading configuration: %v\n%v", configFile, err.Error())
	}

	//specified logging configuration
	log.SetOutput(&lumberjack.Logger{
		Dir:        config.LogDirectory,
		MaxSize:    100 * lumberjack.Megabyte,
		MaxBackups: 3,
		MaxAge:     28, //days
	})

	//creating redirections and names maps
	redirectionsList, servicesTypesList := services.Init(config.RedirectionConfig, config.StateDirectory)
	//setting app context
	context := handler.AppContext(
		redirectionsList,
		servicesTypesList,
		config.LoadBalancerScheme,
		config.Verbose)

	//disabling certificate checking
	TLSClientConfigCert := &tls.Config{InsecureSkipVerify: true}
	TransportCert := &http.Transport{
		TLSClientConfig: TLSClientConfigCert,
	}

	//setting routing
	director := handler.ReverseProxyDirector(context)
	reverseProxy := &httputil.ReverseProxy{Director: director, Transport: TransportCert}
	http.Handle("/", handler.ContextWithoutLogging(nil, handler.Websocket(director, reverseProxy)))

	// setting registrations handlers
	registrationHandler := handler.Context(context, handler.ServicesManagment(handler.Registration))
	deregistrationHandler := handler.Context(context, handler.ServicesManagment(handler.Deregistration))
	// wrapping registration handlers into host filter
	if !config.DisableRegistrationHostFilter {
		registrationHandler = handler.Authentication(config.PrivateLoadBalancerAddress, registrationHandler)
		deregistrationHandler = handler.Authentication(config.PrivateLoadBalancerAddress, deregistrationHandler)
	}
	//wrapping registration handlers into basic auth
	if config.EnableBasicAuth {
		credentials := map[string]string{config.BasicAuthLogin: config.BasicAuthPassword}
		registrationHandler = tigertonic.HTTPBasicAuth(credentials, "scalarm", registrationHandler)
		deregistrationHandler = tigertonic.HTTPBasicAuth(credentials, "scalarm", deregistrationHandler)
	}
	// setting routes
	http.Handle("/register", registrationHandler)
	http.Handle("/deregister", deregistrationHandler)
	http.Handle("/list", handler.Context(context, handler.List))
	http.HandleFunc("/error", handler.RedirectionError)

	//starting periodical multicast addres sending
	go StartMulticastAddressSender(config.PrivateLoadBalancerAddress, config.MulticastAddress)

	//setting up server
	server := &http.Server{
		Addr:      ":" + config.Port,
		TLSConfig: TLSClientConfigCert,
	}

	if config.LoadBalancerScheme == "http" {
		err = server.ListenAndServe()
	} else { // "https"
		//redirect http to https
		if config.Port == "443" {
			go func() {
				serverHTTP := &http.Server{
					Addr: ":80",
					Handler: http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							http.Redirect(w, r, fmt.Sprintf("https://%v%v", r.Host, r.RequestURI),
								http.StatusMovedPermanently)
						}),
				}
				err = serverHTTP.ListenAndServe()
				if err != nil {
					log.Fatalf("An error occurred while running service on port 80\n%v", err.Error())
				}
			}()
		}

		err = server.ListenAndServeTLS(config.CertFilePath, config.KeyFilePath)
	}
	if err != nil {
		log.Fatalf("An error occurred while running service on port %v\n%v", config.Port, err.Error())
	}
}
