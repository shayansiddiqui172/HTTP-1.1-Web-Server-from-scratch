package headers

import (
	"fmt"
	"strings"
)

type Headers map[string]string

func (h Headers) Set(key, value string) {
	h[strings.ToLower(key)] = value
}

func NewHeaders() Headers {
	return Headers{}
}

func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

func isValidTokenChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	str := string(data)

	idx := strings.Index(str, "\r\n")
	if idx == -1 {
		return 0, false, nil
	}

	if idx == 0 {
		return 2, true, nil
	}

	line := str[:idx]
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return 0, false, fmt.Errorf("invalid header: missing colon")
	}

	key := line[:colonIdx]
	value := line[colonIdx+1:]

	if strings.TrimSpace(key) != key || strings.Contains(key, " ") {
		return 0, false, fmt.Errorf("invalid header: bad key format %q", key)
	}

	for i := 0; i < len(key); i++ {
		if !isValidTokenChar(key[i]) {
			return 0, false, fmt.Errorf("invalid header: invalid character %q in key", key[i])
		}
	}

	key = strings.ToLower(key)
	value = strings.TrimSpace(value)
	if existing, ok := h[key]; ok {
		h[key] = existing + ", " + value
	} else {
		h[key] = value
	}

	return idx + 2, false, nil
}
