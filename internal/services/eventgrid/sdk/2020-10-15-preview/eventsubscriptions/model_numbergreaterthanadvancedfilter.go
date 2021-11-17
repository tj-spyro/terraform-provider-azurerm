package eventsubscriptions

import (
	"encoding/json"
	"fmt"
)

var _ AdvancedFilter = NumberGreaterThanAdvancedFilter{}

type NumberGreaterThanAdvancedFilter struct {
	Value *float64 `json:"value,omitempty"`

	// Fields inherited from AdvancedFilter
	Key *string `json:"key,omitempty"`
}

var _ json.Marshaler = NumberGreaterThanAdvancedFilter{}

func (s NumberGreaterThanAdvancedFilter) MarshalJSON() ([]byte, error) {
	type wrapper NumberGreaterThanAdvancedFilter
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling NumberGreaterThanAdvancedFilter: %+v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling NumberGreaterThanAdvancedFilter: %+v", err)
	}
	decoded["operatorType"] = "NumberGreaterThan"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling NumberGreaterThanAdvancedFilter: %+v", err)
	}

	return encoded, nil
}
