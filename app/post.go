// post handlers
package app

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Post struct {
	Author           map[string]interface{} `json:"author"`
	Id               string                 `json:"id"`
	Title            string                 `json:"title"`
	Type             string                 `json:"type"`
	Body             string                 `json:"body"`
	Status           string                 `json:"status"`
	PublishDate      int                    `json:"publishDate"`
	Upvotes          int                    `json:"upvotes"`
	Downvotes        int                    `json:"downvotes"`
	ViewCount        int                    `json:"viewCount"`
	CreateTime       int                    `json:"createTime"`
	LastModifiedTime int                    `json:"lastModifiedTime"`
}

// handler for GET /posts
func PostGetAll(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	postGetAll := `
		MATCH (author:USER)-[r:CREATED]->(p:POST)
		RETURN p.id as id, p.title as title, p.type as type,
		p.body as body, p.status as status, p.publishDate as publishDate,
		p.upvotes as upvotes, p.downvotes as downvotes,
		p.viewCount as viewCount, r.createTime as createTime,
		p.lastModifiedTime as lastModifiedTime, author
	`

	queryReqPostGetAll := QueryRequest{
		Name:   "post-get-all",
		Result: &[]Post{},
		Query: MakeQuery(
			postGetAll,
			nil,
			nil,
		),
	}

	result := QueryResult{}
	err := context.DB.RunSingleQuery(queryReqPostGetAll, &result)
	if err != nil {
		return http.StatusBadRequest, err
	}
	log.Println(result.Result)
	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for POST /posts/query
func PostQuery(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	body, err := getReqBody(r)
	if err != nil {
		return http.StatusBadRequest, err
	}
	log.Println("query body: " + body)

	postFind := `
		MATCH (author:USER)-[r:CREATED]->(p:POST` + body + `)
		RETURN p.id as id, p.title as title, p.type as type,
		p.body as body, p.status as status, p.publishDate as publishDate,
		p.upvotes as upvotes, p.downvotes as downvotes,
		p.viewCount as viewCount, r.createTime as createTime,
		p.lastModifiedTime as lastModifiedTime, author
	`

	queryReqFindPost := QueryRequest{
		Name:   "find-post",
		Result: &[]Post{},
		Query:  MakeQuery(postFind, nil, nil),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqFindPost, &result)
	if err != nil {
		return http.StatusBadRequest, err
	}

	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, json.NewEncoder(w).Encode(res)
}

// handler for POST /posts
func PostCreate(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	var props map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&props)
	log.Printf("query map: %v\n", props)

	postCreate := `
		MATCH (author:USER {id:{uid}})
		CREATE (author)-[r:CREATED {createTime: timestamp()}]->(p:POST {props})
		RETURN p.id as id, p.title as title, p.type as type,
		p.body as body, p.status as status, p.publishDate as publishDate,
		p.upvotes as upvotes, p.downvotes as downvotes,
		p.viewCount as viewCount, r.createTime as createTime,
		p.lastModifiedTime as lastModifiedTime, author
	`

	queryReqCreatePost := QueryRequest{
		Name:   "create-post",
		Result: &[]Post{},
		Query: MakeQuery(
			postCreate,
			Props{"uid": props["author"], "props": props},
			nil,
		),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqCreatePost, &result)
	if err != nil {
		return http.StatusBadRequest, err
	}

	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(res)
}
