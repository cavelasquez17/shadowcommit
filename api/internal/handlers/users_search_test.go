package handlers_test

import (
	"net/url"
	"testing"
)

func usernames(t *testing.T, tsURL, path string) []string {
	t.Helper()
	out := decodeJSON[struct {
		Items []struct {
			Username string `json:"username"`
		} `json:"items"`
	}](t, mustGet(t, tsURL+path))
	names := make([]string, 0, len(out.Items))
	for _, i := range out.Items {
		names = append(names, i.Username)
	}
	return names
}

// A bare % or _ typed into a search box must be taken literally, not as a wildcard that matches
// every member.
func TestUserSearchTreatsWildcardsLiterally(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	for _, n := range []string{"under_score", "plainuser", "otherperson"} {
		devLoginAs(t, ts, n)
	}

	cases := []struct {
		name, path string
		want       []string
	}{
		{"autocomplete: % matches nobody", "/api/users/search?q=" + url.QueryEscape("%"), nil},
		{"autocomplete: _ matches only names containing an underscore", "/api/users/search?q=" + url.QueryEscape("_"), []string{"under_score"}},
		{"autocomplete: a normal prefix still works", "/api/users/search?q=plain", []string{"plainuser"}},
		{"directory: % matches nobody", "/api/users?q=" + url.QueryEscape("%"), nil},
		{"directory: _ matches only names containing an underscore", "/api/users?q=" + url.QueryEscape("_"), []string{"under_score"}},
		{"directory: a normal substring still works", "/api/users?q=person", []string{"otherperson"}},
		{"directory: language % matches nobody", "/api/users?language=" + url.QueryEscape("%"), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := usernames(t, ts.URL, tc.path)
			if len(got) != len(tc.want) || (len(got) == 1 && got[0] != tc.want[0]) {
				t.Errorf("GET %s = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
