package main

import (
	"context"
	"encoding/json"
	"html"
	"html/template"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var domainRe = regexp.MustCompile(`^[A-Za-z0-9.*%_][A-Za-z0-9.*%_\-]*$`)

type dnsResult struct {
	Value   string
	Kind    string // "forward" | "reverse"
	OK      bool
	Output  string
	Err     string
	Records []string
}

func dnsContent(value string, res *dnsResult) template.HTML {
	esc := html.EscapeString

	var b strings.Builder
	b.WriteString(`<div class="pageheader">`)
	b.WriteString(`<div><h1>DNS Lookup</h1>`)
	b.WriteString(`<p class="pageheader__desc">Forward or reverse DNS lookup</p></div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<section class="card form-card">`)
	b.WriteString(`<form method="post" action="/dns" class="form">`)
	b.WriteString(`<div class="field">`)
	b.WriteString(`<label class="field__label" for="value">Domain or IP address <span class="req" aria-hidden="true">*</span></label>`)
	b.WriteString(`<input class="field__input" id="value" name="value" type="text" required autocomplete="off" placeholder="e.g. example.com or 1.1.1.1 or 2606:4700:4700::1111" value="` + esc(value) + `">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="actions"><button type="submit" class="btn btn--primary">Run Lookup</button></div>`)
	b.WriteString(`</form>`)
	b.WriteString(`</section>`)

	if res == nil {
		b.WriteString(`<section class="card"><div class="empty">`)
		b.WriteString(`<p class="empty__title">No results yet</p>`)
		b.WriteString(`<p class="empty__copy">Enter a domain or an IP address and run a lookup to see the records here.</p>`)
		b.WriteString(`</div></section>`)
		return template.HTML(b.String())
	}

	b.WriteString(`<section class="card result">`)
	b.WriteString(`<div class="result__head"><h2>Last Result</h2>`)
	if res.OK {
		b.WriteString(`<span class="chip chip--success">` + esc(res.Kind) + ` · ` + strconv.Itoa(len(res.Records)) + ` record(s)</span>`)
	} else {
		b.WriteString(`<span class="chip chip--danger">No result</span>`)
	}
	b.WriteString(`</div>`)

	if res.Err != "" {
		b.WriteString(`<div class="error"><p class="error__title">Lookup failed</p>`)
		b.WriteString(`<p class="error__copy">` + esc(res.Err) + `</p></div>`)
	}

	b.WriteString(`<div class="kpis kpis--inline">`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Query</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono" title="` + esc(res.Value) + `">` + esc(res.Value) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Type</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + esc(res.Kind) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Records</div>`)
	b.WriteString(`<div class="metric-card__value">` + strconv.Itoa(len(res.Records)) + `</div></div>`)
	b.WriteString(`</div>`)

	if res.Output != "" {
		b.WriteString(`<pre class="output">` + esc(res.Output) + `</pre>`)
	}
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

func dnsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderShell(w, shellData{
			Title:             "DNS Lookup · EchoHeaders",
			Active:            "dns",
			BreadcrumbCurrent: "DNS Lookup",
			Body:              dnsContent("", nil),
		})
		return
	}

	if !checkRateLimit(dnsLimiter, w, r, func(ra time.Duration) {
		renderShell(w, shellData{
			Title:             "DNS Lookup · EchoHeaders",
			Active:            "dns",
			BreadcrumbCurrent: "DNS Lookup",
			Status:            http.StatusTooManyRequests,
			BannerTitle:       "Rate limit exceeded",
			BannerCopy:        "Too many DNS lookups from your IP address. Try again in " + ra.String() + ".",
			Body:              dnsContent("", nil),
		})
	}) {
		return
	}

	_ = r.ParseForm()
	value := strings.TrimSpace(r.Form.Get("value"))

	var res *dnsResult
	switch {
	case value == "":
		res = &dnsResult{Value: value, Err: "Domain or IP address is required"}
	case net.ParseIP(value) != nil:
		res = runDNS(r, value, "reverse", []string{"+short", "-x", value})
	case len(value) <= 253 && domainRe.MatchString(value):
		res = runDNS(r, value, "forward", []string{"+short", value})
	default:
		res = &dnsResult{Value: value, Err: "Invalid input: expected a valid IP address or domain name"}
	}

	if wantsJSON(r) {
		dnsJSON(w, res)
	} else {
		renderShell(w, shellData{
			Title:             "DNS Lookup · EchoHeaders",
			Active:            "dns",
			BreadcrumbCurrent: "DNS Lookup",
			Body:              dnsContent(res.Value, res),
		})
	}
}

func runDNS(r *http.Request, value, kind string, digArgs []string) *dnsResult {
	res := &dnsResult{Value: value, Kind: kind}

	timeout := 15 * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	out, err := runCmd(ctx, "dig", digArgs...)
	res.Output = strings.TrimSpace(out)

	if ctx.Err() == context.DeadlineExceeded {
		res.Err = "timed out after " + timeout.String()
	} else if err != nil {
		res.Err = err.Error()
	}

	for _, line := range strings.Split(res.Output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res.Records = append(res.Records, line)
		}
	}
	res.OK = res.Err == "" && len(res.Records) > 0
	return res
}

func dnsJSON(w http.ResponseWriter, res *dnsResult) {
	payload := map[string]any{
		"ok":      res.OK,
		"value":   res.Value,
		"type":    res.Kind,
		"output":  res.Output,
		"records": res.Records,
		"error":   res.Err,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logJSON(map[string]any{"level": "error", "msg": "encode failed", "error": err.Error()})
	}
}
