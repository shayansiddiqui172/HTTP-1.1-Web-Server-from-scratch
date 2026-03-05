package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"httpfromtcp/internal/headers"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"httpfromtcp/internal/server"
)

const port = 10702

func proxyHandler(w *response.Writer, req *request.Request) {
	path := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")
	url := "https://httpbin.org" + path

	resp, err := http.Get(url)
	if err != nil {
		body := []byte(fmt.Sprintf("error proxying request: %v", err))
		w.WriteStatusLine(response.StatusInternalServerError)
		h := response.GetDefaultHeaders(len(body))
		w.WriteHeaders(h)
		w.WriteBody(body)
		return
	}
	defer resp.Body.Close()

	w.WriteStatusLine(response.StatusOK)
	h := response.GetDefaultHeaders(0)
	h.Set("content-type", "text/plain")
	h.Set("transfer-encoding", "chunked")
	h.Set("trailer", "X-Content-SHA256, X-Content-Length")
	delete(h, "content-length")
	w.WriteHeaders(h)

	var fullBody []byte
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			fmt.Println("read", n, "bytes")
			fullBody = append(fullBody, buf[:n]...)
			w.WriteChunkedBody(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	w.WriteChunkedBodyDone()

	hash := sha256.Sum256(fullBody)
	trailers := headers.NewHeaders()
	trailers.Set("X-Content-SHA256", fmt.Sprintf("%x", hash))
	trailers.Set("X-Content-Length", fmt.Sprintf("%d", len(fullBody)))
	w.WriteTrailers(trailers)
}

func handler(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin") {
		proxyHandler(w, req)
		return
	}

	if req.RequestLine.RequestTarget == "/video" {
		videoData, err := os.ReadFile("assets/vim.mp4")
		if err != nil {
			body := []byte("video not found\n")
			w.WriteStatusLine(response.StatusInternalServerError)
			h := response.GetDefaultHeaders(len(body))
			w.WriteHeaders(h)
			w.WriteBody(body)
			return
		}
		w.WriteStatusLine(response.StatusOK)
		h := response.GetDefaultHeaders(len(videoData))
		h.Set("content-type", "video/mp4")
		w.WriteHeaders(h)
		w.WriteBody(videoData)
		return
	}

	if req.RequestLine.RequestTarget == "/yourproblem" {
		body := []byte(`<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`)
		w.WriteStatusLine(response.StatusBadRequest)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "text/html")
		w.WriteHeaders(h)
		w.WriteBody(body)
		return
	}

	if req.RequestLine.RequestTarget == "/myproblem" {
		body := []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`)
		w.WriteStatusLine(response.StatusInternalServerError)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "text/html")
		w.WriteHeaders(h)
		w.WriteBody(body)
		return
	}

	body := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)
	w.WriteStatusLine(response.StatusOK)
	h := response.GetDefaultHeaders(len(body))
	h.Set("content-type", "text/html")
	w.WriteHeaders(h)
	w.WriteBody(body)
}

func main() {
	s, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
