package httpapi

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chronograph/internal/clock"
	"chronograph/internal/realtime"
	"chronograph/internal/store"
)

type API struct {
	store store.Store
	hub   *realtime.Hub
	now   func() time.Time
}

func NewRouter(s store.Store, h *realtime.Hub) http.Handler {
	a := &API{store: s, hub: h, now: func() time.Time { return time.Now().UTC() }}
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	m.HandleFunc("POST /api/v1/instances", a.create)
	m.HandleFunc("GET /api/v1/control/{token}", a.snapshot(store.Control))
	m.HandleFunc("PATCH /api/v1/control/{token}", a.updateInstance)
	m.HandleFunc("POST /api/v1/control/{token}/capabilities/{scope}/rotate", a.rotateCapability)
	m.HandleFunc("GET /api/v1/view/{token}", a.snapshot(store.View))
	m.HandleFunc("POST /api/v1/control/{token}/clocks", a.addClock)
	m.HandleFunc("PATCH /api/v1/control/{token}/clocks/{clock}", a.updateClock)
	m.HandleFunc("DELETE /api/v1/control/{token}/clocks/{clock}", a.removeClock)
	m.HandleFunc("POST /api/v1/control/{token}/clocks/{clock}/commands", a.command)
	m.HandleFunc("POST /api/v1/control/{token}/clocks/{clock}/undo", a.undo)
	m.HandleFunc("GET /api/v1/{scope}/{token}/events", a.sse)
	m.HandleFunc("GET /api/v1/control/{token}/export", a.export)
	m.HandleFunc("DELETE /api/v1/control/{token}", a.delete)
	m.Handle("/", staticHandler())
	return security(m)
}
func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panic", "error", v)
				writeError(w, 500, "internal_error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func status(err error) int {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return 404
	case errors.Is(err, store.ErrForbidden):
		return 403
	case errors.Is(err, store.ErrArchived):
		return 409
	case errors.Is(err, store.ErrLimit):
		return 422
	case errors.Is(err, clock.ErrInvalidTransition), errors.Is(err, clock.ErrLapUnsupported), errors.Is(err, clock.ErrUndoExpired), errors.Is(err, clock.ErrUndoSuperseded):
		return 409
	default:
		return 400
	}
}
func (a *API) cap(ctx context.Context, token string, scope store.Scope) (store.Capability, error) {
	c, e := a.store.ResolveCapability(ctx, token)
	if e == nil && c.Scope != scope {
		e = store.ErrForbidden
	}
	return c, e
}
func (a *API) create(w http.ResponseWriter, r *http.Request) {
	c, e := a.store.CreateInstance(r.Context(), a.now())
	if e != nil {
		writeError(w, 500, "create_failed")
		return
	}
	writeJSON(w, 201, map[string]string{"instance_id": c.InstanceID, "control_url": "/c/" + c.ControlToken, "view_url": "/v/" + c.ViewToken})
}
func (a *API) snapshot(scope store.Scope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, e := a.cap(r.Context(), r.PathValue("token"), scope)
		if e != nil {
			writeError(w, status(e), "not_found")
			return
		}
		s, e := a.store.Snapshot(r.Context(), c.InstanceID)
		if e != nil {
			writeError(w, status(e), "not_found")
			return
		}
		writeJSON(w, 200, envelope(s, a.now()))
	}
}
func (a *API) updateInstance(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	var patch store.InstancePatch
	if e = decode(r, &patch); e != nil || patch.Title == nil {
		writeError(w, 400, "invalid_json")
		return
	}
	s, e := a.store.UpdateInstance(r.Context(), c, patch, a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 200, envelope(s, a.now()))
}
func (a *API) rotateCapability(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	scope := store.Scope(r.PathValue("scope"))
	if scope != store.Control && scope != store.View {
		writeError(w, 400, "invalid_scope")
		return
	}
	rotated, e := a.store.RotateCapability(r.Context(), c, scope)
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	if s, e := a.store.Snapshot(r.Context(), c.InstanceID); e == nil {
		a.publish(s)
	}
	raw := rotated.ViewToken
	path := "/v/" + raw
	if scope == store.Control {
		raw = rotated.ControlToken
		path = "/c/" + raw
	}
	writeJSON(w, 200, map[string]string{"url": path, string(scope) + "_url": path})
}
func envelope(s store.InstanceSnapshot, now time.Time) any {
	return struct {
		store.InstanceSnapshot
		ServerTime time.Time `json:"server_time"`
	}{s, now}
}
func (a *API) publish(s store.InstanceSnapshot) {
	b, _ := json.Marshal(envelope(s, a.now()))
	a.hub.Publish(s.ID, realtime.Message{ID: s.Version, Event: "update", Data: b})
}
func (a *API) command(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	var cmd clock.Command
	if e = decode(r, &cmd); e != nil {
		writeError(w, 400, "invalid_json")
		return
	}
	cmd.ID = r.Header.Get("Idempotency-Key")
	if cmd.ID == "" {
		writeError(w, 400, "idempotency_key_required")
		return
	}
	s, _, e := a.store.ApplyCommand(r.Context(), c, r.PathValue("clock"), cmd, a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 200, envelope(s, a.now()))
}
func (a *API) addClock(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	var in struct {
		Type       clock.ClockType `json:"type"`
		DurationMS int64           `json:"duration_ms"`
	}
	if e = decode(r, &in); e != nil {
		writeError(w, 400, "invalid_json")
		return
	}
	s, e := a.store.AddClock(r.Context(), c, in.Type, time.Duration(in.DurationMS)*time.Millisecond, a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 201, envelope(s, a.now()))
}
func (a *API) updateClock(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	var p store.ClockPatch
	if e = decode(r, &p); e != nil {
		writeError(w, 400, "invalid_json")
		return
	}
	s, e := a.store.UpdateClock(r.Context(), c, r.PathValue("clock"), p, a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 200, envelope(s, a.now()))
}
func (a *API) removeClock(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	s, e := a.store.RemoveClock(r.Context(), c, r.PathValue("clock"), a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 200, envelope(s, a.now()))
}
func (a *API) undo(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	s, _, e := a.store.Undo(r.Context(), c, r.PathValue("clock"), r.Header.Get("Idempotency-Key"), a.now())
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	a.publish(s)
	writeJSON(w, 200, envelope(s, a.now()))
}
func (a *API) sse(w http.ResponseWriter, r *http.Request) {
	scope := store.View
	if r.PathValue("scope") == "control" {
		scope = store.Control
	}
	c, e := a.cap(r.Context(), r.PathValue("token"), scope)
	if e != nil {
		writeError(w, status(e), "not_found")
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "stream_unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, cancel := a.hub.Subscribe(c.InstanceID)
	defer cancel()
	s, e := a.store.Snapshot(r.Context(), c.InstanceID)
	if e != nil {
		return
	}
	b, _ := json.Marshal(envelope(s, a.now()))
	fmt.Fprintf(w, "id: %d\nevent: snapshot\ndata: %s\n\n", s.Version, b)
	f.Flush()
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		select {
		case msg, open := <-ch:
			if !open {
				return
			}
			if _, e := a.cap(r.Context(), r.PathValue("token"), scope); e != nil {
				return
			}
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, msg.Event, msg.Data)
			f.Flush()
		case <-tick.C:
			if _, e := a.cap(r.Context(), r.PathValue("token"), scope); e != nil {
				return
			}
			fmt.Fprint(w, ": keepalive\n\n")
			f.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
func (a *API) export(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e != nil {
		writeError(w, status(e), "forbidden")
		return
	}
	data, e := a.store.Export(r.Context(), c)
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	if r.URL.Query().Get("format") != "csv" {
		w.Header().Set("Content-Disposition", "attachment; filename=chronograph.json")
		writeJSON(w, 200, data)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=chronograph-csv.zip")
	z := zip.NewWriter(w)
	cw, _ := z.Create("clocks.csv")
	csvw := csv.NewWriter(cw)
	_ = csvw.Write([]string{"id", "type", "label", "state", "duration_ms", "accumulated_ms"})
	for _, c := range data.Instance.Clocks {
		_ = csvw.Write([]string{c.ID, string(c.Type), c.Label, string(c.State), strconv.FormatInt(c.Duration.Milliseconds(), 10), strconv.FormatInt(c.Accumulated.Milliseconds(), 10)})
	}
	csvw.Flush()
	_ = z.Close()
}
func (a *API) delete(w http.ResponseWriter, r *http.Request) {
	c, e := a.cap(r.Context(), r.PathValue("token"), store.Control)
	if e == nil {
		e = a.store.DeleteInstance(r.Context(), c)
	}
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	w.WriteHeader(204)
}
func tokenFromPath(path string) string { return strings.Trim(path, "/") }
