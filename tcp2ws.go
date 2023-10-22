package tcp2ws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

// ForwardTCP handles a TCP connection and forwards it to a WebSocket server.
// Handle blocks until the connection is closed or ctx is canceled.
func ForwardTCP(ctx context.Context, tcpConn net.Conn, websocketURL string) error {
	defer tcpConn.Close()

	wsConn, _, err := websocket.Dial(ctx, websocketURL, nil)
	if err != nil {
		return err
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	return Pipe(ctx, wsConn, tcpConn, false)
}

// Pipe forwards data between a WebSocket connection and a TCP connection.
func Pipe(ctx context.Context, ws *websocket.Conn, tcp net.Conn, wsPing bool) error {
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

	if wsPing {
		go func() {
			defer cancel()
			for {
				tctx, tcancel := context.WithTimeout(ctx, 4800*time.Millisecond)
				if err := ws.Ping(tctx); err != nil {
					tcancel()
					return
				}
				tcancel()
				time.Sleep(time.Second * 5)
			}
		}()
	}

	<-ctx.Done()

	var err error
	select {
	case err = <-errCh:
	default:
	}

	return err
}

// ForwardWebsocket handles a WebSocket connection and forwards it to a TCP server.
func ForwardWebsocket(w http.ResponseWriter, r *http.Request, tcpAddr string) {
	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		http.Error(w, "websocket accept error", http.StatusInternalServerError)
		return
	}
	// FIXME This is not an optimal status for all exit paths.
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	tcpConn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		log.Printf("tcp connect error: %v", err)
		return
	}
	defer tcpConn.Close()

	if err := Pipe(r.Context(), wsConn, tcpConn, true); err != nil {
		log.Print(err)
		return
	}
}
