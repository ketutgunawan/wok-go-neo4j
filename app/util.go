// utils
package app

import (
	"io"
	"net/http"
	"regexp"
)

// Format {"name": "Jon Snow"} to {name: "Jon Snow"}
func formatToNeoJson(jsonString string) string {
	return regexp.MustCompile(`"([a-zA-Z0-9]+)":`).ReplaceAllString(jsonString, "${1}:")
}

// read the request body and format it
func getReqBody(r *http.Request) (string, error) {
	b := make([]byte, 200)
	n, err := r.Body.Read(b)
	if err != nil && err != io.EOF {
		return "", err
	}
	return formatToNeoJson(string(b[:n])), nil
}
