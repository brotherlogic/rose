package vision

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	trailingCommaRegex = regexp.MustCompile(`,(\s*[}\]])`)
	sentenceEndRegex   = regexp.MustCompile(`[.!?]+(?:\s+|$)`)
)

// ParseAndValidateAnalysisResult parses the multimodal model response into an AnalysisResult.
// It applies resilient parsing including markdown code fence stripping, substring JSON extraction,
// trailing comma repair, and required field validation.
func ParseAndValidateAnalysisResult(content string) (*AnalysisResult, error) {
	content = strings.TrimSpace(content)

	// Markdown fence stripping: Strip ```json ... ``` and ``` ... ``` enclosing fences
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.Index(content, "\n"); idx != -1 {
			firstLine := strings.TrimSpace(content[:idx])
			if firstLine == "json" || !strings.Contains(firstLine, "{") {
				content = content[idx+1:]
			}
		}
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	// Substring extraction: Locate the outermost {...} substring to isolate the JSON payload
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || start >= end {
		return nil, errors.New("no valid json object found in content")
	}
	jsonStr := content[start : end+1]

	// Syntax repair: Automatically remove trailing commas before closing braces or brackets
	jsonStr = trailingCommaRegex.ReplaceAllString(jsonStr, "$1")

	// JSON unmarshaling into AnalysisResult
	var res AnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis result json: %w", err)
	}

	// Required field validation: Assert Title, Medium, Description, and Theme are non-empty after trimming
	if strings.TrimSpace(res.Title) == "" {
		return nil, errors.New("analysis result missing required field: title")
	}
	if strings.TrimSpace(res.Medium) == "" {
		return nil, errors.New("analysis result missing required field: medium")
	}
	if strings.TrimSpace(res.Description) == "" {
		return nil, errors.New("analysis result missing required field: description")
	}
	if strings.TrimSpace(res.Theme) == "" {
		return nil, errors.New("analysis result missing required field: theme")
	}

	return &res, nil
}

// ValidateCuratorialRules validates the analysis result against curatorial persona constraints:
// 1. Prohibited vocabulary check: res.Description must not contain any forbidden terms.
// 2. Sentence count validation: res.Description must contain between 2 and 4 sentences.
// 3. Word count validation: res.Description must contain between 50 and 120 words.
func ValidateCuratorialRules(res *AnalysisResult) error {
	if res == nil {
		return errors.New("analysis result is nil")
	}

	desc := strings.TrimSpace(res.Description)
	if desc == "" {
		return errors.New("analysis result description is empty")
	}

	// 1. Prohibited vocabulary check
	for _, word := range ForbiddenCuratorialWords {
		pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(word))
		matched, err := regexp.MatchString(pattern, desc)
		if err != nil {
			return fmt.Errorf("failed to evaluate prohibited vocabulary regex for %q: %w", word, err)
		}
		if matched {
			return fmt.Errorf("curatorial description violates prohibited vocabulary rule: contains forbidden word %q", word)
		}
	}

	// 2. Word count validation: 50 to 120 words
	words := strings.Fields(desc)
	wordCount := len(words)
	if wordCount < 50 || wordCount > 120 {
		return fmt.Errorf("curatorial description word count %d outside allowed range [50, 120]", wordCount)
	}

	// 3. Sentence count validation: 2 to 4 sentences
	sentenceCount := countSentences(desc)
	if sentenceCount < 2 || sentenceCount > 4 {
		return fmt.Errorf("curatorial description sentence count %d outside allowed range [2, 4]", sentenceCount)
	}

	return nil
}

func countSentences(text string) int {
	tokens := sentenceEndRegex.Split(strings.TrimSpace(text), -1)
	count := 0
	for _, t := range tokens {
		if strings.TrimSpace(t) != "" {
			count++
		}
	}
	return count
}
