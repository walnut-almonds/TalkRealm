package handler //nolint:testpackage

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetOGPreview_ImageContentTypeReturnsMinimalPreview(t *testing.T) {
	targetURL := "https://example.com/avatar"

	prevClient := ogHTTPClient
	ogHTTPClient = &http.Client{
		Timeout: 3 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, targetURL, req.URL.String())

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
				Body:       io.NopCloser(strings.NewReader("fake-jpeg")),
				Request:    req,
			}, nil
		}),
	}

	t.Cleanup(func() { ogHTTPClient = prevClient })

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/og", GetOGPreview)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/og?url="+url.QueryEscape(targetURL),
		nil,
	)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(
		t,
		`{"title":"","description":"","image":"`+targetURL+`","site_name":"","url":"`+targetURL+`"}`,
		w.Body.String(),
	)
}

func TestGetOGPreview_NonHTMLNonImageReturnsEmptyPreview(t *testing.T) {
	targetURL := "https://example.com/data"

	prevClient := ogHTTPClient
	ogHTTPClient = &http.Client{
		Timeout: 3 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, targetURL, req.URL.String())

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Request:    req,
			}, nil
		}),
	}

	t.Cleanup(func() { ogHTTPClient = prevClient })

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/og", GetOGPreview)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/og?url="+url.QueryEscape(targetURL),
		nil,
	)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(
		t,
		`{"title":"","description":"","image":"","site_name":"","url":"`+targetURL+`"}`,
		w.Body.String(),
	)
}

func TestIsPrivateIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.1.2.3", "10.0.0.1", "172.16.0.1", "172.31.255.255",
		"192.168.1.1", "169.254.169.254", "0.0.0.0", "::1", "fe80::1",
		"::ffff:127.0.0.1", "::ffff:169.254.169.254", "fc00::1", "100.64.0.1",
	}
	for _, s := range blocked {
		require.True(t, isPrivateIP(net.ParseIP(s)), "expected %s to be blocked", s)
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "172.32.0.1", "100.128.0.1", "2606:4700:4700::1111"}
	for _, s := range allowed {
		require.False(t, isPrivateIP(net.ParseIP(s)), "expected %s to be allowed", s)
	}
}
