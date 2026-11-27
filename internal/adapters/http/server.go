package http

import (
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"path/filepath"
	"strings"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

// Server is the loopback review UI. It calls use cases and does not hash
// files or write SQL.
type Server struct {
	token    string
	sessions app.SessionRepository
	library  app.LibraryRepository
	trash    app.Trasher
	disk     app.Relocator
	walk     app.Walker
	read     app.FileReader
	identify app.Identifier
	similar  app.Similarizer
	pick     app.FolderPicker
	mux      *nethttp.ServeMux
}

// New returns the review server. sessions and library may be the same store.
func New(token string, sessions app.SessionRepository, library app.LibraryRepository, trash app.Trasher, disk app.Relocator) *Server {
	s := &Server{
		token:    token,
		sessions: sessions,
		library:  library,
		trash:    trash,
		disk:     disk,
		mux:      nethttp.NewServeMux(),
	}
	s.mux.HandleFunc("GET /{$}", s.home)
	s.mux.HandleFunc("POST /scans", s.postScan)
	s.mux.HandleFunc("GET /scans/{id}", s.getSession)
	s.mux.HandleFunc("GET /scans/{id}/files/{fileID}/preview", s.preview)
	s.mux.HandleFunc("POST /scans/{id}/decisions", s.postDecision)
	s.mux.HandleFunc("POST /scans/{id}/dry-run", s.dryRun)
	s.mux.HandleFunc("POST /scans/{id}/apply", s.apply)
	return s
}

// WithScan wires the ports POST /scans and preview call.
// pick is the folder dialog. Tests pass a fake. cmd/hdsort passes the native dialog.
func (s *Server) WithScan(walk app.Walker, read app.FileReader, identify app.Identifier, similar app.Similarizer, pick app.FolderPicker) *Server {
	s.walk = walk
	s.read = read
	s.identify = identify
	s.similar = similar
	s.pick = pick
	return s
}

// ServeHTTP authenticates, then dispatches. The package name is http, so
// this method is the net/http handler.
func (s *Server) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !s.allow(w, r) {
		nethttp.Error(w, "unauthorized", nethttp.StatusUnauthorized)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) allow(w nethttp.ResponseWriter, r *nethttp.Request) bool {
	if r.Method == nethttp.MethodGet && r.URL.Path == "/" {
		if query := r.URL.Query().Get("token"); query != "" {
			if tokensEqual(query, s.token) {
				s.setCookie(w)
				return true
			}
			return false
		}
	}
	if bearer, ok := bearerToken(r); ok && tokensEqual(bearer, s.token) {
		return true
	}
	if cookie, err := r.Cookie("hdsort_token"); err == nil && tokensEqual(cookie.Value, s.token) {
		return true
	}
	return false
}

func bearerToken(r *nethttp.Request) (string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

func (s *Server) setCookie(w nethttp.ResponseWriter) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     "hdsort_token",
		Value:    s.token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: nethttp.SameSiteStrictMode,
	})
}

func (s *Server) home(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.URL.Query().Get("scan")
	if id == "" {
		s.writeReview(w, nethttp.StatusOK, "", domain.NewSession(""), nil)
		return
	}
	session, err := s.sessions.Load(id)
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	s.writeReview(w, nethttp.StatusOK, id, session, nil)
}

type scanBody struct {
	Path      string `json:"path"`
	ExactOnly bool   `json:"exactOnly"`
}

func (s *Server) postScan(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.walk == nil || s.read == nil || s.identify == nil || s.similar == nil || s.pick == nil {
		nethttp.Error(w, "scan is not configured", nethttp.StatusInternalServerError)
		return
	}
	body, err := decodeScan(r)
	if err != nil {
		nethttp.Error(w, "bad scan", nethttp.StatusBadRequest)
		return
	}
	root := body.Path
	if root == "" {
		root, err = s.pick.Pick()
		if err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}
	}
	result, err := app.Scan(root, "", s.walk, s.read, s.identify, s.similar, body.ExactOnly)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
		return
	}
	session, err := app.AttachFiles(result.Session, result.Files)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	id, err := s.sessions.Save(session)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	writeJSON(w, nethttp.StatusOK, sessionJSON(id, session))
}

func decodeScan(r *nethttp.Request) (scanBody, error) {
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			return scanBody{}, err
		}
		return scanBody{
			Path:      r.PostForm.Get("path"),
			ExactOnly: r.PostForm.Get("exactOnly") == "true",
		}, nil
	}
	var body scanBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if errors.Is(err, io.EOF) {
		return scanBody{}, nil
	}
	return body, err
}

func (s *Server) getSession(w nethttp.ResponseWriter, r *nethttp.Request) {
	session, err := s.sessions.Load(r.PathValue("id"))
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, sessionJSON(r.PathValue("id"), session))
}

type decisionBody struct {
	Path     string `json:"path"`
	Choice   string `json:"choice"`
	MoveFrom string `json:"moveFrom"`
}

func (s *Server) postDecision(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	session, err := s.sessions.Load(id)
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	var body decisionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		nethttp.Error(w, "bad decision", nethttp.StatusBadRequest)
		return
	}
	next, err := applyDecision(session, body)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
		return
	}
	if err := s.sessions.Replace(id, next); err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	writeJSON(w, nethttp.StatusOK, sessionJSON(id, next))
}

func applyDecision(session domain.Session, body decisionBody) (domain.Session, error) {
	switch {
	case body.MoveFrom != "" && body.Path == "" && body.Choice == "":
		return domain.ConfirmMove(session, body.MoveFrom)
	case body.Path != "" && body.Choice != "" && body.MoveFrom == "":
		return domain.RecordDecision(session, body.Path, domain.Choice(body.Choice))
	default:
		return domain.Session{}, errors.New("decision needs a path and a choice, or a moveFrom path")
	}
}

