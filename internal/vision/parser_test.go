package vision

import (
	"strings"
	"testing"
)

func TestParseAndValidate_ValidJSON(t *testing.T) {
	input := `{
		"title": "Chromatic Fracture",
		"medium": "Wax pigment on reclaimed cellulose matrix",
		"description": "An aggressive confrontation with domestic spatial constraints. The wax marks assert an unyielding material presence across the surface.",
		"theme": "The Crayon Period"
	}`

	res, err := ParseAndValidateAnalysisResult(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Title != "Chromatic Fracture" {
		t.Errorf("expected title 'Chromatic Fracture', got: %s", res.Title)
	}
	if res.Medium != "Wax pigment on reclaimed cellulose matrix" {
		t.Errorf("expected medium 'Wax pigment on reclaimed cellulose matrix', got: %s", res.Medium)
	}
	if res.Theme != "The Crayon Period" {
		t.Errorf("expected theme 'The Crayon Period', got: %s", res.Theme)
	}
}

func TestParseAndValidate_MarkdownFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "json code fence",
			input: "```json\n" +
				`{"title": "Void", "medium": "Ink", "description": "A study in absence.", "theme": "Monochrome Nihilism"}` +
				"\n```",
		},
		{
			name: "plain code fence",
			input: "```\n" +
				`{"title": "Void", "medium": "Ink", "description": "A study in absence.", "theme": "Monochrome Nihilism"}` +
				"\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseAndValidateAnalysisResult(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.Title != "Void" {
				t.Errorf("expected title 'Void', got: %s", res.Title)
			}
		})
	}
}

