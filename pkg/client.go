package pkg

import (
	"bufio"
	"encoding/json"
	"github.com/ManouchehrRasoulli/rfswatcher/internal"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/protocol"
	"log"
	"net"
	"time"
)

type Client struct {
	address string
	logger  *log.Logger
	f       *internal.Handler
	exit    chan struct{}
}

func NewClient(address string, logger *log.Logger) *Client {
	c := Client{
		address: address,
		logger:  logger,
		f:       nil,
		exit:    make(chan struct{}),
	}

	return &c
}

func (c *Client) Exit() error {
	close(c.exit)
	return nil
}

func (c *Client) Run() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	c.logger.Printf("client :: connected to host %s ...\n", c.address)

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
				c.logger.Printf("client error :: got error %v, %T...\n", err, err)
				continue
			}

			d := protocol.Data{}
			err = json.Unmarshal(data[:len(data)-1], &d)
			if err != nil {
				c.logger.Printf("client error :: got error %v on unmarshalling data %s\n", err, string(data))
				continue
			}

			c.logger.Printf("client :: got data %v\n", d)
		}
	}
}