func (s *Server) dryRun(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	session, err := s.sessions.Load(id)
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	report, err := app.DryRun(session, fileRefs(session), s.library, s.trash, s.disk)
	if err != nil {
		s.writePlanError(w, err)
		return
	}
	if wantsJSON(r) {
		writeJSON(w, nethttp.StatusOK, reportJSON(report))
		return
	}
	s.writeReview(w, nethttp.StatusOK, id, session, &report)
}

func (s *Server) apply(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	session, err := s.sessions.Load(id)
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	next, err := app.Apply(session, fileRefs(session), s.library, s.trash, s.disk)
	if err != nil {
		s.writePlanError(w, err)
		return
	}
	if err := s.sessions.Replace(id, next); err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	if wantsJSON(r) {
		writeJSON(w, nethttp.StatusOK, sessionJSON(id, next))
		return
	}
	s.writeReview(w, nethttp.StatusOK, id, next, nil)
}

func (s *Server) writePlanError(w nethttp.ResponseWriter, err error) {
	status := nethttp.StatusInternalServerError
	if errors.Is(err, domain.ErrUnconfirmedDelete) || errors.Is(err, domain.ErrLastCopy) || errors.Is(err, domain.ErrSessionNotOpen) || errors.Is(err, domain.ErrMoveOutsideRoot) {
		status = nethttp.StatusConflict
	}
	nethttp.Error(w, err.Error(), status)
}

func (s *Server) writeLoadError(w nethttp.ResponseWriter, err error) {
	if errors.Is(err, app.ErrSessionNotFound) {
		nethttp.Error(w, err.Error(), nethttp.StatusNotFound)
		return
	}
	nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
}

func wantsJSON(r *nethttp.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w nethttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fileRefs(session domain.Session) []domain.FileRef {
	var out []domain.FileRef
	seen := map[string]struct{}{}
	add := func(files []domain.FileRef) {
		for _, file := range files {
			key := filepath.Clean(file.Path)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, file)
		}
	}
	for _, group := range session.Exact {
		add(group.Members())
	}
	for _, group := range session.Similar {
		add(group.Members())
	}
	return out
}

type sessionView struct {
	ID        string         `json:"id"`
	Root      string         `json:"root"`
	Status    string         `json:"status"`
	ExactOnly bool           `json:"exactOnly"`
	Exact     []exactView    `json:"exact"`
	Similar   []similarView  `json:"similar"`
	Decisions []decisionView `json:"decisions"`
	Files     []fileView     `json:"files"`
}

type exactView struct {
	Members        []memberView `json:"members"`
	Recommendation *recView     `json:"recommendation,omitempty"`
}

type similarView struct {
	Members []memberView `json:"members"`
}

type memberView struct {
	Path      string `json:"path"`
	ContentID string `json:"contentId,omitempty"`
}

type recView struct {
	Keeper string   `json:"keeper"`
	Trash  []string `json:"trash"`
	Reason string   `json:"reason"`
	Text   string   `json:"text,omitempty"`
}

type decisionView struct {
	Path   string `json:"path"`
	Choice string `json:"choice"`
}

type fileView struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func sessionJSON(id string, session domain.Session) sessionView {
	view := sessionView{
		ID:        id,
		Root:      session.Root,
		Status:    string(session.Status()),
		ExactOnly: session.ExactOnly,
	}
	for _, group := range session.Exact {
		item := exactView{Members: membersOf(group.Members())}
		if rec, ok := group.Recommendation(); ok {
			item.Recommendation = &recView{
				Keeper: rec.Keeper,
				Trash:  append([]string(nil), rec.Trash...),
				Reason: string(rec.Reason),
				Text:   rec.Text,
			}
		}
		view.Exact = append(view.Exact, item)
	}
	for _, group := range session.Similar {
		view.Similar = append(view.Similar, similarView{Members: membersOf(group.Members())})
	}
	for _, decision := range session.Decisions {
		view.Decisions = append(view.Decisions, decisionView{Path: decision.Path, Choice: string(decision.Choice)})
	}
	for _, file := range session.Files {
		view.Files = append(view.Files, fileView{ID: file.ID, Path: file.File.Path})
	}
	if view.Exact == nil {
		view.Exact = []exactView{}
	}
	if view.Similar == nil {
		view.Similar = []similarView{}
	}
	if view.Decisions == nil {
		view.Decisions = []decisionView{}
	}
	if view.Files == nil {
		view.Files = []fileView{}
	}
	return view
}

func membersOf(files []domain.FileRef) []memberView {
	out := make([]memberView, len(files))
	for i, file := range files {
		out[i] = memberView{Path: file.Path, ContentID: string(file.ContentID)}
	}
	return out
}

type reportJSONBody struct {
	Changed bool     `json:"changed"`
	Trash   []string `json:"trash"`
	Moves   []string `json:"moves"`
	Upserts []string `json:"upserts"`
}

func reportJSON(report app.PlanReport) reportJSONBody {
	moves := make([]string, len(report.Moves))
	for i, move := range report.Moves {
		moves[i] = move.From + " -> " + move.Destination
	}
	trash := report.Trash
	if trash == nil {
		trash = []string{}
	}
	upserts := report.Upserts
	if upserts == nil {
		upserts = []string{}
	}
	return reportJSONBody{Changed: false, Trash: trash, Moves: moves, Upserts: upserts}
}
