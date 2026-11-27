package http

import (
	"html/template"
	nethttp "net/http"
	"path/filepath"
	"strings"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

const bannerText = "Deleting the local file can delete the synced copy. This check is only a path-segment hint."

var reviewTemplate = template.Must(template.New("review").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>hdsort</title></head>
<body>
{{if .Banner}}<p id="sync-banner">{{.Banner}}</p>{{end}}
<section id="exact">
<h2>Exact duplicates</h2>
{{range .Exact}}
{{if .HasRecommendation}}
<p>Keeper: {{.Keeper}}</p>
<p>Reason: {{.Reason}}</p>
{{range .Trash}}<p>Delete: {{.}}</p>{{end}}
{{else}}
{{range .Members}}<p>{{.Path}}</p>{{end}}
{{end}}
{{end}}
</section>
<section id="similar">
<h2>Similar files</h2>
{{range .Similar}}
{{range .Members}}<p>{{.Path}}</p>{{end}}
{{end}}
</section>
{{if .Report}}
<p>Nothing was changed.</p>
{{range .Report.Trash}}<p>{{.}}</p>{{end}}
{{end}}
{{if .ID}}
<form method="post" action="/scans/{{.ID}}/dry-run"><button type="submit">Dry run</button></form>
<form method="post" action="/scans/{{.ID}}/apply"><button type="submit">Apply</button></form>
{{end}}
</body>
</html>
`))

type reviewPage struct {
	ID      string
	Banner  string
	Exact   []pageExact
	Similar []pageSimilar
	Report  *app.PlanReport
}

type pageExact struct {
	HasRecommendation bool
	Keeper            string
	Reason            string
	Trash             []string
	Members           []pageMember
}

type pageSimilar struct {
	Members []pageMember
}

type pageMember struct {
	Path string
}

func (s *Server) writeReview(w nethttp.ResponseWriter, status int, id string, session domain.Session, report *app.PlanReport) {
	page := reviewPage{ID: id, Report: report}
	if syncedRoot(session.Root) {
		page.Banner = bannerText
	}
	for _, group := range session.Exact {
		item := pageExact{}
		if rec, ok := group.Recommendation(); ok {
			item.HasRecommendation = true
			item.Keeper = rec.Keeper
			item.Reason = string(rec.Reason)
			item.Trash = rec.Trash
		}
		for _, member := range group.Members() {
			item.Members = append(item.Members, pageMember{Path: member.Path})
		}
		page.Exact = append(page.Exact, item)
	}
	for _, group := range session.Similar {
		item := pageSimilar{}
		for _, member := range group.Members() {
			item.Members = append(item.Members, pageMember{Path: member.Path})
		}
		page.Similar = append(page.Similar, item)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = reviewTemplate.Execute(w, page)
}

func syncedRoot(root string) bool {
	names := map[string]struct{}{
		"CloudStorage":     {},
		"Mobile Documents": {},
		"Dropbox":          {},
		"Google Drive":     {},
	}
	for _, part := range strings.Split(filepath.Clean(root), string(filepath.Separator)) {
		if _, ok := names[part]; ok {
			return true
		}
	}
	return false
}
