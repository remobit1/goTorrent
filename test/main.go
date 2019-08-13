package main

import (
	"fmt"
	"net"
)

func main() {

	converse := make(chan []byte)
	go listen(converse)

	raddr, err := net.ResolveTCPAddr("tcp", "localhost:4950")

	if err != nil {
		fmt.Println(err.Error())
	}

	laddr, err := net.ResolveTCPAddr("tcp", "localhost:5000")

	if err != nil {
		fmt.Println(err.Error())
	}

	conn, err := net.DialTCP("tcp", laddr, raddr)

	if err != nil {
		fmt.Println(err.Error())
	}

	defer conn.Close()

	hi := []byte("Hello!")

	_, err = conn.Write(hi)

	if err != nil {
		fmt.Println(err.Error())
	}

	select {
	case greeting := <-converse:
		fmt.Println(greeting)
	}
}

func listen(ch chan []byte) {
	listener, err := net.Listen("tcp", "localhost:50000")

	defer listener.Close()

	if err != nil {
		fmt.Println(err.Error())
	}

	conn, err := listener.Accept()

	if err != nil {
		fmt.Println(err.Error())
	}

	var b []byte

	for {
		n, err := conn.Read(b)

		if err != nil {
			fmt.Println(err)
		}

		if n > 0 {
			ch <- b
		}
	}
}
