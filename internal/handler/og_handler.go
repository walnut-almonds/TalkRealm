package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

// OGData holds parsed Open Graph metadata.
type OGData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
	URL         string `json:"url"`
}

// ogMaxBodyBytes 限制解析的回應大小，避免惡意連結拖著伺服器讀無上限的 body。
const ogMaxBodyBytes = 2 << 20 // 2MB

// errBlockedHost URL 解析到內網／loopback 位址時回傳。
var errBlockedHost = errors.New("blocked host")

var ogHTTPClient = &http.Client{
	Timeout: 8 * time.Second,
	// 位址檢查放在 dial 階段：redirect 目標、DNS 名稱與各種 IP 編碼
	// （127.1、十進位 IP、::ffff:127.0.0.1 …）一次全涵蓋，
	// 只比對 URL 字串擋不住這些。
	Transport: &http.Transport{DialContext: safeDialContext},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}

		return nil
	},
}

// safeDialContext 先解析 host，再只對公開位址建立連線。
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	var dialer net.Dialer

	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			continue
		}

		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if dialErr == nil {
			return conn, nil
		}

		err = dialErr
	}

	if err != nil {
		return nil, err
	}

	return nil, errBlockedHost
}

// isPrivateIP 判斷位址是否屬於不該對外代訪的範圍（loopback / 私有網段 / link-local / multicast）。
func isPrivateIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}

	addr = addr.Unmap()

	// IsPrivate 只涵蓋 RFC1918 / ULA，不含 CGNAT（100.64.0.0/10）——
	// 雲端與電信業者的內部網段常落在那裡。
	if cgnat.Contains(addr) {
		return true
	}

	return !addr.IsValid() || addr.IsLoopback() || addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified()
}

// cgnat RFC 6598 shared address space。
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// GetOGPreview fetches Open Graph metadata for a given URL.
//
//	GET /api/v1/og?url=<encoded-url>
func GetOGPreview(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url query parameter is required"})
		return
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	// Block requests to private/loopback addresses to prevent SSRF.
	host := strings.ToLower(parsed.Hostname())
	if isPrivateHost(host) {
		c.JSON(http.StatusForbidden, gin.H{"error": "url not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	req.Header.Set("User-Agent", "TalkRealm-OGBot/1.0 (+https://talkrealm.app)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := ogHTTPClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch url"})
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		og := OGData{URL: rawURL}

		// Direct image URLs are common in chat messages; return a minimal preview
		// instead of treating them as an error.
		if strings.HasPrefix(strings.ToLower(contentType), "image/") {
			og.Image = rawURL
		}

		c.JSON(http.StatusOK, og)

		return
	}

	og := parseOGTags(resp, rawURL)
	c.JSON(http.StatusOK, og)
}

// parseOGTags extracts OG meta tags from the HTML response.
//
//nolint:gocognit
func parseOGTags(resp *http.Response, originalURL string) OGData {
	og := OGData{URL: originalURL}

	doc, err := html.Parse(io.LimitReader(resp.Body, ogMaxBodyBytes))
	if err != nil {
		return og
	}

	var walk func(*html.Node)

	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "meta":
				property, name, content := "", "", ""

				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "property":
						property = strings.ToLower(a.Val)
					case "name":
						name = strings.ToLower(a.Val)
					case "content":
						content = a.Val
					}
				}

				switch property {
				case "og:title":
					og.Title = content
				case "og:description":
					og.Description = content
				case "og:image":
					og.Image = resolveURL(content, originalURL)
				case "og:site_name":
					og.SiteName = content
				case "og:url":
					og.URL = content
				}

				// Fallback: standard <meta name="description">
				if name == "description" && og.Description == "" {
					og.Description = content
				}

			case "title":
				// Fallback: <title> tag
				if og.Title == "" && n.FirstChild != nil {
					og.Title = strings.TrimSpace(n.FirstChild.Data)
				}

			case "head":
				// Only walk head for performance; stop after </head>.
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}

				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	return og
}

// resolveURL resolves a potentially relative image URL against the page URL.
func resolveURL(imgURL, pageURL string) string {
	if imgURL == "" {
		return ""
	}

	parsed, err := url.Parse(imgURL)
	if err != nil {
		return imgURL
	}

	if parsed.IsAbs() {
		return imgURL
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return imgURL
	}

	return base.ResolveReference(parsed).String()
}

// isPrivateHost returns true for loopback, link-local, and RFC 1918 hostnames.
func isPrivateHost(host string) bool {
	private := []string{
		"localhost", "127.", "::1", "0.", "10.", "192.168.", "169.254.",
	}

	for _, p := range private {
		if host == p || strings.HasPrefix(host, p) {
			return true
		}
	}

	// 172.16.0.0/12
	if strings.HasPrefix(host, "172.") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			second, err := strconv.Atoi(parts[1])
			if err == nil && second >= 16 && second <= 31 {
				return true
			}
		}
	}

	return false
}
