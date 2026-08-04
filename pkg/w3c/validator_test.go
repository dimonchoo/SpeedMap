package w3c

import (
	"testing"
)

func TestParseW3CJSONResponse(t *testing.T) {
	mockResponse := []byte(`{
		"messages": [
			{
				"type": "error",
				"lastLine": 14,
				"lastColumn": 32,
				"message": "End tag “div” seen, but there were open elements.",
				"extract": "<div><p>Unclosed paragraph</div>"
			},
			{
				"type": "info",
				"subType": "warning",
				"lastLine": 25,
				"lastColumn": 10,
				"message": "Section lacks heading. Consider using “h2”-“h6” elements.",
				"extract": "<section class=\"hero\">"
			}
		]
	}`)

	report, err := ParseW3CJSONResponse("https://example.com", mockResponse)
	if err != nil {
		t.Fatalf("ParseW3CJSONResponse returned error: %v", err)
	}

	if report.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got %s", report.URL)
	}

	if report.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount = 1, got %d", report.ErrorCount)
	}

	if report.WarningCount != 1 {
		t.Errorf("Expected WarningCount = 1, got %d", report.WarningCount)
	}

	if report.IsValid {
		t.Errorf("Expected IsValid = false")
	}

	if report.Status != "invalid" {
		t.Errorf("Expected Status = 'invalid', got %s", report.Status)
	}

	if len(report.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(report.Messages))
	}
}
