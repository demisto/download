package util

import (
	"encoding/json"
	"strings"
)

// removeFromArr the given filter
func removeFromArr(a []interface{}, filter string) []interface{} {
	var arr []interface{}
	ok := true
	for i := range a {
		switch cell := a[i].(type) {
		case map[string]interface{}:
			arr = append(arr, removeFromMap(cell, filter))
		default:
			ok = false
		}
	}
	if ok {
		return arr
	}
	return a
}

// removeFromMap the given filter where filter is in the form of key1.key2
func removeFromMap(m map[string]interface{}, filter string) map[string]interface{} {
	parts := strings.Split(filter, ".")
	if len(parts) == 1 {
		delete(m, parts[0])
		return m
	} else if len(parts) > 1 {
		if v, has := m[parts[0]]; has {
			switch v := v.(type) {
			case map[string]interface{}:
				m[parts[0]] = removeFromMap(v, strings.Join(parts[1:], "."))
			case []interface{}:
				m[parts[0]] = removeFromArr(v, strings.Join(parts[1:], "."))
			default:
				// Be lenient and ignore weird filters
			}
		}
	}
	return m
}

// ToGenericFilter translates an object to generic slices and maps while filtering the given fields
func ToGenericFilter(v interface{}, filters ...string) (interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(filters) == 0 {
		return b, nil
	}
	var intermediate interface{}
	err = json.Unmarshal(b, &intermediate)
	if err != nil {
		return nil, err
	}
	switch intermediate := intermediate.(type) {
	case map[string]interface{}:
		for _, filter := range filters {
			intermediate = removeFromMap(intermediate, filter)
		}
	case []interface{}:
		for _, filter := range filters {
			intermediate = removeFromArr(intermediate, filter)
		}
	}
	return intermediate, nil
}

// MarshalWithFilter the given struct while filtering out the given Fields
// The filters should be in the form of x.y where x and y are the JSON names (as in the JSON tags)
// Naive and slow implementation - might be rewritten if needed
func MarshalWithFilter(v interface{}, filters ...string) ([]byte, error) {
	intermediate, err := ToGenericFilter(v, filters...)
	if err != nil {
		return nil, err
	}
	return json.Marshal(intermediate)
}

// AddPropToJSONString receives the JSON string, key in the form of "part1.part2" and value
// and adds all the relevant parts to resulting JSON string
// For example, passing empty string and "Security.SessionKey" as key will create {"Security": {"SessionKey": val}}
// Does not support array values
func AddPropToJSONString(s, key string, prop interface{}) (string, error) {
	var sMap map[string]interface{}
	if len(s) > 0 {
		err := json.Unmarshal([]byte(s), &sMap)
		if err != nil {
			return s, err
		}
	} else {
		sMap = make(map[string]interface{})
	}
	curr := sMap
	keys := strings.Split(key, ".")
	for _, k := range keys[0 : len(keys)-1] {
		currIfc, ok := curr[k]
		if ok {
			curr = currIfc.(map[string]interface{})
		} else {
			sub := make(map[string]interface{})
			curr[k] = sub
			curr = sub
		}
	}
	curr[keys[len(keys)-1]] = prop
	b, err := json.Marshal(sMap)
	if err != nil {
		return s, err
	}
	return string(b), nil
}
