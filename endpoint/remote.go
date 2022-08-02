package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	co "github.com/a1comms/ssh-reverse-concentrator/shared"
)

const (
	// Time allowed to write a message to the peer.
	ClientWSwriteWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	ClientWSpongWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	ClientWSpingPeriod = 2 * time.Second
	// Status Period - how often should we update the mothership on our System Status.
	ClientWSstatusPeriod = 10 * time.Second
)

type ClientWS struct {
	// The websocket connection.
	ws *websocket.Conn
	// Web Socket write mutex.
	writeMutex *sync.Mutex

	// Current Reader
	reader io.Reader
}

var clientConn *ClientWS
var wsDialer *websocket.Dialer = &websocket.Dialer{
	Proxy: http.ProxyFromEnvironment,
}

func RemoteConnect() (io.ReadWriteCloser, error) {
	clientConn := &ClientWS{
		writeMutex: &sync.Mutex{},
	}

	err := clientConn.Connect()
	if err != nil {
		return nil, err
	}

	go clientConn.writePump()
	go clientConn.readPump()

	return clientConn, nil
}

func (c *ClientWS) Connect() error {
	ws, response, err := wsDialer.Dial(globalConfig.RemoteEndpoint, nil)
	if err != nil {
		if response != nil {
			return fmt.Errorf("%s - %s", response.Status, err)
		} else {
			return fmt.Errorf("%s - %s", 550, err)
		}
	} else {
		c.ws = ws
		log.Printf("Connected to WebSocket, performing handshake...")
		if err := c.PerformHandshake(); err != nil {
			return err
		}
		log.Printf("Connected, handshake complete.")

		return nil
	}
}

func (c *ClientWS) Write(p []byte) (n int, err error) {
	return len(p), c.writeData(websocket.BinaryMessage, p)
}

func (c *ClientWS) Read(p []byte) (n int, err error) {
	if c.reader == nil {
		_, c.reader, err = c.ws.NextReader()
		if err != nil {
			return 0, err
		}
	}

	i, err := c.reader.Read(p)
	if err == io.EOF {
		c.reader = nil
		err = nil
	}
	return i, err
}

func (c *ClientWS) Close() error {
	c.CloseCleanup()

	return nil
}

func (c *ClientWS) writeData(mt int, payload []byte) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	c.ws.SetWriteDeadline(time.Now().Add(ClientWSwriteWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *ClientWS) PerformHandshake() error {
	var err error = nil
	var mType int = 0
	var message []byte = nil
	c1 := make(chan bool, 1)

	// Step 1. Wait for the identity request from the server...
	log.Printf("WebSocket: Handshake Step 1: Waiting for identityRequest...")
	go func() {
		for mType != 1 && err == nil {
			mType, message, err = c.ws.ReadMessage()
			log.Printf("WebSocket: Read message of type %d from WebSocket during handshake", mType)
		}
		c1 <- true
	}()
	select {
	case <-c1:
		log.Printf("WebSocket: Completed Handshake Step 1 Before Timeout, Wuhoo!")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("WebSocket: Timeout Waiting for Handshake Step 1")
	}

	response := co.JSONType{}
	if err = json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("WebSocket: Failed to Unmarshal JSON during Handshake Step 1: %s", err)
	}
	if response.MType != "identityRequest" {
		return fmt.Errorf("WebSocket: Expected identityRequest during handshake, but got \"%s\"", response.MType)
	}

	// Step 2. Send our identity notification...
	project, service, version, instance, err := GetRunningIdentity()
	if err != nil {
		return err
	}

	notify := &co.JSONIdentityNotify{
		MType:    "identityNotify",
		Project:  project,
		Service:  service,
		Version:  version,
		Instance: instance,
	}
	log.Printf("WebSocket: Handshake Step 2: Sending identityNotify, %#v", notify)
	notifyJSON, err := json.Marshal(notify)
	if err != nil {
		return fmt.Errorf("WebSocket: Failed to Marshal identifyConfirm JSON: %s", err)
	}
	err = c.writeData(websocket.TextMessage, notifyJSON)
	if err != nil {
		return fmt.Errorf("WebSocket: Handshake Step 2: Problem writing to WebSocket: %s", err)
	}

	// Step 3. Confirmation from the server & get our UUID...
	log.Printf("WebSocket: Handshake Step 3: Waiting for \"identityConfirm\"")
	mType = 0
	message = nil

	go func() {
		for mType != 1 && err == nil {
			mType, message, err = c.ws.ReadMessage()
			log.Printf("WebSocket: Read message of type %d from WebSocket during handshake", mType)
		}
		c1 <- true
	}()
	select {
	case <-c1:
		log.Printf("WebSocket: Completed Handshake Step 3 Before Timeout, Wuhoo x2")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("WebSocket: Timeout Waiting for Handshake Step 3")
	}

	resp := co.JSONIdentityConfirm{}
	if err = json.Unmarshal(message, &resp); err != nil {
		return fmt.Errorf("WebSocket: Failed to Unmarshal JSON during Handshake Step 3: %s", err)
	}
	if !(resp.MType == "identityConfirm") {
		return fmt.Errorf("WebSocket: Expected \"identityConfirm\" during handshake, but got \"%s\"", resp.MType)
	} else if resp.Accepted != true {
		return fmt.Errorf("WebSocket: Server told us our handshake wasn't accepted, on noes! :(...")
	}

	// Looks like we're valid, add the user to the user MAP.
	log.Printf("Completed handshake, proceeding with connection...")
	return nil
}

func (c *ClientWS) writePump() {
	pingTicker := time.NewTicker(ClientWSpingPeriod)
	defer func() {
		pingTicker.Stop()
		c.CloseCleanup()
	}()
	for {
		select {
		case <-pingTicker.C:
			if err := c.writeData(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("Remote WebSocket - Write Error in writePump on ping: %s", err)
				return
			}
		}
	}
}

func (c *ClientWS) readPump() {
	c.ws.SetReadDeadline(time.Now().Add(ClientWSpongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(ClientWSpongWait)); return nil })
}

func (c *ClientWS) CloseCleanup() {
	log.Printf("WebSocket Connection Closed")
	c.ws.Close()
}
