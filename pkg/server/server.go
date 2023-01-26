package server

import (
	"bufio"
	"encoding/json"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
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
	e       chan model.Event
	f       *filehandler.Handler
	exit    chan struct{}
	path    string
}

func NewServer(address string, path string, logger *log.Logger, f *filehandler.Handler) *Server {
	s := Server{
		address: address,
		logger:  logger,
		e:       make(chan model.Event, 2),
		f:       f,
		exit:    make(chan struct{}, 0),
		path:    path,
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
		var conn net.Conn
		conn, err = l.Accept() // handle single connection
		if err != nil {
			return err
		}

		r := bufio.NewReader(conn)
		var data []byte
		data, err = r.ReadBytes('@')
		if err != nil {
			s.logger.Printf("server error :: got error %v, %T...\n", err, err)
			continue
		}
		req := protocol.Data{}
		err = json.Unmarshal(data[:len(data)-1], &req)
		if err != nil {
			s.logger.Printf("server error :: got error invalid request %v, %T...\n", err, err)
			continue
		}

		switch req.Type {
		case protocol.SubscribePath:
			{
				go func(conn net.Conn) {
					for {
						select {
						case e := <-s.e:
							{
								if strings.HasPrefix(e.Name, "exit") ||
									e.Op.Has(model.Create) {
									continue
								}

								e.Name = strings.TrimPrefix(e.Name, s.path)
								fMeta := s.f.GetMeta(e.Name)
								if fMeta == nil {
									s.logger.Printf("server error :: nil meta data !! for event %v\n", e)
									continue
								}

								resPaylod, _ := json.Marshal(protocol.FileMetaPayload{
									Path:       s.path,
									FileName:   e.Name,
									Op:         e.Op,
									Size:       fMeta.Size,
									ChangeDate: fMeta.ModifyTime,
								})
								resData := protocol.Data{
									Sec:     0,
									Time:    time.Now(),
									Type:    protocol.ChangeNotify,
									Heading: nil,
									Payload: resPaylod,
								}

								dataByte, cerr := json.Marshal(resData)
								if cerr != nil {
									s.logger.Printf("server error :: %v on marshalling data %v\n", cerr, data)
									continue
								}

								dataByte = append(dataByte, '@')

								n, werr := conn.Write(dataByte)
								if werr != nil {
									s.logger.Printf("server error :: %v writing data\n", werr)
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
		case protocol.RequestFile:
			{ // download changes
				reqPayload := protocol.RequestFilePayload{}
				err = json.Unmarshal(req.Payload, &reqPayload)
				if err != nil {
					s.logger.Printf("server error :: error invalid payload for request file %v !!\n", err)
					continue
				}
				data, err = s.f.ReadFile(reqPayload.FileName)
				if err != nil {
					s.logger.Printf("server error :: error on reading file %v !!\n", err)
					continue
				}

				res := protocol.Data{
					Sec:     req.Sec + 1,
					Time:    time.Now(),
					Type:    protocol.ResponseFile,
					Heading: req.Heading,
					Payload: data,
				}

				data, err = json.Marshal(res)
				if err != nil {
					s.logger.Printf("server error :: %v on marshalling data %v --> response file\n", err, data)
					continue
				}
				data = append(data, '@')
				_, err = conn.Write(data)
				if err != nil {
					s.logger.Printf("server error :: %v on writing file data into socket !!\n", err)
				}
			}
		default:
			s.logger.Printf("server error :: bad request from client %s !!\n", string(data[:len(data)-1]))
			_ = conn.Close()
			continue
		}
	}
}
