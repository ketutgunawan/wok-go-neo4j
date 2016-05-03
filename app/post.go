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

type VoteRel struct {
	Created int  `json:"created"`
	Found   bool `json:"found"`
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
		return http.StatusInternalServerError, err
	}
	log.Println(result.Result)
	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, json.NewEncoder(w).Encode(res)
}

// handler for GET /posts/:id
func PostGetOne(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	postFindById := `
		MATCH (author:USER)-[r:CREATED]->(p:POST {id:{id}})
		RETURN p.id as id, p.title as title, p.type as type,
		p.body as body, p.status as status, p.publishDate as publishDate,
		p.upvotes as upvotes, p.downvotes as downvotes,
		p.viewCount as viewCount, r.createTime as createTime,
		p.lastModifiedTime as lastModifiedTime, author
	`
	queryReqPostFindById := QueryRequest{
		Name:   "find-post-by-id",
		Result: &[]Post{},
		Query:  MakeQuery(postFindById, Props{"id": ps.ByName("id")}, nil),
	}

	result := QueryResult{}
	err := context.DB.RunSingleQuery(queryReqPostFindById, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, json.NewEncoder(w).Encode(res)
}

// handler for POST /posts/query
func PostQuery(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	// able to get query string later
	r.ParseForm()
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
	if order := r.Form.Get("orderBy"); order != "" {
		postFind += "\nORDER BY p." + order
		if desc := r.Form.Get("desc"); desc == "true" {
			postFind += " DESC"
		}
	}
	if skip := r.Form.Get("skip"); skip != "" {
		postFind += "\nSKIP " + skip
	}
	if limit := r.Form.Get("limit"); limit != "" {
		postFind += "\nLIMIT " + limit
	}
	log.Println(postFind)

	queryReqFindPost := QueryRequest{
		Name:   "find-post",
		Result: &[]Post{},
		Query:  MakeQuery(postFind, nil, nil),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqFindPost, &result)
	if err != nil {
		return http.StatusInternalServerError, err
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
		SET p.publishDate = r.createTime
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
		return http.StatusInternalServerError, err
	}

	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(res)
}

// handler for PUT /posts/:id
// This will update the post or create one if not exists
func PostUpdate(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	var props map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&props)
	log.Printf("query map: %v\n", props)

	updateOrCreatePost := `
		MATCH (author:USER {id:{uid}})
		MERGE (author)-[r:CREATED]->(p:POST {id:{id}})
		ON CREATE SET p={props}, r.createTime=timestamp()
		ON MATCH SET p={props}, p.lastModifiedTime=timestamp()
		RETURN p.id as id, p.title as title, p.type as type,
		p.body as body, p.status as status, p.publishDate as publishDate,
		p.upvotes as upvotes, p.downvotes as downvotes,
		p.viewCount as viewCount, r.createTime as createTime,
		p.lastModifiedTime as lastModifiedTime, author 
	`
	queryReqPostUpdateOrCreate := QueryRequest{
		Name:   "update-or-create-post",
		Result: &[]Post{},
		Query: MakeQuery(
			updateOrCreatePost,
			Props{"uid": props["author"], "id": ps.ByName("id"), "props": props},
			nil,
		),
	}
	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqPostUpdateOrCreate, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	var res interface{}
	res, err = getAuthorData(result.Result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(res)
}

// handler for DELETE /posts/:id
func PostDestroy(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	postDestroy := `
		MATCH (p:POST {id:{id}})
		DETACH DELETE p
	`

	queryReqPostDestroy := QueryRequest{
		Name:   "delete-post",
		Result: nil,
		Query:  MakeQuery(postDestroy, Props{"id": ps.ByName("id")}, nil),
	}

	result := QueryResult{}
	err := context.DB.RunSingleQuery(queryReqPostDestroy, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode("Delete post ok.")
}

// vote a post
func PostVote(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	var props map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&props)
	if err != nil {
		log.Println("Parse request body error.")
	}

	postVote := `
		MATCH (u:USER {id: {props}.id}), (p:POST {id: {id}})
		MERGE (u)-[r:VOTED]->(p)
		ON CREATE SET r.created=timestamp(), r.found=false, p.upvotes=p.upvotes+1
		ON MATCH SET r.found=true
		RETURN r.created as craeted, r.found as found
	`

	queryReqPostVote := QueryRequest{
		Name:   "vote-post",
		Result: &[]VoteRel{},
		Query: MakeQuery(
			postVote,
			Props{"id": ps.ByName("id"), "props": props},
			nil,
		),
	}
	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqPostVote, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusNoContent, json.NewEncoder(w).Encode("Vote post ok.")
}

// devote a post
func PostDeleteVote(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	var props map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&props)
	if err != nil {
		log.Println("Parse request body error.")
	}

	postDeleteVote := `
		MATCH (u:USER {id: {props}.id})-[r:VOTED]->(p:POST {id: {id}})
		SET p.upvotes=p.upvotes-1
		DELETE r
	`

	queryReqPostDeleteVote := QueryRequest{
		Name:   "delete-post-vote",
		Result: nil,
		Query:  MakeQuery(postDeleteVote, Props{"id": ps.ByName("id"), "props": props}, nil),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqPostDeleteVote, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode("Delete post vote ok.")
}

// get a post's vote count
func PostGetVote(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	postGetVote := `
		MATCH (p:POST {id: {id}})
		RETURN p.upvotes as votes
	`

	queryReqPostGetVote := QueryRequest{
		Name: "get-post-vote",
		Result: &[]struct {
			Votes int `json:"votes"`
		}{},
		Query: MakeQuery(postGetVote, Props{"id": ps.ByName("id")}, nil),
	}

	result := QueryResult{}
	err := context.DB.RunSingleQuery(queryReqPostGetVote, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}
