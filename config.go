package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/scalarm/scalarm_load_balancer/services"
)

type Config struct {
	Port                          string
	MulticastAddress              string
	PrivateLoadBalancerAddress    string
	LoadBalancerScheme            string
	CertFilePath                  string
	KeyFilePath                   string
	LogDirectory                  string
	StateDirectory                string
	Verbose                       bool
	RedirectionConfig             []services.RedirectionPolicy
	DisableRegistrationHostFilter bool
	EnableBasicAuth               bool
	BasicAuthLogin                string
	BasicAuthPassword             string
}

func LoadConfig(filename string) (*Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}

	if config.MulticastAddress == "" {
		return nil, fmt.Errorf("Multicast address is missing")
	}

	if config.LoadBalancerScheme == "" {
		config.LoadBalancerScheme = "https"
	}

	if config.Port == "" {
		if config.LoadBalancerScheme == "https" {
			config.Port = "443"
		} else if config.LoadBalancerScheme == "http" {
			config.Port = "80"
		}
	}

	if config.PrivateLoadBalancerAddress == "" {
		config.PrivateLoadBalancerAddress = "localhost:" + config.Port
	}

	if config.LoadBalancerScheme != "https" && config.LoadBalancerScheme != "http" {
		return nil, fmt.Errorf("Unsuported protocol in LoadBalancerScheme")
	}
	if config.CertFilePath == "" {
		config.CertFilePath = "cert.pem"
	}
	if config.KeyFilePath == "" {
		config.KeyFilePath = "key.pem"
	}
	if config.RedirectionConfig == nil {
		return nil, fmt.Errorf("RedirectionConfig is missing")
	}
	if config.LogDirectory == "" {
		config.LogDirectory = "log"
	}
	if !strings.HasSuffix(config.StateDirectory, "/") && config.StateDirectory != "" {
		config.StateDirectory += "/"
	}
	if config.EnableBasicAuth && (config.BasicAuthLogin == "" || config.BasicAuthPassword == "") {
		return nil, fmt.Errorf("Missing basic auth credentials")
	}

	return config, nil
}
