# RISP: Realtime Integer Streaming Protocol

- [RISP: Realtime Integer Streaming Protocol](#risp-realtime-integer-streaming-protocol)
  - [Overview](#overview)
    - [Features](#features)
    - [Available Commands](#available-commands)
      - [Getting Help](#getting-help)
      - [RISP Client](#risp-client)
      - [RISP Server](#risp-server)
    - [Protocol](#protocol)
      - [Message Structure](#message-structure)
      - [Choreography](#choreography)
    - [Project Structure](#project-structure)
  - [Getting Started](#getting-started)
    - [Application Setup](#application-setup)
    - [Additional Help](#additional-help)
    - [Running Tests](#running-tests)
    - [Run a Demo](#run-a-demo)
  - [Additional Notes](#additional-notes)
    - [Further Improvements](#further-improvements)

## Overview

RISP is an application implementing an demo protocol for streaming a sequence of integers from a server to a client.

### Features

RISP supports the following features:

- The server can handle multiple clients concurrently.
- A checksum sent by the server is used to verify the sequence received by the client is correct.
- A dynamic window size is used to adapt to connection stability.
- An acknowledgement strategy is implemented to ensure that dropped messages are resent by the server.
- The connection is stateful and the server uses a session store to persist client state. Clients can thus freely disconnect and reconnect (within 30s) to resume receiving the sequence.

### Available Commands

#### Getting Help

```
❯ risp help
Usage:
   [flags]
   [command]

Available Commands:
  client      Starts a RISP client.
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  server      Starts a RISP server.

Flags:
      --env string           Describes the current environment and should be one of: local, test, dev, prod. (default "local")
      --health_port int      The port the health server should listen on. (default 8080)
  -h, --help                 help for this command
      --log_level string     Sets the log level and should be one of: debug, info, warn, error. (default "debug")
      --max_goroutines int   The maximum allowed number of goroutines that can be spawned before healthchecks fail. (default 200)
      --port int             The port the gRPC server should listen on. (default 8081)

Use " [command] --help" for more information about a command.
```

#### RISP Client

```
❯ risp client --help
Starts a RISP client.

Usage:
   client [sequence_length] [flags]

Flags:
      --client_killswitch_ms int   The number of milliseconds between client disconnections. Leave unset to not trigger this behaviour.
      --client_ticker_ms int       The number of milliseconds between client messages. (default 2000)
  -h, --help                       help for client

Global Flags:
      --env string           Describes the current environment and should be one of: local, test, dev, prod. (default "local")
      --health_port int      The port the health server should listen on. (default 8080)
      --log_level string     Sets the log level and should be one of: debug, info, warn, error. (default "debug")
      --max_goroutines int   The maximum allowed number of goroutines that can be spawned before healthchecks fail. (default 200)
      --port int             The port the gRPC server should listen on. (default 8081)
```

#### RISP Server

```
❯ risp server --help
Starts a RISP server.

Usage:
   server [flags]

Flags:
  -h, --help                   help for server
      --server_ticker_ms int   The number of milliseconds between server messages. (default 1000)

Global Flags:
      --env string           Describes the current environment and should be one of: local, test, dev, prod. (default "local")
      --health_port int      The port the health server should listen on. (default 8080)
      --log_level string     Sets the log level and should be one of: debug, info, warn, error. (default "debug")
      --max_goroutines int   The maximum allowed number of goroutines that can be spawned before healthchecks fail. (default 200)
      --port int             The port the gRPC server should listen on. (default 8081)
```

### Protocol

The RISP protocol is _loosely_ modelled on TCP.

#### Message Structure

A message can be in one of four states:

1. `CONNECTING` - this state is used for the setup handshake. It is only sent by a client and indicates that it is trying to (re)connect to the server.
2. `CONNECTED` - this indicates that the connection is healthy and data transfer is occuring.
3. `CLOSING` - this state is used for the closing handshake. The client sends a `CLOSING` message to request for the checksum, which the server returns on a `CLOSING` message.
4. `CLOSED` - this state is used for the closing handshake. The client sends a `CLOSED` message to indicate that the sequence has been received, to which the server responds with a `CLOSED` message.

A client message includes the following fields:

- `state` is one of the aforementioned message states.
- `len` is the length of the sequence requested
- `uuid` is the client's UUID
- `window` is the current window size
- `ack` is the position of the last successfully received message

A server message includes the following fields:

- `state` is one of the aforementioned message states.
- `index` is the index of the payload in the sequence
- `payload` is the value in the sequence at the given index
- `checksum` is the sum of all values in the sequence

#### Choreography

The usual correspondence is as follows:

1. The client sends a `CONNECTING` message to the server with its UUID, a sequence length and an initial window size.
2. The server sends back up to _window_ `CONNECTED` messages to the client with the sequence values on the payload.
3. The client when all messages in the window are received, or a timeout occurs, the client sends a `CONNECTED` message to the server with the `ack` field set to the index of the last known sequence element. If messages were received without issue, the client can increase the window size.
4. Steps 2 and 3 repeat until the client receives the entire sequence, at which point it sends a `CLOSING` message with the `ack` value set to the sequence length.
5. The server responds with a `CLOSING` message containing the checksum of the sequence.
6. The client sends a `CLOSED` message on receipt of the checksum, and the server replied with a `CLOSED` message before terminating the connection.

A client can disconnect at any point in the flow. If it reconnects with a `CONNECTING` message and the same UUID, the server will restore the session state.

If messages sent by the server are lost, the client can request them again by sending a `CONNECTING` message with the `ack` flag set to the first missing index in the sequence. The server will then resend sequence values from that point forwards.

### Project Structure

This project is organised in accordance with the best practices described [here](https://github.com/golang-standards/project-layout).

The folder structure is as follows:

- `/bin` holds any built executable binaries.
- `/cmd/<name>` holds the top-level entrypoints. Each folder `<name>` should hold a `main.go` to produce a standalone binary for that application.
- `/pkg` holds public packages that can be imported and used by external projects. Strict semantic versioning must be followed here.
- `/scripts` holds utility bash scripts and the like.
- `/configs` holds application config.
- `/api` holds api and service definitions.
- `/internal` holds logic internal to the application which should not be imported by external projects (this is enforced by the Go compiler)
- `/internal/pkg` holds internal package code
- `/internal/app/apps` holds different application entrypoints. For example, a single binary can have multiple execution modes by leveraging different app implementations in here, while keeping dependencies separate.
- `/internal/app/cfg` holds application configuration to connect to external dependencies (databases, services, etc). Each configuration is implemented just once but can be used to configure any app type that needs it.

## Getting Started

### Application Setup

To build the application binary (you need Go 1.17):

```
make build
```

This puts the application binary in `./bin/risp`. For ease of use, you can create an alias for your current session:

```
alias risp="./bin/risp"
```

Now all you need to do is set up your environment for the current shell session:

```
export $(cat ./configs/dev.env | xargs)
```

And now you're ready to start running `risp` commands!

### Additional Help

To learn more about what you can do, run `make help`.

```
❯ make help
 help:                          Shows help messages.
 clean:                         Cleans up build artefacts.
 lint:                          Runs linters.
 docs:                          Starts the Go documentation server.
 mocks:                         Generate mocks in all packages.
 proto:                         Builds the proto files.
 build_dependencies:            Builds the application dependencies.
 build:                         Builds the application. [cmd]
 test_integ_deps:               Prepares dependencies for running integration tests, creating and starting all containers.
 test_start_containers:         Starts all the existing containers for the test environment.
 test_integ:                    Runs integration tests. [timeout, dir, flags, run]
 test_unit:                     Runs unit tests. [run, flags, timeout, dir]
 exec:                          Executes the built application binary. [cmd, subcommand, flags]
 run:                           Runs the application using go run. [cmd, subcommand, flags]
```

### Running Tests

**To run unit tests:**

```
make test_unit
```

**To run integration tests:**

```
make test_integ
```


### Run a Demo

Start the server:

```
risp server
```

In another terminal, start the client (you can run many of these at once):

```
risp client 50
```

You can also watch the demo video in the project root.

## Additional Notes

### Further Improvements

- Dockerise server application. I didn't have time for this, but I hope you don't have too much trouble getting up and running.
- Comprehensive unit and integration testing. I didn't have time for this, but I included some small example tests to demonstrate my awareness of the topic.
- Secure the gRPC connection with TLS.
- Use a _selective acknowledgement_ strategy to further minimise re-sending messages that the client has successfully received. (Currently, everything is resent from the first undelivered item in the sequence onwards).
- Add a session cleanup strategy for the server, to clean up the session state after 30s. I didn't have time to implement this.
