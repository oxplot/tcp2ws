# tcp2ws

`tcp2ws` is a Go-based tool that provides bidirectional TCP to WebSocket
protocol conversion. It consists of two commands that allow you to:

1. Forward TCP connections to a WebSocket server (`tcp2ws`)
2. Forward WebSocket connections to a TCP server (`ws2tcp`)

This enables TCP-only clients to communicate with WebSocket servers and vice
versa, making it useful for scenarios where protocol conversion is needed, such
as connecting legacy TCP applications to modern WebSocket services.

## Requirements

- Go 1.21 or later

## Installation

```bash
go install github.com/oxplot/tcp2ws/cmd/tcp2ws@latest
go install github.com/oxplot/tcp2ws/cmd/ws2tcp@latest
```

## Usage

### tcp2ws

`tcp2ws` listens for TCP connections and forwards them to a WebSocket server.

```bash
tcp2ws [options] ws[s]://host:port/path

Options:
  -listen string
        TCP listen address:port (default ":7101")
```

Example:

```bash
# Forward TCP connections from port 7101 to a WebSocket server
tcp2ws -listen :7101 ws://example.com:8080/ws
```

### ws2tcp

`ws2tcp` runs a WebSocket server that forwards connections to a TCP server.

```bash
ws2tcp [options] tcp-host:port

Options:
  -listen string
        WebSocket listen address:port (default ":8080")
```

Example:

```bash
# Accept WebSocket connections on port 8080 and forward them to a TCP server
ws2tcp -listen :8080 localhost:6379
```

## Features

- Bidirectional data forwarding between TCP and WebSocket protocols
- Support for both ws:// and wss:// (secure WebSocket) protocols
- Configurable listen addresses and ports
- Automatic connection cleanup and resource management
- Error handling and logging
