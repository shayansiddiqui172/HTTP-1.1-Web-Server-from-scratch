package main

import (
	"fmt"
	"log"
	"net"

	"httpfromtcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Println("Server is listening on :42069...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		r, err := request.RequestFromReader(conn)
		if err != nil {
			log.Printf("error parsing request: %v", err)
			conn.Close()
			continue
		}

		fmt.Printf("Request line:\n- Method: %s\n- Target: %s\n- Version: %s\n",
			r.RequestLine.Method,
			r.RequestLine.RequestTarget,
			r.RequestLine.HttpVersion,
		)

		fmt.Println("Headers:")
		for key, value := range r.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Printf("Body:\n%s\n", string(r.Body))

		conn.Close()
	}
}
