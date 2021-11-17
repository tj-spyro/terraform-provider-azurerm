package eventsubscriptions

import (
	"encoding/json"
	"fmt"
)

var _ AdvancedFilter = NumberInRangeAdvancedFilter{}

type NumberInRangeAdvancedFilter struct {
	Values *[][]float64 `json:"values,omitempty"`

	// Fields inherited from AdvancedFilter
	Key *string `json:"key,omitempty"`
}

var _ json.Marshaler = NumberInRangeAdvancedFilter{}

func (s NumberInRangeAdvancedFilter) MarshalJSON() ([]byte, error) {
	type wrapper NumberInRangeAdvancedFilter
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling NumberInRangeAdvancedFilter: %+v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling NumberInRangeAdvancedFilter: %+v", err)
	}
	decoded["operatorType"] = "NumberInRange"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling NumberInRangeAdvancedFilter: %+v", err)
	}

	return encoded, nil
}
