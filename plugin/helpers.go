package plugin

import "encoding/json"

// JSONToJSONMap converts JSON to a map of its top-level keys to JSON serialized values.
func JSONToJSONMap(inrec []byte) (map[string][]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(inrec, &data); err != nil {
		return nil, err
	}

	var err error
	d := make(map[string][]byte)
	for k, v := range data {
		d[k], err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}
