package httpapi_test

import (
	"chronograph/internal/httpapi"
	"chronograph/internal/realtime"
	"chronograph/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaticAppAndSPAFlight(t *testing.T) {
	h := httpapi.NewRouter(store.NewMemory(), realtime.NewHub(2))
	for _, path := range []string{"/", "/c/some-token", "/v/public-token"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "<html") {
			t.Fatalf("%s status=%d body=%q", path, w.Code, w.Body.String())
		}
	}
}
