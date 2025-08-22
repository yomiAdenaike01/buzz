package utils

import "encoding/json"

func MustJSON(data any) string {
	d, _ := json.Marshal(data)
	return string(d)
}
