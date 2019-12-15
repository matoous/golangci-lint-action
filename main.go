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
	envRepo = "GITHUB_REPOSITORY"
	envSHA  = "GITHUB_SHA"
	//nolint:gosec
	envToken = "GITHUB_TOKEN"
)

var (
	ghToken   string
	repoOwner string
	repoName  string
	headSHA   string
)

const defaultRequestTimeout = 30 * time.Second

const maxFailureCount = 50

var client *github.Client

func loadConfig() error {
	if env := os.Getenv(envToken); env != "" {
		ghToken = env
	} else {
		return fmt.Errorf("missing environment variable: %s", envToken)
	}

	if env := os.Getenv(envRepo); env != "" {
		s := strings.SplitN(env, "/", 2)
		repoOwner, repoName = s[0], s[1]
	} else {
		return fmt.Errorf("missing environment variable: %s", envRepo)
	}

	if env := os.Getenv(envSHA); env != "" {
		headSHA = env
	} else {
		return fmt.Errorf("missing environment variable: %s", envSHA)
	}

	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	))

	client = github.NewClient(tc)
	return nil
}

func createCheck() (*github.CheckRun, error) {
	opts := github.CreateCheckRunOptions{
		Name:    name,
		HeadSHA: headSHA,
		Status:  github.String("in_progress"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	check, _, err := client.Checks.CreateCheckRun(ctx, repoOwner, repoName, opts)
	if err != nil {
		return nil, fmt.Errorf("error while creating check-run: %s", err)
	}

	return check, nil
}

type conclusion int

const (
	conclSuccess conclusion = iota
	conclFailure
)

func (c conclusion) String() string {
	return [...]string{"success", "failure"}[c]
}

func completeCheck(check *github.CheckRun, concl conclusion, errCount int) error {
	opts := github.UpdateCheckRunOptions{
		Name:       name,
		HeadSHA:    github.String(headSHA),
		Conclusion: github.String(concl.String()),
		Output: &github.CheckRunOutput{
			Title:   github.String("Result"),
			Summary: github.String(fmt.Sprintf("%d errors", errCount)),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	if _, _, err := client.Checks.UpdateCheckRun(ctx, repoOwner, repoName, check.GetID(), opts); err != nil {
		return fmt.Errorf("error while completing check-run: %s", err)
	}
	return nil
}

// Report contains the data returned by golangci lint parsed from json
type Report struct {
	Issues []result.Issue `json:"Issues"`
}

func createAnnotations(issues []result.Issue) []*github.CheckRunAnnotation {
	ann := make([]*github.CheckRunAnnotation, len(issues))
	for i := range issues {
		r := issues[i].GetLineRange()
		ann[i] = &github.CheckRunAnnotation{
			Path:            github.String(issues[i].Pos.Filename),
			StartLine:       github.Int(r.From),
			EndLine:         github.Int(r.To),
			AnnotationLevel: github.String("failure"),
			Title:           github.String(issues[i].FromLinter),
			Message:         github.String(issues[i].Text),
		}
	}

	return ann
}

func pushFailures(check *github.CheckRun, failures []result.Issue) error {
	failuresCount := len(failures)
	if failuresCount > maxFailureCount {
		failures = failures[:maxFailureCount]
	}
	opts := github.UpdateCheckRunOptions{
		Name:    name,
		HeadSHA: github.String(headSHA),
		Output: &github.CheckRunOutput{
			Title:       github.String("Result"),
			Summary:     github.String(fmt.Sprintf("%d errors", failuresCount)),
			Annotations: createAnnotations(failures),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	if _, _, err := client.Checks.UpdateCheckRun(ctx, repoOwner, repoName, check.GetID(), opts); err != nil {
		return fmt.Errorf("error while updating check-run: %s", err)
	}
	return nil
}

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}

	concl := conclSuccess
	check, err := createCheck()
	if err != nil {
		panic(err)
	}

	var report Report
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&report); err != nil {
		panic(err)
	}

	if len(report.Issues) > 0 {
		concl = conclFailure
		if err := pushFailures(check, report.Issues); err != nil {
			panic(err)
		}
	}

	if err := completeCheck(check, concl, len(report.Issues)); err != nil {
		panic(err)
	}

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
