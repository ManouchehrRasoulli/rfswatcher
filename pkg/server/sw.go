package server

import (
	"github.com/ManouchehrRasoulli/rfswatcher/internal"
	"log"
	"net"
)

type Server struct {
	address string
	w       *internal.Watcher
	l       net.Listener
	log     *log.Logger
}

func NewServer(address string, w *internal.Watcher, lg *log.Logger) *Server {
	s := Server{
		address: address,
		w:       w,
		l:       nil,
		log:     lg,
	}

	return &s
}

func (s *Server) Listen() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.l = l
	return nil
}

func (s *Server) Run() error {
	for {
		c, err := s.l.Accept()
		if err != nil {
			s.log.Printf("Server :: got error (%v) on accepting connection !!", err)
			return Exit
		}

		s.log.Printf("Server :: accept handle connection --> {remote-address: %s, network: %s}", c.RemoteAddr().String(), c.RemoteAddr().Network())
		go s.serve(c)
	}
}

func (s *Server) Close() error {
	if s.w != nil {
		s.w.Close()
	}
	return s.l.Close()
}

func (s *Server) serve(c net.Conn) {
}
