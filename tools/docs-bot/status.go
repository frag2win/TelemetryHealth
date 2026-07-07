package main

import (
	"fmt"
	"os"
	"strings"
)

func updateImplementationStatus(info *CommitInfo, prdSection string) error {
	filePath := "../../app/DOCS/Implementation_Status.md"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	// Table format: | Feature | PRD Section | Status | Milestone | Commit/Link |
	for i, line := range lines {
		if strings.Contains(line, "|") && strings.Contains(line, prdSection) {
			parts := strings.Split(line, "|")
			if len(parts) >= 6 {
				// Trim spaces to check accurately
				for j := range parts {
					parts[j] = strings.TrimSpace(parts[j])
				}
				
				// parts[0] is empty before first |
				// parts[1] is Feature
				// parts[2] is PRD Section
				// parts[3] is Status
				// parts[4] is Milestone
				// parts[5] is Commit/Link
				// parts[6] is empty after last |
				
				if parts[2] == prdSection {
					parts[3] = "Complete"
					parts[5] = fmt.Sprintf("[%s](https://github.com/frag2win/TelemetryHealth/commit/%s)", info.SHA[:7], info.SHA)
					
					// Rebuild the line
					lines[i] = fmt.Sprintf("| %s | %s | %s | %s | %s |", parts[1], parts[2], parts[3], parts[4], parts[5])
					updated = true
					break
				}
			}
		}
	}

	if !updated {
		return fmt.Errorf("could not find PRD section %s in Implementation_Status.md", prdSection)
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}
