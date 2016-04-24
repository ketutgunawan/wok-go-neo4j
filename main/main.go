/**
 * Serves the requests for data stored in Neo4j.
 *
 */
package main

import (
	"log"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
	"github.com/leozhucong/wok-go-neo4j/app"
)

var (
	// regexp
	urlQuery = regexp.MustCompile("^/(query)/([a-zA-Z0-9]+)$")
)

var (
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

// Http handler for `/query` route.
//func httpQueryHandler(w http.ResponseWriter, r *http.Request) {
//	fmt.Println("Request from " + r.URL.Path)

//	queryType, err := getUrlQueryType(w, r)
//	if err != nil {
//		fmt.Println(err.Error())
//	}

//	fmt.Println("Query type: " + queryType)

//	queryReqRecommendFriend := QueryRequest{
//		Name:   "recommend-friend-with-limit",
//		Result: &[]Person{},
//		Query: makeCypherQuery(
//			RECOMMENDED_FIRENDS_WITH_LIMIT,
//			neoism.Props{"name": "Tom Hanks", "limit": 5},
//			nil,
//		),
//	}

//	queryReqMutualFriend := QueryRequest{
//		Name:   "mutual-friend",
//		Result: &[]Person{},
//		Query: makeCypherQuery(
//			MUTUAL_FRIENDS,
//			neoism.Props{"aName": "Tom Cruise", "bName": "Tom Hanks"},
//			nil,
//		),
//	}

//	results, _ := runConcurrentQuery(db, []QueryRequest{queryReqRecommendFriend, queryReqMutualFriend}, mergeHandler)

//	json.NewEncoder(w).Encode(results)
//}

//func getUrlQueryType(w http.ResponseWriter, r *http.Request) (string, error) {
//	q := urlQuery.FindStringSubmatch(r.URL.Path)
//	if q == nil {
//		http.NotFound(w, r)
//		return "", errors.New("Invalid query type")
//	}
//	return q[2], nil
//}

type appHandlerFunc func(*app.AppContext, http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

func makeHandler(context *app.AppContext, handle appHandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Println("Request: " + r.Method + " " + r.URL.Path)
		if status, err := handle(context, w, r, ps); err != nil {
			switch status {
			case http.StatusNotFound:
				http.NotFound(w, r)
				//	case http.StatusBadRequest:
				//		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case http.StatusInternalServerError:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				log.Println(err.Error())
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func main() {

	db, err := app.OpenDB("http://neo4j:wiirok@localhost:7474/db/data")
	if err != nil {
		panic("Open database failed")
	}
	context := &app.AppContext{db}

	router := httprouter.New()

	// user handlers
	router.GET("/users", makeHandler(context, app.UserGetAll))
	router.GET("/users/:id", makeHandler(context, app.UserGetOne))
	router.POST("/users/query", makeHandler(context, app.UserQuery))
	router.POST("/users", makeHandler(context, app.UserCreate))
	router.PUT("/users/:id", makeHandler(context, app.UserUpdate))
	router.DELETE("/users/:id", makeHandler(context, app.UserDestroy))

	// post handlers
	router.GET("/posts", makeHandler(context, app.PostGetAll))
	router.GET("/posts/:id", makeHandler(context, app.PostGetOne))
	router.POST("/posts/query", makeHandler(context, app.PostQuery))
	router.POST("/posts", makeHandler(context, app.PostCreate))
	router.PUT("/posts/:id", makeHandler(context, app.PostUpdate))
	router.DELETE("/posts/:id", makeHandler(context, app.PostDestroy))

	log.Fatal(http.ListenAndServe(":8888", router))
}
