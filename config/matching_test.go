package config

import (
	"strings"
	"testing"
)

// TestBoolExprSimpleBody tests basic body field matching
func TestBoolExprSimpleBody(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4"}}
	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile model pattern: %v", err)
	}

	expr := &BoolExpr{
		Body: map[string]PatternField{
			"model": modelPattern,
		},
	}

	body := map[string]any{"model": "gpt-4", "temperature": 0.7}
	headers := map[string]string{}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for model=gpt-4")
	}

	body["model"] = "claude-3"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for model=claude-3")
	}
}

// TestBoolExprSimpleQuery tests query parameter matching
func TestBoolExprSimpleQuery(t *testing.T) {
	providerPattern := PatternField{Patterns: []string{"openai"}}
	if err := providerPattern.Validate(); err != nil {
		t.Fatalf("failed to compile provider pattern: %v", err)
	}

	expr := &BoolExpr{
		Query: map[string]PatternField{
			"provider": providerPattern,
		},
	}

	body := map[string]any{}
	headers := map[string]string{}
	query := map[string]string{"provider": "openai", "version": "v1"}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for provider=openai")
	}

	query["provider"] = "anthropic"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for provider=anthropic")
	}
}

// TestBoolExprSimpleHeaders tests header matching with case insensitivity
func TestBoolExprSimpleHeaders(t *testing.T) {
	authPattern := PatternField{Patterns: []string{"Bearer.*"}}
	if err := authPattern.Validate(); err != nil {
		t.Fatalf("failed to compile auth pattern: %v", err)
	}

	expr := &BoolExpr{
		Headers: map[string]PatternField{
			"Authorization": authPattern,
		},
	}

	body := map[string]any{}
	headers := map[string]string{"Authorization": "Bearer token123"}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for Authorization header")
	}

	// Test case insensitivity of header keys
	headers = map[string]string{"authorization": "Bearer xyz"}
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected case-insensitive match for authorization header")
	}

	headers = map[string]string{"AUTHORIZATION": "Bearer abc"}
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected case-insensitive match for AUTHORIZATION header")
	}

	headers["Authorization"] = "Basic user:pass"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for Basic auth")
	}
}

// TestBoolExprImplicitAnd tests that multiple fields at the same level are AND'ed
func TestBoolExprImplicitAnd(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4"}}
	providerPattern := PatternField{Patterns: []string{"openai"}}
	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile model pattern: %v", err)
	}
	if err := providerPattern.Validate(); err != nil {
		t.Fatalf("failed to compile provider pattern: %v", err)
	}

	expr := &BoolExpr{
		Body: map[string]PatternField{
			"model": modelPattern,
		},
		Query: map[string]PatternField{
			"provider": providerPattern,
		},
	}

	body := map[string]any{"model": "gpt-4"}
	headers := map[string]string{}
	query := map[string]string{"provider": "openai"}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match when both body and query match")
	}

	// Missing query parameter
	query = map[string]string{}
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when query doesn't match")
	}

	// Wrong body value
	query = map[string]string{"provider": "openai"}
	body["model"] = "claude-3"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when body doesn't match")
	}
}

// TestBoolExprOr tests explicit OR logic
func TestBoolExprOr(t *testing.T) {
	gptPattern := PatternField{Patterns: []string{"gpt-4"}}
	claudePattern := PatternField{Patterns: []string{"claude-3"}}
	if err := gptPattern.Validate(); err != nil {
		t.Fatalf("failed to compile gpt pattern: %v", err)
	}
	if err := claudePattern.Validate(); err != nil {
		t.Fatalf("failed to compile claude pattern: %v", err)
	}

	expr := &BoolExpr{
		Or: []BoolExpr{
			{
				Body: map[string]PatternField{
					"model": gptPattern,
				},
			},
			{
				Body: map[string]PatternField{
					"model": claudePattern,
				},
			},
		},
	}

	body := map[string]any{"model": "gpt-4"}
	headers := map[string]string{}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4 via OR")
	}

	body["model"] = "claude-3"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3 via OR")
	}

	body["model"] = "llama-2"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for llama-2")
	}
}

// TestBoolExprAnd tests explicit AND logic
func TestBoolExprAnd(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4"}}
	streamPattern := PatternField{Patterns: []string{"false"}}
	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile model pattern: %v", err)
	}
	if err := streamPattern.Validate(); err != nil {
		t.Fatalf("failed to compile stream pattern: %v", err)
	}

	expr := &BoolExpr{
		And: []BoolExpr{
			{
				Body: map[string]PatternField{
					"model": modelPattern,
				},
			},
			{
				Body: map[string]PatternField{
					"stream": streamPattern,
				},
			},
		},
	}

	body := map[string]any{"model": "gpt-4", "stream": false}
	headers := map[string]string{}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match when both conditions are true")
	}

	body["stream"] = true
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when stream=true")
	}

	body["stream"] = false
	body["model"] = "claude-3"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when model is wrong")
	}
}

