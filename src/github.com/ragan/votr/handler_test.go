package votr

import (
	"testing"
	"net/http"
	"net/http/httptest"
)

func TestRootHandlerStatusCodes(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RootHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTemporaryRedirect {
		t.Errorf("handler returned wrong status code: expected %v, got %v",
			http.StatusTemporaryRedirect, status)
	}

}

func BenchmarkRootHandler(b *testing.B) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RootHandler)
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rr, req)
	}
}
