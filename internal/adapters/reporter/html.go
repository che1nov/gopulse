package reporter

import (
	"fmt"
	"html"
	"io"
	"math"

	"github.com/che1nov/gopulse/internal/domain"
)

type HTML struct{}

func NewHTML() HTML {
	return HTML{}
}

func (r HTML) PrintSnapshot(w io.Writer, snapshot domain.Snapshot) error {
	if _, err := fmt.Fprint(w, htmlHeader("Benchmark results")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, `<section><h1>Benchmark results</h1><table><thead><tr><th>Benchmark</th><th>ns/op</th><th>B/op</th><th>allocs/op</th></tr></thead><tbody>`); err != nil {
		return err
	}
	for _, b := range snapshot.Benchmarks {
		if _, err := fmt.Fprintf(w, `<tr><td>%s</td><td>%.0f</td><td>%.0f</td><td>%.0f</td></tr>`+"\n", html.EscapeString(b.Name), b.NsPerOp, b.BytesPerOp, b.AllocsPerOp); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "</tbody></table></section></body></html>\n")
	return err
}

func (r HTML) PrintCheck(w io.Writer, result domain.CheckResult) error {
	if _, err := fmt.Fprint(w, htmlHeader("Performance report")); err != nil {
		return err
	}

	resultClass := "ok"
	resultText := "OK"
	if result.Failed {
		resultClass = "failed"
		resultText = "FAILED"
	}

	summary := summarize(result)
	if _, err := fmt.Fprintf(w, `<section><header><h1>Performance report</h1><p class="result %s">Result: %s</p></header><div class="summary"><div><strong>%d</strong><span>benchmarks</span></div><div><strong>%d</strong><span>metrics</span></div><div><strong>%d</strong><span>regressions</span></div></div>`, resultClass, resultText, summary.benchmarks, summary.metrics, summary.regressions); err != nil {
		return err
	}
	if result.Reason != "" {
		if _, err := fmt.Fprintf(w, `<p class="reason">%s</p>`, html.EscapeString(result.Reason)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, `<table><thead><tr><th>Benchmark</th><th>Metric</th><th>Baseline</th><th>Current</th><th>Change</th><th>Chart</th><th>Status</th></tr></thead><tbody>`); err != nil {
		return err
	}

	for _, cmp := range result.Comparisons {
		if cmp.Missing || cmp.New {
			status := "new"
			if cmp.Missing {
				status = "missing"
			}
			if _, err := fmt.Fprintf(w, `<tr><td>%s</td><td colspan="5"></td><td><span class="badge neutral">%s</span></td></tr>`+"\n", html.EscapeString(cmp.Name), status); err != nil {
				return err
			}
			continue
		}

		for _, metric := range cmp.Metrics {
			status := "OK"
			badgeClass := "ok"
			if metric.Regression {
				status = "REGRESSION"
				badgeClass = "failed"
			}
			if _, err := fmt.Fprintf(
				w,
				`<tr><td>%s</td><td>%s</td><td>%.0f</td><td>%.0f</td><td>%+0.1f%%</td><td>%s</td><td><span class="badge %s">%s</span></td></tr>`+"\n",
				html.EscapeString(cmp.Name),
				html.EscapeString(metric.Name),
				metric.Baseline,
				metric.Current,
				metric.ChangePct,
				changeBar(metric.ChangePct),
				badgeClass,
				status,
			); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprint(w, "</tbody></table></section></body></html>\n")
	return err
}

func htmlHeader(title string) string {
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(title) + `</title>
<style>
:root{color-scheme:light dark;--bg:#f7f8fb;--panel:#fff;--text:#18202f;--muted:#647084;--line:#d8dde8;--ok:#16833a;--bad:#c62828;--warn:#a15c00;--track:#e8ecf3}
body{margin:0;background:var(--bg);color:var(--text);font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
section{max-width:1120px;margin:32px auto;padding:0 20px}
header{display:flex;align-items:center;gap:16px;margin-bottom:16px}h1{font-size:28px;margin:0}
.summary{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin:0 0 18px}.summary div{background:var(--panel);border:1px solid var(--line);padding:14px}.summary strong{display:block;font-size:24px}.summary span{color:var(--muted);font-size:13px}
table{width:100%;border-collapse:collapse;background:var(--panel);border:1px solid var(--line)}
th,td{padding:12px 14px;border-bottom:1px solid var(--line);text-align:left;vertical-align:middle}
th{font-size:12px;text-transform:uppercase;letter-spacing:.04em;color:var(--muted)}
td:nth-child(3),td:nth-child(4),td:nth-child(5){font-variant-numeric:tabular-nums}
.result{display:inline-block;margin:0;padding:6px 10px;border-radius:6px;font-weight:700}
.result.ok{background:#e8f5ed;color:var(--ok)}.result.failed{background:#fdeaea;color:var(--bad)}
.reason{margin:0 0 16px;color:var(--muted)}
.badge{display:inline-block;padding:4px 8px;border-radius:999px;font-size:12px;font-weight:700}
.badge.ok{background:#e8f5ed;color:var(--ok)}.badge.failed{background:#fdeaea;color:var(--bad)}.badge.neutral{background:#edf0f6;color:var(--muted)}
.bar{display:grid;grid-template-columns:180px 64px;align-items:center;gap:10px;min-width:260px}.track{position:relative;width:180px;height:18px;background:var(--track);border-radius:4px;overflow:hidden}.track::after{content:"";position:absolute;left:50%;top:0;bottom:0;width:1px;background:var(--muted);opacity:.65}.fill{position:absolute;top:4px;height:10px;border-radius:999px;min-width:3px}.fill.ok{background:var(--ok)}.fill.bad{background:var(--bad)}.bar small{color:var(--muted);font-variant-numeric:tabular-nums}
@media (prefers-color-scheme:dark){:root{--bg:#11151d;--panel:#171c26;--text:#eef2f8;--muted:#9ca7ba;--line:#2a3241;--track:#252d3a}.result.ok,.badge.ok{background:#12331f}.result.failed,.badge.failed{background:#3a1616}.badge.neutral{background:#252d3a}}
</style>
</head>
<body>
`
}

func changeBar(changePct float64) string {
	width := math.Min(math.Abs(changePct), 100) / 2
	className := "ok"
	left := 50 - width
	if changePct > 0 {
		className = "bad"
		left = 50
	}
	label := fmt.Sprintf("%+0.1f%%", changePct)
	return `<div class="bar"><span class="track"><span class="fill ` + className + `" style="left:` + fmt.Sprintf("%.1f", left) + `%;width:` + fmt.Sprintf("%.1f", width) + `%"></span></span><small>` + html.EscapeString(label) + `</small></div>`
}

type reportSummary struct {
	benchmarks  int
	metrics     int
	regressions int
}

func summarize(result domain.CheckResult) reportSummary {
	var summary reportSummary
	for _, cmp := range result.Comparisons {
		summary.benchmarks++
		for _, metric := range cmp.Metrics {
			summary.metrics++
			if metric.Regression {
				summary.regressions++
			}
		}
	}
	return summary
}
