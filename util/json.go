package util

import "encoding/json"

func MustJson(v any) []byte {
	if v == nil {
		return make([]byte, 0)
	}
	data, _ := json.Marshal(v)
	return data
}

func MustJsonString(v any) string {
	return string(MustJson(v))
}

func MustJsonIdent(v any) []byte {
	if v == nil {
		return make([]byte, 0)
	}
	data, _ := json.MarshalIndent(v, "", " ")
	return data
}

func MustJsonIndentString(v any) string {
	return string(MustJsonIdent(v))
}
