package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"strings"
)

type Message struct {
	conn     net.Conn
	username string
	payload  string
}

var clients = make(map[net.Conn]bool)
var msgqueue = make(chan Message)

func main() {
	go handleMessages()

	log.Print("Loading certificate...")

	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Print("Creating TLS listener on port 8443...")
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", ":8443", config)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer func(ln net.Listener) {
		err := ln.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(ln)

	log.Print("Listening for connections...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	clients[conn] = true

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			delete(clients, conn)
			log.Print(err)
		}
	}(conn)

	var mobj Message
	mobj.conn = conn

	r := bufio.NewReader(conn)
	n, err := conn.Write([]byte("What username do you want to use?\n"))
	resp, err := r.ReadString('\n')
	if err != nil {
		delete(clients, conn)
		log.Print(n, err)
		return
	}

	mobj.username = strings.TrimSuffix(resp, "\n")

	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			delete(clients, conn)
			log.Print(err)
			return
		}

		println(mobj.username + ": " + msg)
		mobj.payload = msg

		n, err := conn.Write([]byte("Received!\n"))
		if err != nil {
			delete(clients, conn)
			log.Print(n, err)
			return
		}
		msgqueue <- mobj
	}
}

func handleMessages() {
	for {
		msg := <-msgqueue

		for client := range clients {
			if client != msg.conn {
				n, err := client.Write([]byte(msg.username + ": " + msg.payload))
				if err != nil {
					delete(clients, client)
					log.Print(n, err)
					return
				}
			}
		}
	}
}