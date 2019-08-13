package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

func main() {

	converse := make(chan []byte)
	var wg sync.WaitGroup
	var gg sync.WaitGroup
	go listen(converse, &wg, &gg)

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

	hi := []byte("6Hello!")
	i := 0

	go func(y int, g *sync.WaitGroup, bg *sync.WaitGroup) {
		for y < 10 {
			g.Add(1)
			_, err := conn.Write(hi)

			if err != nil {
				fmt.Println(err.Error())
			}
			y++

			g.Wait()
			bg.Done()

		}
	}(i, &wg, &gg)
Loop:
	for {
		select {
		case greeting := <-converse:
			fmt.Println(string(greeting) + "\n")
			continue Loop
		}
	}
}

func listen(ch chan []byte, wg *sync.WaitGroup, gg *sync.WaitGroup) {
	listener, err := net.Listen("tcp", "localhost:4950")

	defer listener.Close()

	if err != nil {
		fmt.Println(err.Error())
	}

	conn, err := listener.Accept()

	if err != nil {
		fmt.Println(err.Error())
	}

	for {
		gg.Add(1)
		rdr := bufio.NewReader(conn)
		// b := make([]byte, 10)

		n, err := rdr.ReadByte()

		if err != nil {
			fmt.Println(err)
		}

		b := make([]byte, int(n))

		b, err = rdr.ReadBytes('!')

		if err != nil {
			fmt.Println(err)
		}

		if len(b) > 0 {
			ch <- b
		}
		wg.Done()
		gg.Wait()
	}
}
