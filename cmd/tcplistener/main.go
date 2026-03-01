package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os" // Added for syncing
	"strings"
)

// Use the getLinesChannel we fixed earlier with the 1024 buffer
func getLinesChannel(f io.ReadCloser) <-chan string {
	lines := make(chan string)
	go func() {
		defer f.Close()
		defer close(lines)
		currentLine := ""
		for {
			data := make([]byte, 1024)
			n, err := f.Read(data)
			if n > 0 {
				parts := strings.Split(string(data[:n]), "\n")
				for i, part := range parts {
					if i == len(parts)-1 {
						currentLine += part
					} else {
						lines <- strings.TrimRight(currentLine+part, "\r")
						currentLine = ""
					}
				}
			}
			if err != nil {
				break
			}
		}
		if currentLine != "" {
			lines <- strings.TrimRight(currentLine, "\r")
		}
	}()
	return lines
}

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// Use log.Println for the "listening" message so it doesn't
	// mess up the test file
	log.Println("Server is listening on :42069...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		fmt.Println("connection accepted")
		os.Stdout.Sync()

		for line := range getLinesChannel(conn) {
			fmt.Println(line)
			os.Stdout.Sync() // This is the fix!
		}

		fmt.Println("connection closed")
		os.Stdout.Sync()
	}
}
