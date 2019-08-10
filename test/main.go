package main

import (
	"fmt"
	"net"
)

func main() {

	laddr, err := net.ResolveTCPAddr("tcp", "96.227.68.183:57745")

	if err != nil {
		fmt.Println(err.Error())
	}

	conn, err := net.DialTCP("tcp", nil, laddr)

	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()

}
