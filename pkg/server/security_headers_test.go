package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router with the security headers middleware
	router := gin.New()
	router.Use(securityHeadersMiddleware())

	// Add a simple test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Create a test request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Verify response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Test all security headers
	tests := []struct {
		name          string
		headerName    string
		expectedValue string
	}{
		{
			name:          "Strict-Transport-Security header",
			headerName:    "Strict-Transport-Security",
			expectedValue: "max-age=31536000; includeSubDomains",
		},
		{
			name:          "X-Frame-Options header",
			headerName:    "X-Frame-Options",
			expectedValue: "DENY",
		},
		{
			name:          "X-Content-Type-Options header",
			headerName:    "X-Content-Type-Options",
			expectedValue: "nosniff",
		},
		{
			name:          "Content-Security-Policy header",
			headerName:    "Content-Security-Policy",
			expectedValue: "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; img-src 'self' data: https:; font-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none';",
		},
		{
			name:          "Referrer-Policy header",
			headerName:    "Referrer-Policy",
			expectedValue: "strict-origin-when-cross-origin",
		},
		{
			name:          "Permissions-Policy header",
			headerName:    "Permissions-Policy",
			expectedValue: "camera=(), microphone=(), geolocation=(), interest-cohort=()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualValue := w.Header().Get(tt.headerName)
			if actualValue != tt.expectedValue {
				t.Errorf("Expected %s header to be %q, got %q", tt.headerName, tt.expectedValue, actualValue)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_MultipleRequests(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router with the security headers middleware
	router := gin.New()
	router.Use(securityHeadersMiddleware())

	router.GET("/endpoint1", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "endpoint1"})
	})

	router.POST("/endpoint2", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "endpoint2"})
	})

	endpoints := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/endpoint1", http.StatusOK},
		{"POST", "/endpoint2", http.StatusCreated},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			req, err := http.NewRequest(endpoint.method, endpoint.path, nil)
			if err != nil {
				t.Fatalf("Failed to create test request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != endpoint.status {
				t.Errorf("Expected status code %d, got %d", endpoint.status, w.Code)
			}

			// Verify critical security headers are present
			criticalHeaders := []string{
				"Strict-Transport-Security",
				"X-Frame-Options",
				"X-Content-Type-Options",
				"Content-Security-Policy",
			}

			for _, header := range criticalHeaders {
				if w.Header().Get(header) == "" {
					t.Errorf("Expected %s header to be set, but it was empty", header)
				}
			}
		})
	}
}

func TestSecurityHeadersMiddleware_WithOtherMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router with multiple middleware
	router := gin.New()

	// Add custom middleware that sets a custom header
	router.Use(func(c *gin.Context) {
		c.Header("X-Custom-Header", "custom-value")
		c.Next()
	})

	// Add security headers middleware
	router.Use(securityHeadersMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify both custom and security headers are present
	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("Custom header from other middleware was not set")
	}

	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("Security headers were not set when combined with other middleware")
	}
}

func TestSecurityHeadersMiddleware_HSTS_Format(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(securityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hstsHeader := w.Header().Get("Strict-Transport-Security")

	// Verify HSTS includes max-age
	if hstsHeader == "" {
		t.Fatal("HSTS header is not set")
	}

	// Verify it includes both max-age and includeSubDomains
	expectedComponents := []string{"max-age=31536000", "includeSubDomains"}
	for _, component := range expectedComponents {
		found := false
		if containsString(hstsHeader, component) {
			found = true
		}
		if !found {
			t.Errorf("HSTS header missing required component: %s", component)
		}
	}
}

func TestSecurityHeadersMiddleware_CSP_Directives(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(securityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	cspHeader := w.Header().Get("Content-Security-Policy")

	if cspHeader == "" {
		t.Fatal("CSP header is not set")
	}

	// Verify important CSP directives are present
	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
	}

	for _, directive := range requiredDirectives {
		if !containsString(cspHeader, directive) {
			t.Errorf("CSP header missing required directive: %s", directive)
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}