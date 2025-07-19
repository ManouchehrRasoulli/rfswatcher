package server

import (
	"crypto/tls"
	"log"
	"net"
	"strings"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/user"
)

type Mode int

type ServerTLS struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type Server struct {
	address string
	logger  *log.Logger
	e       chan model.Event
	f       *filehandler.Handler
	exit    chan struct{}
	path    string
	tls     *ServerTLS
	um      *user.UserManager
}

func NewServer(address string, path string, tls *ServerTLS, um *user.UserManager, logger *log.Logger, f *filehandler.Handler) *Server {
	s := Server{
		address: address,
		logger:  logger,
		e:       make(chan model.Event, 2),
		f:       f,
		exit:    make(chan struct{}, 0),
		path:    path,
		tls:     tls,
		um:      um,
	}

	return &s
}

func (s *Server) Exit() error {
	close(s.exit)
	return nil
}

func (s *Server) EventHook(event model.Event, err error) {
	if err != nil {
		s.logger.Printf("server error :: receive error %v on hook\n", err)
		return
	}

	if strings.Contains(event.Name, "swp") ||
		strings.Contains(event.Name, ".goutputstream") ||
		strings.HasSuffix(event.Name, "~") ||
		event.Op.Has(model.Chmod) {
		return
	}

	s.e <- event
}

func (s *Server) Run() error {
	var l net.Listener

	if s.tls == nil {
		s.logger.Println("server warn :: TLS doesn't set")
		ln, err := net.Listen("tcp", s.address)
		if err != nil {
			return err
		}

		l = ln
	} else {
		cert, err := tls.LoadX509KeyPair(s.tls.Cert, s.tls.Key)
		if err != nil {
			return err
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		ln, err := tls.Listen("tcp", s.address, tlsConfig)
		if err != nil {
			return err
		}

		l = ln
	}

	host, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}

	s.logger.Printf("server :: running on host %s, port %s ...\n", host, port)

	for {
		var conn net.Conn
		conn, err = l.Accept() // handle single connection
		if err != nil {
			return err
		}

		username, err := s.joinHandler(conn)
		if err != nil {
			s.logger.Printf("server error :: %v\n", err)
			conn.Close()
			continue
		}

		go s.handleAuthenticatedConnection(conn, username)
	}
}
