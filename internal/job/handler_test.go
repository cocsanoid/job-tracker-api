package job

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	s, _ := NewStore("")
	return NewHandler(s).Routes()
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateApplication(t *testing.T) {
	h := newHandler(t)
	rec := do(h, "POST", "/applications",
		`{"company":"AIparking","position":"Golang Intern","city":"Almaty","status":"applied"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var a Application
	json.Unmarshal(rec.Body.Bytes(), &a)
	if a.ID != 1 || a.Status != StatusApplied || a.AppliedAt == nil {
		t.Errorf("unexpected: %+v", a)
	}
}

func TestCreateApplication_BadRequests(t *testing.T) {
	h := newHandler(t)
	tests := []struct{ name, body string }{
		{"broken json", `{oops`},
		{"no company", `{"position":"Dev"}`},
		{"bad status", `{"company":"X","position":"Dev","status":"maybe"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(h, "POST", "/applications", tt.body); rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestGet_NotFoundAndBadID(t *testing.T) {
	h := newHandler(t)
	if rec := do(h, "GET", "/applications/5", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if rec := do(h, "GET", "/applications/abc", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestPatchStatusAndStats(t *testing.T) {
	h := newHandler(t)
	do(h, "POST", "/applications", `{"company":"X","position":"Dev"}`)

	rec := do(h, "PATCH", "/applications/1/status", `{"status":"interview"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d", rec.Code)
	}
	if rec := do(h, "PATCH", "/applications/1/status", `{"status":"???"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("bad status code = %d, want 400", rec.Code)
	}

	rec = do(h, "GET", "/stats", "")
	var st Stats
	json.Unmarshal(rec.Body.Bytes(), &st)
	if st.Total != 1 || st.ByStatus[StatusInterview] != 1 {
		t.Errorf("unexpected stats: %+v", st)
	}
}

func TestListFilterAndDelete(t *testing.T) {
	h := newHandler(t)
	do(h, "POST", "/applications", `{"company":"A","position":"Dev","status":"applied"}`)
	do(h, "POST", "/applications", `{"company":"B","position":"Dev"}`)

	rec := do(h, "GET", "/applications?status=applied", "")
	var list []Application
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].Company != "A" {
		t.Errorf("unexpected list: %+v", list)
	}

	if rec := do(h, "GET", "/applications?status=nope", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad filter code = %d, want 400", rec.Code)
	}
	if rec := do(h, "DELETE", "/applications/1", ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete code = %d, want 204", rec.Code)
	}
	if rec := do(h, "GET", "/applications/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("after delete code = %d, want 404", rec.Code)
	}
}
