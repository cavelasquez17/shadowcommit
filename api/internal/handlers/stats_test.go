package handlers_test

import "testing"

// The web app iterates these as arrays (stats.by_language.map(...)), so an empty table must
// serialise as [] and never as null, or the page crashes while the community is still empty.
func TestStatsListsAreArraysEvenWhenEmpty(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil) // no users, no threads, no jobs

	cases := []struct {
		path string
		keys []string
	}{
		{"/api/stats", []string{"by_cause", "by_country", "by_seniority", "by_language"}},
		{"/api/jobs/stats", []string{"by_seniority"}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			body := decodeJSON[map[string]any](t, mustGet(t, ts.URL+tc.path))
			for _, key := range tc.keys {
				v, present := body[key]
				if !present {
					t.Errorf("%s: %q is missing", tc.path, key)
					continue
				}
				if _, isArray := v.([]any); !isArray {
					t.Errorf("%s: %q = %v (%T), want an empty array", tc.path, key, v, v)
				}
			}
		})
	}
}