// TestBoolExprNot tests NOT logic
func TestBoolExprNot(t *testing.T) {
	streamPattern := PatternField{Patterns: []string{"true"}}
	if err := streamPattern.Validate(); err != nil {
		t.Fatalf("failed to compile stream pattern: %v", err)
	}

	expr := &BoolExpr{
		Not: &BoolExpr{
			Body: map[string]PatternField{
				"stream": streamPattern,
			},
		},
	}

	body := map[string]any{"stream": false}
	headers := map[string]string{}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match when stream is NOT true")
	}

	body["stream"] = true
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when stream IS true")
	}
}

// TestBoolExprNestedOrAnd tests complex nested logic: (A OR B) AND C
func TestBoolExprNestedOrAnd(t *testing.T) {
	gptPattern := PatternField{Patterns: []string{"gpt-4"}}
	claudePattern := PatternField{Patterns: []string{"claude-3"}}
	providerPattern := PatternField{Patterns: []string{"openai|anthropic"}}
	if err := gptPattern.Validate(); err != nil {
		t.Fatalf("failed to compile gpt pattern: %v", err)
	}
	if err := claudePattern.Validate(); err != nil {
		t.Fatalf("failed to compile claude pattern: %v", err)
	}
	if err := providerPattern.Validate(); err != nil {
		t.Fatalf("failed to compile provider pattern: %v", err)
	}

	// (model=gpt-4 OR model=claude-3) AND query.provider=(openai|anthropic)
	expr := &BoolExpr{
		And: []BoolExpr{
			{
				Or: []BoolExpr{
					{
						Body: map[string]PatternField{
							"model": gptPattern,
						},
					},
					{
						Body: map[string]PatternField{
							"model": claudePattern,
						},
					},
				},
			},
			{
				Query: map[string]PatternField{
					"provider": providerPattern,
				},
			},
		},
	}

	body := map[string]any{"model": "gpt-4"}
	headers := map[string]string{}
	query := map[string]string{"provider": "openai"}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4 + openai")
	}

	body["model"] = "claude-3"
	query["provider"] = "anthropic"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3 + anthropic")
	}

	body["model"] = "llama-2"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when model is neither gpt-4 nor claude-3")
	}

	body["model"] = "gpt-4"
	query["provider"] = "cohere"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when provider doesn't match")
	}
}

// TestBoolExprComplexNesting tests deeply nested logic
func TestBoolExprComplexNesting(t *testing.T) {
	gptPattern := PatternField{Patterns: []string{"gpt-4"}}
	claudePattern := PatternField{Patterns: []string{"claude-3"}}
	streamPattern := PatternField{Patterns: []string{"true"}}
	authPattern := PatternField{Patterns: []string{"Bearer.*"}}
	if err := gptPattern.Validate(); err != nil {
		t.Fatalf("failed to compile gpt pattern: %v", err)
	}
	if err := claudePattern.Validate(); err != nil {
		t.Fatalf("failed to compile claude pattern: %v", err)
	}
	if err := streamPattern.Validate(); err != nil {
		t.Fatalf("failed to compile stream pattern: %v", err)
	}
	if err := authPattern.Validate(); err != nil {
		t.Fatalf("failed to compile auth pattern: %v", err)
	}

	// ((model=gpt-4 OR model=claude-3) AND NOT stream=true) AND Authorization=Bearer
	expr := &BoolExpr{
		And: []BoolExpr{
			{
				And: []BoolExpr{
					{
						Or: []BoolExpr{
							{
								Body: map[string]PatternField{
									"model": gptPattern,
								},
							},
							{
								Body: map[string]PatternField{
									"model": claudePattern,
								},
							},
						},
					},
					{
						Not: &BoolExpr{
							Body: map[string]PatternField{
								"stream": streamPattern,
							},
						},
					},
				},
			},
			{
				Headers: map[string]PatternField{
					"Authorization": authPattern,
				},
			},
		},
	}

	body := map[string]any{"model": "gpt-4", "stream": false}
	headers := map[string]string{"Authorization": "Bearer token123"}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4, stream=false, with Bearer token")
	}

	body["model"] = "claude-3"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3, stream=false, with Bearer token")
	}

	// Fail when stream=true
	body["model"] = "gpt-4"
	body["stream"] = true
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when stream=true")
	}

	// Fail when no auth header
	body["stream"] = false
	headers = map[string]string{}
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match without Authorization header")
	}

	// Fail when wrong model
	body["model"] = "llama-2"
	headers = map[string]string{"Authorization": "Bearer token123"}
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for llama-2")
	}
}

