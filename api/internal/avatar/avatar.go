// Package avatar draws the default identicon for a user. It runs entirely on our side, so a
// visitor's browser never contacts a third party, and the same username always gets the same
// picture.
package avatar

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// palette holds the accent colours of the site's design system (see index.css).
var palette = []string{"#ccff00", "#ff3355", "#00e5ff", "#39ff88", "#a855f7", "#ff8a1f", "#ff5ecf", "#ffe14d"}

// URLPath is the same-origin address that serves a user's default avatar.
func URLPath(username string) string {
	return "/api/avatars/" + username
}

// SVG returns a 5x5 symmetric identicon derived from the SHA-256 of the username. It is small, has
// no external references and is valid XML.
func SVG(username string) []byte {
	sum := sha256.Sum256([]byte(username))
	color := palette[int(sum[0])%len(palette)]

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 5 5" width="96" height="96" shape-rendering="crispEdges">`)
	b.WriteString(`<rect width="5" height="5" fill="#0a0a0f"/>`)
	fmt.Fprintf(&b, `<g fill="%s">`, color)
	// Columns 0-2 come from the hash bits; columns 3-4 mirror 1-0 so every face is symmetric.
	for row := range 5 {
		for col := range 3 {
			bit := row*3 + col
			if sum[1+bit/8]>>(bit%8)&1 == 0 {
				continue
			}
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1"/>`, col, row)
			if mirror := 4 - col; mirror != col {
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1"/>`, mirror, row)
			}
		}
	}
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}
