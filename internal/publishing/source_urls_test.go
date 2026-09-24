package publishing

import (
	"reflect"
	"testing"
)

// Source URLs are rendered as anchor hrefs in a browser, so the scheme gate is
// the security-relevant part of this validator: `javascript:` and `data:` must
// never survive normalization.
func TestNormalizeSourceURLs(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{name: "nil input stays nil", input: nil, want: nil},
		{name: "empty input stays nil", input: []string{}, want: nil},
		{name: "blank entries are dropped", input: []string{"  ", ""}, want: nil},
		{name: "https is accepted", input: []string{"https://example.com/page"}, want: []string{"https://example.com/page"}},
		{name: "http is accepted", input: []string{"http://example.com"}, want: []string{"http://example.com"}},
		{
			name:  "entries are trimmed and order preserved",
			input: []string{"  https://example.com/b  ", "https://example.com/a"},
			want:  []string{"https://example.com/b", "https://example.com/a"},
		},
		{
			name:  "duplicates collapse to the first occurrence",
			input: []string{"https://example.com/a", "https://example.com/a"},
			want:  []string{"https://example.com/a"},
		},
		{name: "javascript scheme is rejected", input: []string{"javascript:alert(1)"}, wantErr: true},
		{name: "data scheme is rejected", input: []string{"data:text/html,<script>alert(1)</script>"}, wantErr: true},
		{name: "file scheme is rejected", input: []string{"file:///etc/passwd"}, wantErr: true},
		{name: "scheme-less relative path is rejected", input: []string{"/just/a/path"}, wantErr: true},
		{name: "bare host without scheme is rejected", input: []string{"example.com/page"}, wantErr: true},
		{name: "http without host is rejected", input: []string{"http://"}, wantErr: true},
		{name: "one bad entry rejects the whole list", input: []string{"https://example.com", "javascript:alert(1)"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeSourceURLs(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeSourceURLs(%#v) = %#v, want error", tc.input, got)
				}
				if code := ErrorCode(err); code != ErrCodeValidationFailed {
					t.Fatalf("ErrorCode() = %q, want %q", code, ErrCodeValidationFailed)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeSourceURLs(%#v) unexpected error = %v", tc.input, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("NormalizeSourceURLs(%#v) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}
