package main

import (
	"log"

	"github.com/leozhucong/wok-go-neo4j/app"
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
)

type User struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	HashedPassword string `json:"hashedPassword"`
	Salt           string `json:"salt"`
}

func main() {
	db, _ := app.OpenDB("http://neo4j:wiirok@localhost:7474/db/data")
	query := app.QueryRequest{
		Name:   "test",
		Result: &[]User{},
		Query: app.MakeQuery(
			FIND_USER_BY_EMAIL,
			app.Props{"email": "test@test.com"},
			nil,
		),
	}
	result := app.QueryResult{}
	db.RunSingleQuery(query, &result)
	log.Println(result.Result)
}
