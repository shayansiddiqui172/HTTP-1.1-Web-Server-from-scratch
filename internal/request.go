package request

import (
	"fmt"
	"io"
	"strings"
)

type parserState int

const (
	stateInitialized parserState = iota
	stateDone
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	state       parserState
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{state: stateInitialized}
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
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	if r.state == stateDone {
		return 0, fmt.Errorf("error: trying to read data in done state")
	}

	consumed, requestLine, err := parseRequestLine(data)
	if err != nil {
		return 0, err
	}
	if consumed == 0 {
		// need more data
		return 0, nil
	}

	r.RequestLine = *requestLine
	r.state = stateDone
	return consumed, nil
}

func parseRequestLine(data []byte) (int, *RequestLine, error) {
	str := string(data)
	idx := strings.Index(str, "\r\n")
	if idx == -1 {
		// no complete line yet
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

	return idx + 2, &RequestLine{ // +2 for \r\n
		Method:        method,
		RequestTarget: parts[1],
		HttpVersion:   versionParts[1],
	}, nil
}
