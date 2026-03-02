package request

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"httpfromtcp/internal/headers"
)

type parserState int

const (
	stateInitialized parserState = iota
	stateParsingHeaders
	stateParsingBody
	stateDone
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	state       parserState
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{
		state:   stateInitialized,
		Headers: headers.NewHeaders(),
	}
	buf := make([]byte, 8)
	unparsed := []byte{}

	for req.state != stateDone {
		n, err := reader.Read(buf)
		if n > 0 {
			unparsed = append(unparsed, buf[:n]...)
			consumed, parseErr := req.parse(unparsed)
			if parseErr != nil {
				return nil, parseErr
			}
			unparsed = unparsed[consumed:]
		}
		if err == io.EOF {
			if req.state == stateParsingBody {
				contentLengthStr := req.Headers.Get("content-length")
				if contentLengthStr != "" {
					contentLength, _ := strconv.Atoi(contentLengthStr)
					if len(req.Body) < contentLength {
						return nil, fmt.Errorf("body shorter than content-length: got %d, expected %d", len(req.Body), contentLength)
					}
				}
			}
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for r.state != stateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			break
		}
		totalBytesParsed += n
	}
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		consumed, requestLine, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if consumed == 0 {
			return 0, nil
		}
		r.RequestLine = *requestLine
		r.state = stateParsingHeaders
		return consumed, nil

	case stateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.state = stateParsingBody
		}
		return n, nil

	case stateParsingBody:
		contentLengthStr := r.Headers.Get("content-length")
		if contentLengthStr == "" {
			r.state = stateDone
			return 0, nil
		}
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return 0, fmt.Errorf("invalid content-length: %s", contentLengthStr)
		}
		r.Body = append(r.Body, data...)
		if len(r.Body) > contentLength {
			return 0, fmt.Errorf("body longer than content-length")
		}
		if len(r.Body) == contentLength {
			r.state = stateDone
		}
		return len(data), nil

	case stateDone:
		return 0, fmt.Errorf("error: trying to read data in done state")
	}

	return 0, fmt.Errorf("unknown state")
}

func parseRequestLine(data []byte) (int, *RequestLine, error) {
	str := string(data)
	idx := strings.Index(str, "\r\n")
	if idx == -1 {
		return 0, nil, nil
	}

	line := str[:idx]
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return 0, nil, fmt.Errorf("invalid request line: expected 3 parts, got %d", len(parts))
	}

	method := parts[0]
	for _, c := range method {
		if c < 'A' || c > 'Z' {
			return 0, nil, fmt.Errorf("invalid method: %s", method)
		}
	}

	versionParts := strings.Split(parts[2], "/")
	if len(versionParts) != 2 || versionParts[0] != "HTTP" || versionParts[1] != "1.1" {
		return 0, nil, fmt.Errorf("invalid HTTP version: %s", parts[2])
	}

	return idx + 2, &RequestLine{
		Method:        method,
		RequestTarget: parts[1],
		HttpVersion:   versionParts[1],
	}, nil
}