// TestBoolExprEmptyMatches tests that empty BoolExpr always matches
func TestBoolExprEmptyMatches(t *testing.T) {
	expr := &BoolExpr{}

	body := map[string]any{"model": "anything"}
	headers := map[string]string{"X-Custom": "value"}
	query := map[string]string{"param": "value"}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected empty BoolExpr to match everything")
	}
}

// TestBoolExprMultipleFieldsInBodyAndQuery tests multiple fields at same level (implicit AND)
func TestBoolExprMultipleFieldsInBodyAndQuery(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4"}}
	tempPattern := PatternField{Patterns: []string{"0\\.[7-9]"}}
	providerPattern := PatternField{Patterns: []string{"openai"}}
	versionPattern := PatternField{Patterns: []string{"v1"}}

	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile model pattern: %v", err)
	}
	if err := tempPattern.Validate(); err != nil {
		t.Fatalf("failed to compile temp pattern: %v", err)
	}
	if err := providerPattern.Validate(); err != nil {
		t.Fatalf("failed to compile provider pattern: %v", err)
	}
	if err := versionPattern.Validate(); err != nil {
		t.Fatalf("failed to compile version pattern: %v", err)
	}

	expr := &BoolExpr{
		Body: map[string]PatternField{
			"model":       modelPattern,
			"temperature": tempPattern,
		},
		Query: map[string]PatternField{
			"provider": providerPattern,
			"version":  versionPattern,
		},
	}

	body := map[string]any{"model": "gpt-4", "temperature": 0.7}
	headers := map[string]string{}
	query := map[string]string{"provider": "openai", "version": "v1"}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match when all fields match")
	}

	// Fail when one body field doesn't match
	body["temperature"] = 0.5
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when temperature doesn't match")
	}

	// Fail when one query field doesn't match
	body["temperature"] = 0.8
	query["version"] = "v2"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match when version doesn't match")
	}
}

// TestBoolExprRegexAlternation tests regex alternation (|) within patterns
func TestBoolExprRegexAlternation(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4|claude-3|llama-2"}}
	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile model pattern: %v", err)
	}

	expr := &BoolExpr{
		Body: map[string]PatternField{
			"model": modelPattern,
		},
	}

	body := map[string]any{}
	headers := map[string]string{}
	query := map[string]string{}

	body["model"] = "gpt-4"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4")
	}

	body["model"] = "claude-3"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3")
	}

	body["model"] = "llama-2"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected match for llama-2")
	}

	body["model"] = "gemini"
	if expr.Evaluate(body, headers, query) {
		t.Fatal("expected no match for gemini")
	}
}

// TestBoolExprCaseInsensitiveHeaderKeys tests that header key matching is case-insensitive
func TestBoolExprCaseInsensitiveHeaderKeys(t *testing.T) {
	contentTypePattern := PatternField{Patterns: []string{"application/json"}}
	if err := contentTypePattern.Validate(); err != nil {
		t.Fatalf("failed to compile content-type pattern: %v", err)
	}

	expr := &BoolExpr{
		Headers: map[string]PatternField{
			"Content-Type": contentTypePattern,
		},
	}

	body := map[string]any{}
	headers := map[string]string{}
	query := map[string]string{}

	// Test various casings of header keys
	testCases := []string{
		"Content-Type",
		"content-type",
		"CONTENT-TYPE",
		"CoNtEnT-TyPe",
	}

	for _, headerKey := range testCases {
		headers = map[string]string{headerKey: "application/json"}
		if !expr.Evaluate(body, headers, query) {
			t.Fatalf("expected case-insensitive match for header key: %s", headerKey)
		}
	}
}

// TestBoolExprCaseInsensitiveHeaderValues tests that header value matching is case-insensitive (via regex flag)
func TestBoolExprCaseInsensitiveHeaderValues(t *testing.T) {
	// Pattern without explicit case flag - should still match case-insensitively due to (?i) prefix
	contentTypePattern := PatternField{Patterns: []string{"Application/Json"}}
	if err := contentTypePattern.Validate(); err != nil {
		t.Fatalf("failed to compile content-type pattern: %v", err)
	}

	expr := &BoolExpr{
		Headers: map[string]PatternField{
			"Content-Type": contentTypePattern,
		},
	}

	body := map[string]any{}
	headers := map[string]string{"Content-Type": "application/json"}
	query := map[string]string{}

	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected case-insensitive match for header value")
	}

	headers["Content-Type"] = "APPLICATION/JSON"
	if !expr.Evaluate(body, headers, query) {
		t.Fatal("expected case-insensitive match for uppercase value")
	}
}

