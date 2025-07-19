package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/protocol"
)

var (
	ErrServerReadPacket            = errors.New("failed to read packet from connection")
	ErrServerUnmarshalPacket       = errors.New("failed to unmarshal packet data")
	ErrServerWritePacket           = errors.New("failed to write packet to connection")
	ErrServerInconsistentWrite     = errors.New("inconsistent data write: bytes written mismatch")
	ErrServerAuthenticationFailed  = errors.New("authentication failed")
	ErrServerInvalidPacketType     = errors.New("invalid packet type received")
	ErrServerMarshalResponsePacket = errors.New("failed to marshal response packet data")
)

func (s *Server) joinHandler(conn net.Conn) (string, error) {
	r := bufio.NewReader(conn)
	var data []byte
	data, err := r.ReadBytes('@')
	if err != nil {
		return "", errors.Join(ErrServerReadPacket, err)
	}
	req := protocol.Data{}
	err = json.Unmarshal(data[:len(data)-1], &req)
	if err != nil {
		return "", errors.Join(ErrServerUnmarshalPacket, err)
	}

	var username string
	var connIP string
	ackJoinPayload := &protocol.AckJoinPayload{Ok: false}

	if s.um == nil {
		ackJoinPayload.Ok = true
	} else if req.Type == protocol.Join {
		joinPayload := &protocol.JoinPayload{}
		err := json.Unmarshal(req.Payload, &joinPayload)
		if err != nil {
			ackJoinPayload.Ok = false
			ackJoinPayload.Msg = fmt.Sprintf("invalid payload. %v", err)
		} else if s.um.CheckUserPassword(joinPayload.Username, joinPayload.Password) {
			host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

			if s.um.CheckUserIP(joinPayload.Username, host) {
				ackJoinPayload.Ok = false
				ackJoinPayload.Msg = "another system has logged in. if something is wrong call the server admin."
			} else {
				ackJoinPayload.Ok = true
				username = joinPayload.Username
				connIP = host
			}
		} else {
			ackJoinPayload.Ok = false
		}
	} else {
		ackJoinPayload.Ok = false
		ackJoinPayload.Msg = "invalid packet type"
	}

	ackJoinBytes, _ := json.Marshal(ackJoinPayload)
	resData := &protocol.Data{
		Sec:     0,
		Time:    time.Now(),
		Type:    protocol.AckJoin,
		Heading: nil,
		Payload: ackJoinBytes,
	}
	resDataByte, _ := json.Marshal(resData)
	resDataByte = append(resDataByte, '@')

	n, err := conn.Write(resDataByte)
	if err != nil {
		return "", errors.Join(ErrServerWritePacket, err)
	}

	if n != len(resDataByte) {
		subErr := fmt.Errorf("%d != %d", n, len(resDataByte))
		return "", errors.Join(ErrServerInconsistentWrite, subErr)
	}

	if s.um != nil && username != "" && connIP != "" {
		s.um.SetAuthenticatedUser(username, connIP)
		return username, nil
	}

	if ackJoinPayload.Ok {
		return "", nil
	}

	subErr := fmt.Errorf("username: %q, ip: %q", username, connIP)
	return "", errors.Join(ErrServerAuthenticationFailed, subErr)
}

func (s *Server) handleAuthenticatedConnection(conn net.Conn, username string) {
	defer func() {
		conn.Close()

		if s.um != nil {
			s.um.UnsetAuthenticatedUser(username)
		}
	}()

	r := bufio.NewReader(conn)

	for {
		data, err := r.ReadBytes('@')
		if err != nil {
			s.logger.Printf("server error :: %v\n", errors.Join(ErrServerReadPacket, err))
			continue
		}

		req := protocol.Data{}
		err = json.Unmarshal(data[:len(data)-1], &req)
		if err != nil {
			s.logger.Printf("server error :: %v\n", errors.Join(ErrServerUnmarshalPacket, err))
			continue
		}

		switch req.Type {
		case protocol.SubscribePath:
			s.handleSubscription(conn)
		case protocol.RequestFile:
			if err := s.handleFileRequest(conn, &req); err != nil {
				s.logger.Println(err)
			}
		default:
			s.logger.Printf("server error :: %v\n", ErrServerInvalidPacketType)
			return
		}
	}
}

func (s *Server) handleSubscription(conn net.Conn) {
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
					s.logger.Printf("server error :: %v\n", errors.Join(ErrServerMarshalResponsePacket, cerr))
					continue
				}

				dataByte = append(dataByte, '@')

				n, werr := conn.Write(dataByte)
				if werr != nil {
					s.logger.Printf("server error :: %v\n", errors.Join(ErrServerWritePacket, werr))
					continue
				}

				if n != len(dataByte) {
					subErr := fmt.Errorf("%d != %d", n, len(dataByte))
					s.logger.Printf("server error :: %v\n", errors.Join(ErrServerInconsistentWrite, subErr))
					continue
				}
			}
		case <-s.exit:
			conn.Close()
			return
		}
	}
}

func (s *Server) handleFileRequest(conn net.Conn, req *protocol.Data) error {
	reqPayload := protocol.RequestFilePayload{}
	err := json.Unmarshal(req.Payload, &reqPayload)
	if err != nil {
		return fmt.Errorf("server error :: %v", errors.Join(ErrServerUnmarshalPacket, err))
	}
	data, err := s.f.ReadFile(reqPayload.FileName)
	if err != nil {
		return fmt.Errorf("server error :: %v", errors.Join(ErrServerReadPacket, err))
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
		return fmt.Errorf("server error :: %v\n", errors.Join(ErrServerMarshalResponsePacket, err))
	}
	data = append(data, '@')
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("server error :: %v", errors.Join(ErrServerWritePacket, err))
	}

	return nil
}
