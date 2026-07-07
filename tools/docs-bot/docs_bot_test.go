package main

import (
	"testing"
)

func TestExtractPRDSection(t *testing.T) {
	body := `This is a commit body.
It fixes a lot of things.
Closes-PRD-Section: §8.2
And some other text.`

	res := extractPRDSection(body)
	if res != "§8.2" {
		t.Errorf("Expected §8.2, got %s", res)
	}

	body2 := `Closes-PRD-Section: §12`
	res2 := extractPRDSection(body2)
	if res2 != "§12" {
		t.Errorf("Expected §12, got %s", res2)
	}
}

func TestPrefixRegex(t *testing.T) {
	tests := []struct {
		subject string
		prefix  string
		desc    string
	}{
		{"FEATURE: add new dashboard", "FEATURE", "add new dashboard"},
		{"BUG: fix nil pointer", "BUG", "fix nil pointer"},
		{"UI: update colors", "UI", "update colors"},
		{"DOCS: update readme", "DOCS", "update readme"},
		{"CHORE: bump deps", "CHORE", "bump deps"},
		{"INVALID format here", "", ""},
	}

	for _, tc := range tests {
		matches := prefixRegex.FindStringSubmatch(tc.subject)
		if tc.prefix == "" {
			if len(matches) > 0 {
				t.Errorf("Expected no match for %q, got %v", tc.subject, matches)
			}
		} else {
			if len(matches) < 3 {
				t.Errorf("Expected match for %q, got none", tc.subject)
			} else if matches[1] != tc.prefix || matches[2] != tc.desc {
				t.Errorf("For %q, expected prefix %q desc %q, got %q %q", tc.subject, tc.prefix, tc.desc, matches[1], matches[2])
			}
		}
	}
}
