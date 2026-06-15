package integration

import "encoding/json"

func unmarshalJSON(raw string, dest interface{}) error {
	return json.Unmarshal([]byte(raw), dest)
}
