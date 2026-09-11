package client

import (
	"encoding/json"
	"strings"
	"testing"
)

// The scan endpoints report description inconsistently: POST /scans carries it,
// PUT /scans/{id} carries it when it answers with a body at all, and
// GET /scans/{id} never does. A *string keeps "not reported" apart from
// "reported as empty", which is the difference between an unverifiable update
// and a description the tenant wiped.
func TestScanDescriptionDistinguishesAbsentFromEmpty(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *string
	}{
		{name: "reported", body: `{"description":"nightly sweep"}`, want: strptr("nightly sweep")},
		{name: "reported empty", body: `{"description":""}`, want: strptr("")},
		{name: "absent", body: `{"name":"Nightly"}`, want: nil},
		{name: "explicit null", body: `{"description":null}`, want: nil},
	}

	for _, tt := range tests {
		t.Run("detail/"+tt.name, func(t *testing.T) {
			var detail ScanDetail
			if err := json.Unmarshal([]byte(tt.body), &detail); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			assertStrPtr(t, "description", detail.Description, tt.want)
		})

		t.Run("info/"+tt.name, func(t *testing.T) {
			var info ScanInfo
			if err := json.Unmarshal([]byte(tt.body), &info); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			assertStrPtr(t, "description", info.Description, tt.want)
		})
	}
}

// PUT /scans/{id} is documented to return the scan object bare, POST /scans
// wraps the same object in "scan", and tenants have answered the update with no
// body at all. All three have to decode, and only the last may yield a nil scan
// -- a zero-valued ScanDetail would claim an echo nobody sent.
func TestScanUpdateResponseAcceptsEveryObservedShape(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantScan    bool
		wantName    string
		wantDesc    *string
		wantInvalid bool
	}{
		{
			name:     "bare object, as documented",
			body:     `{"id":7,"name":"Nightly","description":"nightly sweep"}`,
			wantScan: true,
			wantName: "Nightly",
			wantDesc: strptr("nightly sweep"),
		},
		{
			name:     "wrapped like the create response",
			body:     `{"scan":{"id":7,"name":"Nightly","description":"nightly sweep"}}`,
			wantScan: true,
			wantName: "Nightly",
			wantDesc: strptr("nightly sweep"),
		},
		{
			name:     "object without a description",
			body:     `{"id":7,"name":"Nightly"}`,
			wantScan: true,
			wantName: "Nightly",
			wantDesc: nil,
		},
		{name: "null body", body: `null`},
		{
			// An empty object is a body, but it reports nothing; the nil
			// description is what stops it being read as a cleared one.
			name:     "empty object",
			body:     `{}`,
			wantScan: true,
		},
		{name: "not an object", body: `"nope"`, wantInvalid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp ScanUpdateResponse
			err := json.Unmarshal([]byte(tt.body), &resp)
			if tt.wantInvalid {
				if err == nil {
					t.Fatal("unmarshal succeeded, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.wantScan != (resp.Scan != nil) {
				t.Fatalf("scan present = %t, want %t", resp.Scan != nil, tt.wantScan)
			}
			if !tt.wantScan {
				return
			}
			if resp.Scan.Name != tt.wantName {
				t.Errorf("name = %q, want %q", resp.Scan.Name, tt.wantName)
			}
			assertStrPtr(t, "description", resp.Scan.Description, tt.wantDesc)
		})
	}
}

// An empty response body never reaches UnmarshalJSON -- Do skips the decode --
// so the zero value has to mean "no echo" on its own.
func TestScanUpdateResponseZeroValueReportsNoEcho(t *testing.T) {
	var resp ScanUpdateResponse
	if resp.Scan != nil {
		t.Errorf("zero value reports a scan: %+v", resp.Scan)
	}
}

func strptr(s string) *string { return &s }

func assertStrPtr(t *testing.T, label string, got, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("%s = %s, want %s", label, showStrPtr(got), showStrPtr(want))
	case *got != *want:
		t.Errorf("%s = %q, want %q", label, *got, *want)
	}
}

func showStrPtr(p *string) string {
	if p == nil {
		return "nil"
	}
	return `"` + *p + `"`
}

// The keys a response carried are recorded for diagnostics: when logs cannot be
// collected from a production run, the error message is the only channel, and
// "the object came back with three keys" is what separates a normalising tenant
// from something else answering for it.
func TestScanResponsesRecordTheKeysTheyCarried(t *testing.T) {
	var detail ScanDetail
	if err := json.Unmarshal([]byte(`{"id":7,"name":"Nightly","description":"x"}`), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := strings.Join(detail.PresentKeys, ","), "description,id,name"; got != want {
		t.Errorf("detail keys = %q, want %q (sorted)", got, want)
	}
	if detail.ID != 7 || detail.Name != "Nightly" {
		t.Errorf("decoding the object itself regressed: %+v", detail)
	}

	var info ScanInfo
	if err := json.Unmarshal([]byte(`{"object_id":7,"name":"Nightly"}`), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := strings.Join(info.PresentKeys, ","), "name,object_id"; got != want {
		t.Errorf("info keys = %q, want %q", got, want)
	}
	if info.ID != 7 {
		t.Errorf("object_id decoded as %d", info.ID)
	}

	// A thin object is the case worth naming, so an empty list must not look
	// like a missing feature.
	var empty ScanDetail
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(empty.PresentKeys) != 0 {
		t.Errorf("keys = %v, want none", empty.PresentKeys)
	}

	// And the update wrapper must carry them through from either shape.
	for _, body := range []string{
		`{"scan":{"id":7,"name":"Nightly"}}`,
		`{"id":7,"name":"Nightly"}`,
	} {
		var resp ScanUpdateResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		if got, want := strings.Join(resp.Scan.PresentKeys, ","), "id,name"; got != want {
			t.Errorf("%s -> keys %q, want %q", body, got, want)
		}
	}
}
