package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func updateCommitLog(info *CommitInfo) error {
	dir := "../../app/DOCS/commit-log"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.md", info.Date))
	
	entry := fmt.Sprintf("- `%s` | %s | **%s** | %s | [Link](https://github.com/frag2win/TelemetryHealth/commit/%s)\n", 
		info.SHA[:7], info.Author, info.Prefix, info.Description, info.SHA)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

func updateBuildIssueReport(info *CommitInfo) error {
	filePath := "../../app/DOCS/Build_Issue_Report.md"
	
	entry := fmt.Sprintf("- [%s](https://github.com/frag2win/TelemetryHealth/commit/%s) | **%s**: %s\n", 
		info.Date, info.SHA, info.Author, info.Description)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}
