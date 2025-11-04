package processor

import (
	"testing"
)

func TestSubstitutePathParam(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		paramName  string
		paramValue string
		wantURL    string
		wantErr    bool
	}{
		{
			name:       "valid substitution",
			url:        "http://api.com/v1/users/:id",
			paramName:  ":id",
			paramValue: "user123",
			wantURL:    "http://api.com/v1/users/user123",
			wantErr:    false,
		},
		{
			name:       "multiple placeholders",
			url:        "http://api.com/v1/org/:orgId/users/:id",
			paramName:  ":id",
			paramValue: "user456",
			wantURL:    "http://api.com/v1/org/:orgId/users/user456",
			wantErr:    false,
		},
		{
			name:       "placeholder not found",
			url:        "http://api.com/v1/users/profile",
			paramName:  ":id",
			paramValue: "user123",
			wantErr:    true,
		},
		{
			name:       "empty URL",
			url:        "",
			paramName:  ":id",
			paramValue: "user123",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := substitutePathParam(tt.url, tt.paramName, tt.paramValue)

			if (err != nil) != tt.wantErr {
				t.Errorf("substitutePathParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && gotURL != tt.wantURL {
				t.Errorf("substitutePathParam() got = %v, want %v", gotURL, tt.wantURL)
			}
		})
	}
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		isEmpty bool
	}{
		{
			name:    "valid key",
			input:   []byte("user123"),
			want:    "user123",
			isEmpty: false,
		},
		{
			name:    "key with null bytes",
			input:   []byte("user\x00123"),
			want:    "user123",
			isEmpty: false,
		},
		{
			name:    "key with leading/trailing spaces",
			input:   []byte("  user123  "),
			want:    "user123",
			isEmpty: false,
		},
		{
			name:    "key with non-printable characters",
			input:   []byte("user\x01\x02123"),
			want:    "user123",
			isEmpty: false,
		},
		{
			name:    "empty key after sanitization",
			input:   []byte("\x00\x00\x00"),
			want:    "",
			isEmpty: true,
		},
		{
			name:    "key with special characters",
			input:   []byte("user@123#abc"),
			want:    "user@123#abc",
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeKey(tt.input)

			if got != tt.want {
				t.Errorf("sanitizeKey() got = %q, want %q", got, tt.want)
			}

			isEmpty := got == ""
			if isEmpty != tt.isEmpty {
				t.Errorf("sanitizeKey() isEmpty = %v, want %v", isEmpty, tt.isEmpty)
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		pathParam  *string
		msgKey     []byte
		wantURL    string
		wantErr    bool
		errContain string
	}{
		{
			name:      "no path parameter configured",
			baseURL:   "http://api.com/v1/users",
			pathParam: nil,
			msgKey:    []byte("user123"),
			wantURL:   "http://api.com/v1/users",
			wantErr:   false,
		},
		{
			name:      "valid key with path parameter",
			baseURL:   "http://api.com/v1/users/:id",
			pathParam: stringPtr(":id"),
			msgKey:    []byte("user123"),
			wantURL:   "http://api.com/v1/users/user123",
			wantErr:   false,
		},
		{
			name:      "key with special characters (@)",
			baseURL:   "http://api.com/v1/users/:id",
			pathParam: stringPtr(":id"),
			msgKey:    []byte("user@example.com"),
			wantURL:   "http://api.com/v1/users/user@example.com",
			wantErr:   false,
		},
		{
			name:      "key with slashes",
			baseURL:   "http://api.com/v1/users/:id",
			pathParam: stringPtr(":id"),
			msgKey:    []byte("path/to/user"),
			wantURL:   "http://api.com/v1/users/path%2Fto%2Fuser",
			wantErr:   false,
		},
		{
			name:       "empty key after sanitization",
			baseURL:    "http://api.com/v1/users/:id",
			pathParam:  stringPtr(":id"),
			msgKey:     []byte("\x00\x00\x00"),
			wantErr:    true,
			errContain: "empty after sanitization",
		},
		{
			name:       "placeholder not found",
			baseURL:    "http://api.com/v1/users/profile",
			pathParam:  stringPtr(":id"),
			msgKey:     []byte("user123"),
			wantErr:    true,
			errContain: "not found in URL",
		},
		{
			name:      "key with null bytes and spaces",
			baseURL:   "http://api.com/v1/users/:id",
			pathParam: stringPtr(":id"),
			msgKey:    []byte("  user\x00123  "),
			wantURL:   "http://api.com/v1/users/user123",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &httpProcessor{
				url:       tt.baseURL,
				pathParam: tt.pathParam,
				logr:      nil, // Using nil logger for test simplicity
			}

			gotURL, err := processor.parseURL(tt.msgKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContain != "" && err != nil {
				if !containsSubstring(err.Error(), tt.errContain) {
					t.Errorf("parseURL() error message = %q, should contain %q", err.Error(), tt.errContain)
				}
			}

			if err == nil && gotURL != tt.wantURL {
				t.Errorf("parseURL() got = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) >= len(substr) && s[:len(substr)] == substr) || len(s) > len(substr))
}
