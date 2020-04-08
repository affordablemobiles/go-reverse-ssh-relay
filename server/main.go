package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

var globalConfig struct {
	ListenPort       int            `yaml:"listenPort"`
	ListenPortStatus int            `yaml:"listenPortStatus"`
	LocalListenStart int            `yaml:"localListenStart"`
	LocalListenEnd   int            `yaml:"localListenEnd"`
	StaticPortMap    map[string]int `yaml:"staticPortMap"`
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

	err = processConfig()
	if err != nil {
		log.Fatalf("Error while processing config file.")
	}

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	// Block until a signal is received.
	go func() {
		for s := range c {
			log.Printf("Got signal: %#v", s)

			err := processConfig()
			if err != nil {
				log.Fatalf("Error while processing config file.")
			}
		}
	}()

	go startWebSocket()
	go startWebStatus()

	done := make(chan bool)
	<-done
}

func processConfig() error {
	data, err := ioutil.ReadFile(opts.Config)
	if err != nil {
		log.Fatalf("Failed to read config file.")
	}

	err = yaml.Unmarshal([]byte(data), &globalConfig)
	if err != nil {
		log.Fatalf("Error while parsing config file.")
	}

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

func saveConfig() {
	data, err := yaml.Marshal(&globalConfig)
	if err != nil {
		log.Fatalf("Failed to marshal config: %s", err)
	}

	err = ioutil.WriteFile(opts.Config, data, 0644)
	if err != nil {
		log.Fatalf("Failed to write new config: %s", err)
	}
}
