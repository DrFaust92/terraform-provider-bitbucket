package bitbucket

import (
	"encoding/json"
	"testing"
)

func TestFlexBool_UnmarshalJSON_BoolTrue(t *testing.T) {
	input := []byte(`{"default_branch_deletion": true}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion == nil || result.DefaultBranchDeletion.Value == nil {
		t.Fatal("expected non-nil value")
	}
	if *result.DefaultBranchDeletion.Value != true {
		t.Fatalf("expected true, got %v", *result.DefaultBranchDeletion.Value)
	}
}

func TestFlexBool_UnmarshalJSON_BoolFalse(t *testing.T) {
	input := []byte(`{"default_branch_deletion": false}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion == nil || result.DefaultBranchDeletion.Value == nil {
		t.Fatal("expected non-nil value")
	}
	if *result.DefaultBranchDeletion.Value != false {
		t.Fatalf("expected false, got %v", *result.DefaultBranchDeletion.Value)
	}
}

func TestFlexBool_UnmarshalJSON_StringTrue(t *testing.T) {
	input := []byte(`{"default_branch_deletion": "true"}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion == nil || result.DefaultBranchDeletion.Value == nil {
		t.Fatal("expected non-nil value")
	}
	if *result.DefaultBranchDeletion.Value != true {
		t.Fatalf("expected true, got %v", *result.DefaultBranchDeletion.Value)
	}
}

func TestFlexBool_UnmarshalJSON_StringFalse(t *testing.T) {
	input := []byte(`{"default_branch_deletion": "false"}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion == nil || result.DefaultBranchDeletion.Value == nil {
		t.Fatal("expected non-nil value")
	}
	if *result.DefaultBranchDeletion.Value != false {
		t.Fatalf("expected false, got %v", *result.DefaultBranchDeletion.Value)
	}
}

func TestFlexBool_UnmarshalJSON_Null(t *testing.T) {
	input := []byte(`{"default_branch_deletion": null}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion == nil {
		// json.Unmarshal with null sets the pointer to nil
		return
	}
	if result.DefaultBranchDeletion.Value != nil {
		t.Fatalf("expected nil value, got %v", *result.DefaultBranchDeletion.Value)
	}
}

func TestFlexBool_UnmarshalJSON_Absent(t *testing.T) {
	input := []byte(`{}`)
	var result struct {
		DefaultBranchDeletion *FlexBool `json:"default_branch_deletion"`
	}
	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DefaultBranchDeletion != nil {
		t.Fatalf("expected nil, got %v", result.DefaultBranchDeletion)
	}
}

func TestFlexBool_MarshalJSON(t *testing.T) {
	val := true
	fb := FlexBool{Value: &val}
	data, err := json.Marshal(fb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "true" {
		t.Fatalf("expected 'true', got %s", string(data))
	}
}

func TestFlexBool_BoolPtr_Nil(t *testing.T) {
	var fb *FlexBool
	if fb.BoolPtr() != nil {
		t.Fatal("expected nil")
	}
}
