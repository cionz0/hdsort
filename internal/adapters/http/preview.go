package http

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	nethttp "net/http"
	"path/filepath"
	"strings"

	_ "image/gif" // register GIF for image.Decode
	_ "image/png" // register PNG for image.Decode

	"github.com/cionz0/hdsort/internal/domain"
)

const (
	// maxInline is the largest file returned unchanged.
	// Images and other types above this size are not sent in full.
	maxInline = 1 << 20
	// maxThumbWidth is the widest JPEG made from a larger image.
	maxThumbWidth = 320
)

func (s *Server) preview(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Query().Has("path") || bodyHasPath(r) {
		nethttp.Error(w, "path is not accepted", nethttp.StatusBadRequest)
		return
	}
	fileID := r.PathValue("fileID")
	if !opaqueID(fileID) {
		nethttp.Error(w, "file id required", nethttp.StatusBadRequest)
		return
	}
	session, err := s.sessions.Load(r.PathValue("id"))
	if err != nil {
		s.writeLoadError(w, err)
		return
	}
	file, ok := sessionFile(session, fileID)
	if !ok {
		nethttp.Error(w, "file not in this session", nethttp.StatusNotFound)
		return
	}
	if s.read == nil {
		nethttp.Error(w, "scan is not configured", nethttp.StatusInternalServerError)
		return
	}
	src, err := s.read.Open(file.File.Path)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	defer src.Close()
	size, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		return
	}
	kind := imageKind(file.File.Path)
	if kind != "" {
		if size <= maxInline {
			w.Header().Set("Content-Type", kind)
			w.WriteHeader(nethttp.StatusOK)
			_, _ = io.Copy(w, src)
			return
		}
		if err := writeThumbnail(w, src); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		}
		return
	}
	if size > maxInline {
		nethttp.Error(w, "file is too large to preview", nethttp.StatusUnsupportedMediaType)
		return
	}
	name := filepath.Base(file.File.Path)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(nethttp.StatusOK)
	_, _ = io.Copy(w, src)
}

func bodyHasPath(r *nethttp.Request) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil || len(raw) == 0 {
		return false
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&body); err != nil {
		return strings.Contains(string(raw), `"path"`)
	}
	return body.Path != ""
}

func opaqueID(id string) bool {
	if id == "" || strings.ContainsAny(id, `/\`) || filepath.IsAbs(id) {
		return false
	}
	return true
}

func sessionFile(session domain.Session, id string) (domain.SessionFile, bool) {
	for _, file := range session.Files {
		if file.ID == id {
			return file, true
		}
	}
	return domain.SessionFile{}, false
}

func imageKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func writeThumbnail(w nethttp.ResponseWriter, src io.Reader) error {
	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}
	fitted := fitWidth(img, maxThumbWidth)
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(nethttp.StatusOK)
	return jpeg.Encode(w, fitted, &jpeg.Options{Quality: 80})
}

func fitWidth(src image.Image, maxW int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxW {
		return src
	}
	newW := maxW
	newH := height * maxW / width
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		sy := bounds.Min.Y + y*height/newH
		for x := 0; x < newW; x++ {
			sx := bounds.Min.X + x*width/newW
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
