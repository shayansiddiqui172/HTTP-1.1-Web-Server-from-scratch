package response

import (
	"fmt"
	"io"

	"httpfromtcp/internal/headers"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type writerState int

const (
	writerStateStatusLine writerState = iota
	writerStateHeaders
	writerStateBody
	writerStateDone
)

type Writer struct {
	w     io.Writer
	state writerState
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w, state: writerStateStatusLine}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != writerStateStatusLine {
		return fmt.Errorf("cannot write status line in current state")
	}
	err := WriteStatusLine(w.w, statusCode)
	if err != nil {
		return err
	}
	w.state = writerStateHeaders
	return nil
}

func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.state != writerStateHeaders {
		return fmt.Errorf("cannot write headers in current state")
	}
	err := WriteHeaders(w.w, h)
	if err != nil {
		return err
	}
	w.state = writerStateBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("cannot write body in current state")
	}
	return w.w.Write(p)
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("cannot write body in current state")
	}
	chunkSize := fmt.Sprintf("%x\r\n", len(p))
	_, err := w.w.Write([]byte(chunkSize))
	if err != nil {
		return 0, err
	}
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}
	_, err = w.w.Write([]byte("\r\n"))
	return n, err
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("cannot write body in current state")
	}
	n, err := w.w.Write([]byte("0\r\n"))
	return n, err
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	for key, value := range h {
		_, err := fmt.Fprintf(w.w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w.w, "\r\n")
	return err
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	var reasonPhrase string
	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	default:
		reasonPhrase = ""
	}
	_, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h["content-length"] = fmt.Sprintf("%d", contentLen)
	h["connection"] = "close"
	h["content-type"] = "text/plain"
	return h
}

func WriteHeaders(w io.Writer, h headers.Headers) error {
	for key, value := range h {
		_, err := fmt.Fprintf(w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\r\n")
	return err
}
