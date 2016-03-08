/**
 * Serves the requests for data stored in Neo4j.
 *
 */
package main

import (
	"encoding/json"
	"fmt"
	"github.com/jmcvetta/neoism"
	zmq "github.com/pebbe/zmq4"
)

var (
	dbUsername = "neo4j"
	dbPassword = "wiirok"
	dbUrl      = "http://" + dbUsername + ":" + dbPassword + "@localhost:7474/db/data"
)

// an empty interface that fits any types it's given
type AnyType interface{}

// Query data struct for decoding json format request data
type ReqZMQ struct {
	Id    string
	Query string
}

// Response data struct for encoding json
type ResZMQ struct {
	Id   string
	Data []Person
}

// for storing :Person data from db
type Person struct {
	Name string `json:"name"`
	Born int    `json:"born"`
}

type QueryRequest struct {
	Name   string
	Query  *neoism.CypherQuery
	Result AnyType
}

type QueryResult struct {
	Name    string
	Columns []string
	Result  AnyType
}

func movieCastCQ(title string, persons *[]Person) *neoism.CypherQuery {
	return &neoism.CypherQuery{
		Statement: `
			MATCH (a:Person)-[:ACTED_IN]->(movie)
			WHERE movie.title={title}
			RETURN a.name AS name, a.born AS born
		`,
		Parameters: neoism.Props{"title": title},
		Result:     persons,
	}
}

func doSingleQuery(db *neoism.Database, query *QueryRequest, result *QueryResult) {
	query.Query.Result = &(query.Result)
	err := db.Cypher(query.Query)
	if err != nil {
		fmt.Print(err.Error())
	}
	result.Name = query.Name
	result.Columns = query.Query.Columns()
	result.Result = query.Result
}

func runConcurrentCQs(db *neoism.Database, queries []QueryRequest, cb func([]QueryResult) (interface{}, error)) (interface{}, error) {
	// TODO: implement this function
	results := make([]QueryResult, len(queries))
	for i, query := range queries {
		go doSingleQuery(db, &query, &results[i])
	}
	return cb(results)
}

func main() {
	db, err := neoism.Connect(dbUrl)
	if err != nil {
		fmt.Println(err.Error())
	}
	persons := []Person{}
	db.Cypher(movieCastCQ("The Matrix", &persons))
	fmt.Println(persons)
	context, _ := zmq.NewContext()
	socket, _ := context.NewSocket(zmq.REP)
	defer context.Term()
	defer socket.Close()
	socket.Bind("tcp://*:8888")

	for {
		// Receive the request json format data from client
		query, _ := socket.Recv(0)

		// Decode the json data
		req := ReqZMQ{}
		err := json.Unmarshal([]byte(query), &req)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("Received", req.Id, req.Query)
		persons = []Person{}
		db.Cypher(movieCastCQ(req.Query, &persons))
		fmt.Println(persons)
		res := ResZMQ{Id: req.Id, Data: persons}
		resJson, err := json.Marshal(res)
		if err != nil {
			fmt.Println(err.Error())
			socket.Send(err.Error(), 0)
		}
		socket.SendBytes(resJson, 0)
	}
}
