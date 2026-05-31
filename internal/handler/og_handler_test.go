package handler //nolint:testpackage

import (
	"context"
	"io"
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
