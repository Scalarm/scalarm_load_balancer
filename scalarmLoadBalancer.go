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
		log.Fatal("An error occurred while loading configuration: " + configFile + "\n" + err.Error())
	}

	//specified logging configuration
	fmt.Println(config.LogDirectory)
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
	http.Handle("/", handler.Context(nil, handler.Websocket(director, reverseProxy)))

	http.Handle("/register", handler.Authentication(
		config.PrivateLoadBalancerAddress,
		handler.Context(
			context,
			handler.ServicesManagment(handler.Registration))))

	http.Handle("/deregister", handler.Authentication(
		config.PrivateLoadBalancerAddress,
		handler.Context(
			context,
			handler.ServicesManagment(handler.Deregistration))))

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
							http.Redirect(w, r, "https://"+r.Host+r.RequestURI,
								http.StatusMovedPermanently)
						}),
				}
				err = serverHTTP.ListenAndServe()
				if err != nil {
					log.Fatal("An error occurred while running service on port 80\n" + err.Error())
				}
			}()
		}

		err = server.ListenAndServeTLS(config.CertFilePath, config.KeyFilePath)
	}
	if err != nil {
		log.Fatal("An error occurred while running service on port " + config.Port + "\n" + err.Error())
	}
}
