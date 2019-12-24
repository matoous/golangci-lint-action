package main

import (
	"encoding/json"
	"path/filepath"
	"fmt"
	"os"

	"github.com/golangci/golangci-lint/pkg/result"
)

const name = "GolangCI-Lint Action"

const (
	// envGithubWorkspace = "GITHUB_WORKSPACE"
	envBasePath = "INPUT_BASEPATH"
)
// Report contains the data returned by golangci lint parsed from json
type Report struct {
	Issues []result.Issue `json:"Issues"`
}

type annotation struct {
	file string
	line int
	col  int
	text string
}

func (a annotation) Output() string {
	return fmt.Sprintf("::error file=%s,line=%d,col=%d::%s\n", a.file, a.line, a.col, a.text)
}

type config struct {
	workspace string
	basePath string
}

func loadConfig() config {
	return config{
		//workspace: os.Getenv(envGithubWorkspace),
		basePath: os.Getenv(envBasePath),
	}
}

func createAnotations(cfg config, issues []result.Issue) []annotation {
	ann := make([]annotation, len(issues))
	for i, issue := range issues {
		pos := issue.Pos
		file := filepath.Join(cfg.basePath, pos.Filename)
		ann[i] = annotation{
			file: file,
			line: pos.Line,
			col:  pos.Column,
			text: fmt.Sprintf("%s - %s", issue.FromLinter, issue.Text),
		}
	}
	return ann
}

func pushFailures(cfg config, failures []result.Issue) {
	anns := createAnotations(cfg, failures)
	for _, ann := range anns {
		fmt.Println(ann.Output())
	}
}

func main() {
	cfg := loadConfig()

	var report Report
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&report); err != nil {
		panic(err)
	}

	if len(report.Issues) > 0 {
		pushFailures(cfg, report.Issues)
		os.Exit(1)
	}

	os.Exit(0)
}
