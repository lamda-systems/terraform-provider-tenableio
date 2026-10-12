package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSanitizerClean(t *testing.T) {
	t.Parallel()

	const template = "731a8e52-3ea6-a291-ec0a-d2ff0619c19d7bd788d6be818b65"
	in := decode(t, `{
		"uuid": "`+template+`",
		"owner": "someone@example.com",
		"user_permissions": 128,
		"settings": {"basic": {"inputs": [
			{"id": "name", "type": "entry", "required": true},
			{"id": "folder_id", "type": "select", "default": 7,
			 "options": [{"name": "Finance scans", "id": 7}]},
			{"id": "report_verbosity", "type": "select", "default": "1",
			 "options": [{"name": "Normal", "value": "1"}]},
			{"id": "smtp_from", "type": "entry", "default": "ops@example.com"}
		], "acls": [{"name": "everyone"}]}}
	}`)

	s := newSanitizer()
	s.known[template] = true
	got := s.clean(in, "policy/basic")

	want := decode(t, `{
		"uuid": "`+template+`",
		"settings": {"basic": {"inputs": [
			{"id": "name", "type": "entry", "required": true},
			{"id": "folder_id", "type": "select", "redacted": true},
			{"id": "report_verbosity", "type": "select", "default": "1",
			 "options": [{"name": "Normal", "value": "1"}]},
			{"id": "smtp_from", "type": "entry", "default": "ops@example.com"}
		]}}
	}`)
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Errorf("clean() =\n%s", gotJSON)
	}

	if !reflect.DeepEqual(s.sortedRedacted(), []string{"folder_id"}) {
		t.Errorf("redacted = %v, want [folder_id]", s.sortedRedacted())
	}
	// The address left in a non-tenant input is flagged, not silently kept;
	// the known template uuid is not.
	if !reflect.DeepEqual(s.review, []string{"policy/basic.settings.basic.inputs[3].default"}) {
		t.Errorf("review = %v", s.review)
	}
}

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	return v
}
