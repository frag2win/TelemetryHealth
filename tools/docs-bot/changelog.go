package main

import (
	"fmt"
	"os"
	"strings"
)

func updateChangelog(info *CommitInfo) error {
	filePath := "../../app/DOCS/CHANGELOG.md"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	
	targetHeader := ""
	switch info.Prefix {
	case "FEATURE":
		targetHeader = "### Added"
	case "BUG", "UI":
		targetHeader = "### Fixed"
	case "REFACTOR", "PERF", "SEC", "DOCS":
		targetHeader = "### Changed"
	case "TEST", "CHORE":
		targetHeader = "### Internal"
	default:
		return fmt.Errorf("unknown prefix %s", info.Prefix)
	}

	entry := fmt.Sprintf("- %s ([%s](https://github.com/frag2win/TelemetryHealth/commit/%s))", info.Description, info.SHA[:7], info.SHA)

	var newLines []string
	inserted := false
	for _, line := range lines {
		newLines = append(newLines, line)
		if !inserted && strings.HasPrefix(line, targetHeader) {
			// Insert a blank line and the new entry
			newLines = append(newLines, "")
			newLines = append(newLines, entry)
			inserted = true
		}
	}

	if !inserted {
		return fmt.Errorf("could not find header %s in CHANGELOG.md", targetHeader)
	}

	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
}
