# wok-go-neo4j

An <strong>UNFINISHED</strong> server written in Go and backed with Neo4j

## REST APIs:
#### User
* GET  /users -- Get all users
* GET  /users/:id  -- Get a user by id
* POST /users -- Create a user (with user data)
* POST /users/query -- Get users by mutiple properties (with prop values)
* POST /users/query/:queryName -- Complex query (with query parameters)
* PUT  /users/:id -- Update a user by id (with user data)

#### POST
* GET  /posts -- Get all posts
* GET  /posts/:id -- Get a post by id
* POST /posts -- Create a post (with post data)
* POST /posts/query -- Get posts by mutiple properties (with prop values)
* POST /posts/query/:queryName -- Complex query (with query parameters)
* PUT  /posts/:id -- Update a post by id (with post data)

#### Relation
* GET  /relation/:id1/:id2 -- Get relation(with properties) between nodes by their ids 
* POST /relation/query/:queryName -- Custom query(with query parameters)



# Credit:

* [Neoism](https://github.com/jmcvetta/neoism)
* [HttpRouter](https://github.com/julienschmidt/httprouter)

