package main

import (
	"encoding/json"
	"html"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type headerRow struct {
	Name  string
	Value string
}

type pageData struct {
	Method      string
	Path        string
	Protocol    string
	HeaderCount int
	Rows        []headerRow
	JSONURL     string
	GeneratedAt string
}

func collect(r *http.Request) pageData {
	names := make([]string, 0, len(r.Header))
	for name := range r.Header {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]headerRow, 0, len(r.Header))
	for _, name := range names {
		for _, value := range r.Header[name] {
			rows = append(rows, headerRow{Name: name, Value: value})
		}
	}

	q := r.URL.Query()
	q.Set("format", "json")
	jsonURL := r.URL.Path
	if encoded := q.Encode(); encoded != "" {
		jsonURL += "?" + encoded
	}

	return pageData{
		Method:      r.Method,
		Path:        r.URL.Path,
		Protocol:    r.Proto,
		HeaderCount: len(r.Header),
		Rows:        rows,
		JSONURL:     jsonURL,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func headersContent(r *http.Request) template.HTML {
	d := collect(r)
	esc := html.EscapeString

	var b strings.Builder
	b.WriteString(`<div class="pageheader">`)
	b.WriteString(`<div><h1>HTTP Headers</h1>`)
	b.WriteString(`<p class="pageheader__desc">Live view of the headers received for this request</p></div>`)
	b.WriteString(`<a class="btn btn--primary" href="` + esc(d.JSONURL) + `">View JSON</a></div>`)

	b.WriteString(`<section class="kpis" aria-label="Request summary">`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Method</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--brand">` + esc(d.Method) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Path</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono" title="` + esc(d.Path) + `">` + esc(d.Path) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Protocol</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + esc(d.Protocol) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Headers</div>`)
	b.WriteString(`<div class="metric-card__value">` + strconv.Itoa(d.HeaderCount) + `</div></div>`)
	b.WriteString(`</section>`)

	b.WriteString(`<section class="card" aria-label="Received headers"><div class="table-wrap"><table>`)
	b.WriteString(`<thead><tr><th scope="col">Header Name</th><th scope="col">Value</th></tr></thead><tbody>`)
	if len(d.Rows) == 0 {
		b.WriteString(`<tr><td colspan="2"><div class="empty">`)
		b.WriteString(`<p class="empty__title">No headers received</p>`)
		b.WriteString(`<p class="empty__copy">Send a request with custom headers to see them echoed here.</p>`)
		b.WriteString(`</div></td></tr>`)
	} else {
		for _, row := range d.Rows {
			b.WriteString(`<tr><td class="name">` + esc(row.Name) + `</td><td class="value">` + esc(row.Value) + `</td></tr>`)
		}
	}
	b.WriteString(`</tbody></table></div></section>`)

	b.WriteString(`<div class="meta"><span>Generated ` + esc(d.GeneratedAt) + `</span></div>`)
	return template.HTML(b.String())
}

func jsonHeadersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	responseHeaders := make(map[string][]string, len(w.Header()))
	for name, values := range w.Header() {
		responseHeaders[name] = values
	}

	payload := map[string]any{
		"request": map[string]any{
			"method":   r.Method,
			"path":     r.URL.Path,
			"protocol": r.Proto,
			"host":     r.Host,
			"query":    r.URL.Query(),
			"headers":  r.Header,
		},
		"response": map[string]any{
			"status":  "200 OK",
			"headers": responseHeaders,
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logJSON(map[string]any{"level": "error", "msg": "encode failed", "error": err.Error()})
	}
}

func headersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if wantsJSON(r) {
		jsonHeadersHandler(w, r)
		return
	}
	renderShell(w, shellData{
		Title:             "HTTP Headers · EchoHeaders",
		Active:            "headers",
		BreadcrumbCurrent: "HTTP Headers",
		Body:              headersContent(r),
	})
}
