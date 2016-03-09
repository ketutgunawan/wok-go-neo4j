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

	/*
	 * Cypher Queries
	 */
	RECOMMENDED_FIRENDS_WITH_LIMIT = `
		MATCH (u:Person)-[:KNOWS]->(f:Person)-[:KNOWS]->(fof:Person)
		WHERE u.name={name}
		AND NOT (u)-[:KNOWS]->(fof)
		RETURN u.name AS me fof.name AS names, count(names) AS c
		ORDER BY c DESC
		LIMIT {limit}
	`
	MUTUAL_FRIENDS = `
		MATCH (a:Person)-[:KNOWS]->(mutal_firend:Person)<-[:KNOWS]-(b:Person)
		WHERE a.name={aName} AND b.name={bName}
		RETURN mutal_firend.name as names, count(*) as c
	`
	MOVIE_CAST = `
		MATCH (a:Person)-[:ACTED_IN]->(movie)
		WHERE movie.title={title}
		RETURN a.name AS name, a.born AS born
	`
)

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

// Cypher query request
type QueryRequest struct {
	Name   string
	Result interface{}
	Query  *neoism.CypherQuery
}

// Cypher query result
type QueryResult struct {
	Name    string
	Columns []string
	Result  interface{}
}

// make a cypher query
func makeCypherQuery(statement string, params neoism.Props, result *interface{}) *neoism.CypherQuery {
	return &neoism.CypherQuery{
		Statement:  statement,
		Parameters: params,
		Result:     result,
	}
}

func runSingleQuery(db *neoism.Database, query *QueryRequest, result *QueryResult) {
	result.Result = query.Result
	query.Query.Result = result.Result
	err := db.Cypher(query.Query)
	if err != nil {
		fmt.Print(err.Error())
	}
	result.Name = query.Name
	result.Columns = query.Query.Columns()
}

func runConcurrentQuery(db *neoism.Database, queries []QueryRequest, handler func([]QueryResult) (interface{}, error)) (interface{}, error) {
	results := make([]QueryResult, len(queries))
	for i, query := range queries {
		go runSingleQuery(db, &query, &results[i])
	}
	return handler(results)
}

func mergeHandler(results []QueryResult) (interface{}, error) {
	finalResult := make(map[string]interface{}, len(results))
	for _, result := range results {
		finalResult[result.Name] = result.Result
	}
	return finalResult, nil
}

func main() {
	db, err := neoism.Connect(dbUrl)
	if err != nil {
		fmt.Println(err.Error())
	}

	queryReqRecommendFriend := QueryRequest{
		Name:   "recommend-friend-with-limit",
		Result: []Person{},
		Query: makeCypherQuery(
			RECOMMENDED_FIRENDS_WITH_LIMIT,
			neoism.Props{"name": "Tom Hanks", "limit": 3},
			nil,
		),
	}

	queryReqMutualFriend := QueryRequest{
		Name:   "mutual-friend",
		Result: []Person{},
		Query: makeCypherQuery(
			MUTUAL_FRIENDS,
			neoism.Props{"aName": "Liv Tyler", "bName": "Tom Hanks"},
			nil,
		),
	}

	//results, _ := runConcurrentQuery(db, []QueryRequest{queryReqRecommendFriend, queryReqMutualFriend}, mergeHandler)

	result1, result2 := QueryResult{}, QueryResult{}
	runSingleQuery(db, &queryReqRecommendFriend, &result1)
	runSingleQuery(db, &queryReqMutualFriend, &result2)
	fmt.Println(result1, result2)

	context, _ := zmq.NewContext()
	socket, _ := context.NewSocket(zmq.REP)
	defer context.Term()
	defer socket.Close()
	socket.Bind("tcp://*:8888")

	// Looping to listen to requests
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
		persons := []Person{}
		//db.Cypher(movieCastCQ(req.Query, &persons))
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
