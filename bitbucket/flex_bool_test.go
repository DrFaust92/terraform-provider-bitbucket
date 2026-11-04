package bitbucket

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlexBoolUnmarshal(t *testing.T) {
	cases := map[string]*bool{
		`{"default_branch_deletion": "true"}`:  new(true),
		`{"default_branch_deletion": "false"}`: new(false),
		`{"default_branch_deletion": true}`:    new(true),
		`{"default_branch_deletion": false}`:   new(false),
		`{"default_branch_deletion": null}`:    nil,
		`{}`:                                   nil,
	}
	for in, want := range cases {
		var m BranchingModel
		if err := json.Unmarshal([]byte(in), &m); err != nil {
			t.Fatalf("%s: %s", in, err)
		}
		got := m.DefaultBranchDeletion
		if (got == nil) != (want == nil) || (got != nil && bool(*got) != *want) {
			t.Fatalf("%s: got %v, want %v", in, got, want)
		}
	}

	var m BranchingModel
	if err := json.Unmarshal([]byte(`{"default_branch_deletion": "maybe"}`), &m); err == nil {
		t.Fatal("expected an error for a non-boolean string")
	}
}

func TestFlexBoolMarshalsAsString(t *testing.T) {
	out, err := json.Marshal(BranchingModel{DefaultBranchDeletion: new(flexBool(true))})
	if err != nil {
		t.Fatal(err)
	}
	if want := `"default_branch_deletion":"true"`; !json.Valid(out) || !strings.Contains(string(out), want) {
		t.Fatalf("got %s, want it to contain %s", out, want)
	}
}
