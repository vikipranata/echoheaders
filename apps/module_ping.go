package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func resolveTarget(ctx context.Context, input string) (string, error) {
	if parsed := net.ParseIP(input); parsed != nil {
		return parsed.String(), nil
	}
	if len(input) > 253 || !domainRe.MatchString(input) {
		return "", fmt.Errorf("invalid input: expected a valid IPv4 address or domain name")
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, input)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s: %v", input, err)
	}
	for _, a := range addrs {
		if a.IP.To4() != nil {
			return a.IP.String(), nil
		}
	}
	if len(addrs) > 0 {
		return addrs[0].IP.String(), nil
	}
	return "", fmt.Errorf("no addresses found for %s", input)
}

func clampInt(s string, min, max, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var (
	pingLossRe = regexp.MustCompile(`(\d+) packets? transmitted, (\d+) received, ([\d.]+)% packet loss`)
	pingRttRe  = regexp.MustCompile(`(?:rtt|round-trip) min/avg/max(?:/mdev)? = ([\d.]+)/([\d.]+)/([\d.]+)`)
)

type pingStats struct {
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	LossPct  float64 `json:"loss_pct"`
	Min      float64 `json:"min_ms"`
	Avg      float64 `json:"avg_ms"`
	Max      float64 `json:"max_ms"`
}

type pingResult struct {
	Target   string
	IP       string
	Count    int
	OK       bool
	Output   string
	Err      string
	Stats    pingStats
	HasStats bool
}

func parsePing(output string) (stats pingStats, hasRTT bool) {
	if m := pingLossRe.FindStringSubmatch(output); m != nil {
		stats.Sent, _ = strconv.Atoi(m[1])
		stats.Received, _ = strconv.Atoi(m[2])
		stats.LossPct, _ = strconv.ParseFloat(m[3], 64)
	}
	if m := pingRttRe.FindStringSubmatch(output); m != nil {
		stats.Min, _ = strconv.ParseFloat(m[1], 64)
		stats.Avg, _ = strconv.ParseFloat(m[2], 64)
		stats.Max, _ = strconv.ParseFloat(m[3], 64)
		hasRTT = true
	}
	return stats, hasRTT
}

func pingContent(target string, count int, res *pingResult) template.HTML {
	esc := html.EscapeString

	var b strings.Builder
	b.WriteString(`<div class="pageheader">`)
	b.WriteString(`<div><h1>Ping</h1>`)
	b.WriteString(`<p class="pageheader__desc">ICMP ping to a specific domain or IPv4 address.</p></div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<section class="card form-card">`)
	b.WriteString(`<form method="post" action="/ping" class="form">`)
	b.WriteString(`<div class="field">`)
	b.WriteString(`<label class="field__label" for="ip">Domain or IPv4 address.<span class="req" aria-hidden="true">*</span></label>`)
	b.WriteString(`<input class="field__input" id="ip" name="ip" type="text" required autocomplete="off" placeholder="e.g. example.com or 1.1.1.1" value="` + esc(target) + `">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="field">`)
	b.WriteString(`<label class="field__label" for="count">Count</label>`)
	b.WriteString(`<input class="field__input" id="count" name="count" type="number" min="1" max="20" value="` + strconv.Itoa(count) + `">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="actions"><button type="submit" class="btn btn--primary">Run Ping</button></div>`)
	b.WriteString(`</form>`)
	b.WriteString(`</section>`)

	if res == nil {
		b.WriteString(`<section class="card"><div class="empty">`)
		b.WriteString(`<p class="empty__title">No results yet</p>`)
		b.WriteString(`<p class="empty__copy">Enter an domain or IPv4 address and run ping to see the output here.</p>`)
		b.WriteString(`</div></section>`)
		return template.HTML(b.String())
	}

	b.WriteString(`<section class="card result">`)
	b.WriteString(`<div class="result__head"><h2>Last Result</h2>`)
	if res.OK {
		b.WriteString(`<span class="chip chip--success">Reachable</span>`)
	} else {
		b.WriteString(`<span class="chip chip--danger">Unreachable</span>`)
	}
	b.WriteString(`</div>`)

	if res.Err != "" {
		b.WriteString(`<div class="error"><p class="error__title">Ping failed</p>`)
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
		s := res.Stats
		b.WriteString(`<div class="kpis kpis--inline">`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Min RTT</div>`)
		b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + strconv.FormatFloat(s.Min, 'f', 3, 64) + ` ms</div></div>`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Avg RTT</div>`)
		b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + strconv.FormatFloat(s.Avg, 'f', 3, 64) + ` ms</div></div>`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Max RTT</div>`)
		b.WriteString(`<div class="metric-card__value metric-card__value--mono">` + strconv.FormatFloat(s.Max, 'f', 3, 64) + ` ms</div></div>`)
		b.WriteString(`<div class="metric-card"><div class="metric-card__label">Loss</div>`)
		b.WriteString(`<div class="metric-card__value">` + strconv.FormatFloat(s.LossPct, 'f', 1, 64) + `%</div></div>`)
		b.WriteString(`</div>`)
	}

	if res.Output != "" {
		b.WriteString(`<pre class="output">` + esc(res.Output) + `</pre>`)
	}
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderShell(w, shellData{
			Title:             "Ping · EchoHeaders",
			Active:            "ping",
			BreadcrumbCurrent: "Ping",
			Body:              pingContent("", 3, nil),
		})
		return
	}

	if !checkRateLimit(pingLimiter, w, r, func(ra time.Duration) {
		renderShell(w, shellData{
			Title:             "Ping · EchoHeaders",
			Active:            "ping",
			BreadcrumbCurrent: "Ping",
			Status:            http.StatusTooManyRequests,
			BannerTitle:       "Rate limit exceeded",
			BannerCopy:        "Too many ping executions from your IP address. Try again in " + ra.String() + ".",
			Body:              pingContent("", 3, nil),
		})
	}) {
		return
	}

	_ = r.ParseForm()
	target := strings.TrimSpace(r.Form.Get("ip"))
	count := clampInt(r.Form.Get("count"), 1, 20, 3)

	if target == "" {
		pingRespond(w, r, &pingResult{Count: count, Err: "Domain or IPv4 address is required"})
		return
	}

	timeout := 4*time.Second + 3*time.Second*time.Duration(count)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	ip, err := resolveTarget(ctx, target)
	if err != nil {
		pingRespond(w, r, &pingResult{Target: target, Count: count, Err: err.Error()})
		return
	}

	out, err := runCmd(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2", ip)

	res := &pingResult{Target: target, IP: ip, Count: count, Output: strings.TrimSpace(out)}
	if ctx.Err() == context.DeadlineExceeded {
		res.Err = "timed out after " + timeout.String()
	} else if err != nil {
		res.Err = err.Error()
	}
	res.Stats, res.HasStats = parsePing(out)
	res.OK = res.Err == "" && res.Stats.Received > 0

	pingRespond(w, r, res)
}

func pingRespond(w http.ResponseWriter, r *http.Request, res *pingResult) {
	if wantsJSON(r) {
		pingJSON(w, res)
	} else {
		renderShell(w, shellData{
			Title:             "Ping · EchoHeaders",
			Active:            "ping",
			BreadcrumbCurrent: "Ping",
			Body:              pingContent(res.Target, res.Count, res),
		})
	}
}

func pingJSON(w http.ResponseWriter, res *pingResult) {
	payload := map[string]any{
		"ok":          res.OK,
		"target":      res.Target,
		"resolved_ip": res.IP,
		"ip":          res.IP,
		"count":       res.Count,
		"output":      res.Output,
		"error":       res.Err,
		"stats":       res.Stats,
		"has_rtt":     res.HasStats,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logJSON(map[string]any{"level": "error", "msg": "encode failed", "error": err.Error()})
	}
}
