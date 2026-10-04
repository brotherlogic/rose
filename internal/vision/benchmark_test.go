package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// canonicalArchetypeCase represents a benchmark visual archetype test specification.
type canonicalArchetypeCase struct {
	Name             string
	Archetype        string
	SyntheticPayload []byte
	ExpectedTheme    string
	RawModelResponse string
}

// colloquialMediumTerms defines everyday, non-elevated colloquial materials
// that should never appear in elevated gallery-grade medium descriptions.
var colloquialMediumTerms = []string{
	"crayon", "crayons",
	"marker", "markers",
	"playdough", "play-doh",
	"slime",
	"spill", "spills",
	"pencil", "pencils",
	"carpet",
	"cardboard",
	"paper",
	"child", "toddler", "kid",
}

// 1x1 minimal valid PNG byte sequence used as synthetic image payload.
var syntheticImagePNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, // IDAT chunk
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, // IEND chunk
	0x42, 0x60, 0x82,
}

// TestBenchmark_VisualArchetypes validates end-to-end vision parsing and curatorial compliance
// across the canonical synthetic visual archetypes:
// 1. Void / Low-Information -> "Monochrome Nihilism"
// 2. Linear Wax / Pigment   -> "The Crayon Period"
// 3. Dynamic Linework       -> "Kinetic Scribblism"
// 4. Spills / Entropy       -> "Domestic Destructionism"
// 5. Object Sculpture       -> "Found Object Assemblage"
func TestBenchmark_VisualArchetypes(t *testing.T) {
	archetypes := []canonicalArchetypeCase{
		{
			Name:             "Void_LowInformation",
			Archetype:        "Void / Low-Information: Blurry/dark/pocket capture",
			SyntheticPayload: syntheticImagePNG,
			ExpectedTheme:    ThemeMonochromeNihilism,
			RawModelResponse: `{
				"title": "Study in Absolute Absence No. 7",
				"medium": "Digital sensor capture in low-illumination field",
				"description": "This uncompromising monochrome study engages deeply with the phenomenology of visual sensory deprivation and atmospheric obscurity within modern spatial discourse. Through the total reduction of optical illumination and figurative reference, the planar surface articulates an unyielding meditation on existential absence. The resulting dark field dissolves conventional spatial hierarchies into an austere, contemplative arena of pure phenomenological contemplation and silence.",
				"theme": "` + ThemeMonochromeNihilism + `"
			}`,
		},
		{
			Name:             "LinearWax_Pigment",
			Archetype:        "Linear Wax / Pigment: Primitive marks and raw wax explorations",
			SyntheticPayload: syntheticImagePNG,
			ExpectedTheme:    ThemeTheCrayonPeriod,
			// Wrapped in markdown code fences and with trailing comma to test 100% resilient parsing
			RawModelResponse: "```json\n" + `{
				"title": "Chromatic Stratification in Vermilion and Ochre",
				"medium": "Wax pigment on reclaimed cellulose matrix",
				"description": "This visceral execution of chromatic density exploits the physical resistance between dense wax pigment and textured cellulose ground. The artist applies deliberate pressure to generate layered striations that oscillate between raw primeval instinct and calculated geometric boundary. Every saturated stroke challenges traditional planar boundaries, demanding that the viewer confront the stark materiality of pigment application.",
				"theme": "` + ThemeTheCrayonPeriod + `",
			}` + "\n```",
		},
		{
			Name:             "DynamicLinework",
			Archetype:        "Dynamic Linework: Frantic energetic gestures and chaotic linework",
			SyntheticPayload: syntheticImagePNG,
			ExpectedTheme:    ThemeKineticScribblism,
			// Surrounded by extraneous conversational text to test resilient extraction
			RawModelResponse: "Certainly, here is the curatorial analysis for the exhibition catalog:\n" + `{
				"title": "Tension and Centrifugal Velocity",
				"medium": "Solvent-based dye transfer on woven substrate",
				"description": "The composition erupts in a storm of frantic, rapid-fire gestural mark-making that transforms the entire surface into an arena of furious energetic release. Unfettered velocity and rhythmic intersections deconstruct formal balance, manifesting a heightened psychological urgency across each directional vector. Through these relentless linear trajectories, the work establishes an uncompromising dialogue with the traditions of historic action painting.",
				"theme": "` + ThemeKineticScribblism + `"
			}` + "\nI hope this meets your cataloging requirements.",
		},
		{
			Name:             "Spills_Entropy",
			Archetype:        "Spills / Entropy: Material spill and structural disruption",
			SyntheticPayload: syntheticImagePNG,
			ExpectedTheme:    ThemeDomesticDestructionism,
			RawModelResponse: `{
				"title": "Entropic Rupture in Viscous Aqueous Medium",
				"medium": "Aqueous fluid on compressed pulp",
				"description": "An accidental yet profound embrace of material entropy dominates this composition, allowing liquid fluid to destabilize the planar integrity of the support. The uncontrolled dissipation and organic seepage challenge intentionality, proposing an aesthetic wherein structural disruption becomes the primary agent of meaning. In staging this confrontation with systemic breakdown, the work elevates transient domestic turbulence to monumental cultural tragedy.",
				"theme": "` + ThemeDomesticDestructionism + `"
			}`,
		},
		{
			Name:             "ObjectSculpture",
			Archetype:        "Object Sculpture: 3D assembly and spatial arrangements",
			SyntheticPayload: syntheticImagePNG,
			ExpectedTheme:    ThemeFoundObjectAssemblage,
			RawModelResponse: `{
				"title": "Spatial Juxtaposition of Found Domestic Polyforms",
				"medium": "Found object assemblage and synthetic composite media",
				"description": "This complex three-dimensional construction investigates architectural equilibrium and commodity fetishism through the rigorous spatial juxtaposition of disparate utilitarian artifacts. By liberating everyday molded materials from their functional domestic bondage, the sculptural mass establishes an uncanny monumental presence within the gallery context. The precarious balance achieved across discordant components interrogates late-capitalist material culture with unflinching intellectual rigor.",
				"theme": "` + ThemeFoundObjectAssemblage + `"
			}`,
		},
	}

	for _, tc := range archetypes {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			requestReceived := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestReceived = true

				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected application/json Content-Type, got %s", r.Header.Get("Content-Type"))
				}

				reqBody, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}

				var req chatCompletionRequest
				if err := json.Unmarshal(reqBody, &req); err != nil {
					t.Fatalf("failed to unmarshal request body: %v", err)
				}

				// Assert system prompt contains curatorial instructions
				if len(req.Messages) < 2 {
					t.Fatalf("expected at least 2 messages in request, got %d", len(req.Messages))
				}
				sysMsg, ok := req.Messages[0].Content.(string)
				if !ok || !strings.Contains(sysMsg, "Tate Modern") {
					t.Errorf("expected curatorial system prompt in first message, got: %v", req.Messages[0].Content)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(fmt.Sprintf(`{
					"choices": [
						{
							"message": {
								"content": %s
							}
						}
					]
				}`, strconvQuote(tc.RawModelResponse))))
			}))
			defer server.Close()

			svc := NewService(
				WithEndpoint(server.URL),
				WithHTTPClient(server.Client()),
			)

			res, err := svc.Analyze(context.Background(), tc.SyntheticPayload)
			if err != nil {
				t.Fatalf("expected clean analysis parsing for archetype %q, got error: %v", tc.Name, err)
			}
			if !requestReceived {
				t.Fatalf("server did not receive inference request for archetype %q", tc.Name)
			}
			if res == nil {
				t.Fatalf("expected non-nil AnalysisResult for archetype %q", tc.Name)
			}

			// Automated Suite Assertions:
			// 1. Presence of all required keys
			if strings.TrimSpace(res.Title) == "" {
				t.Errorf("archetype %q: missing or empty 'title'", tc.Name)
			}
			if strings.TrimSpace(res.Medium) == "" {
				t.Errorf("archetype %q: missing or empty 'medium'", tc.Name)
			}
			if strings.TrimSpace(res.Description) == "" {
				t.Errorf("archetype %q: missing or empty 'description'", tc.Name)
			}
			if strings.TrimSpace(res.Theme) == "" {
				t.Errorf("archetype %q: missing or empty 'theme'", tc.Name)
			}

			// 2. Expected canonical thematic movement classification
			if res.Theme != tc.ExpectedTheme {
				t.Errorf("archetype %q: expected theme %q, got %q", tc.Name, tc.ExpectedTheme, res.Theme)
			}

			// 3. Zero occurrences of prohibited diminutive vocabulary
			for _, forbidden := range ForbiddenCuratorialWords {
				pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(forbidden))
				matched, err := regexp.MatchString(pattern, res.Description)
				if err != nil {
					t.Fatalf("failed to evaluate regex for forbidden word %q: %v", forbidden, err)
				}
				if matched {
					t.Errorf("archetype %q description contains forbidden diminutive word %q: %s", tc.Name, forbidden, res.Description)
				}
			}

			// 4. Adherence to plaque critique length constraints (2–4 sentences, 50–120 words)
			words := strings.Fields(res.Description)
			if len(words) < 50 || len(words) > 120 {
				t.Errorf("archetype %q: description word count %d outside allowed range [50, 120]", tc.Name, len(words))
			}
			sentences := countSentences(res.Description)
			if sentences < 2 || sentences > 4 {
				t.Errorf("archetype %q: description sentence count %d outside allowed range [2, 4]", tc.Name, sentences)
			}

			// 5. Elevated medium transformation (no colloquial material descriptions)
			for _, colloquial := range colloquialMediumTerms {
				pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(colloquial))
				matched, err := regexp.MatchString(pattern, res.Medium)
				if err != nil {
					t.Fatalf("failed to evaluate regex for colloquial term %q: %v", colloquial, err)
				}
				if matched {
					t.Errorf("archetype %q medium contains colloquial non-elevated term %q: %s", tc.Name, colloquial, res.Medium)
				}
			}

			// 6. Curatorial rules validator check
			if err := ValidateCuratorialRules(res); err != nil {
				t.Errorf("archetype %q: ValidateCuratorialRules failed: %v", tc.Name, err)
			}
		})
	}
}

