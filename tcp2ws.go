package tcp2ws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"nhooyr.io/websocket"
)

// TCPHandler handles a TCP connection and forwards it to a WebSocket server.
type TCPHandler struct {
	wsURL string
}

// NewTCPHandler creates a new TCPHandler that forwards TCP connections to the
// given WebSocket server.
func NewTCPHandler(wsURL string) *TCPHandler {
	return &TCPHandler{
		wsURL: wsURL,
	}
}

// Handle handles a TCP connection and forwards it to a WebSocket server.
// Handle blocks until the connection is closed or ctx is canceled.
// Handle can be called multiple times concurrently.
func (h *TCPHandler) Handle(ctx context.Context, c net.Conn) error {
	defer c.Close()

	wsConn, _, err := websocket.Dial(ctx, h.wsURL, nil)
	if err != nil {
		return err
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	return pipeConns(ctx, wsConn, c)
}

func pipeConns(ctx context.Context, ws *websocket.Conn, tcp net.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 5)

	go func() {
		defer cancel()
		for {
			t, b, err := ws.Read(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.As(err, &websocket.CloseError{}) {
					return
				}
				errCh <- fmt.Errorf("conn %s: websocket read error: %v", tcp.RemoteAddr(), err)
				return
			}
			if t != websocket.MessageBinary {
				continue
			}
			if _, err = tcp.Write(b); err != nil {
				errCh <- fmt.Errorf("conn %s: tcp write error: %v", tcp.RemoteAddr(), err)
				return
			}
		}
	}()

	go func() {
		defer cancel()
		b := make([]byte, 1024)
		for {
			n, err := tcp.Read(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errCh <- fmt.Errorf("conn %s: tcp read error: %v", tcp.RemoteAddr(), err)
				return
			}
			if n < 1 {
				continue
			}
			if err := ws.Write(ctx, websocket.MessageBinary, b[:n]); err != nil {
				errCh <- fmt.Errorf("conn %s: websocket write error: %v", tcp.RemoteAddr(), err)
				return
			}
		}
	}()

	// go func() {
	// 	for {
	// 		time.Sleep(time.Second * 10)
	// 		if err := ws.Ping(ctx); err != nil {
	// 			return
	// 		}
	// 	}
	// }()

	<-ctx.Done()

	var err error
	select {
	case err = <-errCh:
	default:
	}

	return err
}

// WSHandler handles a WebSocket connection and forwards it to a TCP server.
type WSHandler struct {
	tcpAddr string
}

// NewWSHandler creates a new WSHandler that forwards WebSocket connections to
// the given TCP server.
func NewWSHandler(tcpAddr string) *WSHandler {
	return &WSHandler{
		tcpAddr: tcpAddr,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		http.Error(w, "websocket accept error", http.StatusInternalServerError)
		return
	}
	// FIXME This is not an optimal status for all exit paths.
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	tcpConn, err := net.Dial("tcp", h.tcpAddr)
	if err != nil {
		log.Printf("tcp connect error: %v", err)
		return
	}
	defer tcpConn.Close()

	if err := pipeConns(r.Context(), wsConn, tcpConn); err != nil {
		log.Print(err)
		return
	}
}
