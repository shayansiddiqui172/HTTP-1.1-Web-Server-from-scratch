package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s := &Server{listener: listener, handler: handler}
	go s.listen()
	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("error accepting connection: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		w := response.NewWriter(conn)
		body := []byte(err.Error())
		w.WriteStatusLine(response.StatusBadRequest)
		h := response.GetDefaultHeaders(len(body))
		w.WriteHeaders(h)
		w.WriteBody(body)
		return
	}

	w := response.NewWriter(conn)
	s.handler(w, req)
}
