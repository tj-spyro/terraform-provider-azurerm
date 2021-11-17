package eventsubscriptions

import (
	"encoding/json"
	"fmt"
)

var _ AdvancedFilter = StringContainsAdvancedFilter{}

type StringContainsAdvancedFilter struct {
	Values *[]string `json:"values,omitempty"`

	// Fields inherited from AdvancedFilter
	Key *string `json:"key,omitempty"`
}

var _ json.Marshaler = StringContainsAdvancedFilter{}

func (s StringContainsAdvancedFilter) MarshalJSON() ([]byte, error) {
	type wrapper StringContainsAdvancedFilter
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling StringContainsAdvancedFilter: %+v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling StringContainsAdvancedFilter: %+v", err)
	}
	decoded["operatorType"] = "StringContains"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling StringContainsAdvancedFilter: %+v", err)
	}

	return encoded, nil
}
