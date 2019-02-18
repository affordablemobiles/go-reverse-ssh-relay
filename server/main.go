package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

var globalConfig struct {
	ListenPort       int `yaml:"listenPort"`
	LocalListenStart int `yaml:"localListenStart"`
	LocalListenEnd   int `yaml:"localListenEnd"`
}

var opts struct {
	Config string `short:"c" long:"config" description:"Configuration YAML file location" required:"true"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if _, err := os.Stat(opts.Config); err != nil {
		log.Fatalf("Specified config file doesn't exist!\n")
	}

	data, err := ioutil.ReadFile(opts.Config)
	if err != nil {
		log.Fatalf("Failed to read config file.")
	}

	err = yaml.Unmarshal([]byte(data), &globalConfig)
	if err != nil {
		log.Fatalf("Error while parsing config file.")
	}
	err = processConfig()
	if err != nil {
		log.Fatalf("Error while processing config file.")
	}

	go startWebSocket()

	done := make(chan bool)
	<-done
}

func processConfig() error {
	if globalConfig.LocalListenStart < 1024 {
		log.Fatalf("Start port must be higher than 1024")
	}
	if globalConfig.LocalListenEnd < 1024 {
		log.Fatalf("End port must be higher than 1024")
	}

	if globalConfig.LocalListenStart > globalConfig.LocalListenEnd {
		log.Fatalf("End Port Must be Higher than Start Port")
	}

	return nil
}
