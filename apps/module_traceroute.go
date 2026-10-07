package main

import (
	"context"
	"encoding/json"
	"html"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var hopLineRe = regexp.MustCompile(`^\s*(\d+)\s+(.*)$`)

type tracerouteStats struct {
	Hops        int    `json:"hops"`
	Destination string `json:"destination"`
}

type tracerouteResult struct {
	Target   string
	IP       string
	MaxHops  int
	OK       bool
	Output   string
	Err      string
	Stats    tracerouteStats
	HasStats bool
}

func parseTraceroute(output string, ip string) (stats tracerouteStats, reached bool) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if m := hopLineRe.FindStringSubmatch(line); m != nil {
			stats.Hops++
			stats.Destination = strings.TrimSpace(m[2])
		}
	}
	if stats.Destination != "" {
		reached = strings.Contains(stats.Destination, ip)
	}
	return stats, reached
}

func tracerouteContent(target string, maxHops int, res *tracerouteResult) template.HTML {
	esc := html.EscapeString

	var b strings.Builder
	b.WriteString(`<div class="pageheader">`)
	b.WriteString(`<div><h1>Traceroute</h1>`)
	b.WriteString(`<p class="pageheader__desc">Traceroute to a specific domain or IPv4 address.</p></div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<section class="card form-card">`)
	b.WriteString(`<form method="post" action="/traceroute" class="form">`)
	b.WriteString(`<div class="field">`)
	b.WriteString(`<label class="field__label" for="ip">Domain or IPv4 address.<span class="req" aria-hidden="true">*</span></label>`)
	b.WriteString(`<input class="field__input" id="ip" name="ip" type="text" required autocomplete="off" placeholder="e.g. example.com or 1.1.1.1" value="` + esc(target) + `">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="field">`)
	b.WriteString(`<label class="field__label" for="maxhops">Max hops</label>`)
	b.WriteString(`<input class="field__input" id="maxhops" name="maxhops" type="number" min="1" max="60" value="` + strconv.Itoa(maxHops) + `">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="actions"><button type="submit" class="btn btn--primary">Run Traceroute</button></div>`)
	b.WriteString(`</form>`)
	b.WriteString(`</section>`)

	if res == nil {
		b.WriteString(`<section class="card"><div class="empty">`)
		b.WriteString(`<p class="empty__title">No results yet</p>`)
		b.WriteString(`<p class="empty__copy">Enter an domain or IPv4 address and run traceroute to see the path here.</p>`)
		b.WriteString(`</div></section>`)
		return template.HTML(b.String())
	}

	b.WriteString(`<section class="card result">`)
	b.WriteString(`<div class="result__head"><h2>Last Result</h2>`)
	if res.OK {
		b.WriteString(`<span class="chip chip--success">Destination reached</span>`)
	} else {
		b.WriteString(`<span class="chip chip--danger">Destination not reached</span>`)
	}
	b.WriteString(`</div>`)

	if res.Err != "" {
		b.WriteString(`<div class="error"><p class="error__title">Traceroute failed</p>`)
		b.WriteString(`<p class="error__copy">` + esc(res.Err) + `</p></div>`)
	}

	resolved := res.IP
	if resolved == "" {
		resolved = "—"
	}
	b.WriteString(`<div class="kpis kpis--inline">`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Target</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono" title="` + esc(res.Target) + `">` + esc(res.Target) + `</div></div>`)
	b.WriteString(`<div class="metric-card"><div class="metric-card__label">Resolved IP</div>`)
	b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + esc(resolved) + `</div></div>`)
	b.WriteString(`</div>`)

	if res.HasStats {
		b.WriteString(`<div class="kpis kpis--inline">`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Hops</div>`)
		b.WriteString(`<div class="metric-card__value">` + strconv.Itoa(res.Stats.Hops) + `</div></div>`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Final Hop</div>`)
		b.WriteString(`<div class="metric-card__value metric-card__value--mono" title="` + esc(res.Stats.Destination) + `">` + esc(res.Stats.Destination) + `</div></div>`)
		b.WriteString(`</div>`)
	}

	if res.Output != "" {
		b.WriteString(`<pre class="output">` + esc(res.Output) + `</pre>`)
	}
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

func tracerouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderShell(w, shellData{
			Title:             "Traceroute · EchoHeaders",
			Active:            "traceroute",
			BreadcrumbCurrent: "Traceroute",
			Body:              tracerouteContent("", 30, nil),
		})
		return
	}

	if !checkRateLimit(tracerouteLimiter, w, r, func(ra time.Duration) {
		renderShell(w, shellData{
			Title:             "Traceroute · EchoHeaders",
			Active:            "traceroute",
			BreadcrumbCurrent: "Traceroute",
			Status:            http.StatusTooManyRequests,
			BannerTitle:       "Rate limit exceeded",
			BannerCopy:        "Too many traceroute executions from your IP address. Try again in " + ra.String() + ".",
			Body:              tracerouteContent("", 30, nil),
		})
	}) {
		return
	}

	_ = r.ParseForm()
	target := strings.TrimSpace(r.Form.Get("ip"))
	maxHops := clampInt(r.Form.Get("maxhops"), 1, 60, 30)

	if target == "" {
		tracerouteRespond(w, r, &tracerouteResult{MaxHops: maxHops, Err: "Domain or IPv4 address is required"})
		return
	}

	timeout := 10*time.Second + 3*time.Second*time.Duration(maxHops)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	ip, err := resolveTarget(ctx, target)
	if err != nil {
		tracerouteRespond(w, r, &tracerouteResult{Target: target, MaxHops: maxHops, Err: err.Error()})
		return
	}

	out, err := runCmd(ctx, "traceroute", "-n", "-w", "2", "-q", "1", "-m", strconv.Itoa(maxHops), ip)

	res := &tracerouteResult{Target: target, IP: ip, MaxHops: maxHops, Output: strings.TrimSpace(out)}
	if ctx.Err() == context.DeadlineExceeded {
		res.Err = "timed out after " + timeout.String()
	} else if err != nil {
		res.Err = err.Error()
	}
	res.Stats, res.OK = parseTraceroute(out, ip)
	res.HasStats = res.Stats.Hops > 0
	if ctx.Err() != nil || err != nil {
		res.OK = false
	}

	tracerouteRespond(w, r, res)
}

func tracerouteRespond(w http.ResponseWriter, r *http.Request, res *tracerouteResult) {
	if wantsJSON(r) {
		tracerouteJSON(w, res)
	} else {
		renderShell(w, shellData{
			Title:             "Traceroute · EchoHeaders",
			Active:            "traceroute",
			BreadcrumbCurrent: "Traceroute",
			Body:              tracerouteContent(res.Target, res.MaxHops, res),
		})
	}
}

func tracerouteJSON(w http.ResponseWriter, res *tracerouteResult) {
	payload := map[string]any{
		"ok":          res.OK,
		"target":      res.Target,
		"resolved_ip": res.IP,
		"ip":          res.IP,
		"max_hops":    res.MaxHops,
		"output":      res.Output,
		"error":       res.Err,
		"stats":       res.Stats,
		"has_stats":   res.HasStats,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logJSON(map[string]any{"level": "error", "msg": "encode failed", "error": err.Error()})
	}
}
