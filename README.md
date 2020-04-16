## Reverse Connecting SSH (via relay) over WebSocket

This app allows inbound SSH into a target, when only outbound connectivity is allowed from the target (reverse shell style).

The SSH server is implemented in the Go endpoint itself, but it could just as easily be modified to redirect to the system SSH service instead.

The outbound connection from the target is made over HTTPS, utilising WebSockets for two way communication.

### About

The "endpoint" connects outbound to the "relay" via a secure WebSocket.

The "relay" starts a local listening socket and forwards connections to it back to the remote "endpoint".

The "endpoint" in turn has an SSH server that accepts those connections and provides a shell & exec functionality.

### Use Case

This was developed to allow us SSH access into App Engine Standard Environment (2nd generation) runtime instances on GCP, for file sync and debugging in a development / pre-production environment.
