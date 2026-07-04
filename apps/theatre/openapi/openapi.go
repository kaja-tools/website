// Package openapi embeds the OpenAPI document that is the source of truth
// for the theatre service's HTTP interface.
package openapi

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
