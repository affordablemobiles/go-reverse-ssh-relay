package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	co "github.com/a1comms/ssh-reverse-concentrator/shared"
	"github.com/gorilla/websocket"
)

// Define a mutex to allow thread-safe access to our client connection map.
var clientMutex = &sync.RWMutex{}

// Create a map (array) to hold pointers to our connected Windows clients.
var clientMAP map[int]*ClientWSconn = make(map[int]*ClientWSconn)

var ClientWSupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 2 * time.Second
)

type ClientWSconn struct {
	// The websocket connection.
	ws *websocket.Conn
	// Web Socket write mutex.
	writeMutex *sync.Mutex
	// Connected since.
	connectedSince time.Time
	// Metadata
	metadata co.JSONIdentityNotify
	// Done channel
	done chan bool
	// Local listen port
	listenPort int
	// Reader
	reader io.Reader
}

func ClientWSHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Caught new Client WebSocket connection, upgrading...")
	ws, err := ClientWSupgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Problem upgrading Client WebSocket connection: %+v", err)
		return
	}

	log.Printf("Creating connection object...")
	c := &ClientWSconn{
		writeMutex: &sync.Mutex{},
		ws:         ws,
		done:       make(chan bool),
	}

	log.Printf("Performing Identity Handshake...")
	err = c.IdentityHandshake()
	if err != nil {
		log.Printf("HANDSHAKE FAILED - CLOSING CONNECTION: %s", err)
		c.CloseCleanup()
		return
	}
	log.Printf("Identity Handshake Complete.")

	c.PostHandshakeInit()

	go c.writePump()
	go c.readPump()
}

// Initial identity handshake before we start doing our usual stuff.
func (c *ClientWSconn) IdentityHandshake() error {
	var err error = nil

	// Step 1. Request identity from client...
	log.Printf("Handshake Step 1: Requesting identity from client...")
	request := &co.JSONType{MType: "identityRequest"}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("Failed to Marshal JSON for \"identityRequest\": %s", err)
	}
	err = c.writeData(websocket.TextMessage, requestJSON)
	if err != nil {
		return fmt.Errorf("Failed to write \"identityRequest\" to WebSocket during handshake: %s", err)
	}

	// Step 2. Read the identity response from the client...
	log.Printf("Handshake Step 2: Waiting for identityNotify from client...")
	var mType int = 0
	var message []byte = nil
	for mType != 1 && err == nil {
		mType, message, err = c.ws.ReadMessage()
		log.Printf("Read message of type %d from WebSocket during handshake.", mType)
	}
	if err != nil {
		return fmt.Errorf("Failed reading from WebSocket during handshake: %s", err)
	}
	response := co.JSONIdentityNotify{}
	if err = json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("Failed to Unmarshal JSON during handshake: %s", err)
	}
	if !(response.MType == "identityNotify" && response.Project != "") {
		return fmt.Errorf("Response type or metadata invalid during handshake")
	}

	// Set metadata against object.
	c.metadata = response
	// Allocate port number.
	c.listenPort = c.AllocatePort()

	// Step 3. Send confirmation to client...
	log.Printf("Handshake Step 3: Sending confirmation to client...")
	confirm := &co.JSONIdentityConfirm{MType: "identityConfirm", Accepted: true}
	confirmJSON, err := json.Marshal(confirm)
	if err != nil {
		return fmt.Errorf("Error marshaling IdentityConfirm in handshake. %s", err)
	}
	log.Printf("Finishing handshake with identity confirmation")
	err = c.writeData(websocket.TextMessage, confirmJSON)
	if err != nil {
		return fmt.Errorf("Problem writing to WebSocket during end of handshake: %s", err)
	}

	c.connectedSince = time.Now()

	log.Printf("Returning from handshake, all completed...")

	return nil
}

func (c *ClientWSconn) AllocatePort() int {
	clientMutex.Lock()

	var port int

	for i := globalConfig.LocalListenStart; i < globalConfig.LocalListenEnd; i++ {
		if _, ok := clientMAP[i]; !ok {
			port = i
			break
		}
	}

	if port < 1 {
		panic(fmt.Errorf("Unable to Allocate Port"))
	}

	clientMAP[port] = c

	clientMutex.Unlock()

	return port
}

func (c *ClientWSconn) PostHandshakeInit() error {
	go StartMuxadoSession(c.listenPort, c, c.done)

	return nil
}

func (c *ClientWSconn) Write(p []byte) (n int, err error) {
	return len(p), c.writeData(websocket.BinaryMessage, p)
}

func (c *ClientWSconn) Read(p []byte) (n int, err error) {
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

func (c *ClientWSconn) Close() error {
	c.CloseCleanup()

	return nil
}

// Thread-safe write to client, takes payload type & payload.
func (c *ClientWSconn) writeData(mt int, payload []byte) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *ClientWSconn) readPump() {
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
}

func (c *ClientWSconn) writePump() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()

		c.CloseCleanup()
	}()
	for {
		select {
		case <-pingTicker.C:
			if err := c.writeData(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *ClientWSconn) CloseCleanup() {
	// Send the done signal to our proxy.
	c.done <- true

	// Close the WebSocket connection.
	c.ws.Close()

	clientMutex.Lock()
	// Remove the user from our user map.
	if _, ok := clientMAP[c.listenPort]; ok {
		delete(clientMAP, c.listenPort)
	}
	clientMutex.Unlock()
}
