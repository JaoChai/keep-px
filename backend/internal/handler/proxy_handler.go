package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ProxyHandler struct {
	httpClient *http.Client
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (h *ProxyHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		ErrorJSON(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid URL")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Pixlinks/1.0)")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		ErrorJSON(w, http.StatusBadGateway, "failed to fetch page")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorJSON(w, http.StatusBadGateway, "failed to read page")
		return
	}

	html := string(body)

	// Remove X-Frame-Options and CSP headers that block iframe
	// Inject postMessage bridge script before </body>
	bridgeScript := `<script>
(function() {
  window.addEventListener('message', function(e) {
    if (e.data && e.data.type === 'pixlinks:activate-setup') {
      if (window.Pixlinks && window.Pixlinks.VisualSetup) {
        var vs = new window.Pixlinks.VisualSetup({ apiKey: '' });
        vs.activate();
      }
    }
  });
})();
</script>`

	html = strings.Replace(html, "</body>", bridgeScript+"</body>", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Remove restrictive headers
	w.Header().Del("X-Frame-Options")
	w.Header().Del("Content-Security-Policy")
	fmt.Fprint(w, html)
}
