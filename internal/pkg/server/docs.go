// Package server implements the server side of the RISP protocol.
//
// The server performs the following steps:
// 	1. Sets up a gRPC server to handle incoming connections from clients.
// 	2. On connection with a client, it receives the initial handshake message with state CONNECTING,
// 	   specifying the client UUID and sequence length and a small window size.
// 	3. The server initialises the client session state with a random sequence, before sending the
// 	   server response with the state CONNECTED, containing the first payload item.
// 	4. The server repeatedly sends sequence items until the window size is exhausted.
// 	5. When the server receives an acknowledgment from the client, it updates the session state for the client
// 	   according to the new known state and continues to send the next payload items.
// 	6. When the server receives a CLOSING message from the client, it sends the CLOSING reply with the expected checksum.
// 	7. The server receives the CLOSED message from the client, and it sends the CLOSED reply.
//
// An instance of Server captures the expected state of the client in memory. When the client confirms the state,
// it updates the client state in a session store. This session store could be adapted to a persistent store like Redis
// to allow multiple server instances to handle client reconnections.
//
// Additional flags can be specified to control the server message sending interval.
//
// TODO: it would be nice to switch up message ordering, to demonstrate how the protocol can deal with this.
// TODO: it would be nice to intermittently drop messages, to demonstrate how the protocol can deal with this.
package server
