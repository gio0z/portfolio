//go:build ignore

// Command statsgen regenerates pkg/api/stats_generated.go from real evidence.
//
// It scans a projects root directory (default $HOME/projects, override with
// -root) and reports only numbers it actually counted:
//
//   - Production systems: tracked engagements present as a git repository.
//   - Commits on client work: the sum of `git rev-list --count HEAD` over
//     those repositories.
//   - Engineering since: the year of the earliest root commit found anywhere.
//   - Clients served: distinct clients in the engagement registry below that
//     still exist on disk. A client counts once no matter how many
//     repositories their product was split across (web plus Android), and a
//     client still counts while their repository is temporarily unavailable,
//     because the engagement is a fact independent of this machine's checkout.
//
// Nothing is estimated, averaged, or inferred. An engagement directory that is
// missing, or present without readable git history, is reported on stderr and
// excluded from the commit and system totals rather than guessed at.
//
// Regenerate the stats file with:
//
//	go run tools/statsgen/main.go > pkg/api/stats_generated.go
//
// Stdlib only, no dependencies.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// engagementClient maps a directory under the projects root to the client that
// engagement was delivered for. Two directories may share one client (one
// product shipped as a web app plus an Android app), and clients are counted
// once. Every client here has been delivered to; entries are never added
// speculatively.
var engagementClient = map[string]string{
	"jam-nguar":          "Pemerintah Kabupaten Blitar",
	"inven-kab-blitar":   "Pemerintah Kabupaten Blitar",
	"ba-rekon":           "BA-Rekon",
	"labtu-web":          "Labtu",
	"labtu-android":      "Labtu",
	"cs-portal":          "CS Portal",
	"mdfashion":          "MD Fashion",
	"nusantara-botanica": "Plantea",
	"tour-travel-web":    "Afsa Tour & Transport",
	"pamonta":            "Pamonta",
}

// repoStat is one engagement with countable history.
type repoStat struct {
	client  string
	commits int
	first   time.Time
}

// scanResult separates what was counted from what was skipped and why.
type scanResult struct {
	repos   []repoStat // engagements backed by a readable git repository
	clients []string   // distinct clients with a directory present on disk
	absent  []string   // engagement directories not present at all
	noGit   []string   // engagement directories present without readable history
}

func main() {
	defaultRoot := filepath.Join(os.Getenv("HOME"), "projects")
	root := flag.String("root", defaultRoot, "directory containing the project repositories")
	flag.Parse()

	res, err := scan(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "statsgen: %v\n", err)
		os.Exit(1)
	}
	for _, name := range res.absent {
		fmt.Fprintf(os.Stderr, "statsgen: skipping %s: directory not found\n", name)
	}
	for _, name := range res.noGit {
		fmt.Fprintf(os.Stderr, "statsgen: %s has no readable git history; counted as a client, excluded from commit and system totals\n", name)
	}
	if len(res.repos) == 0 || len(res.clients) == 0 {
		fmt.Fprintf(os.Stderr, "statsgen: no tracked engagements found under %s\n", *root)
		os.Exit(1)
	}

	commits := 0
	earliest := time.Time{}
	for _, r := range res.repos {
		commits += r.commits
		if earliest.IsZero() || r.first.Before(earliest) {
			earliest = r.first
		}
	}

	fmt.Print(render(len(res.repos), commits, len(res.clients), earliest.Year()))
}

// scan counts every known engagement under root.
func scan(root string) (scanResult, error) {
	fi, err := os.Stat(root)
	if err != nil {
		return scanResult{}, fmt.Errorf("reading projects root: %w", err)
	}
	if !fi.IsDir() {
		return scanResult{}, fmt.Errorf("projects root %s is not a directory", root)
	}

	names := make([]string, 0, len(engagementClient))
	for name := range engagementClient {
		names = append(names, name)
	}
	sort.Strings(names)

	var res scanResult
	seenClients := make(map[string]bool, len(names))
	for _, name := range names {
		dir := filepath.Join(root, name)
		if !isDir(dir) {
			res.absent = append(res.absent, name)
			continue
		}
		seenClients[engagementClient[name]] = true
		if !isDir(filepath.Join(dir, ".git")) {
			res.noGit = append(res.noGit, name)
			continue
		}
		commits, err := commitCount(dir)
		if err != nil || commits == 0 {
			res.noGit = append(res.noGit, name)
			continue
		}
		first, err := firstCommit(dir)
		if err != nil {
			res.noGit = append(res.noGit, name)
			continue
		}
		res.repos = append(res.repos, repoStat{client: engagementClient[name], commits: commits, first: first})
	}

	clients := make([]string, 0, len(seenClients))
	for client := range seenClients {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	res.clients = clients
	return res, nil
}

// commitCount asks git for the number of commits reachable from HEAD.
func commitCount(dir string) (int, error) {
	out, err := git(dir, "rev-list", "--count", "HEAD")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// firstCommit resolves the root commit and returns its committer date, which
// is the moment the engagement started.
func firstCommit(dir string) (time.Time, error) {
	roots, err := git(dir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return time.Time{}, err
	}
	shas := strings.Fields(roots)
	if len(shas) == 0 {
		return time.Time{}, fmt.Errorf("no root commit in %s", dir)
	}
	// A repository may have several root commits; the oldest one dates it.
	oldest := time.Time{}
	for _, sha := range shas {
		out, err := git(dir, "show", "-s", "--format=%cI", sha)
		if err != nil {
			return time.Time{}, err
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(out))
		if err != nil {
			return time.Time{}, err
		}
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
	}
	return oldest, nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return string(out), nil
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// generatedTemplate is the exact source of pkg/api/stats_generated.go. Keeping
// the whole file in one template means the generated output can be diffed
// against the checked-in file verbatim.
const generatedTemplate = `// Code generated by tools/statsgen/main.go. DO NOT EDIT BY HAND.
//
// Regenerate from real repository history with:
//
//	go run tools/statsgen/main.go > pkg/api/stats_generated.go
//
// Every number below is counted from repositories on disk: production systems
// and commit totals come from ` + "`git rev-list --count HEAD`" + ` per tracked
// repository, and the engineering start year comes from the earliest commit
// date found. Edit tools/statsgen/main.go rather than this file.

package api

// GeneratedStats backs the statistics block on the public profile endpoint.
var GeneratedStats = []StatMetric{
	{Label: "Production systems", Value: %q, Sub: "Government, retail, travel"},
	{Label: "Commits on client work", Value: %q, Sub: "Across %d repositories"},
	{Label: "Engineering since", Value: %q, Sub: "Current practice"},
	{Label: "Clients served", Value: %q, Sub: "Blitar region and beyond"},
}
`

// render formats the generated Go source, and is the single source of truth
// for the shape of pkg/api/stats_generated.go.
func render(systems, commits, clients, since int) string {
	return fmt.Sprintf(generatedTemplate,
		strconv.Itoa(systems),
		strconv.Itoa(commits), systems,
		strconv.Itoa(since),
		strconv.Itoa(clients),
	)
}
