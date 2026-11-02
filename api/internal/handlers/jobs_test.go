package handlers_test

import (
	"net/http"
	"testing"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
)

type jobResp struct {
	ID          string `json:"id"`
	Company     string `json:"company"`
	CountryCode string `json:"country_code"`
	Currency    string `json:"currency"`
	Seniority   string `json:"seniority"`
	Remote      bool   `json:"remote"`
	IsSponsored bool   `json:"is_sponsored"`
	IsFilled    bool   `json:"is_filled"`
	ViewCount   int    `json:"view_count"`
}

type jobList struct {
	Items []jobResp `json:"items"`
	Total int       `json:"total"`
}

func jobBody(company string, extra map[string]any) map[string]any {
	b := map[string]any{
		"company": company, "title": "Staff Engineer", "apply_url": "https://example.com/apply",
		"body": "A job description that is long enough to be valid.",
	}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func newJob(t *testing.T, tsURL string, c *http.Client, company string, extra map[string]any) jobResp {
	t.Helper()
	resp := postJSON(t, c, tsURL+"/api/jobs", jobBody(company, extra))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create job: got %d", resp.StatusCode)
	}
	return decodeJSON[struct {
		Job jobResp `json:"job"`
	}](t, resp).Job
}

func listJobs(t *testing.T, tsURL, query string) jobList {
	t.Helper()
	return decodeJSON[jobList](t, mustGet(t, tsURL+"/api/jobs"+query))
}

func TestCreateJobNormalizesFieldsAndAppliesDefaults(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	c := devLoginAs(t, ts, "job_poster")

	j := newJob(t, ts.URL, c, "Acme", map[string]any{"country_code": "co", "currency": "cop"})
	if j.CountryCode != "CO" || j.Currency != "COP" {
		t.Errorf("country/currency should be upper-cased, got %q/%q", j.CountryCode, j.Currency)
	}

	d := newJob(t, ts.URL, c, "Defaults Inc", nil)
	if d.Currency != "USD" || d.Seniority != string(models.SeniorityMid) {
		t.Errorf("defaults should be USD/mid, got %q/%q", d.Currency, d.Seniority)
	}
}

func TestCreateJobValidation(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	c := devLoginAs(t, ts, "job_validator")

	cases := []struct {
		name  string
		extra map[string]any
	}{
		{"javascript url", map[string]any{"apply_url": "javascript:alert(1)"}},
		{"not a url", map[string]any{"apply_url": "not a url"}},
		{"ftp scheme", map[string]any{"apply_url": "ftp://example.com/file"}},
		{"body too short", map[string]any{"body": "short"}},
		{"company too short", map[string]any{"company": "x"}},
		{"country code too long", map[string]any{"country_code": "COL"}},
	}
	t.Run("rejected", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resp := postJSON(t, c, ts.URL+"/api/jobs", jobBody("Valid Co", tc.extra))
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("got %d, want 400", resp.StatusCode)
				}
			})
		}
	})
	if got := listJobs(t, ts.URL, "").Total; got != 0 {
		t.Errorf("rejected postings must not be stored, got %d", got)
	}

	resp := postJSON(t, &http.Client{}, ts.URL+"/api/jobs", jobBody("Anon Co", nil))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous create: got %d, want 401", resp.StatusCode)
	}
}

func TestListJobsIsPublicSortsSponsoredFirstAndFilters(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	c := devLoginAs(t, ts, "job_lister")
	// The sponsored job is created FIRST (oldest), so only the sponsorship ordering can
	// put it on top; plain recency would leave it last.
	spons := newJob(t, ts.URL, c, "Sponsored Co", nil)
	newJob(t, ts.URL, c, "Plain Co", map[string]any{"country_code": "US"})
	remote := newJob(t, ts.URL, c, "Remote Co", map[string]any{"remote": true, "country_code": "CO", "seniority": "senior"})
	if err := db.Model(&models.JobPosting{}).Where("public_id = ?", spons.ID).Update("is_sponsored", true).Error; err != nil {
		t.Fatal(err)
	}

	all := listJobs(t, ts.URL, "") // no login: the board is public
	if all.Total != 3 || len(all.Items) != 3 {
		t.Fatalf("want 3 jobs, got total=%d items=%d", all.Total, len(all.Items))
	}
	if all.Items[0].ID != spons.ID {
		t.Errorf("sponsored job must come first, got %s", all.Items[0].Company)
	}

	for q, want := range map[string]string{
		"?remote=true":      remote.ID,
		"?country=co":       remote.ID,
		"?seniority=senior": remote.ID,
	} {
		got := listJobs(t, ts.URL, q)
		if got.Total != 1 || got.Items[0].ID != want {
			t.Errorf("filter %s: want only %s, got %+v", q, want, got)
		}
	}
	if got := listJobs(t, ts.URL, "?limit=1"); len(got.Items) != 1 || got.Total != 3 {
		t.Errorf("limit=1 should return 1 item of 3 total, got %d of %d", len(got.Items), got.Total)
	}
}

