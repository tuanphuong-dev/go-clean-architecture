package utils

import "encoding/json"

func ToJSONString(value any) (string, error) {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