// TestBenchmark_SuiteAssertions_ComplianceFailures tests that the benchmark validation
// suite rigorously catches and rejects compliance failures across archetypal evaluations.
func TestBenchmark_SuiteAssertions_ComplianceFailures(t *testing.T) {
	tests := []struct {
		name        string
		rawJSON     string
		expectError string
	}{
		{
			name: "Prohibited_Vocabulary_Child",
			rawJSON: `{
				"title": "Study in Yellow",
				"medium": "Wax pigment on cellulose matrix",
				"description": "This monumental composition embodies the austere dialectic of space and mark-making in contemporary practice. The gestural urgency establishes an unrelenting rhythm that interrogates the viewer phenomenological expectations across the entire picture plane. Each intentional intervention transforms the child substrate into an arena of profound metaphysical inquiry and rigorous aesthetic contemplation.",
				"theme": "The Crayon Period"
			}`,
			expectError: "forbidden word \"child\"",
		},
		{
			name: "Prohibited_Vocabulary_Scribble",
			rawJSON: `{
				"title": "Kinetic Gesture No. 1",
				"medium": "Solvent-based dye on substrate",
				"description": "This monumental composition embodies the austere dialectic of space and mark-making in contemporary practice. The gestural urgency establishes an unrelenting rhythm that interrogates the viewer phenomenological expectations across the entire picture plane. Each scribble transforms the everyday substrate into an arena of profound metaphysical inquiry and rigorous aesthetic contemplation.",
				"theme": "Kinetic Scribblism"
			}`,
			expectError: "forbidden word \"scribble\"",
		},
		{
			name: "WordCount_TooShort",
			rawJSON: `{
				"title": "Void Study",
				"medium": "Sensor capture in low illumination",
				"description": "This is a brief critique that is far too short to satisfy the required curatorial depth and rigor. It fails constraints.",
				"theme": "Monochrome Nihilism"
			}`,
			expectError: "outside allowed range [50, 120]",
		},
		{
			name: "SentenceCount_TooLow",
			rawJSON: `{
				"title": "Singular Run-on",
				"medium": "Wax pigment on reclaimed cellulose matrix",
				"description": "This monumental composition embodies the austere dialectic of space and mark-making in contemporary practice while the gestural urgency establishes an unrelenting rhythm that interrogates the viewer phenomenological expectations across the entire picture plane where each intentional intervention transforms the everyday substrate into an arena of profound metaphysical inquiry and rigorous aesthetic contemplation without any period separators anywhere.",
				"theme": "The Crayon Period"
			}`,
			expectError: "sentence count 1 outside allowed range [2, 4]",
		},
		{
			name: "MissingRequiredField_Theme",
			rawJSON: `{
				"title": "Untitled Piece",
				"medium": "Aqueous fluid on compressed pulp",
				"description": "This monumental composition embodies the austere dialectic of space and mark-making in contemporary practice. The gestural urgency establishes an unrelenting rhythm that interrogates the viewer phenomenological expectations across the entire picture plane. Each intentional intervention transforms the everyday substrate into an arena of profound metaphysical inquiry and rigorous aesthetic contemplation.",
				"theme": ""
			}`,
			expectError: "missing required field: theme",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseAndValidateAnalysisResult(tt.rawJSON)
			if err != nil {
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Fatalf("expected error containing %q, got %v", tt.expectError, err)
				}
				return
			}

			err = ValidateCuratorialRules(res)
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected validation error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

// TestBenchmark_ElevatedMediumVerification tests that colloquial materials are flagged
// while elevated museum-grade media are accepted.
func TestBenchmark_ElevatedMediumVerification(t *testing.T) {
	colloquialCases := []struct {
		medium     string
		colloquial bool
	}{
		{"Wax crayons on construction paper", true},
		{"Felt tip marker on cardboard box", true},
		{"Playdough and slime on bedroom carpet", true},
		{"Toddler juice spill on floor", true},
		{"Wax pigment on reclaimed cellulose matrix", false},
		{"Solvent-based dye transfer on woven ground", false},
		{"Organic polymer paste on synthetic fiber substrate", false},
		{"Aqueous fluid on compressed pulp", false},
		{"Found object assemblage and mixed media", false},
	}

	for _, tc := range colloquialCases {
		isColloquial := false
		var detected string
		for _, term := range colloquialMediumTerms {
			pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(term))
			if matched, _ := regexp.MatchString(pattern, tc.medium); matched {
				isColloquial = true
				detected = term
				break
			}
		}

		if isColloquial != tc.colloquial {
			t.Errorf("medium %q: expected colloquial=%v, got %v (detected: %q)", tc.medium, tc.colloquial, isColloquial, detected)
		}
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
