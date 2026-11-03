package avatar

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestSVGIsStableForAUsername(t *testing.T) {
	t.Parallel()
	first, second := SVG("alice"), SVG("alice")
	if string(first) != string(second) {
		t.Fatal("the same username must always get the same avatar")
	}
}

func TestSVGDiffersBetweenUsernames(t *testing.T) {
	t.Parallel()
	seen := map[string]string{}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "frank", "grace", "heidi"} {
		s := string(SVG(name))
		if other, dup := seen[s]; dup {
			t.Fatalf("%s and %s produce the same avatar", name, other)
		}
		seen[s] = name
	}
}

func TestSVGIsWellFormedAndSelfContained(t *testing.T) {
	t.Parallel()
	out := SVG("carlos")
	var doc struct {
		XMLName xml.Name `xml:"svg"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not well-formed XML: %v", err)
	}
	// No external references of any kind: the whole point is that no third party is contacted.
	// The xmlns attribute holds a namespace identifier, which is not fetched; only real references count.
	for _, forbidden := range []string{"href", "src=", "url(", "<script", "<image", "@import"} {
		if strings.Contains(strings.ToLower(string(out)), forbidden) {
			t.Errorf("avatar must be self-contained, found %q", forbidden)
		}
	}
	if len(out) > 4096 {
		t.Errorf("avatar is %d bytes; keep it small", len(out))
	}
}

func TestSVGHandlesEmptyAndOddInput(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", "a", strings.Repeat("x", 500), "ñandú"} {
		if _, err := xmlCheck(SVG(name)); err != nil {
			t.Errorf("SVG(%q) is not valid XML: %v", name, err)
		}
	}
}

func TestURLPath(t *testing.T) {
	t.Parallel()
	if got := URLPath("alice"); got != "/api/avatars/alice" {
		t.Errorf("URLPath = %q, want a same-origin API path", got)
	}
}

func xmlCheck(b []byte) (int, error) {
	d := xml.NewDecoder(strings.NewReader(string(b)))
	n := 0
	for {
		_, err := d.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return n, nil
			}
			return n, err
		}
		n++
	}
}
