package eventsubscriptions

import (
	"encoding/json"
	"fmt"
)

var _ AdvancedFilter = IsNotNullAdvancedFilter{}

type IsNotNullAdvancedFilter struct {

	// Fields inherited from AdvancedFilter
	Key *string `json:"key,omitempty"`
}

var _ json.Marshaler = IsNotNullAdvancedFilter{}

func (s IsNotNullAdvancedFilter) MarshalJSON() ([]byte, error) {
	type wrapper IsNotNullAdvancedFilter
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling IsNotNullAdvancedFilter: %+v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling IsNotNullAdvancedFilter: %+v", err)
	}
	decoded["operatorType"] = "IsNotNull"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling IsNotNullAdvancedFilter: %+v", err)
	}

	return encoded, nil
}
