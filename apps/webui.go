package main

import (
	"html/template"
	"net/http"
	"strings"
)

const sharedCSS = `
:root {
  --style-font-sans: "Plus Jakarta Sans", Inter, system-ui, sans-serif;
  --style-font-mono: "JetBrains Mono", SFMono-Regular, Consolas, monospace;

  --style-bg-app: #F8FAFC;
  --style-bg-surface: #FFFFFF;
  --style-bg-surface-muted: #F1F5F9;
  --style-bg-brand-soft: #E9FBF6;
  --style-row-hover: #F8FAFC;

  --style-text-primary: #0F172A;
  --style-text-secondary: #475569;
  --style-text-muted: #64748B;
  --style-text-inverse: #FFFFFF;
  --style-text-brand: #0f556f;

  --style-border-default: #E2E8F0;
  --style-border-strong: #CBD5E1;
  --style-border-brand: #259dcc;

  --style-action-primary: #0f556f;
  --style-action-primary-hover: #0b595d;
  --style-action-primary-active: #083B30;

  --style-status-info-soft: #EFF6FF;
  --style-status-info-text: #1D4ED8;
  --style-status-danger: #DC2626;
  --style-status-danger-soft: #FEF2F2;
  --style-status-danger-text: #B91C1C;

  --style-spacing-2: 8px;
  --style-spacing-3: 12px;
  --style-spacing-4: 16px;
  --style-spacing-5: 20px;
  --style-spacing-6: 24px;
  --style-spacing-8: 32px;
  --style-spacing-12: 48px;

  --style-radius-md: 8px;
  --style-radius-lg: 12px;
  --style-radius-full: 9999px;

  --style-shadow-sm: 0 1px 2px rgba(15, 23, 42, 0.06);
  --style-shadow-md: 0 4px 12px rgba(15, 23, 42, 0.08);

  --style-topbar-height: 64px;
  --style-page-padding: 24px;
  --style-sidebar-width: 240px;
  --style-row-height: 48px;
  --style-button-height: 40px;

  --style-text-xs: 12px;
  --style-text-sm: 14px;
  --style-text-base: 16px;
  --style-text-xl: 20px;
  --style-text-2xl: 24px;
  --style-leading-tight: 1.25;
  --style-leading-normal: 1.5;

  --style-motion: 180ms cubic-bezier(0.2, 0, 0, 1);
}

* { box-sizing: border-box; }

body {
  margin: 0;
  background: var(--style-bg-app);
  color: var(--style-text-primary);
  font-family: var(--style-font-sans);
  font-size: var(--style-text-base);
  line-height: var(--style-leading-normal);
  -webkit-font-smoothing: antialiased;
}

/* TopCommandBar */
.topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--style-topbar-height);
  padding: 0 var(--style-page-padding);
  background: var(--style-bg-surface);
  border-bottom: 1px solid var(--style-border-default);
}
.topbar__left { display: flex; align-items: center; gap: var(--style-spacing-2); }
.topbar__brand { font-size: var(--style-text-base); font-weight: 700; letter-spacing: -0.01em; }
.topbar__brand em { font-style: normal; color: var(--style-text-brand); }

/* Hamburger (CSS-drawn, no icon font) */
.hamburger {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: var(--style-button-height);
  height: var(--style-button-height);
  padding: 0 var(--style-spacing-3);
  background: transparent;
  border: none;
  border-radius: var(--style-radius-md);
  cursor: pointer;
  transition: background var(--style-motion);
}
.hamburger:hover { background: var(--style-bg-surface-muted); }
.hamburger:focus-visible { outline: none; box-shadow: 0 0 0 3px rgba(37, 157, 220, 0.32); }
.hamburger__line {
  display: block;
  width: 100%;
  height: 2px;
  border-radius: 1px;
  background: var(--style-text-secondary);
  transition: transform var(--style-motion), opacity var(--style-motion);
}
.shell[data-menu="open"] .hamburger__line:nth-child(1) { transform: translateY(6px) rotate(45deg); }
.shell[data-menu="open"] .hamburger__line:nth-child(2) { opacity: 0; }
.shell[data-menu="open"] .hamburger__line:nth-child(3) { transform: translateY(-6px) rotate(-45deg); }

/* AppShell: sidebar + main */
.shell { display: flex; align-items: stretch; }
.sidebar {
  position: sticky;
  top: var(--style-topbar-height);
  width: var(--style-sidebar-width);
  flex-shrink: 0;
  height: calc(100vh - var(--style-topbar-height));
  padding: var(--style-spacing-4) var(--style-spacing-3);
  background: var(--style-bg-surface);
  border-right: 1px solid var(--style-border-default);
}
.sidebar__nav { display: flex; flex-direction: column; gap: var(--style-spacing-1, 4px); }
.nav-item {
  display: block;
  padding: var(--style-spacing-2) var(--style-spacing-3);
  border-radius: var(--style-radius-md);
  font-size: var(--style-text-sm);
  font-weight: 500;
  color: var(--style-text-secondary);
  text-decoration: none;
  transition: background var(--style-motion), color var(--style-motion);
}
.nav-item:hover { background: var(--style-bg-surface-muted); color: var(--style-text-primary); }
.nav-item.is-active { background: var(--style-bg-brand-soft); color: var(--style-text-brand); font-weight: 600; }

/* Menu open/close (hamburger): desktop defaults open, mobile defaults closed */
.shell[data-menu="closed"] .sidebar { display: none; }
@media (max-width: 900px) {
  .sidebar {
    position: fixed;
    top: var(--style-topbar-height);
    left: 0;
    bottom: 0;
    z-index: 20;
    height: auto;
    overflow-y: auto;
    box-shadow: var(--style-shadow-md);
  }
  .shell:not([data-menu="open"]) .sidebar { display: none; }
}

.main {
  flex: 1;
  min-width: 0;
  max-width: 1440px;
  padding: var(--style-page-padding);
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--style-spacing-2);
  margin-bottom: var(--style-spacing-3);
  font-size: var(--style-text-sm);
}
.breadcrumb a { color: var(--style-text-muted); text-decoration: none; transition: color var(--style-motion); }
.breadcrumb a:hover { color: var(--style-text-brand); text-decoration: underline; }
.breadcrumb__sep { color: var(--style-border-strong); }
.breadcrumb__current { color: var(--style-text-secondary); font-weight: 500; }

/* PageHeader */
.pageheader {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--style-spacing-4);
  margin-bottom: var(--style-spacing-6);
}
.pageheader h1 {
  margin: 0 0 4px;
  font-size: var(--style-text-2xl);
  font-weight: 700;
  line-height: var(--style-leading-tight);
  letter-spacing: -0.02em;
}
.pageheader__desc {
  margin: 0;
  max-width: 640px;
  font-size: var(--style-text-sm);
  color: var(--style-text-secondary);
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--style-button-height);
  padding: 0 var(--style-spacing-4);
  border-radius: var(--style-radius-md);
  font-family: var(--style-font-sans);
  font-size: var(--style-text-sm);
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  border: none;
  transition: background var(--style-motion), border-color var(--style-motion);
}
.btn--primary {
  background: var(--style-action-primary);
  color: var(--style-text-inverse);
}
.btn--primary:hover { background: var(--style-action-primary-hover); }
.btn--primary:active { background: var(--style-action-primary-active); }
.btn--primary:focus-visible { outline: none; box-shadow: 0 0 0 3px rgba(37, 157, 220, 0.32); }

/* MetricCards */
.kpis {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--style-spacing-4);
  margin-bottom: var(--style-spacing-6);
}
.kpis--inline { margin-bottom: var(--style-spacing-4); }
.metric-card {
  padding: var(--style-spacing-4);
  background: var(--style-bg-surface);
  border: 1px solid var(--style-border-default);
  border-radius: var(--style-radius-lg);
  box-shadow: var(--style-shadow-sm);
}
.metric-card__label {
  margin-bottom: var(--style-spacing-2);
  font-size: var(--style-text-xs);
  font-weight: 600;
  letter-spacing: 0.6px;
  text-transform: uppercase;
  color: var(--style-text-muted);
}
.metric-card__value {
  font-size: var(--style-text-xl);
  font-weight: 700;
  line-height: var(--style-leading-tight);
}
.metric-card__value--brand { color: var(--style-text-brand); }
.metric-card__value--mono {
  font-family: var(--style-font-mono);
  font-size: var(--style-text-base);
  font-weight: 500;
  word-break: break-all;
}

/* Card */
.card {
  background: var(--style-bg-surface);
  border: 1px solid var(--style-border-default);
  border-radius: var(--style-radius-lg);
  box-shadow: var(--style-shadow-sm);
  overflow: hidden;
  margin-bottom: var(--style-spacing-6);
}
.card:last-child { margin-bottom: 0; }

/* DataTable */
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
thead th {
  padding: 10px var(--style-spacing-4);
  background: var(--style-bg-surface-muted);
  border-bottom: 1px solid var(--style-border-default);
  text-align: left;
  font-size: var(--style-text-xs);
  font-weight: 600;
  letter-spacing: 0.6px;
  text-transform: uppercase;
  color: var(--style-text-secondary);
  white-space: nowrap;
}
tbody tr { height: var(--style-row-height); transition: background var(--style-motion); }
tbody tr:hover { background: var(--style-row-hover); }
tbody td {
  padding: var(--style-spacing-3) var(--style-spacing-4);
  border-bottom: 1px solid var(--style-border-default);
  font-size: var(--style-text-sm);
  vertical-align: middle;
}
tbody tr:last-child td { border-bottom: none; }
td.name {
  font-family: var(--style-font-mono);
  font-weight: 500;
  white-space: nowrap;
}
td.value {
  font-family: var(--style-font-mono);
  color: var(--style-text-secondary);
  word-break: break-all;
}

/* Form (FormField + Input) */
.form-card { padding: var(--style-spacing-6); }
.form { display: flex; flex-direction: column; gap: var(--style-spacing-4); max-width: 480px; }
.field { display: flex; flex-direction: column; gap: var(--style-spacing-2); }
.field__label { font-size: var(--style-text-sm); font-weight: 500; }
.field__label .req { color: var(--style-status-danger-text); }
.field input {
  height: var(--style-button-height);
  padding: 0 var(--style-spacing-3);
  background: var(--style-bg-surface);
  border: 1px solid var(--style-border-strong);
  border-radius: var(--style-radius-md);
  font-family: var(--style-font-mono);
  font-size: var(--style-text-sm);
  color: var(--style-text-primary);
  transition: border-color var(--style-motion), box-shadow var(--style-motion);
}
.field input::placeholder { color: var(--style-text-muted); font-family: var(--style-font-sans); }
.field input:focus {
  outline: none;
  border-color: var(--style-border-brand);
  box-shadow: 0 0 0 3px rgba(37, 157, 220, 0.16);
}
.actions { display: flex; gap: var(--style-spacing-2); margin-top: var(--style-spacing-2); }

/* Result + status chips */
.result { padding: var(--style-spacing-6); }
.result__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--style-spacing-2);
  margin-bottom: var(--style-spacing-4);
}
.result__head h2 { margin: 0; font-size: var(--style-text-xl); font-weight: 700; line-height: var(--style-leading-tight); }
.chip {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: var(--style-radius-full);
  font-size: var(--style-text-xs);
  font-weight: 600;
  white-space: nowrap;
}
.chip--success { background: var(--style-bg-brand-soft); color: var(--style-text-brand); }
.chip--danger { background: var(--style-status-danger-soft); color: var(--style-status-danger-text); }

/* ErrorState */
.error {
  margin-bottom: var(--style-spacing-4);
  padding: var(--style-spacing-4);
  background: var(--style-status-danger-soft);
  border: 1px solid var(--style-border-default);
  border-left: 3px solid var(--style-status-danger);
  border-radius: var(--style-radius-md);
}
.error__title { margin: 0 0 4px; font-weight: 600; }
.error__copy { margin: 0; font-size: var(--style-text-sm); color: var(--style-text-secondary); word-break: break-word; }

/* Terminal output */
.output {
  margin: 0;
  padding: var(--style-spacing-4);
  background: var(--style-bg-surface-muted);
  border-radius: var(--style-radius-md);
  font-family: var(--style-font-mono);
  font-size: var(--style-text-xs);
  line-height: 1.6;
  color: var(--style-text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

/* EmptyState */
.empty { padding: var(--style-spacing-12) var(--style-spacing-6); text-align: center; }
.empty__title { margin: 0 0 4px; font-weight: 600; }
.empty__copy { margin: 0; font-size: var(--style-text-sm); color: var(--style-text-muted); }

/* Footer meta */
.meta {
  display: flex;
  justify-content: space-between;
  gap: var(--style-spacing-2);
  margin-top: var(--style-spacing-3);
  font-size: var(--style-text-xs);
  color: var(--style-text-muted);
}
`

const shellTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>{{.CSS}}</style>
</head>
<body>
<header class="topbar">
  <div class="topbar__left">
    <button class="hamburger" type="button" aria-label="Toggle module menu" aria-expanded="true" aria-controls="module-menu">
      <span class="hamburger__line"></span>
      <span class="hamburger__line"></span>
      <span class="hamburger__line"></span>
    </button>
    <div class="topbar__brand">Echo<em>Headers</em></div>
  </div>
</header>
<div class="shell">
  <aside class="sidebar" id="module-menu">
    <nav class="sidebar__nav" aria-label="Modules">
      <a class="nav-item{{if eq .Active "headers"}} is-active{{end}}" href="/">HTTP Headers</a>
      <a class="nav-item{{if eq .Active "ping"}} is-active{{end}}" href="/ping">Ping</a>
      <a class="nav-item{{if eq .Active "traceroute"}} is-active{{end}}" href="/traceroute">Traceroute</a>
      <a class="nav-item{{if eq .Active "dns"}} is-active{{end}}" href="/dns">DNS Lookup</a>
    </nav>
  </aside>
  <main class="main">
    <nav class="breadcrumb" aria-label="Breadcrumb">
      <a href="/">Echo</a>
      <span class="breadcrumb__sep">/</span>
      <span class="breadcrumb__current">{{.BreadcrumbCurrent}}</span>
    </nav>
    {{if .BannerTitle}}
    <div class="error" role="alert">
      <p class="error__title">{{.BannerTitle}}</p>
      <p class="error__copy">{{.BannerCopy}}</p>
    </div>
    {{end}}
    {{.Body}}
  </main>
