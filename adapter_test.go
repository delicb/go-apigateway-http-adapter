package apigateway_adapter

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProxyResponseWriter_Write(t *testing.T) {
	w := newProxyResponseWriter()
	_, err := w.Write([]byte("something"))
	if err != nil {
		t.Errorf("unexpected error when writing to proxyResponseWriter: %v", err)
	}
	if w.status == 0 {
		t.Errorf("writer status code still uninitialized even after write")
	}
}

func TestProxyResponseWriter_WriteHeader(t *testing.T) {
	w := newProxyResponseWriter()
	w.WriteHeader(http.StatusNotFound)
	if w.status != http.StatusNotFound {
		t.Errorf("writer status unexpected, got: %v, expected: %v", w.status, http.StatusNotFound)
	}
}

func TestProxyResponseWriter_toApiGatewayResponse(t *testing.T) {
	w := newProxyResponseWriter()
	w.Header().Set("h1", "v1")
	w.Header().Add("multi", "v1")
	w.Header().Add("multi", "v2")
	w.Header().Add("Set-Cookie", "c1=val1")
	w.Header().Add("Set-Cookie", "c2=val2")

	_, err := w.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected error on proxy writer write: %v", err)
	}

	resp := w.toApiGatewayResponse()

	t.Log(resp)
	if resp.IsBase64Encoded == true {
		t.Error("Base64Encoding is set to true, expected false")
	}
	if resp.Headers["H1"] != "v1" {
		t.Errorf("wrong value for header %q, expected %q, got %q", "h1", "v1", resp.Headers["h1"])
	}

	expectMultiValueHeaders := []string{"v1", "v2"}
	if !compareIgnoreOrder(resp.MultiValueHeaders["Multi"], expectMultiValueHeaders) {
		t.Errorf("unexpected multi value headers, got %v, expected %v", resp.MultiValueHeaders["Multi"], expectMultiValueHeaders)
	}

	expectCookies := []string{"c2=val2", "c1=val1"}
	if !compareIgnoreOrder(resp.Cookies, expectCookies) {
		t.Errorf("unexpected value for cookies, got: %v, expected: %v", resp.Cookies, expectCookies)
	}
}

func compareIgnoreOrder(s1, s2 []string) bool {
	return cmp.Equal(s1, s2, cmp.Transformer("sliceToMap", func(in []string) map[string]struct{} {
		m := make(map[string]struct{})
		for _, v := range in {
			m[v] = struct{}{}
		}
		return m
	}))
}
