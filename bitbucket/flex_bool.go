package bitbucket

import (
	"encoding/json"
	"strconv"
)

// FlexBool is a bool type that can be unmarshaled from both JSON boolean and
// JSON string representations (e.g. true, "true", "false").
// This is needed because the Bitbucket API inconsistently returns some boolean
// fields as strings (e.g. "default_branch_deletion": "false").
type FlexBool struct {
	Value *bool
}

func (fb *FlexBool) UnmarshalJSON(data []byte) error {
	// Try bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		fb.Value = &b
		return nil
	}

	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fb.Value = &parsed
		return nil
	}

	// null
	fb.Value = nil
	return nil
}

func (fb FlexBool) MarshalJSON() ([]byte, error) {
	if fb.Value == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(*fb.Value)
}

// BoolPtr returns the underlying *bool value.
func (fb *FlexBool) BoolPtr() *bool {
	if fb == nil {
		return nil
	}
	return fb.Value
}
