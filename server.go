/**
 * Serves the requests for data stored in Neo4j.
 *
 */
package main

import (
	"encoding/json"
	"fmt"
	"github.com/jmcvetta/neoism"
	"net/http"
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
		AND NOT u=fof
		RETURN u.name AS me, fof.name AS names, count(fof.name) AS c
		ORDER BY c DESC
		LIMIT {limit}
	`
	MUTUAL_FRIENDS = `
		MATCH (a:Person)-[:KNOWS]->(mutal_firend:Person)<-[:KNOWS]-(b:Person)
		WHERE a.name={aName} AND b.name={bName}
		RETURN mutal_firend.name AS names, count(*) AS c
	`
	MOVIE_CAST = `
		MATCH (a:Person)-[:ACTED_IN]->(movie)
		WHERE movie.title={title}
		RETURN a.name AS name, a.born AS born
	`
)

// for storing :Person data from db
type Person struct {
	Name string `json:"names"`
	Born int    `json:"born"`
}

// Cypher query request
// Note: `Result` field only accept pointer(address) of
// slice of struct type. Ex. &[]Person{}
type QueryRequest struct {
	Name   string
	Query  *neoism.CypherQuery
	Result interface{}
}

// Cypher query result
// Note: For general use, don't assign any of the fields mannualy.
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

// Helper function for runConcurrentQuery function.
// Run a single query and send the result to the channel `ch`.
func runSingleQuery(db *neoism.Database, query QueryRequest, ch chan QueryResult) {
	result := QueryResult{}
	result.Result = query.Result
	query.Query.Result = result.Result
	err := db.Cypher(query.Query)
	if err != nil {
		fmt.Printf("Err in runSingleQuery: %s\n", err.Error())
	}
	result.Name = query.Name
	result.Columns = query.Query.Columns()
	ch <- result
}

// Run the queries concurrently and accept a handler function to do some
// post-processing of the result retrieved from database
func runConcurrentQuery(db *neoism.Database, queries []QueryRequest, handler func([]QueryResult) (interface{}, error)) (interface{}, error) {
	queryLen := len(queries)
	results := make([]QueryResult, queryLen)
	ch := make(chan QueryResult, queryLen)
	for _, query := range queries {
		// Note: we pass an copy of query to runSingleQuery,
		// not the address of it, otherwise the following goroutines will
		// get uncertain query since they all have the same address.
		// Therefore all the result information should be retrieved
		// from QueryResult rather than QueryRequest
		go runSingleQuery(db, query, ch)
	}
	for j, _ := range results {
		r := <-ch
		results[j] = r
	}
	return handler(results)
}

// A simple handler function to merge results together.
func mergeHandler(results []QueryResult) (interface{}, error) {
	finalResult := make(map[string]interface{}, len(results))
	for _, result := range results {
		finalResult[result.Name] = result.Result
	}
	return finalResult, nil
}

// Http handler for `/query` route.
func httpQueryHandler(w http.ResponseWriter, req *http.Request) {
	db, err := neoism.Connect(dbUrl)
	if err != nil {
		fmt.Println(err.Error())
	}

	queryReqRecommendFriend := QueryRequest{
		Name:   "recommend-friend-with-limit",
		Result: &[]Person{},
		Query: makeCypherQuery(
			RECOMMENDED_FIRENDS_WITH_LIMIT,
			neoism.Props{"name": "Tou Hanks", "limit": 5},
			nil,
		),
	}

	queryReqMutualFriend := QueryRequest{
		Name:   "mutual-friend",
		Result: &[]Person{},
		Query: makeCypherQuery(
			MUTUAL_FRIENDS,
			neoism.Props{"aName": "Tom Cruise", "bName": "Tom Hanks"},
			nil,
		),
	}

	results, _ := runConcurrentQuery(db, []QueryRequest{queryReqRecommendFriend, queryReqMutualFriend}, mergeHandler)

	json.NewEncoder(w).Encode(results)
}

func main() {

	http.HandleFunc("/query", httpQueryHandler)
	http.ListenAndServe(":8888", nil)
}
