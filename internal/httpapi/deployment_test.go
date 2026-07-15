package httpapi_test

import (
	"encoding/json"
	"image/png"
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
		"Start timing",
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

func TestLandingSEOContract(t *testing.T) {
	index := readProjectFile(t, "../../landing/index.html")
	for _, want := range []string{
		`property="og:title"`,
		`property="og:description"`,
		`property="og:image" content="https://chronoflag.com/social-card.png"`,
		`name="twitter:card" content="summary_large_image"`,
		`rel="icon" href="/favicon.svg"`,
		`rel="manifest" href="/site.webmanifest"`,
		`type="application/ld+json"`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("landing metadata does not contain %q", want)
		}
	}

	jsonLD := between(t, index, `<script type="application/ld+json">`, `</script>`)
	var schema map[string]any
	if err := json.Unmarshal([]byte(jsonLD), &schema); err != nil {
		t.Fatalf("decode JSON-LD: %v", err)
	}
	if schema["@type"] != "SoftwareApplication" || schema["name"] != "Chronoflag" {
		t.Errorf("unexpected JSON-LD identity: %#v", schema)
	}

	robots := readProjectFile(t, "../../landing/robots.txt")
	for _, want := range []string{"User-agent: *", "Allow: /", "Sitemap: https://chronoflag.com/sitemap.xml"} {
		if !strings.Contains(robots, want) {
			t.Errorf("robots.txt does not contain %q", want)
		}
	}

	sitemap := readProjectFile(t, "../../landing/sitemap.xml")
	for _, want := range []string{
		`xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
		"<loc>https://chronoflag.com/</loc>",
	} {
		if !strings.Contains(sitemap, want) {
			t.Errorf("sitemap.xml does not contain %q", want)
		}
	}
	if strings.Contains(sitemap, "app.chronoflag.com") {
		t.Error("sitemap must not index private application routes")
	}

	manifestBody := readProjectFile(t, "../../landing/site.webmanifest")
	var manifest struct {
		Name  string `json:"name"`
		Icons []struct {
			Src   string `json:"src"`
			Sizes string `json:"sizes"`
		} `json:"icons"`
	}
	if err := json.Unmarshal([]byte(manifestBody), &manifest); err != nil {
		t.Fatalf("decode landing manifest: %v", err)
	}
	if manifest.Name != "Chronoflag" || len(manifest.Icons) < 2 {
		t.Errorf("incomplete landing manifest: %#v", manifest)
	}

	assertPNGDimensions(t, "../../landing/social-card.png", 1200, 630)
	assertPNGDimensions(t, "../../landing/icon-192.png", 192, 192)
	assertPNGDimensions(t, "../../landing/icon-512.png", 512, 512)
	assertPNGDimensions(t, "../../landing/apple-touch-icon.png", 180, 180)
}

func TestLandingContainerIncludesSEOAssets(t *testing.T) {
	dockerfile := readProjectFile(t, "../../landing/Dockerfile")
	nginx := readProjectFile(t, "../../landing/nginx.conf")
	for _, asset := range []string{
		"robots.txt",
		"sitemap.xml",
		"site.webmanifest",
		"favicon.svg",
		"social-card.png",
		"icon-192.png",
		"icon-512.png",
		"apple-touch-icon.png",
	} {
		if !strings.Contains(dockerfile, asset) {
			t.Errorf("landing Dockerfile does not package %s", asset)
		}
		escapedAsset := strings.ReplaceAll(asset, ".", `\.`)
		if !strings.Contains(nginx, asset) && !strings.Contains(nginx, escapedAsset) {
			t.Errorf("landing Nginx configuration does not serve %s", asset)
		}
	}
}

func between(t *testing.T, body, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(body, startMarker)
	if start < 0 {
		t.Fatalf("missing start marker %q", startMarker)
	}
	start += len(startMarker)
	end := strings.Index(body[start:], endMarker)
	if end < 0 {
		t.Fatalf("missing end marker %q", endMarker)
	}
	return strings.TrimSpace(body[start : start+end])
}

func assertPNGDimensions(t *testing.T, name string, width, height int) {
	t.Helper()
	file, err := os.Open(name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	if config.Width != width || config.Height != height {
		t.Errorf("%s dimensions are %dx%d, want %dx%d", name, config.Width, config.Height, width, height)
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
