//go:build ignore

// Command contentexport writes the portfolio content to JSON for the static build.
//
// pkg/api holds the single authored copy of the profile, the six projects, and
// the four skill categories. The prerendered public pages need that content as
// data at build time, and they must never fetch it from a live API: a build that
// required a running server would break the project invariant that static assets
// render independently of backend availability.
//
// This command is the bridge. It marshals exactly the content the public API
// serves, so the prerendered pages and the runtime endpoints cannot drift apart.
//
// Regenerate the frontend content file with:
//
//	go run tools/contentexport/main.go > frontend/src/content/portfolio.json
//
// The output is committed. It is generated, never hand-edited: edit the Go
// content and re-run the command above.
//
// Stdlib only, no dependencies.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"portfolio/pkg/api"
)

func main() {
	enc := json.NewEncoder(os.Stdout)
	// Portfolio prose contains ampersands and dashes; escaping them would make
	// the generated file harder to diff for no benefit, since it is written to
	// disk rather than interpolated into HTML.
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	if err := enc.Encode(api.PortfolioContent()); err != nil {
		fmt.Fprintf(os.Stderr, "contentexport: %v\n", err)
		os.Exit(1)
	}
}
