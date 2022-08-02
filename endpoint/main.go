package main

import (
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/sevlyar/go-daemon"
	"gopkg.in/yaml.v2"
)

var globalConfig struct {
	RemoteEndpoint        string   `yaml:"remoteEndpoint"`
	ServerKeyEncoded      string   `yaml:"serverKey"`
	AllowedClientsEncoded []string `yaml:"allowedClientKeys"`
	HealthcheckListenPort int      `yaml:"healthcheckListenPort"`

	serverKey      string
	allowedClients []string
}

var opts struct {
	Config string `short:"c" long:"config" description:"Configuration YAML file location" required:"true"`
}

var logger, _ = syslog.New(syslog.LOG_DAEMON, "ssh-dev-endpoint")

var startTime = time.Now()

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

	daemon.SetSigHandler(termHandler, syscall.SIGQUIT)
	daemon.SetSigHandler(termHandler, syscall.SIGTERM)
	daemon.SetSigHandler(reloadHandler, syscall.SIGHUP)

	args := []string{"[ssh-dev-endpoint]"}
	args = append(args, os.Args[1:]...)

	cntxt := &daemon.Context{
		// What's our run directory location on the router?
		PidFileName: "/tmp/ssh-dev-endpoint.pid",
		PidFilePerm: 0644,
		LogFileName: "/tmp/ssh-dev-endpoint.log",
		LogFilePerm: 0640,
		WorkDir:     "/tmp",
		Umask:       027,
		Args:        args,
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("daemon started")

	log.SetPrefix("")
	log.SetOutput(logger)

	go worker()

	go startSSH()

	go startwebsrv()

	err = daemon.ServeSignals()
	if err != nil {
		log.Println("Error:", err)
	}
	log.Println("daemon terminated")
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func worker() {
LOOP:
	for {
		time.Sleep(time.Second) // this is work to be done by worker.
		select {
		case <-stop:
			break LOOP
		default:
		}
	}
	done <- struct{}{}
}

func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}

func reloadHandler(sig os.Signal) error {
	log.Println("configuration reloaded")
	return nil
}
