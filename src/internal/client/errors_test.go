package client

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"bare 404", &APIError{StatusCode: http.StatusNotFound}, true},
		{"bare 400", &APIError{StatusCode: http.StatusBadRequest}, false},
		{
			// The case that actually occurs: every Client method wraps its
			// error, so a type assertion on *APIError never matches.
			"wrapped 404",
			fmt.Errorf("getting tag category: %w", &APIError{StatusCode: http.StatusNotFound}),
			true,
		},
		{
			"doubly wrapped 404",
			fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &APIError{StatusCode: http.StatusNotFound})),
			true,
		},
		{
			"wrapped 500",
			fmt.Errorf("getting tag category: %w", &APIError{StatusCode: http.StatusInternalServerError}),
			false,
		},
		{"unrelated error", fmt.Errorf("dial tcp: connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
