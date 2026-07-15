package httpapi_test

import (
	"os"
	"strings"
	"testing"
)

func TestLandingTargetsCanonicalApp(t *testing.T) {
	body := readProjectFile(t, "../../landing/index.html")

	for _, want := range []string{
		`href="https://app.chronoflag.com"`,
		"Server-authoritative",
		"Control link",
		"Live view",
		"Multiple clocks",
		"Export",
		"<main",
		"<nav",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("landing page does not contain %q", want)
		}
	}

	for _, forbidden := range []string{
		"fonts.googleapis.com",
		"fonts.gstatic.com",
		`src="http://`,
		`src="https://`,
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("landing page contains third-party or insecure resource %q", forbidden)
		}
	}
}

func TestComposeSeparatesLandingAndApp(t *testing.T) {
	body := readProjectFile(t, "../../compose.yaml")

	for _, want := range []string{
		"  landing:\n",
		"  app:\n",
		`"${LANDING_HTTP_PORT:-8081}:8080"`,
		`"${APP_HTTP_PORT:-8080}:8080"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("compose configuration does not contain %q", want)
		}
	}

	if count := strings.Count(body, "    healthcheck:\n"); count != 3 {
		t.Errorf("compose configuration has %d health checks, want 3", count)
	}
}

func TestPublicSurfacesUseChronoflagBrand(t *testing.T) {
	for _, name := range []string{
		"../../web/src/routes/+page.svelte",
		"../../web/src/lib/Board.svelte",
		"../../web/src/lib/Modal.svelte",
		"../../web/static/manifest.webmanifest",
	} {
		body := readProjectFile(t, name)
		if !strings.Contains(body, "Chronoflag") {
			t.Errorf("%s does not contain the Chronoflag brand", name)
		}
		if strings.Contains(body, "Chronograph") {
			t.Errorf("%s still contains the legacy Chronograph brand", name)
		}
	}
}

func readProjectFile(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(body)
}
