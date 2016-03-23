// utils
package app

import (
	"errors"
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

// Retrieve data field of node returned from neo4j
// Ex. For cypher queries like `MATCH (u:USER) RETURN u`, if we want to get
// the properties of `u` node like {id: "111111", name: "Bob"}, we need to
// first unmarshal the `u` field to a map[string]interface{}, and what we get
// is something like {"all_relationships": ......,"data":"{"id":...}",...},
// since we just want the `data` field, we have to retrieve it and discard
// others.
func getAuthorData(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	res, ok := v.(*[]Post)
	if ok == false {
		return nil, errors.New("interface of *[]Post expected")
	}
	for i, _ := range *res {
		// get the `data` field
		d := (*res)[i].Author["data"]
		(*res)[i].Author = d.(map[string]interface{})
	}
	return *res, nil
}
