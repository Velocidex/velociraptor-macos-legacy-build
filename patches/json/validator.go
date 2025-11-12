package json

import (
	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/velociraptor/utils"
)

type ValidationOptions struct {
	// Allows fetching http schemas using a custom http client.
	Client utils.HTTPClient
}

func ParseJsonToObjectWithSchema(
	json_data string, schemas []string,
	options ValidationOptions) (res *ordereddict.Dict, errs []error) {
	return nil, []error{utils.NotImplementedError}
}

func ParseJsonToMapWithSchema(
	json_string string, schemas []string,
	options ValidationOptions) (
	res map[string]interface{},
	schema *int, errs []error) {
	return nil, nil, []error{utils.NotImplementedError}
}

func PopulateDefaults(dest *ordereddict.Dict,
	src map[string]interface{}, schema *int) {
}
