package vision

import (
	"strings"
	"testing"
)

func TestPrompt_ExportedConstants(t *testing.T) {
	expectedWords := []string{
		"child", "children", "cute", "toddler", "kid", "kids", "drawing", "drawings", "mess", "scribble",
	}

	if len(ForbiddenCuratorialWords) != len(expectedWords) {
		t.Fatalf("expected %d forbidden words, got %d", len(expectedWords), len(ForbiddenCuratorialWords))
	}

	wordMap := make(map[string]bool)
	for _, w := range ForbiddenCuratorialWords {
		wordMap[w] = true
	}
	for _, expected := range expectedWords {
		if !wordMap[expected] {
			t.Errorf("missing expected forbidden word: %q", expected)
		}
	}

	expectedThemes := []string{
		"The Crayon Period",
		"Domestic Destructionism",
		"Kinetic Scribblism",
		"Found Object Assemblage",
		"Monochrome Nihilism",
	}

	if len(CanonicalThemes) != len(expectedThemes) {
		t.Fatalf("expected %d canonical themes, got %d", len(expectedThemes), len(CanonicalThemes))
	}

	themeMap := make(map[string]bool)
	for _, th := range CanonicalThemes {
		themeMap[th] = true
	}
	for _, expected := range expectedThemes {
		if !themeMap[expected] {
			t.Errorf("missing expected canonical theme: %q", expected)
		}
	}

	if ThemeTheCrayonPeriod != "The Crayon Period" {
		t.Errorf("unexpected ThemeTheCrayonPeriod: %s", ThemeTheCrayonPeriod)
	}
	if ThemeDomesticDestructionism != "Domestic Destructionism" {
		t.Errorf("unexpected ThemeDomesticDestructionism: %s", ThemeDomesticDestructionism)
	}
	if ThemeKineticScribblism != "Kinetic Scribblism" {
		t.Errorf("unexpected ThemeKineticScribblism: %s", ThemeKineticScribblism)
	}
	if ThemeFoundObjectAssemblage != "Found Object Assemblage" {
		t.Errorf("unexpected ThemeFoundObjectAssemblage: %s", ThemeFoundObjectAssemblage)
	}
	if ThemeMonochromeNihilism != "Monochrome Nihilism" {
		t.Errorf("unexpected ThemeMonochromeNihilism: %s", ThemeMonochromeNihilism)
	}
}

func TestPrompt_CuratorialSystemPromptContent(t *testing.T) {
	if CuratorialSystemPrompt == "" {
		t.Fatal("CuratorialSystemPrompt is empty")
	}

	// 1. JSON Schema instructions
	for _, key := range []string{`"title"`, `"medium"`, `"description"`, `"theme"`} {
		if !strings.Contains(CuratorialSystemPrompt, key) {
			t.Errorf("CuratorialSystemPrompt missing JSON schema key directive: %s", key)
		}
	}

	// 2. Persona and gravity directives
	personaKeywords := []string{
		"curator",
		"philosophical gravity",
	}
	for _, kw := range personaKeywords {
		if !strings.Contains(strings.ToLower(CuratorialSystemPrompt), strings.ToLower(kw)) {
			t.Errorf("CuratorialSystemPrompt missing persona keyword: %s", kw)
		}
	}

	// 3. Negative constraints and prohibited vocabulary
	for _, forbidden := range ForbiddenCuratorialWords {
		if !strings.Contains(strings.ToLower(CuratorialSystemPrompt), forbidden) {
			t.Errorf("CuratorialSystemPrompt missing forbidden word instruction for: %s", forbidden)
		}
	}

	// 4. Length constraints
	if !strings.Contains(CuratorialSystemPrompt, "2–4") && !strings.Contains(CuratorialSystemPrompt, "2-4") {
		t.Errorf("CuratorialSystemPrompt should specify 2-4 sentences constraint")
	}
	if !strings.Contains(CuratorialSystemPrompt, "50–120") && !strings.Contains(CuratorialSystemPrompt, "50-120") {
		t.Errorf("CuratorialSystemPrompt should specify 50-120 words constraint")
	}

	// 5. Material transformation rules
	materialTerms := []string{
		"Wax pigment",
		"Organic polymer paste",
		"Aqueous fluid",
		"Solvent-based dye transfer",
	}
	for _, term := range materialTerms {
		if !strings.Contains(CuratorialSystemPrompt, term) {
			t.Errorf("CuratorialSystemPrompt missing material transformation guideline: %s", term)
		}
	}

	// 6. Extensible taxonomy and degenerate image handling
	for _, theme := range CanonicalThemes {
		if !strings.Contains(CuratorialSystemPrompt, theme) {
			t.Errorf("CuratorialSystemPrompt missing canonical theme: %s", theme)
		}
	}

	if !strings.Contains(strings.ToLower(CuratorialSystemPrompt), "title case") {
		t.Errorf("CuratorialSystemPrompt should mention Title Case for extensible themes")
	}
	if !strings.Contains(CuratorialSystemPrompt, "Monochrome Nihilism") {
		t.Errorf("CuratorialSystemPrompt should direct degenerate/void images to Monochrome Nihilism")
	}
}
