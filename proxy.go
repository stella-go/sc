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
	debug_print("server from: ", from)
	c, err := net.Dial(network, out)
	if err != nil {
		fmt.Printf("dial out error: %v\n", err)
		return
	}

	stop1 := make(chan bool, 1)
	stop2 := make(chan bool, 1)
	stop3 := make(chan bool, 1)
	stop4 := make(chan bool, 1)

	inch := make(chan []byte)
	outch := make(chan []byte)

	go func() { // client to proxy
		for {
			select {
			case <-stop1:
				conn.Close()
				debug_print("close 1: ", from)
				return
			default:
				buffer := make([]byte, 1024)
				n, err := conn.Read(buffer)
				if err != nil {
					conn.Close()
					stop2 <- true
					debug_print("close 1: ", from)
					return
				}
				inch <- buffer[:n]
			}
		}
	}()

	go func() { // proxy to server
		for {
			select {
			case <-stop2:
				c.Close()
				debug_print("close 2: ", from)
				return
			case buffer := <-inch:
				_, err := c.Write(buffer)
				if err != nil {
					c.Close()
					stop1 <- true
					debug_print("close 2: ", from)
					return
				}
			}
		}
	}()

	go func() { // server to proxy
		for {
			select {
			case <-stop3:
				c.Close()
				debug_print("close 3: ", from)
				return
			default:
				buffer := make([]byte, 1024)
				n, err := c.Read(buffer)
				if err != nil {
					c.Close()
					stop4 <- true
					debug_print("close 3: ", from)
					return
				}
				outch <- buffer[:n]
			}
		}
	}()

	go func() { // proxy to client
		for {
			select {
			case <-stop4:
				conn.Close()
				debug_print("close 4: ", from)
				return
			case buffer := <-outch:
				_, err := conn.Write(buffer)
				if err != nil {
					conn.Close()
					stop3 <- true
					debug_print("close 4: ", from)
					return
				}
			}
		}
	}()
}

func debug_print(s ...any) {
	// fmt.Println(s...)
}
