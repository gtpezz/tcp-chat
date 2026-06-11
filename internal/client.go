package internal

import "net"

type Client struct {
	Name string
	Conn net.Conn
	Send chan string
}

func (c *Client) WritePump() {
	for msg := range c.Send {
		c.Conn.Write([]byte(msg + "\n"))
	}
}
