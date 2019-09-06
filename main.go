package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golangci/golangci-lint/pkg/result"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

const name = "GolangCI-Lint Action"

const (
	envRepo  = "GITHUB_REPOSITORY"
	envSHA   = "GITHUB_SHA"
	envToken = "GITHUB_TOKEN"
)

var (
	ghToken   string
	repoOwner string
	repoName  string
	headSHA   string
)

var client *github.Client

func init() {
	if env := os.Getenv(envToken); env != "" {
		ghToken = env
	} else {
		fmt.Fprintln(os.Stderr, "Missing environment variable:", envToken)
		os.Exit(2)
	}

	if env := os.Getenv(envRepo); env != "" {
		s := strings.SplitN(env, "/", 2)
		repoOwner, repoName = s[0], s[1]
	} else {
		fmt.Fprintln(os.Stderr, "Missing environment variable:", envRepo)
		os.Exit(2)
	}

	if env := os.Getenv(envSHA); env != "" {
		headSHA = env
	} else {
		fmt.Fprintln(os.Stderr, "Missing environment variable:", envSHA)
		os.Exit(2)
	}

	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	))

	client = github.NewClient(tc)
}

func createCheck() *github.CheckRun {
	opts := github.CreateCheckRunOptions{
		Name:    name,
		HeadSHA: headSHA,
		Status:  github.String("in_progress"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	check, _, err := client.Checks.CreateCheckRun(ctx, repoOwner, repoName, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while creating check-run:", err)
		os.Exit(1)
	}

	return check
}

type conclusion int

const (
	conclSuccess conclusion = iota
	conclFailure
)

func (c conclusion) String() string {
	return [...]string{"success", "failure"}[c]
}

func completeCheck(check *github.CheckRun, concl conclusion, errCount int) {
	opts := github.UpdateCheckRunOptions{
		Name:       name,
		HeadSHA:    github.String(headSHA),
		Conclusion: github.String(concl.String()),
		Output: &github.CheckRunOutput{
			Title:   github.String("Result"),
			Summary: github.String(fmt.Sprintf("%d errors", errCount)),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, _, err := client.Checks.UpdateCheckRun(
		ctx, repoOwner, repoName, check.GetID(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error while completing check-run:", err)
		os.Exit(1)
	}
}

// Report contains the data returned by golangci lint parsed from json
type Report struct {
	Issues []result.Issue `json:"Issues"`
}

func createAnnotations(issues []result.Issue) []*github.CheckRunAnnotation {
	ann := make([]*github.CheckRunAnnotation, len(issues))
	for i, f := range issues {
		var level string
		level = "failure"
		r := f.GetLineRange()
		ann[i] = &github.CheckRunAnnotation{
			Path:            github.String(f.Pos.Filename),
			StartLine:       github.Int(r.From),
			EndLine:         github.Int(r.To),
			AnnotationLevel: github.String(level),
			Title: github.String(
				fmt.Sprintf("%s", f.FromLinter),
			),
			Message: github.String(f.Text),
		}
	}

	return ann
}

func pushFailures(check *github.CheckRun, failures []result.Issue) {
	opts := github.UpdateCheckRunOptions{
		Name:    name,
		HeadSHA: github.String(headSHA),
		Output: &github.CheckRunOutput{
			Title:       github.String("Result"),
			Summary:     github.String(fmt.Sprintf("%d errors", len(failures))),
			Annotations: createAnnotations(failures),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, _, err := client.Checks.UpdateCheckRun(
		ctx, repoOwner, repoName, check.GetID(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error while updating check-run:", err)
		os.Exit(1)
	}
}

func main() {
	concl := conclSuccess

	check := createCheck()

	var report Report
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&report); err != nil {
		panic(err)
	}

	if len(report.Issues) > 0 {
		concl = conclFailure
		pushFailures(check, report.Issues)
	}
	completeCheck(check, concl, len(report.Issues))

	if concl == conclSuccess {
		fmt.Println("Successful run")
	} else {
		fmt.Printf("Failed run with %d errors\n", len(report.Issues))
	}

	// Always exit with 0, zero means that the Linter run successfully and created separate check,
	// the separate check must exit with 0 if any errors are found. If we return status 1 here
	// this action will fail and as will the GolangCI-Lint check. E.g. you will get 2 failed actions
	// caused by one problem.
	os.Exit(0)
}
