package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Success(c, gin.H{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("Response.Code = %d, want 0", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("Response.Message = %q, want ok", resp.Message)
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	Created(c, gin.H{"id": 123})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("Response.Code = %d, want 0", resp.Code)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	BadRequest(c, "invalid param")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 400 {
		t.Errorf("Response.Code = %d, want 400", resp.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Unauthorized(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 401 {
		t.Errorf("Response.Code = %d, want 401", resp.Code)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Forbidden(c, "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 403001 {
		t.Errorf("Response.Code = %d, want 403001", resp.Code)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	NotFound(c, "user not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 404 {
		t.Errorf("Response.Code = %d, want 404", resp.Code)
	}
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	Conflict(c, "username taken")

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 409 {
		t.Errorf("Response.Code = %d, want 409", resp.Code)
	}
}

func TestTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	TooManyRequests(c)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 429 {
		t.Errorf("Response.Code = %d, want 429", resp.Code)
	}
	if resp.Message != "rate limit exceeded" {
		t.Errorf("Response.Message = %q, want 'rate limit exceeded'", resp.Message)
	}
}

func TestInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	InternalServerError(c, "db error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 500 {
		t.Errorf("Response.Code = %d, want 500", resp.Code)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Error(c, http.StatusGatewayTimeout, 504, "gateway timeout")

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", w.Code, http.StatusGatewayTimeout)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 504 {
		t.Errorf("Response.Code = %d, want 504", resp.Code)
	}
	if resp.Message != "gateway timeout" {
		t.Errorf("Response.Message = %q, want gateway timeout", resp.Message)
	}
	if resp.Data != nil {
		t.Errorf("Response.Data = %v, want nil", resp.Data)
	}
}

func TestPageResponse(t *testing.T) {
	pr := PageResponse{
		Items:      []string{"a", "b"},
		NextCursor: strPtr("next123"),
		HasMore:    true,
	}

	data, _ := json.Marshal(pr)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if _, ok := m["items"]; !ok {
		t.Error("PageResponse missing items field")
	}
	if _, ok := m["next_cursor"]; !ok {
		t.Error("PageResponse missing next_cursor field")
	}
	if _, ok := m["has_more"]; !ok {
		t.Error("PageResponse missing has_more field")
	}
}

func TestTokenResponse(t *testing.T) {
	tr := TokenResponse{
		AccountID:    42,
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		ExpiresIn:    3600,
	}

	data, _ := json.Marshal(tr)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["account_id"] != float64(42) {
		t.Errorf("TokenResponse.account_id = %v, want 42", m["account_id"])
	}
	if m["access_token"] != "access_token" {
		t.Errorf("TokenResponse.access_token = %v, want access_token", m["access_token"])
	}
	if m["refresh_token"] != "refresh_token" {
		t.Errorf("TokenResponse.refresh_token = %v, want refresh_token", m["refresh_token"])
	}
	if m["expires_in"] != float64(3600) {
		t.Errorf("TokenResponse.expires_in = %v, want 3600", m["expires_in"])
	}
}

func strPtr(s string) *string {
	return &s
}
