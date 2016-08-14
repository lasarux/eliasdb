/* 
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. 
 */

/*
REST endpoint to return general database information.

/info

The info endpoint returns general database information such as known
node kinds, known attributes, etc ..

The return data is a key-value map:

{
	<info name> : <info value>,
	...
}
*/
package v1

import (
	"encoding/json"
	"net/http"

	"devt.de/eliasdb/api"
)

/*
Query endpoint definition (rooted). Handles everything under info/...
*/
const ENDPOINT_INFO_QUERY = api.API_ROOT + API_VERSION_V1 + "/info/"

/*
InfoEndpointInst creates a new endpoint handler.
*/
func InfoEndpointInst() api.RestEndpointHandler {
	return &infoEndpoint{}
}

/*
Handler object for info queries.
*/
type infoEndpoint struct {
	*api.DefaultEndpointHandler
}

/*
HandleGET handles a search query REST call.
*/
func (eq *infoEndpoint) HandleGET(w http.ResponseWriter, r *http.Request, resources []string) {

	data := make(map[string]interface{})

	// Get information

	data["partitions"] = api.GM.Partitions()

	nks := api.GM.NodeKinds()
	data["node_kinds"] = nks

	ncs := make(map[string]uint64)
	for _, nk := range nks {
		ncs[nk] = api.GM.NodeCount(nk)
	}

	data["node_counts"] = ncs

	eks := api.GM.EdgeKinds()
	data["edge_kinds"] = eks

	ecs := make(map[string]uint64)
	for _, ek := range eks {
		ecs[ek] = api.GM.EdgeCount(ek)
	}

	data["edge_counts"] = ecs

	// Write data

	w.Header().Set("content-type", "application/json; charset=utf-8")

	ret := json.NewEncoder(w)
	ret.Encode(data)
}

/*
SwaggerDefs is used to describe the endpoint in swagger.
*/
func (ge *infoEndpoint) SwaggerDefs(s map[string]interface{}) {

	s["paths"].(map[string]interface{})["/v1/info"] = map[string]interface{}{
		"get": map[string]interface{}{
			"summary":     "Return general datastore information.",
			"description": "The info endpoint returns general database information such as known node kinds, known attributes, etc .",
			"produces": []string{
				"text/plain",
				"application/json",
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "A key-value map.",
				},
				"default": map[string]interface{}{
					"description": "Error response",
					"schema": map[string]interface{}{
						"$ref": "#/definitions/Error",
					},
				},
			},
		},
	}

	// Add generic error object to definition

	s["definitions"].(map[string]interface{})["Error"] = map[string]interface{}{
		"description": "A human readable error mesage.",
		"type":        "string",
	}
}
