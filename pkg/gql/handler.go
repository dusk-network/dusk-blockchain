package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/go-chi/render"
	"github.com/graphql-go/graphql"
)

type data struct {
	Query     string                 `json:"query"`
	Operation string                 `json:"operationName,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// handleQuery to process graphQL query
func handleQuery(schema *graphql.Schema, w http.ResponseWriter, r *http.Request, db database.DB) {

	if r.Body == nil {
		http.Error(w, "Must provide graphql query in request body", 400)
		return
	}

	// Read and close JSON request body
	body, err := ioutil.ReadAll(r.Body)
	defer func() {
		_ = r.Body.Close()
	}()
	if err != nil {
		msg := fmt.Sprintf("%d error request: %v", http.StatusBadRequest, err)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var req data
	if err := json.Unmarshal(body, &req); err != nil {
		msg := fmt.Sprintf("Unmarshal request: %v", err)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	log.Tracef("Query: %s", req.Query)
	log.Tracef("Variables: %s", req.Variables)
	log.Tracef("Operation: %s", req.Operation)

	// Execute graphql query
	result := graphql.Do(graphql.Params{
		Schema:         *schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.Operation,
		Context:        context.WithValue(context.Background(), "database", db), //nolint
	})

	// Error check
	if len(result.Errors) > 0 {
		log.Warnf("Execute query error(s): %v", result.Errors)
	}

	render.JSON(w, r, result)
}