// TestActionWhenAny tests the when_any sugar for OR operations
func TestActionWhenAny(t *testing.T) {
	gpt4Pattern := PatternField{Patterns: []string{"gpt-4"}}
	claude3Pattern := PatternField{Patterns: []string{"claude-3"}}
	geminiPattern := PatternField{Patterns: []string{"gemini-pro"}}

	if err := gpt4Pattern.Validate(); err != nil {
		t.Fatalf("failed to compile gpt4 pattern: %v", err)
	}
	if err := claude3Pattern.Validate(); err != nil {
		t.Fatalf("failed to compile claude3 pattern: %v", err)
	}
	if err := geminiPattern.Validate(); err != nil {
		t.Fatalf("failed to compile gemini pattern: %v", err)
	}

	action := Action{
		WhenAny: []BoolExpr{
			{
				Body: map[string]PatternField{
					"model": gpt4Pattern,
				},
			},
			{
				Body: map[string]PatternField{
					"model": claude3Pattern,
				},
			},
			{
				Body: map[string]PatternField{
					"model": geminiPattern,
				},
			},
		},
		Merge: map[string]any{"matched": true},
	}

	// Validate should convert WhenAny to When
	if err := validateAction(&action, 0, 0, "on_request"); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Should have been converted to When with OR
	if action.When == nil {
		t.Fatal("expected When to be set after validation")
	}
	if len(action.When.Or) != 3 {
		t.Fatalf("expected 3 OR expressions, got %d", len(action.When.Or))
	}

	// Test evaluation
	body := map[string]any{"model": "gpt-4"}
	headers := map[string]string{}
	query := map[string]string{}

	if !action.When.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4")
	}

	body["model"] = "claude-3"
	if !action.When.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3")
	}

	body["model"] = "gemini-pro"
	if !action.When.Evaluate(body, headers, query) {
		t.Fatal("expected match for gemini-pro")
	}

	body["model"] = "llama-2"
	if action.When.Evaluate(body, headers, query) {
		t.Fatal("expected no match for llama-2")
	}
}

// TestActionWhenAndWhenAnyMutuallyExclusive tests that both cannot be specified
func TestActionWhenAndWhenAnyMutuallyExclusive(t *testing.T) {
	modelPattern := PatternField{Patterns: []string{"gpt-4"}}
	if err := modelPattern.Validate(); err != nil {
		t.Fatalf("failed to compile pattern: %v", err)
	}

	action := Action{
		When: &BoolExpr{
			Body: map[string]PatternField{
				"model": modelPattern,
			},
		},
		WhenAny: []BoolExpr{
			{
				Body: map[string]PatternField{
					"model": modelPattern,
				},
			},
		},
		Merge: map[string]any{"test": true},
	}

	err := validateAction(&action, 0, 0, "on_request")
	if err == nil {
		t.Fatal("expected error when both when and when_any are specified")
	}
	if !containsString(err.Error(), "cannot specify both when and when_any") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestActionWhenAnyWithComplexExpressions tests when_any with more complex expressions
func TestActionWhenAnyWithComplexExpressions(t *testing.T) {
	gpt4Pattern := PatternField{Patterns: []string{"gpt-4"}}
	premiumPattern := PatternField{Patterns: []string{"premium"}}
	claudePattern := PatternField{Patterns: []string{"claude-3"}}

	if err := gpt4Pattern.Validate(); err != nil {
		t.Fatalf("failed to compile gpt4 pattern: %v", err)
	}
	if err := premiumPattern.Validate(); err != nil {
		t.Fatalf("failed to compile premium pattern: %v", err)
	}
	if err := claudePattern.Validate(); err != nil {
		t.Fatalf("failed to compile claude pattern: %v", err)
	}

	// (model=gpt-4 AND tier=premium) OR (model=claude-3)
	action := Action{
		WhenAny: []BoolExpr{
			{
				Body: map[string]PatternField{
					"model": gpt4Pattern,
				},
				Query: map[string]PatternField{
					"tier": premiumPattern,
				},
			},
			{
				Body: map[string]PatternField{
					"model": claudePattern,
				},
			},
		},
		Merge: map[string]any{"matched": true},
	}

	if err := validateAction(&action, 0, 0, "on_request"); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	body := map[string]any{"model": "gpt-4"}
	headers := map[string]string{}
	query := map[string]string{"tier": "premium"}

	// Should match first expression (gpt-4 AND premium)
	if !action.When.Evaluate(body, headers, query) {
		t.Fatal("expected match for gpt-4 with premium tier")
	}

	// Should not match first expression (missing tier)
	query = map[string]string{}
	if action.When.Evaluate(body, headers, query) {
		t.Fatal("expected no match for gpt-4 without premium tier")
	}

	// Should match second expression (claude-3)
	body["model"] = "claude-3"
	if !action.When.Evaluate(body, headers, query) {
		t.Fatal("expected match for claude-3")
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
