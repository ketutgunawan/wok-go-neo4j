/**
 * Serves the requests for data stored in Neo4j.
 *
 */
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/jmcvetta/neoism"
	"github.com/julienschmidt/httprouter"
)

var (
	// database envs
	dbUsername = "neo4j"
	dbPassword = "wiirok"
	dbUrl      = "http://" + dbUsername + ":" + dbPassword + "@localhost:7474/db/data"
	db         *neoism.Database
)

var (
	// regexp
	urlQuery = regexp.MustCompile("^/(query)/([a-zA-Z0-9]+)$")

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
	ALL_USER = `
		MATCH (u:USER)
		RETURN u.name as name, u.email as email, u.role as role,
				u.hashedPassword as hashedPassword, u.salt as salt,
				u._id as _id
	`
	FIND_USER_BY_EMAIL = `
		MATCH (u:USER)
		WHERE u.email={email}
		RETURN u.name as name, u.email as email, u.role as role,
				u.hashedPassword as hashedPassword, u.salt as salt,
				u.id as id
	`
	FIND_USER_BY_ID = `
		MATCH (u:USER)
		WHERE u.id={id}
		RETURN u.name as name, u.email as email, u.role as role,
				u.hashedPassword as hashedPassword, u.salt as salt,
				u.id as id
	
	`
)

// for req body
type UserBody struct {
	Id    string
	Name  string
	Email string
	Role  string
}

// for storing :Person data from db
type Person struct {
	Name string `json:"names"`
	Born int    `json:"born"`
}

// for storing User
type User struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	HashedPassword string `json:"hashedPassword"`
	Salt           string `json:"salt"`
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

// Run a single query and save the result in result
func runSingleQuerySync(db *neoism.Database, query QueryRequest, result *QueryResult) error {
	result.Result = query.Result
	query.Query.Result = result.Result
	err := db.Cypher(query.Query)
	if err != nil {
		return err
	}
	result.Name = query.Name
	result.Columns = query.Query.Columns()
	return nil
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
func httpQueryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request from " + r.URL.Path)

	queryType, err := getUrlQueryType(w, r)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("Query type: " + queryType)

	queryReqRecommendFriend := QueryRequest{
		Name:   "recommend-friend-with-limit",
		Result: &[]Person{},
		Query: makeCypherQuery(
			RECOMMENDED_FIRENDS_WITH_LIMIT,
			neoism.Props{"name": "Tom Hanks", "limit": 5},
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

func httpUserHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request from " + r.URL.Path)

	urlQueryMap := r.URL.Query()
	id, email := "", ""
	if val, ok := urlQueryMap["id"]; ok {
		id = val[0]
	} else {
		email = urlQueryMap["email"][0]
	}
	fmt.Printf("email=%s\n", email)
	fmt.Printf("id=%s\n", id)

	queryReqFindUserByEmail := QueryRequest{
		Name:   "find-user-by-email",
		Result: &[]User{},
		Query: makeCypherQuery(
			FIND_USER_BY_EMAIL,
			neoism.Props{"email": email},
			nil,
		),
	}

	queryReqFindUserById := QueryRequest{
		Name:   "find-user-by-id",
		Result: &[]User{},
		Query: makeCypherQuery(
			FIND_USER_BY_ID,
			neoism.Props{"id": id},
			nil,
		),
	}

	qq := queryReqFindUserById
	if id == "" {
		qq = queryReqFindUserByEmail
	}
	result1 := QueryResult{}
	//runSingleQuerySync(db, queryReqFindUserByEmail, &result1)
	runSingleQuerySync(db, qq, &result1)
	json.NewEncoder(w).Encode(result1.Result)

}

func getUrlQueryType(w http.ResponseWriter, r *http.Request) (string, error) {
	q := urlQuery.FindStringSubmatch(r.URL.Path)
	if q == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid query type")
	}
	return q[2], nil
}

func UserGetOne(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println("Request from " + r.Host + r.URL.Path)

	queryReqFindUserById := QueryRequest{
		Name:   "find-user-by-id",
		Result: &[]User{},
		Query: makeCypherQuery(
			FIND_USER_BY_ID,
			neoism.Props{"id": ps.ByName("id")},
			nil,
		),
	}

	result := QueryResult{}
	runSingleQuerySync(db, queryReqFindUserById, &result)
	json.NewEncoder(w).Encode(result.Result)
}

func UserQuery(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println("Request from " + r.Host + r.URL.Path)

	// TODO: Maybe 200 is too much?
	b := make([]byte, 100)
	n, err := r.Body.Read(b)
	if err != nil && err != io.EOF {
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	jsonFormatter := regexp.MustCompile(`"([a-zA-Z0-9]+)":`)
	params := jsonFormatter.ReplaceAllString(string(b[:n]), "${1}:")

	finUser := "MATCH (u:USER " + params + ")" +
		"RETURN u.name as name, u.email as email, u.role as role," +
		"u.hashedPassword as hashedPassword, u.salt as salt," +
		"u.id as id"

	queryReqFindUser := QueryRequest{
		Name:   "find-user",
		Result: &[]User{},
		Query: makeCypherQuery(
			finUser,
			nil,
			nil,
		),
	}

	result := QueryResult{}
	err = runSingleQuerySync(db, queryReqFindUser, &result)
	if err != nil {
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	json.NewEncoder(w).Encode(result.Result)
}

func init() {
	var err error
	db, err = neoism.Connect(dbUrl)
	if err != nil {
		panic(err)
	}
}

func main() {
	router := httprouter.New()
	router.GET("/users/:id", UserGetOne)
	router.POST("/users/query", UserQuery)
	//	http.HandleFunc("/query/", httpQueryHandler)
	//	http.HandleFunc("/users/", httpUserHandler)
	log.Fatal(http.ListenAndServe(":8888", router))
}
