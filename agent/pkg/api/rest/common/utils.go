package common

import (
	"encoding/json"
	"github.com/itchyny/gojq"
)

func Unmarshal(data interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GoJq(_input map[string]interface{}, _query string) interface{} {
	query, _ := gojq.Parse(_query)
	iter := query.Run(_input)
	var results []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if _, ok := v.(error); ok {
			continue
		}
		results = append(results, v)
	}
	if len(results) == 1 {
		return results[0]
	}
	return results
}
