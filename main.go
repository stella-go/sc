// Copyright 2010-2022 the original author or authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const VERSION = "v1.0.0"

func main() {
	flag.Usage = func() {
		fmt.Printf(`sc: a proxy forwarding tool %s

usage:
        netwnork topology:
            192.1.9.2 ----> 172.1.7.2
            192.1.9.2 <-x-- 172.1.7.2
            192.2.9.1 ----> 172.1.7.2
            192.2.9.1 <-x-- 172.1.7.2
            192.2.9.1 --x-> 192.1.9.2
            192.2.9.1 <-x-- 192.1.9.2

        except access:
            192.2.9.1 ----> 192.1.9.2:8080
        
        commands:
            [172.1.7.2]: sc -a foobar 8888 9999
            [192.1.9.2]: sc -a foobar -c 172.1.7.2:9999 192.1.9.2:8080
            [192.2.9.1]: curl 172.1.7.2:8888
        
`, VERSION)
		flag.PrintDefaults()
	}
	c := flag.Bool("c", false, "client side")
	u := flag.Bool("u", false, "udp")
	v := flag.Bool("v", false, "version")
	auth := flag.String("a", "", "password for authentication, required")
	flag.Parse()
	if *v {
		fmt.Println(VERSION)
		return
	}
	if len(flag.Args()) != 2 {
		flag.Usage()
		return
	}
	if *auth == "" {
		fmt.Printf("auth is empty\n")
		return
	}

	network := "tcp"
	if *u {
		network = "udp"
	}
	if *c {
		client(*auth, flag.Arg(0), flag.Arg(1), network)
	} else {
		server(*auth, flag.Arg(0), flag.Arg(1), network)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Message struct {
	From    string
	Content []byte
}

func (p *Message) String() string {
	return fmt.Sprintf("[%s] %s", p.From, string(p.Content))
}

func server(auth string, in string, out string, network string) {
	m := &sync.Map{}
	inch := make(chan *Message)  // server to client
	outch := make(chan *Message) // client to server

	go func() {
		// server-client inner communication
		l, err := net.Listen(network, fmt.Sprintf(":%s", out))
		if err != nil {
			panic(err)
		}
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			bts := make([]byte, 4)
			_, err = conn.Read(bts)
			if err != nil {
				conn.Close()
				continue
			}
			lenth := binary.BigEndian.Uint32(bts)
			bts = make([]byte, lenth)
			_, err = conn.Read(bts)
			if err != nil {
				conn.Close()
				continue
			}
			if string(bts) != auth {
				fmt.Printf("from: %s, auth not match: %s\n", conn.RemoteAddr().String(), string(bts))
				conn.Close()
				continue
			}
			if err != nil {
				conn.Close()
				continue
			}
			_, err = conn.Write(authBytes("ok"))
			if err != nil {
				conn.Close()
				continue
			}
			innerHandle(inch, outch, conn)
		}
	}()

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
			serverHandle(outch, conn, m)
		}
	}()

	go func() {
		for {
			message := <-inch
			from := message.From
			if conn, ok := m.Load(from); ok {
				_, err := conn.(net.Conn).Write(message.Content)
				if err != nil {
					conn.(net.Conn).Close()
					m.Delete(from)
				}
			}
		}
	}()
}

func client(auth string, in string, out string, network string) {
	m := &sync.Map{}

	inch := make(chan *Message)  // server to client
	outch := make(chan *Message) // client to server

	go func() {
		// server-client inner communication
		conn, err := net.Dial(network, in)
		if err != nil {
			return
		}
		_, err = conn.Write(authBytes(auth))
		if err != nil {
			conn.Close()
		}
		bts := make([]byte, 4)
		_, err = conn.Read(bts)
		if err != nil {
			conn.Close()
			return
		}
		lenth := binary.BigEndian.Uint32(bts)
		bts = make([]byte, lenth)
		_, err = conn.Read(bts)
		if err != nil {
			conn.Close()
			return
		}
		if string(bts) != "ok" {
			conn.Close()
			return
		}

		innerHandle(inch, outch, conn)
	}()

	clientHandle(outch, inch, network, out, m)
}

// read to in, from out to write
func innerHandle(in chan *Message, out chan *Message, conn net.Conn) {
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	go func() {
		for {
			var message Message
			err := dec.Decode(&message)
			if err != nil {
				conn.Close()
				return
			}
			// fmt.Printf("in: %s\n", message)
			in <- &message

		}
	}()
	go func() {
		for {
			message := <-out
			// fmt.Printf("out: %s\n", message)
			err := enc.Encode(message)
			if err != nil {
				conn.Close()
				return
			}
		}
	}()
}

// read to in
func serverHandle(in chan *Message, conn net.Conn, m *sync.Map) {
	from := conn.RemoteAddr().String()
	m.Store(from, conn)
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				conn.Close()
				m.Delete(from)
				return
			}
			in <- &Message{From: from, Content: buffer[:n]}
		}
	}()
}

// read to in, from out to write
func clientHandle(in chan *Message, out chan *Message, network string, address string, m *sync.Map) {
	go func() {
		for {
			message := <-out
			if conn, ok := m.Load(message.From); ok {
				_, err := conn.(net.Conn).Write(message.Content)
				if err == nil {
					continue
				}
			}
			conn, err := net.Dial(network, address)
			if err != nil {
				continue
			}
			_, err = conn.Write(message.Content)
			if err != nil {
				continue
			}
			go func() {
				for {
					buffer := make([]byte, 1024)
					n, err := conn.Read(buffer)
					if err != nil {
						conn.Close()
						m.Delete(message.From)
						return
					}
					in <- &Message{From: message.From, Content: buffer[:n]}
				}
			}()
			m.Store(message.From, conn)
		}
	}()
}

func authBytes(auth string) []byte {
	bts := make([]byte, 4)
	binary.BigEndian.PutUint32(bts, uint32(len(auth)))
	bts = append(bts, []byte(auth)...)
	return bts
}
