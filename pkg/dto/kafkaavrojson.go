package dto

import (
	"fmt"
	"strings"
)

func ConvertKafkaNativeToJson(bin interface{}) ([]byte, error) {
	bbin, ok := bin.(map[string]interface{})
	if !ok {
		return []byte{}, fmt.Errorf("not parsable")
	}

	for _, val := range bbin {
		cc, err := convert(val)
		if err != nil {
			return []byte{}, err
		}
		return []byte(`{` + cc + `}`), nil
	}

	return []byte{}, fmt.Errorf("empty payload")
}

func convert(bin interface{}) (string, error) {
	bbin, ok := bin.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("not parsable")
	}

	var retAll []string

	for key, val := range bbin {
		ret := `"` + key + `":`
		if val == nil {
			ret += "null"
		}
		if guessIsUnion(val) {
			ret += guessTakeValueFromUnion(val)
			retAll = append(retAll, ret)
		}
	}

	return strings.Join(retAll, ","), nil
}

func guessIsUnion(bin interface{}) bool {
	bbin, ok := bin.(map[string]interface{})
	if !ok {
		return false
	}

	for key := range bbin {
		if key == "string" || key == "int" || key == "boolean" || key == "float" || key == "long" || key == "enum" {
			continue
		}
		return false
	}

	return true
}

func guessTakeValueFromUnion(bin interface{}) string {
	bbin, ok := bin.(map[string]interface{})
	if !ok {
		return ""
	}

	for key, val := range bbin {
		if key == "string" || key == "enum" {
			if vval, ok := val.(string); ok {
				return `"` + vval + `"`
			}
			return ""
		}

		if key == "boolean" {
			if vval, ok := val.(bool); ok {
				if vval {
					return "true"
				}
			}
			return "false"
		}

		if key == "int" || key == "long" {
			if vval, ok := val.(int32); ok {
				return fmt.Sprint(vval)
			}
			if vval, ok := val.(int64); ok {
				return fmt.Sprint(vval)
			}
			return "0"
		}
		break
	}

	return ""
}
