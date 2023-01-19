package pkg

import (
	"encoding/json"
	"github.com/ManouchehrRasoulli/rfswatcher/internal"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/protocol"
	"log"
	"net"
	"strings"
	"time"
)

type Mode int

type Server struct {
	address string
	logger  *log.Logger
	e       chan internal.Event
	f       *internal.Handler
	exit    chan struct{}
}

func NewServer(address string, logger *log.Logger, f *internal.Handler) *Server {
	s := Server{
		address: address,
		logger:  logger,
		e:       make(chan internal.Event, 2),
		f:       f,
		exit:    make(chan struct{}, 0),
	}

	return &s
}

func (s *Server) Exit() error {
	close(s.exit)
	return nil
}

func (s *Server) EventHook(event internal.Event, err error) {
	if err != nil {
		s.logger.Printf("server error :: receive error %v on hook\n", err)
		return
	}

	if strings.Contains(event.Name, "swp") ||
		strings.Contains(event.Name, ".goutputstream") ||
		strings.HasSuffix(event.Name, "~") ||
		event.Op.Has(internal.Chmod) {
		return
	}

	s.e <- event
}

func (s *Server) Run() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}

	s.logger.Printf("server :: running on host %s, port %s ...\n", host, port)

	for {
		conn, err := l.Accept() // handle single connection
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			for {
				select {
				case e := <-s.e:
					{
						fMeta := s.f.GetMeta(e.Name)
						if fMeta == nil {
							s.logger.Printf("server error :: nil meta data !! for event %v\n", e)
							continue
						}

						data := protocol.Data{
							Sec:     0,
							Time:    time.Now(),
							Type:    protocol.ChangeNotify,
							Heading: nil,
							Payload: protocol.FileMetaPayload{
								Path:       "",
								FileName:   e.Name,
								Op:         e.Op,
								Size:       fMeta.Size,
								ChangeDate: fMeta.ModifyTime,
							},
						}

						dataByte, err := json.Marshal(data)
						if err != nil {
							s.logger.Printf("server error :: %v on marshalling data %v\n", err, data)
							continue
						}

						dataByte = append(dataByte, '@')

						n, err := conn.Write(dataByte)
						if err != nil {
							s.logger.Printf("server error :: %v writing data\n", err)
							continue
						}

						if n != len(dataByte) {
							s.logger.Printf("server error :: inconsistent data write !! %d != %d\n", n, len(dataByte))
							continue
						}
					}
				case <-s.exit:
					_ = conn.Close()
				}
			}
		}(conn)
	}
}
