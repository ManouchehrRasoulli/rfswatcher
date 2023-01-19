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
	address  string
	logger   *log.Logger
	f        *internal.Handler
	exit     chan struct{}
	download chan protocol.FileMetaPayload
}

func NewClient(address string, logger *log.Logger) *Client {
	c := Client{
		address:  address,
		logger:   logger,
		f:        nil,
		exit:     make(chan struct{}),
		download: make(chan protocol.FileMetaPayload, 1),
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

	req := protocol.Data{
		Sec:     0,
		Time:    time.Time{},
		Type:    protocol.SubscribePath,
		Heading: nil,
		Payload: protocol.SubscribePathPayload{
			Path: "root",
			Id:   "10",
		},
	}
	rb, err := json.Marshal(req)
	if err != nil {
		c.logger.Printf("client error :: got error %v on marshal subscribe request\n", err)
		return err
	}

	rb = append(rb, '@')
	n, err := conn.Write(rb)
	if n != len(rb) {
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
				c.logger.Printf("client error :: got error %v, %T...\n", err, err)
				continue
			}

			d := protocol.Data{}
			err = json.Unmarshal(data[:len(data)-1], &d)
			if err != nil {
				c.logger.Printf("client error :: got error %v on unmarshalling data %s\n", err, string(data))
				continue
			}

			if d.Type == protocol.ChangeNotify {
				c.logger.Printf("client :: got modification notification %v\n", d)

				dp, ok := d.Payload.(protocol.FileMetaPayload)
				if !ok {
					c.logger.Printf("client :: error invalid file notify change payload %v, %T\n", d.Payload, d.Payload)
					continue
				}

				c.download <- dp
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
					if e.Op.Has(internal.Write | internal.Create) {
						// download file
						_, _ = net.Dial("tcp", c.address)

					}
					if e.Op.Has(internal.Remove) {
						// remove files
						c.logger.Printf("client worker :: remove file notification %v !!", e)
						continue
					}
				}
			case <-c.exit:
				return
			}
		}
	}()
}
