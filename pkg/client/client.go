package client

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/protocol"
)

var (
	ErrClientReadPacket              = errors.New("failed to read packet from connection")
	ErrClientMarshalPacket           = errors.New("failed to marshal packet data")
	ErrClientWritePacket             = errors.New("failed to write packet to connection")
	ErrClientInconsistentWrite       = errors.New("inconsistent data write: bytes written mismatch")
	ErrClientAuthenticationFailed    = errors.New("authentication failed")
	ErrClientInvalidPacketType       = errors.New("invalid packet type received")
	ErrClientUnmarshalResponsePacket = errors.New("failed to unmarshal response packet data")
	ErrClientReadDeadline            = errors.New("failed to set read deadline on connection")
)

type Client struct {
	tls      *tls.Config
	address  string
	username string
	password string
	logger   *log.Logger
	f        *filehandler.Handler
	exit     chan struct{}
	download chan protocol.FileMetaPayload
}

func NewClient(address string, username string, password string, tls *tls.Config, logger *log.Logger, f *filehandler.Handler) *Client {
	c := Client{
		tls:      tls,
		address:  address,
		username: username,
		password: password,
		logger:   logger,
		f:        f,
		exit:     make(chan struct{}),
		download: make(chan protocol.FileMetaPayload, 1),
	}

	go c.downloader() // run download daemon

	return &c
}

func (c *Client) Exit() error {
	close(c.exit)
	return nil
}

func (c *Client) Run() error {
	var conn net.Conn
	if c.tls != nil {
		_conn, err := tls.Dial("tcp", c.address, c.tls)
		if err != nil {
			return err
		}

		conn = _conn
	} else {
		_conn, err := net.Dial("tcp", c.address)
		if err != nil {
			return err
		}

		conn = _conn
	}

	c.Auth(conn, c.username, c.password)

	c.logger.Printf("client :: connected to host %s ...\n", c.address)

	req := protocol.Data{
		Sec:     0,
		Time:    time.Now(),
		Type:    protocol.SubscribePath,
		Heading: nil,
		Payload: nil,
	}
	rb, err := json.Marshal(req)
	if err != nil {
		c.logger.Printf("client error :: got error %v on marshal subscribe request\n", err)
		return err
	}

	rb = append(rb, '@')
	n, err := conn.Write(rb)
	if n != len(rb) || err != nil {
		c.logger.Printf("client error :: send subscribe request %v\n", err)
		return err
	}

	r := bufio.NewReader(conn)
	for {
		select {
		case <-c.exit:
			return conn.Close()
		default:
			err = conn.SetReadDeadline(time.Now().Add(time.Second * 10))
			if err != nil {
				c.logger.Printf("client error :: got error %v\n", err)
				continue
			}
			data, err := r.ReadBytes('@')
			if err != nil {
				continue
			}

			d := protocol.Data{}
			err = json.Unmarshal(data[:len(data)-1], &d)
			if err != nil {
				c.logger.Printf("client error :: got error %v on unmarshalling data %s\n", err, string(data))
				continue
			}

			if d.Type == protocol.ChangeNotify {
				payload := protocol.FileMetaPayload{}
				err = json.Unmarshal(d.Payload, &payload)
				if err != nil {
					c.logger.Printf("client :: error invalid file notify change payload %v, %T\n", string(d.Payload), d.Payload)
					continue
				}

				c.download <- payload
			} else {
				c.logger.Printf("client :: got data %v !!\n", d)
			}
		}
	}
}

func (c *Client) downloader() {
	go func() {
		for {
			select {
			case e := <-c.download:
				{
					if e.Op.Has(model.Write) {
						// download file
						var conn net.Conn
						if c.tls != nil {
							_conn, err := tls.Dial("tcp", c.address, c.tls)
							if err != nil {
								c.logger.Printf("client worker :: error establish download connection %v\n", err)
								continue
							}

							conn = _conn
						} else {
							_conn, err := net.Dial("tcp", c.address)
							if err != nil {
								c.logger.Printf("client worker :: error establish download connection %v\n", err)
								continue
							}

							conn = _conn
						}

						if err := c.Auth(conn, c.username, c.password); err != nil {
							continue
						}

						reqPayload, _ := json.Marshal(protocol.RequestFilePayload{
							Path:       e.Path,
							FileName:   e.FileName,
							ChangeDate: e.ChangeDate,
						})
						req := protocol.Data{
							Sec:     0,
							Time:    time.Now(),
							Type:    protocol.RequestFile,
							Heading: nil,
							Payload: reqPayload,
						}

						rb, err := json.Marshal(req)
						if err != nil {
							c.logger.Printf("client worker :: got error %v on marshal subscribe request\n", err)
							continue
						}

						rb = append(rb, '@')
						n, err := conn.Write(rb)
						if n != len(rb) || err != nil {
							c.logger.Printf("client worker :: send subscribe request %v\n", err)
							continue
						}

						err = conn.SetReadDeadline(time.Now().Add(time.Second * 30))
						if err != nil {
							c.logger.Printf("client worker :: read file deadline %v\n", err)
							continue
						}

						r := bufio.NewReader(conn)
						data, err := r.ReadBytes('@')
						if err != nil {
							c.logger.Printf("client worker :: got error %v, %T... on reading file\n", err, err)
							continue
						}

						data = data[:len(data)-1]

						response := protocol.Data{}
						err = json.Unmarshal(data, &response)
						if err != nil {
							c.logger.Printf("client worker ERROR :: error %v on reading file request response !!", err)
							continue
						}

						err = c.f.WriteFile(e.FileName, response.Payload)
						if err != nil {
							c.logger.Printf("client worker ERROR :: error %v on write data into file %s!!\n", e, e.FileName)
						}
						continue
						// read connection and tell file management to create file
					}
					if e.Op.Has(model.Remove) {
						// remove files
						c.logger.Printf("client worker :: remove file notification %v !!\n", e)
						err := c.f.RemoveFile(e.FileName)
						if err != nil {
							c.logger.Printf("client worker ERROR :: error %v on remove file %s !!\n", e, e.FileName)
						}
						continue
					}
				}
			case <-c.exit:
				return
			}
		}
	}()
}
