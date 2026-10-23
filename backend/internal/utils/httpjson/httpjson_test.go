package httpjson

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid object", `{"name":"x"}`, false},
		{"empty body", ``, true},
		{"malformed JSON", `{"name":`, true},
		{"unknown field", `{"name":"x","extra":1}`, true},
		{"trailing object", `{"name":"x"}{}`, true},
		{"over 1 MB", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			var dst struct {
				Name string `json:"name"`
			}
			err := Decode(httptest.NewRecorder(), r, &dst)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusNotFound, "not_found", "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var got struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got.Error.Code != "not_found" || got.Error.Message != "resource not found" {
		t.Errorf("body = %s", w.Body)
	}
}

func TestWriteRawSendsBodyUnchanged(t *testing.T) {
	w := httptest.NewRecorder()
	WriteRaw(w, http.StatusCreated, []byte(`{"id":"1"}`))

	if w.Code != http.StatusCreated || w.Body.String() != `{"id":"1"}` {
		t.Errorf("got %d %s", w.Code, w.Body)
	}
}
