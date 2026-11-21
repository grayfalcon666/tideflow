package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"tideflow/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRecoveryReturnsJSONGeneric500(t *testing.T) {
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Code != 500 {
		t.Errorf("Response.Code = %d, want 500", resp.Code)
	}
	if resp.Message != response.InternalErrorMessage {
		t.Errorf("Response.Message = %q, want generic message %q", resp.Message, response.InternalErrorMessage)
	}
}
