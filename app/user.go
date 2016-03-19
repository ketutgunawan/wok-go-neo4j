// user handlers
package app

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var (
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

// handler for GET `/users`
func UserGetAll(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	queryReqFindAllUser := QueryRequest{
		Name:   "find-all-user",
		Result: &[]User{},
		Query: MakeQuery(
			ALL_USER,
			nil,
			nil,
		),
	}

	result := QueryResult{}
	err := context.DB.RunSingleQuery(queryReqFindAllUser, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for GET `/users/:id`
func UserGetOne(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	queryReqFindUserById := QueryRequest{
		Name:   "find-user-by-id",
		Result: &[]User{},
		Query: MakeQuery(
			FIND_USER_BY_ID,
			Props{"id": ps.ByName("id")},
			nil,
		),
	}

	result := QueryResult{}
	context.DB.RunSingleQuery(queryReqFindUserById, &result)

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for POST `/users/query`
func UserQuery(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	body, err := getReqBody(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	log.Println("query body: " + body)

	finUserCQ := "MATCH (u:USER " + body + ")" +
		"RETURN u.name as name, u.email as email, u.role as role," +
		"u.hashedPassword as hashedPassword, u.salt as salt," +
		"u.id as id"

	queryReqFindUser := QueryRequest{
		Name:   "find-user",
		Result: &[]User{},
		Query: MakeQuery(
			finUserCQ,
			nil,
			nil,
		),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqFindUser, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for POST `/users`
// Create a user, need to check duplications
func UserCreate(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	body, err := getReqBody(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	createUserCQ := "CREATE (u:USER " + body + ")" +
		"RETURN u.name as name, u.email as email, u.role as role," +
		"u.hashedPassword as hashedPassword, u.salt as salt," +
		"u.id as id"

	queryReqCreateUser := QueryRequest{
		Name:   "create-user",
		Result: &[]User{},
		Query: MakeQuery(
			createUserCQ,
			nil,
			nil,
		),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqCreateUser, &result)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for PUT `/users/:id`
func UserUpdate(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	var props map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&props)

	log.Printf("query map: %v\n", props)

	saveUserCQ := `
		MERGE (u:USER {id: {id}})
		ON CREATE SET u = {props}
		ON MATCH SET u = {props}
		RETURN u.name as name, u.email as email, u.role as role,
		u.hashedPassword as hashedPassword, u.salt as salt, u.id as id
	`
	queryReqUpdateUser := QueryRequest{
		Name:   "update-user",
		Result: &[]User{},
		Query: MakeQuery(
			saveUserCQ,
			Props{"id": ps.ByName("id"), "props": props},
			nil,
		),
	}

	result := QueryResult{}
	err = context.DB.RunSingleQuery(queryReqUpdateUser, &result)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, json.NewEncoder(w).Encode(result.Result)
}

// handler for POST /users/query/:queryName
func UserComplexQuery(context *AppContext, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (int, error) {
	queryName := ps.ByName("queryName")
	return http.StatusNotImplemented, json.NewEncoder(w).Encode("Not implemented yet")
}
