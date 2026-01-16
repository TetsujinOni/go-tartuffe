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

// TestCreateImposterWithInjection tests that imposters can be created with injection
func TestCreateImposterWithInjection(t *testing.T) {
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
			name: "predicate injection",
			body: `{
				"protocol": "http",
				"port": 3000,
				"stubs": [{
					"predicates": [{"inject": "function(request) { return request.path === '/test'; }"}],
					"responses": [{"is": {"body": "MATCHED"}}]
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
				if len(imp.Stubs[0].Predicates) != 1 {
					t.Fatalf("expected 1 predicate, got %d", len(imp.Stubs[0].Predicates))
				}
				if imp.Stubs[0].Predicates[0].Inject == "" {
					t.Error("expected predicate inject to be set")
				}
			},
		},
		{
			name: "response injection",
			body: `{
				"protocol": "http",
				"port": 3001,
				"stubs": [{
					"responses": [{"inject": "function(request) { return {body: 'INJECTED'}; }"}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if imp.Stubs[0].Responses[0].Inject == "" {
					t.Error("expected response inject to be set")
				}
			},
		},
		{
			name: "combined predicate and response injection",
			body: `{
				"protocol": "http",
				"port": 3002,
				"stubs": [{
					"predicates": [{"inject": "config => config.request.path === '/api'"}],
					"responses": [{"inject": "config => ({body: config.request.method})"}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if imp.Stubs[0].Predicates[0].Inject == "" {
					t.Error("expected predicate inject to be set")
				}
				if imp.Stubs[0].Responses[0].Inject == "" {
					t.Error("expected response inject to be set")
				}
			},
		},
		{
			name: "old interface (single argument)",
			body: `{
				"protocol": "http",
				"port": 3003,
				"stubs": [{
					"responses": [{"inject": "function(request) { return {body: request.method}; }"}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if imp.Stubs[0].Responses[0].Inject == "" {
					t.Error("expected response inject to be set")
				}
			},
		},
		{
			name: "new interface (config argument)",
			body: `{
				"protocol": "http",
				"port": 3004,
				"stubs": [{
					"responses": [{"inject": "config => ({body: config.request.method})"}]
				}]
			}`,
			wantCode: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var imp models.Imposter
				if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if imp.Stubs[0].Responses[0].Inject == "" {
					t.Error("expected response inject to be set")
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

// TestInjectionWithEndOfRequestResolver tests TCP end-of-request resolver injection
func TestInjectionWithEndOfRequestResolver(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	body := `{
		"protocol": "tcp",
		"port": 3000,
		"mode": "text",
		"endOfRequestResolver": {
			"inject": "function(requestData, logger) { return requestData.indexOf('END') > -1; }"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/imposters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateImposter(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateImposter() status = %d, want %d", rec.Code, http.StatusCreated)
		t.Logf("Response body: %s", rec.Body.String())
		return
	}

	var imp models.Imposter
	if err := json.Unmarshal(rec.Body.Bytes(), &imp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if imp.EndOfRequestResolver == nil {
		t.Fatal("expected endOfRequestResolver to be set")
	}
	if imp.EndOfRequestResolver.Inject == "" {
		t.Error("expected endOfRequestResolver inject to be set")
	}

	// Cleanup
	repo.DeleteAll()
	manager.StopAll()
}

// TestInjectionValidationNonStrict tests that injection validation is non-strict
// Mountebank doesn't validate injection code at creation time
func TestInjectionValidationNonStrict(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name: "invalid JavaScript should still create imposter",
			body: `{
				"protocol": "http",
				"port": 3000,
				"stubs": [{
					"responses": [{"inject": "return true;"}]
				}]
			}`,
			wantCode: http.StatusCreated, // Should succeed - validation happens at runtime
		},
		{
			name: "syntax error should still create imposter",
			body: `{
				"protocol": "http",
				"port": 3001,
				"stubs": [{
					"responses": [{"inject": "function() { throw new Error('BOOM'); }"}]
				}]
			}`,
			wantCode: http.StatusCreated, // Should succeed - errors happen at runtime
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
