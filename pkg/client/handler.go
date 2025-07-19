package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg/protocol"
)

// Login into the server(send join packet).
func (c *Client) Auth(conn net.Conn, username string, password string) error {
	if username == "" {
		return nil
	}

	reqPayload, _ := json.Marshal(protocol.JoinPayload{
		Username: username,
		Password: password,
	})
	req := protocol.Data{
		Sec:     0,
		Time:    time.Now(),
		Type:    protocol.Join,
		Heading: nil,
		Payload: reqPayload,
	}

	rb, err := json.Marshal(req)
	if err != nil {
		return errors.Join(ErrClientMarshalPacket, err)
	}

	rb = append(rb, '@')
	n, err := conn.Write(rb)
	if err != nil {
		return errors.Join(ErrClientWritePacket, err)
	}

	reqBytesLen := len(rb)
	if n != reqBytesLen {
		subErr := fmt.Errorf("%d != %d", n, reqBytesLen)
		return errors.Join(ErrClientInconsistentWrite, subErr)
	}

	err = conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	if err != nil {
		return errors.Join(ErrClientReadDeadline, err)
	}

	r := bufio.NewReader(conn)
	data, err := r.ReadBytes('@')
	if err != nil {
		return errors.Join(ErrClientReadPacket, err)
	}

	data = data[:len(data)-1]

	response := protocol.Data{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return errors.Join(ErrClientUnmarshalResponsePacket, err)
	}

	if response.Type != protocol.AckJoin {
		subErr := fmt.Errorf("expect %d(ack join) but received %d", protocol.AckJoin, response.Type)
		return errors.Join(ErrClientInvalidPacketType, subErr)
	}

	ackJoinPayload := &protocol.AckJoinPayload{}
	err = json.Unmarshal(response.Payload, &ackJoinPayload)
	if err != nil {
		return errors.Join(ErrClientUnmarshalResponsePacket, err)
	}

	if !ackJoinPayload.Ok {
		var subErr error
		if ackJoinPayload.Msg != "" {
			subErr = errors.New(ackJoinPayload.Msg)
		}
		return errors.Join(ErrClientAuthenticationFailed, subErr)
	}

	return nil
}