func TestGetJobCountsViewsAndHandlesUnknown(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	c := devLoginAs(t, ts, "job_viewer")
	j := newJob(t, ts.URL, c, "Viewed Co", nil)

	for range 2 {
		mustGet(t, ts.URL+"/api/jobs/"+j.ID).Body.Close()
	}
	resp := mustGet(t, ts.URL+"/api/jobs/"+j.ID)
	got := decodeJSON[struct {
		Job jobResp `json:"job"`
	}](t, resp).Job
	// The response reflects the count before this request's own increment.
	if got.ViewCount != 2 {
		t.Errorf("view_count = %d, want 2 after two prior views", got.ViewCount)
	}

	resp = mustGet(t, ts.URL+"/api/jobs/job_unknown")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown job: got %d, want 404", resp.StatusCode)
	}
}

func TestSponsorIsAdminOnlyAndToggles(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "spons_owner")
	j := newJob(t, ts.URL, owner, "Sponsor Me Co", nil)

	resp := postJSON(t, owner, ts.URL+"/api/jobs/"+j.ID+"/sponsor", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a member (even the owner) sponsoring: got %d, want 403", resp.StatusCode)
	}
}

func TestSponsorByAdmin(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "spons2_owner")
	admin := devLoginAs(t, ts, "spons2_admin")
	makeAdmin(t, db, "spons2_admin")
	j := newJob(t, ts.URL, owner, "Sponsor Me Co", nil)

	for _, want := range []bool{true, false} {
		resp := postJSON(t, admin, ts.URL+"/api/jobs/"+j.ID+"/sponsor", nil)
		out := decodeJSON[struct {
			Sponsored bool `json:"sponsored"`
		}](t, resp)
		if out.Sponsored != want {
			t.Fatalf("sponsored = %v, want %v (it must toggle)", out.Sponsored, want)
		}
	}

	resp := postJSON(t, admin, ts.URL+"/api/jobs/job_unknown/sponsor", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("sponsoring an unknown job: got %d, want 404", resp.StatusCode)
	}
}

func TestMarkFilledPermissionsAndVisibility(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "fill_owner")
	other := devLoginAs(t, ts, "fill_other")
	admin := devLoginAs(t, ts, "fill_admin")
	makeAdmin(t, db, "fill_admin")
	mine := newJob(t, ts.URL, owner, "Mine Co", nil)
	theirs := newJob(t, ts.URL, owner, "Admin Fills Co", nil)

	resp := postJSON(t, other, ts.URL+"/api/jobs/"+mine.ID+"/filled", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("other member marking filled: got %d, want 403", resp.StatusCode)
	}
	if got := listJobs(t, ts.URL, "").Total; got != 2 {
		t.Fatalf("a forbidden attempt must not hide the job, total=%d", got)
	}

	for _, who := range []struct {
		c  *http.Client
		id string
	}{{owner, mine.ID}, {admin, theirs.ID}} {
		resp := postJSON(t, who.c, ts.URL+"/api/jobs/"+who.id+"/filled", nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("mark filled: got %d, want 200", resp.StatusCode)
		}
	}
	if got := listJobs(t, ts.URL, "").Total; got != 0 {
		t.Errorf("filled jobs must leave the board, total=%d", got)
	}
}

func TestDeleteJobPermissions(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	owner := devLoginAs(t, ts, "jdel_owner")
	other := devLoginAs(t, ts, "jdel_other")
	j := newJob(t, ts.URL, owner, "Delete Me Co", nil)

	resp := deleteReq(t, other, ts.URL+"/api/jobs/"+j.ID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("other member deleting: got %d, want 403", resp.StatusCode)
	}
	resp = deleteReq(t, owner, ts.URL+"/api/jobs/"+j.ID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner deleting: got %d, want 200", resp.StatusCode)
	}
	resp = deleteReq(t, owner, ts.URL+"/api/jobs/"+j.ID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleting twice: got %d, want 404", resp.StatusCode)
	}
	if got := listJobs(t, ts.URL, "").Total; got != 0 {
		t.Errorf("deleted job still listed, total=%d", got)
	}
}

func TestJobStatsCountsOpenAndSponsored(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)
	c := devLoginAs(t, ts, "stats_poster")
	newJob(t, ts.URL, c, "One Co", map[string]any{"seniority": "senior"})
	s := newJob(t, ts.URL, c, "Two Co", map[string]any{"seniority": "senior"})
	f := newJob(t, ts.URL, c, "Three Co", nil)
	db.Model(&models.JobPosting{}).Where("public_id = ?", s.ID).Update("is_sponsored", true)
	postJSON(t, c, ts.URL+"/api/jobs/"+f.ID+"/filled", nil).Body.Close()

	resp := mustGet(t, ts.URL+"/api/jobs/stats")
	got := decodeJSON[struct {
		Open       int `json:"open"`
		Sponsored  int `json:"sponsored"`
		BySeniorty []struct {
			Seniority string `json:"seniority"`
			Count     int    `json:"count"`
		} `json:"by_seniority"`
	}](t, resp)
	if got.Open != 2 || got.Sponsored != 1 {
		t.Errorf("open/sponsored = %d/%d, want 2/1 (filled excluded)", got.Open, got.Sponsored)
	}
	if len(got.BySeniorty) != 1 || got.BySeniorty[0].Seniority != "senior" || got.BySeniorty[0].Count != 2 {
		t.Errorf("by_seniority = %+v, want [senior:2]", got.BySeniorty)
	}
}

func mustGet(t *testing.T, u string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, http.NoBody)
	if err != nil {
		t.Fatalf("build GET %s: %v", u, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
	return resp
}
