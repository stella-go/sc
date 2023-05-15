package main

import (
	"fmt"
	"net"
)

func proxy(in string, out string, network string) {
	go func() {
		l, err := net.Listen(network, fmt.Sprintf(":%s", in))
		if err != nil {
			panic(err)
		}
		defer l.Close()

		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			proxyHandler(out, network, conn)
		}
	}()
}

func proxyHandler(out string, network string, conn net.Conn) {
	from := conn.RemoteAddr().String()
	fmt.Println("server from: ", from)
	c, err := net.Dial(network, out)
	if err != nil {
		return
	}

	stop1 := make(chan bool, 2)
	stop2 := make(chan bool, 2)

	inch := make(chan []byte)
	outch := make(chan []byte)

	go func() { // client to proxy
		for {
			select {
			case <-stop1:
				return
			default:
				buffer := make([]byte, 1024)
				n, err := conn.Read(buffer)
				if err != nil {
					conn.Close()
					stop1 <- true
					return
				}
				inch <- buffer[:n]
			}
		}
	}()

	go func() { // proxy to server
		for {
			select {
			case <-stop1:
				return
			case buffer := <-inch:
				_, err := c.Write(buffer)
				if err != nil {
					c.Close()
					stop1 <- true
					return
				}
			}
		}
	}()

	go func() { // server to proxy
		for {
			select {
			case <-stop2:
				return
			default:
				buffer := make([]byte, 1024)
				n, err := c.Read(buffer)
				if err != nil {
					c.Close()
					stop2 <- true
					return
				}
				outch <- buffer[:n]
			}
		}
	}()

	go func() { // proxy to client
		for {
			select {
			case <-stop2:
				return
			case buffer := <-outch:
				_, err := conn.Write(buffer)
				if err != nil {
					conn.Close()
					stop2 <- true
					return
				}
			}
		}
	}()
}
