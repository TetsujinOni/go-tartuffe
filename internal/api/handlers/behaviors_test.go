package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// TestCreateImposterWithBehaviorsObject tests that imposters can be created with _behaviors as an object
// This reproduces the mountebank behavior test failures
func TestCreateImposterWithBehaviorsObject(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	tests := []struct {
		name     string
		body     string
		wantCode int
		validate func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "single wait behavior as object",
			body: `{
				"protocol": "http",
				"port": 3000,
				"stubs": [{
					"responses": [{
						"is": {"statusCode": 200, "body": "test"},
						"_behaviors": {"wait": 1000}
					}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(imp.Stubs) != 1 {
					t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
				}
				if len(imp.Stubs[0].Responses) != 1 {
					t.Fatalf("expected 1 response, got %d", len(imp.Stubs[0].Responses))
				}
				if len(imp.Stubs[0].Responses[0].Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(imp.Stubs[0].Responses[0].Behaviors))
				}
				if imp.Stubs[0].Responses[0].Behaviors[0].Wait == nil {
					t.Error("expected wait behavior to be set")
				}
			},
		},
		{
			name: "single decorate behavior as object",
			body: `{
				"protocol": "http",
				"port": 3001,
				"stubs": [{
					"responses": [{
						"is": {"body": "year is ${YEAR}"},
						"_behaviors": {"decorate": "function() { return response; }"}
					}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(imp.Stubs[0].Responses[0].Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(imp.Stubs[0].Responses[0].Behaviors))
				}
				if imp.Stubs[0].Responses[0].Behaviors[0].Decorate == "" {
					t.Error("expected decorate behavior to be set")
				}
			},
		},
		{
			name: "multiple fields in behavior object",
			body: `{
				"protocol": "http",
				"port": 3002,
				"stubs": [{
					"responses": [{
						"is": {"body": "test"},
						"_behaviors": {"wait": 500, "repeat": 2}
					}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				behavior := imp.Stubs[0].Responses[0].Behaviors[0]
				if behavior.Wait == nil {
					t.Error("expected wait to be set")
				}
				if behavior.Repeat != 2 {
					t.Errorf("expected repeat=2, got %d", behavior.Repeat)
				}
			},
		},
		{
			name: "behaviors as array (backward compatibility)",
			body: `{
				"protocol": "http",
				"port": 3003,
				"stubs": [{
					"responses": [{
						"is": {"body": "test"},
						"_behaviors": [{"wait": 1000}]
					}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(imp.Stubs[0].Responses[0].Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(imp.Stubs[0].Responses[0].Behaviors))
				}
			},
		},
		{
			name: "copy behavior as object",
			body: `{
				"protocol": "http",
				"port": 3004,
				"stubs": [{
					"responses": [{
						"is": {"body": "test"},
						"_behaviors": {
							"copy": {
								"from": "path",
								"into": "${DEST}",
								"using": {"method": "regex", "selector": ".*"}
							}
						}
					}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(imp.Stubs[0].Responses[0].Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(imp.Stubs[0].Responses[0].Behaviors))
				}
				if imp.Stubs[0].Responses[0].Behaviors[0].Copy == nil {
					t.Fatal("expected copy behavior to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/imposters", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateImposter(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("CreateImposter() status = %d, want %d", rec.Code, tt.wantCode)
				t.Logf("Response body: %s", rec.Body.String())
				return
			}

			if tt.validate != nil {
				tt.validate(t, rec)
			}

			// Cleanup
			repo.DeleteAll()
			manager.StopAll()
		})
	}
}

// TestCreateImposterWithBehaviorsRejectsInvalid tests that invalid behavior formats are properly rejected
func TestCreateImposterWithBehaviorsRejectsInvalid(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name: "behaviors as string (invalid)",
			body: `{
				"protocol": "http",
				"port": 3000,
				"stubs": [{
					"responses": [{
						"is": {"body": "test"},
						"_behaviors": "invalid"
					}]
				}]
			}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "behaviors as number (invalid)",
			body: `{
				"protocol": "http",
				"port": 3001,
				"stubs": [{
					"responses": [{
						"is": {"body": "test"},
						"_behaviors": 123
					}]
				}]
			}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/imposters", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateImposter(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("CreateImposter() status = %d, want %d", rec.Code, tt.wantCode)
				t.Logf("Response body: %s", rec.Body.String())
			}

			// Cleanup
			repo.DeleteAll()
			manager.StopAll()
		})
	}
}
