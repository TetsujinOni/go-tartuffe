package imposter

import (
	"os"
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestLookupWithXPathNamespace tests lookup behavior using xpath with namespaces
func TestLookupWithXPathNamespace(t *testing.T) {
	jsEngine := NewJSEngine()
	executor := NewBehaviorExecutor(jsEngine)

	// Create CSV file for lookup
	csvContent := `name,occupation,location
mountebank,tester,worldwide
Brandon,mountebank,Dallas
Bob Barker,"The Price Is Right","Darrington, Washington"`

	csvPath := "/tmp/test_lookup_xpath.csv"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("Failed to create CSV: %v", err)
	}
	defer os.Remove(csvPath)

	req := &models.Request{
		Method: "POST",
		Path:   "/",
		Body:   `<doc xmlns:mb="http://example.com/mb"><mb:name>mountebank</mb:name></doc>`,
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "Hello, YOU[name]! How is YOU['location'] today?",
	}

	behavior := models.Behavior{
		Lookup: []models.Lookup{
			{
				Key: map[string]interface{}{
					"from": "body",
					"using": map[string]interface{}{
						"method":   "xpath",
						"selector": "//mb:name",
						"ns": map[string]interface{}{
							"mb": "http://example.com/mb",
						},
					},
				},
				FromDataSource: &models.DataSource{
					CSV: &models.CSVSource{
						Path:      csvPath,
						KeyColumn: "occupation",
					},
				},
				Into: "YOU",
			},
		},
	}

	result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	wantBody := "Hello, Brandon! How is Dallas today?"
	if bodyStr, ok := result.Body.(string); ok {
		if bodyStr != wantBody {
			t.Errorf("Body = %q, want %q", bodyStr, wantBody)
		}
	} else {
		t.Errorf("Body is not a string: %v", result.Body)
	}
}
