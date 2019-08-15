package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

/*
	Let's make a server that can handle multiple connections
	to the same port, read length-prefixed messages and forward
	them to be processed (printing).
*/

func main() {
	processor := make(chan []byte)
	defer close(processor)
	port := ":5032"

	laddr, err := net.ResolveTCPAddr("tcp", port)

	if err != nil {
		panic(err)
	}

	go listen(processor, laddr)
	go imitateClientSwarm(laddr)

	for {
		select {
		case message := <-processor:
			fmt.Println(string(message))
		}
	}

}

func listen(processor chan []byte, laddr *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", laddr)

	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("Unable to establish connection: %s \n", err.Error())
			continue
		}
		go handleConnection(conn, processor)
	}
}

func handleConnection(conn net.Conn, processor chan []byte) {
	rdr := bufio.NewReader(conn)

	for {
		msgSize, err := rdr.ReadByte()

		if err != nil {
			fmt.Printf("Unable to read msgSize: %s \n", err.Error())
		}

		msgSizeInt, err := strconv.Atoi(string(msgSize))

		if err != nil {
			fmt.Printf("Couldn't convert given string to int: %s", err.Error())
		}

		msg := make([]byte, msgSizeInt)

		n, err := rdr.Read(msg[:msgSizeInt])

		if err != nil {
			fmt.Printf("Unable to read msg: %s \n", err.Error())
			continue
		}

		if n > 0 {
			fmt.Printf("%v bytes read from %s \n", n, conn.RemoteAddr().String())

			processor <- msg
		}

	}
}

func imitateClientSwarm(raddr *net.TCPAddr) {
	ports := []string{":4326", ":4327", ":4328"}

	for _, port := range ports {
		laddr, err := net.ResolveTCPAddr("tcp", port)
		if err != nil {
			fmt.Printf("Could not resolve client address: %s \n", err.Error())
		}

		conn, err := net.DialTCP("tcp", laddr, raddr)

		if err != nil {
			fmt.Printf("Could not connect to provided server over tcp: %s \n", err.Error())
		}
		go writeShitToConnection(conn)

	}
}

func writeShitToConnection(conn *net.TCPConn) {

	msgs := [][]byte{[]byte("5hello"), []byte("7goodbye"), []byte("5nice!")}

	for _, msg := range msgs {
		_, err := conn.Write(msg)

		if err != nil {
			fmt.Println(err.Error())
		}
	}
}
