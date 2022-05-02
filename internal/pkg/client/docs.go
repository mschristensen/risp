// Package client implements the client side of the RISP protocol.
//
// The client performs the following steps:
//	1. Connect to the server.
//	2. Send the initial handshake message with state CONNECTING, specifying the client UUID and sequence length and a small window size.
// 	3. Receive the server response with the state CONNECTED, containing the first payload item.
// 	4. The client repeatedly receives payloads and stores them at the correct place in the sequence.
//  5. When the window size is exhausted, the client sends a CONNECTED message to the server acknowledging the payloads received in the window.
// 	6. When all messages have been received, the client sends a CLOSING message to the server
// 	7. The client receives the CLOSING reply from the server with the expected checksum.
// 	8. The client sends the CLOSED message to the server, and receives the CLOSED reply.
//  9. The client finally logs the result to stdout and disconnects from the server.
//
// An instance of Client captures the known state of the sequence in memory.
//
// The client also implements a dynamic window sizing strategy, whereby the window size
// doubles when all values in the window are received without a disconnection by the client,
// but is reset to a default small value on disconnection. This strategy therefore tries to
// optimise the data exchange according to the connection stability.
//
// When the client disconnects, it returns ErrClientDisconnected. The reconnection must be performed by the caller.
//
// Additional flags can be specified to control the client message sending interval and the killswitch interval (to trigger disconnections).
//
package client
