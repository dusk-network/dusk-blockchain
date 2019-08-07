package gql

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/render"
	"github.com/graphql-go/graphql"
)

type reqBody struct {
	Query string `json:"query"`
}

// handleQuery to process graphQL query
func handleQuery(schema *graphql.Schema, w http.ResponseWriter, r http.Request) {

	if r.Body == nil {
		http.Error(w, "Must provide graphql query in request body", 400)
		return
	}

	decBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Execute graphql query
	result := executeQuery(string(decBody), *schema)

	// render.JSON comes from the chi/render package and handles
	// marshalling to json, automatically escaping HTML and setting
	// the Content-Type as application/json.
	render.JSON(w, &r, result)
}

// ExecuteQuery runs our graphql queries
func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})

	// Error check
	if len(result.Errors) > 0 {
		fmt.Printf("Unexpected errors inside ExecuteQuery: %v", result.Errors)
	}

	return result
}
