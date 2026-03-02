package version

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
		wantErr             bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"v1.2.3", 1, 2, 3, false},
		{"0.0.1", 0, 0, 1, false},
		{"10.20.30", 10, 20, 30, false},
		{"1.2", 0, 0, 0, true},
		{"1.2.3.4", 0, 0, 0, true},
		{"abc", 0, 0, 0, true},
		{"1.x.3", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maj, min, pat, err := parseSemver(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if maj != tt.major || min != tt.minor || pat != tt.patch {
				t.Errorf("parseSemver(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, maj, min, pat, tt.major, tt.minor, tt.patch)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.10.0", "1.9.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "2.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.9.0", "1.10.0", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_vs_%s", tt.latest, tt.current)
		t.Run(name, func(t *testing.T) {
			got, err := isNewer(tt.latest, tt.current)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	latest, err := CheckForUpdate("dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "" {
		t.Errorf("expected empty string for dev version, got %q", latest)
	}
}

func TestCheckForUpdate_NewerAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"tag_name": "v2.0.0"}`)
	}))
	defer srv.Close()

	old := apiURL
	apiURL = srv.URL
	defer func() { apiURL = old }()

	latest, err := CheckForUpdate("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "2.0.0" {
		t.Errorf("expected %q, got %q", "2.0.0", latest)
	}
}

func TestCheckForUpdate_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"tag_name": "v1.0.0"}`)
	}))
	defer srv.Close()

	old := apiURL
	apiURL = srv.URL
	defer func() { apiURL = old }()

	latest, err := CheckForUpdate("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "" {
		t.Errorf("expected empty string when up to date, got %q", latest)
	}
}

func TestCheckForUpdate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	old := apiURL
	apiURL = srv.URL
	defer func() { apiURL = old }()

	latest, err := CheckForUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
	if latest != "" {
		t.Errorf("expected empty string on error, got %q", latest)
	}
}
