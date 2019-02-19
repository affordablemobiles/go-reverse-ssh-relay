## Reverse Connecting SSH (via concentrator) over WebSocket

This is not an officially supported A1comms product; it was developed demonstration project on a one-off basis.

### About

The "endpoint" connects outbound to the "concentrator" via a secure WebSocket.

The "concentrator" starts a local listening socket and forwards connections to it back to the remote "endpoint".

The "endpoint" in turn has an SSH server that accepts those connections and provides a shell & exec functionality.