func TestParseAndValidate_ExtraneousConversationalText(t *testing.T) {
	input := `Greetings! Here is the curatorial analysis for the requested artwork:

	{
		"title": "Domestic Rupture",
		"medium": "Organic polymer on synthetic fiber",
		"description": "A visceral disruption of domestic equilibrium.",
		"theme": "Domestic Destructionism"
	}

	I hope this meets your curatorial standards!`

	res, err := ParseAndValidateAnalysisResult(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Title != "Domestic Rupture" {
		t.Errorf("expected title 'Domestic Rupture', got: %s", res.Title)
	}
}

func TestParseAndValidate_TrailingCommas(t *testing.T) {
	input := `{
		"title": "Entropic Spill",
		"medium": "Aqueous fluid on pulp",
		"description": "An entropic manifestation across the horizontal axis.",
		"theme": "Domestic Destructionism",
	}`

	res, err := ParseAndValidateAnalysisResult(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Title != "Entropic Spill" {
		t.Errorf("expected title 'Entropic Spill', got: %s", res.Title)
	}
}

func TestParseAndValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing title",
			input: `{"medium": "Ink", "description": "A work.", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "empty title",
			input: `{"title": "   ", "medium": "Ink", "description": "A work.", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "missing medium",
			input: `{"title": "Void", "description": "A work.", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "empty medium",
			input: `{"title": "Void", "medium": "\t", "description": "A work.", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "missing description",
			input: `{"title": "Void", "medium": "Ink", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "empty description",
			input: `{"title": "Void", "medium": "Ink", "description": "  ", "theme": "Monochrome Nihilism"}`,
		},
		{
			name:  "missing theme",
			input: `{"title": "Void", "medium": "Ink", "description": "A work."}`,
		},
		{
			name:  "empty theme",
			input: `{"title": "Void", "medium": "Ink", "description": "A work.", "theme": "\n"}`,
		},
		{
			name:  "invalid json",
			input: `not a json object at all`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAndValidateAnalysisResult(tt.input)
			if err == nil {
				t.Fatalf("expected error for missing/empty field, got nil")
			}
		})
	}
}

func TestValidateCuratorialRules_ForbiddenWords(t *testing.T) {
	validDesc := "This monumental composition embodies the austere dialectic of space and mark-making in contemporary practice. The gestural urgency establishes an unrelenting rhythm that interrogates the viewer's phenomenological expectations across the entire picture plane. Each intentional intervention transforms the everyday substrate into an arena of profound metaphysical inquiry and rigorous aesthetic contemplation."

	// Ensure base valid description passes
	validResult := &AnalysisResult{
		Title:       "Valid Title",
		Medium:      "Valid Medium",
		Description: validDesc,
		Theme:       ThemeTheCrayonPeriod,
	}
	if err := ValidateCuratorialRules(validResult); err != nil {
		t.Fatalf("expected valid result to pass curatorial rules, got: %v", err)
	}

	for _, word := range ForbiddenCuratorialWords {
		t.Run("forbidden_"+word, func(t *testing.T) {
			badDesc := "This " + word + " composition embodies the austere dialectic of space and mark-making in contemporary practice. The gestural urgency establishes an unrelenting rhythm that interrogates the viewer's phenomenological expectations. Each intentional intervention transforms the everyday substrate into an arena of metaphysical inquiry."
			res := &AnalysisResult{
				Title:       "Title",
				Medium:      "Medium",
				Description: badDesc,
				Theme:       ThemeTheCrayonPeriod,
			}
			err := ValidateCuratorialRules(res)
			if err == nil {
				t.Fatalf("expected error for forbidden word '%s', got nil", word)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(word)) {
				t.Errorf("expected error message to mention forbidden word '%s', got: %v", word, err)
			}
		})
	}
}

func TestValidateCuratorialRules_SentenceAndWordCount(t *testing.T) {
	t.Run("sentence count too low", func(t *testing.T) {
		res := &AnalysisResult{
			Title:  "Title",
			Medium: "Medium",
			// 1 sentence, 52 words
			Description: "This monumental single sentence contains enough vocabulary to easily exceed fifty words in total length while strictly refusing to incorporate any additional sentence-terminating punctuation marks thereby triggering the validation check for insufficient sentence count despite meeting the word volume requirements of the curatorial rule specification completely and thoroughly without any issue.",
			Theme:       ThemeTheCrayonPeriod,
		}
		if err := ValidateCuratorialRules(res); err == nil {
			t.Fatalf("expected error for sentence count < 2, got nil")
		}
	})

	t.Run("sentence count too high", func(t *testing.T) {
		res := &AnalysisResult{
			Title:  "Title",
			Medium: "Medium",
			// 5 sentences, ~55 words
			Description: "First sentence introduces the conceptual framework of the piece. Second sentence articulates the chromatic tension across the surface. Third sentence investigates the phenomenological resonance of each mark. Fourth sentence connects the material presence to historical movements. Fifth sentence concludes the analysis with an uncompromising assertion of metaphysical intent.",
			Theme:       ThemeTheCrayonPeriod,
		}
		if err := ValidateCuratorialRules(res); err == nil {
			t.Fatalf("expected error for sentence count > 4, got nil")
		}
	})

	t.Run("word count too low", func(t *testing.T) {
		res := &AnalysisResult{
			Title:       "Title",
			Medium:      "Medium",
			Description: "First sentence here. Second sentence here.",
			Theme:       ThemeTheCrayonPeriod,
		}
		if err := ValidateCuratorialRules(res); err == nil {
			t.Fatalf("expected error for word count < 50, got nil")
		}
	})

	t.Run("word count too high", func(t *testing.T) {
		words := make([]string, 130)
		for i := range words {
			words[i] = "monumental"
		}
		// 3 sentences, 130 words
		part1 := strings.Join(words[:40], " ") + "."
		part2 := strings.Join(words[40:80], " ") + "."
		part3 := strings.Join(words[80:], " ") + "."
		res := &AnalysisResult{
			Title:       "Title",
			Medium:      "Medium",
			Description: part1 + " " + part2 + " " + part3,
			Theme:       ThemeTheCrayonPeriod,
		}
		if err := ValidateCuratorialRules(res); err == nil {
			t.Fatalf("expected error for word count > 120, got nil")
		}
	})

	t.Run("valid sentence and word counts", func(t *testing.T) {
		for sentences := 2; sentences <= 4; sentences++ {
			// Construct description with ~60 words across `sentences` sentences
			parts := make([]string, sentences)
			wordsPerPart := 60 / sentences
			for i := 0; i < sentences; i++ {
				w := make([]string, wordsPerPart)
				for j := range w {
					w[j] = "compositional"
				}
				parts[i] = strings.ToUpper(string(w[0][0])) + w[0][1:] + " " + strings.Join(w[1:], " ") + "."
			}
			desc := strings.Join(parts, " ")
			res := &AnalysisResult{
				Title:       "Title",
				Medium:      "Medium",
				Description: desc,
				Theme:       ThemeTheCrayonPeriod,
			}
			if err := ValidateCuratorialRules(res); err != nil {
				t.Errorf("expected %d sentences to pass, got err: %v", sentences, err)
			}
		}
	})
}
