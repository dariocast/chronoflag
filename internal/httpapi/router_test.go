package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chronograph/internal/httpapi"
	"chronograph/internal/realtime"
	"chronograph/internal/store"
)

func TestRouterCreatesAndControlsInstance(t *testing.T) {
	s := store.NewMemory()
	h := realtime.NewHub(8)
	ts := httptest.NewServer(httpapi.NewRouter(s, h))
	defer ts.Close()
	res, err := http.Post(ts.URL+"/api/v1/instances", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", res.StatusCode)
	}
	var created struct {
		ControlURL string `json:"control_url"`
		ViewURL    string `json:"view_url"`
	}
	json.NewDecoder(res.Body).Decode(&created)
	res.Body.Close()
	if created.ControlURL == created.ViewURL || !strings.HasPrefix(created.ControlURL, "/c/") {
		t.Fatalf("bad links: %#v", created)
	}
	controlToken := strings.TrimPrefix(created.ControlURL, "/c/")
	viewToken := strings.TrimPrefix(created.ViewURL, "/v/")
	controlRes, _ := http.Get(ts.URL + "/api/v1/control/" + controlToken)
	if controlRes.StatusCode != 200 {
		t.Fatalf("control=%d", controlRes.StatusCode)
	}
	var snap store.InstanceSnapshot
	json.NewDecoder(controlRes.Body).Decode(&snap)
	controlRes.Body.Close()
	id := snap.Clocks[0].ID
	body := bytes.NewBufferString(`{"type":"start","device_id":"phone"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/control/"+controlToken+"/clocks/"+id+"/commands", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "cmd-1")
	cmdRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if cmdRes.StatusCode != 200 {
		t.Fatalf("command=%d", cmdRes.StatusCode)
	}
	json.NewDecoder(cmdRes.Body).Decode(&snap)
	cmdRes.Body.Close()
	if snap.Clocks[0].State != "running" {
		t.Fatalf("state=%s", snap.Clocks[0].State)
	}
	bad, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/control/"+viewToken+"/clocks/"+id+"/commands", bytes.NewBufferString(`{"type":"pause"}`))
	bad.Header.Set("Idempotency-Key", "bad")
	badRes, _ := http.DefaultClient.Do(bad)
	if badRes.StatusCode != http.StatusNotFound && badRes.StatusCode != http.StatusForbidden {
		t.Fatalf("view mutation status=%d", badRes.StatusCode)
	}
}

func TestRouterSecurityAndNoLazyCreationOnRoot(t *testing.T) {
	ts := httptest.NewServer(httpapi.NewRouter(store.NewMemory(), realtime.NewHub(2)))
	defer ts.Close()
	res, _ := http.Get(ts.URL + "/healthz")
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	if res.Header.Get("Referrer-Policy") != "no-referrer" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing security headers: %v", res.Header)
	}
	unknown, _ := http.Get(ts.URL + "/api/v1/view/nope")
	if unknown.StatusCode != 404 {
		t.Fatalf("unknown=%d", unknown.StatusCode)
	}
}

func TestSSEStartsWithSnapshot(t *testing.T) {
	s := store.NewMemory()
	h := realtime.NewHub(4)
	created, _ := s.CreateInstance(t.Context(), time.Now())
	ts := httptest.NewServer(httpapi.NewRouter(s, h))
	defer ts.Close()
	client := &http.Client{Timeout: time.Second}
	res, err := client.Get(ts.URL + "/api/v1/view/" + created.ViewToken + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	buf := make([]byte, 512)
	n, _ := res.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), "event: snapshot") {
		t.Fatalf("sse=%q", buf[:n])
	}
}

func TestRouterUpdatesTitleAndRotatesCapability(t *testing.T) {
	ts := httptest.NewServer(httpapi.NewRouter(store.NewMemory(), realtime.NewHub(2)))
	defer ts.Close()
	createdRes, err := http.Post(ts.URL+"/api/v1/instances", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		ControlURL string `json:"control_url"`
		ViewURL    string `json:"view_url"`
	}
	json.NewDecoder(createdRes.Body).Decode(&created)
	createdRes.Body.Close()
	token := strings.TrimPrefix(created.ControlURL, "/c/")
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/control/"+token, strings.NewReader(`{"title":"City relay"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("title status=%v err=%v", res.Status, err)
	}
	var snap struct {
		Title string `json:"title"`
	}
	json.NewDecoder(res.Body).Decode(&snap)
	res.Body.Close()
	if snap.Title != "City relay" {
		t.Fatalf("title=%q", snap.Title)
	}
	rotate, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/control/"+token+"/capabilities/control/rotate", nil)
	rotated, err := http.DefaultClient.Do(rotate)
	if err != nil || rotated.StatusCode != http.StatusOK {
		t.Fatalf("rotate status=%v err=%v", rotated.Status, err)
	}
	var links struct {
		ControlURL string `json:"control_url"`
	}
	json.NewDecoder(rotated.Body).Decode(&links)
	rotated.Body.Close()
	if links.ControlURL == created.ControlURL {
		t.Fatal("control URL did not rotate")
	}
	old, _ := http.Get(ts.URL + "/api/v1/control/" + token)
	if old.StatusCode != http.StatusNotFound {
		t.Fatalf("old status=%d", old.StatusCode)
	}
}
