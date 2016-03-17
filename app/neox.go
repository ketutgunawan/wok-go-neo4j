// A Neo4j client that wraps around `neoism`
package app

import (
	"github.com/jmcvetta/neoism"
)

type DB struct {
	*neoism.Database
}

type Query struct {
	*neoism.CypherQuery
}

// synonym for neoism.Props
type Props map[string]interface{}

// Cypher query request
// Note: `Result` field only accept pointer(address) of
// slice of struct type. Ex. &[]Person{}
type QueryRequest struct {
	Name   string
	Query  *Query
	Result interface{}
}

// Cypher query result
// Note: For general use, don't assign any of the fields mannualy.
type QueryResult struct {
	Name    string
	Columns []string
	Result  interface{}
}

// Wrapper for neoism's connect but return `Client` struct
func OpenDB(uri string) (*DB, error) {
	db, err := neoism.Connect(uri)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

// Run a single query and save the result in result
func (db *DB) RunSingleQuery(query QueryRequest, result *QueryResult) error {
	result.Result = query.Result
	query.Query.Result = result.Result
	err := db.Cypher(query.Query.CypherQuery)
	if err != nil {
		return err
	}
	result.Name = query.Name
	result.Columns = query.Query.Columns()
	return nil
}

// Run the queries concurrently and accept a handler function to do some
// post-processing of the result retrieved from database
func (db *DB) RunConcurrentQueries(queries []QueryRequest, handler func([]QueryResult) (interface{}, error)) (interface{}, error) {
	queryLen := len(queries)
	// TODO use lock for maps to garantee sync concurrency
	results := make([]QueryResult, queryLen)
	ch := make(chan QueryResult, queryLen)
	for _, query := range queries {
		// Note: we pass an copy of query to runSingleQuery,
		// not the address of it, otherwise the following goroutines will
		// get uncertain query since they all have the same address.
		// Therefore all the result information should be retrieved
		// from QueryResult rather than QueryRequest
		go func() {
			result := QueryResult{}
			result.Result = query.Result
			query.Query.Result = result.Result
			err := db.Cypher(query.Query.CypherQuery)
			if err != nil {
				panic("Err in runSingleQuery: " + err.Error())
			}
			result.Name = query.Name
			result.Columns = query.Query.Columns()
			ch <- result
		}()
	}
	for j, _ := range results {
		r := <-ch
		results[j] = r
	}
	return handler(results)
}

// make a cypher query
func MakeQuery(statement string, params map[string]interface{}, result *interface{}) *Query {
	return &Query{
		&neoism.CypherQuery{
			Statement:  statement,
			Parameters: neoism.Props(params),
			Result:     result,
		},
	}
}

// A simple handler function to merge results together.
func mergeHandler(results []QueryResult) (interface{}, error) {
	finalResult := make(map[string]interface{}, len(results))
	for _, result := range results {
		finalResult[result.Name] = result.Result
	}
	return finalResult, nil
}

// Helper function for runConcurrentQuery function.
// Run a single query and send the result to the channel `ch`.
//func runSingleQuery(db *neoism.Database, query QueryRequest, ch chan QueryResult) {
//	result := QueryResult{}
//	result.Result = query.Result
//	query.Query.Result = result.Result
//	err := db.Cypher(query.Query)
//	if err != nil {
//		panic("Err in runSingleQuery: " + err.Error())
//	}
//	result.Name = query.Name
//	result.Columns = query.Query.Columns()
//	ch <- result
//}
