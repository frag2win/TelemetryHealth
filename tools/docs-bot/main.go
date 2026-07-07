package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	prefixRegex = regexp.MustCompile(`^(FEATURE|BUG|UI|PERF|SEC|DOCS|REFACTOR|TEST|CHORE):\s*(.*)`)
)

type CommitInfo struct {
	SHA         string
	Author      string
	Subject     string
	Date        string // YYYY-MM-DD
	Body        string
	Prefix      string
	Description string
}

func main() {
	shaPtr := flag.String("sha", "", "Commit SHA to parse")
	flag.Parse()

	if *shaPtr == "" {
		log.Fatal("Must provide --sha flag")
	}
	sha := *shaPtr

	info, err := getCommitInfo(sha)
	if err != nil {
		log.Fatalf("failed to get commit info: %v", err)
	}

	matches := prefixRegex.FindStringSubmatch(info.Subject)
	if len(matches) < 3 {
		log.Printf("Commit %s does not match required prefix format. Skipping docs update.", sha)
		os.Exit(0)
	}
	info.Prefix = matches[1]
	info.Description = strings.TrimSpace(matches[2])

	// 1. Update CHANGELOG.md
	if err := updateChangelog(info); err != nil {
		log.Printf("Failed to update CHANGELOG.md: %v", err)
	}

	// 2. Append to commit-log/YYYY-MM-DD.md
	if err := updateCommitLog(info); err != nil {
		log.Printf("Failed to update commit log: %v", err)
	}

	// 3. Update Build_Issue_Report.md if BUG touching infra
	if info.Prefix == "BUG" {
		infraTouched, err := touchedInfraFiles(info.SHA)
		if err != nil {
			log.Printf("Failed to check infra files: %v", err)
		} else if infraTouched {
			if err := updateBuildIssueReport(info); err != nil {
				log.Printf("Failed to update build issue report: %v", err)
			}
		}
	}

	// 4. Update Implementation_Status.md if Closes-PRD-Section is present
	if prdSec := extractPRDSection(info.Body); prdSec != "" {
		if err := updateImplementationStatus(info, prdSec); err != nil {
			log.Printf("Failed to update implementation status: %v", err)
		}
	}

	fmt.Println("Docs-bot completed successfully.")
}

func getCommitInfo(sha string) (*CommitInfo, error) {
	// Format: %H (SHA) | %an (Author) | %s (Subject) | %cs (Date: YYYY-MM-DD)
	cmd := exec.Command("git", "show", "-s", "--format=%H|%an|%s|%cs", sha)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show format error: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected git show output: %s", string(out))
	}

	cmdBody := exec.Command("git", "show", "-s", "--format=%b", sha)
	bodyOut, err := cmdBody.Output()
	if err != nil {
		return nil, fmt.Errorf("git show body error: %w", err)
	}

	return &CommitInfo{
		SHA:     parts[0],
		Author:  parts[1],
		Subject: parts[2],
		Date:    parts[3],
		Body:    strings.TrimSpace(string(bodyOut)),
	}, nil
}

func touchedInfraFiles(sha string) (bool, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	
	files := strings.Split(string(out), "\n")
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if strings.Contains(f, "Dockerfile") || 
		   strings.Contains(f, "go.mod") || 
		   strings.Contains(f, "go.sum") || 
		   strings.HasPrefix(f, ".github/") || 
		   strings.HasPrefix(f, "deployments/helm/") || 
		   strings.HasPrefix(f, "deployments/terraform/") {
			return true, nil
		}
	}
	return false, nil
}

func extractPRDSection(body string) string {
	re := regexp.MustCompile(`Closes-PRD-Section:\s*(§\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(body)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
