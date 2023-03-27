package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket"
	"os"
	"os/signal"
	"time"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	listener, err := net.Listen("tcp", "127.0.0.1:8020")
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler: echoServer{},
	}
	errc := make(chan error, 1)
	go func() {
		errc <- server.Serve(listener)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		panic(err)
	case sig := <-sigs:
		fmt.Println(sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return server.Shutdown(ctx)
}

type echoServer struct{}

func (echoServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"chat"},
	})
	if err != nil {
		fmt.Println(err)
	}

	defer c.Close(websocket.StatusInternalError, "spontaneous connection closure")

	if c.Subprotocol() != "chat" {
		c.Close(websocket.StatusPolicyViolation, "unsupported protocol")
	}

	for {
		typ, r, err := c.Reader(context.Background())
		if err != nil {
			fmt.Println(err)
		}

		w, err := c.Writer(context.Background(), typ)
		if err != nil {
			fmt.Println(err)
		}

		var p []byte
		_, err = r.Read(p)

		if err != nil {
			fmt.Println(err)
		}

		_, err = io.Copy(w, r)
		if err != nil {
			fmt.Println(err)
		}

		err = w.Close()

		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}

		if err != nil {
			fmt.Println(err)
		}
	}
}