</div>
<script>
(function () {
  var shell = document.querySelector(".shell");
  var btn = document.querySelector(".hamburger");
  if (!shell || !btn) { return; }
  var mq = window.matchMedia("(min-width: 901px)");
  function setState(open) {
    shell.dataset.menu = open ? "open" : "closed";
    btn.setAttribute("aria-expanded", open ? "true" : "false");
  }
  function sync() { setState(mq.matches); }
  btn.addEventListener("click", function () {
    setState(shell.dataset.menu !== "open");
  });
  if (mq.addEventListener) { mq.addEventListener("change", sync); }
  else { mq.addListener(sync); }
  document.querySelectorAll(".nav-item").forEach(function (a) {
    a.addEventListener("click", function () {
      if (!mq.matches) { setState(false); }
    });
  });
  sync();
})();
</script>
</body>
</html>
`

var shell = template.Must(template.New("shell").Parse(shellTemplate))

type shellData struct {
	Title             string
	Active            string
	BreadcrumbCurrent string
	// Status is the HTTP status to emit (0 = 200).
	Status int
	// BannerTitle/BannerCopy render an alert banner under the breadcrumb.
	BannerTitle string
	BannerCopy  string
	// CSS must be template.CSS (not string): html/template refuses to emit
	// plain strings inside <style> and prints ZgotmplZ instead.
	CSS  template.CSS
	Body template.HTML
}

func renderShell(w http.ResponseWriter, d shellData) {
	if d.Status == 0 {
		d.Status = http.StatusOK
	}
	d.CSS = template.CSS(sharedCSS)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(d.Status)
	if err := shell.Execute(w, d); err != nil {
		logJSON(map[string]any{"level": "error", "msg": "render failed", "error": err.Error()})
	}
}

// wantsJSON reports whether the client asked for a JSON representation.
func wantsJSON(r *http.Request) bool {
	switch r.URL.Query().Get("format") {
	case "json":
		return true
	case "html":
		return false
	}
	return !strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html")
}
