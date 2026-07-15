# Landing and App Service Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build independently deployable Docker services for `chronoflag.com` and `app.chronoflag.com`.

**Architecture:** Keep the existing Go/Svelte application image as the stateful `app` service. Add a self-contained static landing served by unprivileged Nginx, and expose each service through its own configurable host port for reverse-proxy routing.

**Tech Stack:** Docker Compose, Go tests, static HTML/CSS, Nginx, Go/Svelte application

---

### Task 1: Specify the service boundary with failing tests

**Files:**
- Create: `internal/httpapi/deployment_test.go`
- Test: `internal/httpapi/deployment_test.go`

- [ ] **Step 1: Write a landing contract test** that reads `../../landing/index.html` and requires the canonical `https://app.chronoflag.com` CTA, the product capabilities, semantic `main` and `nav` landmarks, and no `http://` resources.
- [ ] **Step 2: Write a Compose contract test** that reads `../../compose.yaml` and requires separate `landing` and `app` services, `${LANDING_HTTP_PORT:-8081}:8080`, `${APP_HTTP_PORT:-8080}:8080`, and health checks for both public containers.
- [ ] **Step 3: Run `go test ./internal/httpapi -run 'TestLanding|TestCompose' -v`** and confirm failure because the landing does not exist and Compose has only the original application publication.

### Task 2: Build the static landing service

**Files:**
- Create: `landing/index.html`
- Create: `landing/styles.css`
- Create: `landing/nginx.conf`
- Create: `landing/Dockerfile`

- [ ] **Step 1: Create semantic landing markup** with a skip link, navigation, product hero, feature explanation, operational use cases, and absolute CTAs to `https://app.chronoflag.com`.
- [ ] **Step 2: Implement the responsive brutalist design** using local/system fonts, high-contrast focus states, reduced-motion handling, and no JavaScript or third-party resources.
- [ ] **Step 3: Configure unprivileged Nginx** to listen on `8080`, return `200 ok` from `/healthz`, serve the static site, and send CSP, referrer, frame, and content-type security headers.
- [ ] **Step 4: Package the site** from `nginxinc/nginx-unprivileged:1.29-alpine` and expose port `8080`.

### Task 3: Separate the Compose services

**Files:**
- Modify: `compose.yaml`
- Modify: `.env.example`

- [ ] **Step 1: Add `landing` to Compose** with its own build, published port, restart policy, and health check, without database dependencies.
- [ ] **Step 2: Publish `app` independently** through `${APP_HTTP_PORT:-8080}:8080` while retaining its database dependency and health check.
- [ ] **Step 3: Document default ports** as `APP_HTTP_PORT=8080` and `LANDING_HTTP_PORT=8081` in `.env.example`.
- [ ] **Step 4: Run `go test ./internal/httpapi -run 'TestLanding|TestCompose' -v`** and confirm the new contract tests pass.

### Task 4: Update operator documentation

**Files:**
- Modify: `README.md`
- Modify: `scripts/smoke.sh`

- [ ] **Step 1: Describe the two-container architecture** and show reverse-proxy routing from each production hostname to its default host port.
- [ ] **Step 2: Update configuration documentation** for `APP_HTTP_PORT` and `LANDING_HTTP_PORT`.
- [ ] **Step 3: Extend the smoke test** to probe both services and verify the landing links to the canonical app origin.

### Task 5: Verify and release

**Files:**
- Modify: plan checkboxes in this file

- [ ] **Step 1: Run `make verify`** and require exit code zero.
- [ ] **Step 2: Run `docker compose config`** and require exit code zero with three services (`db`, `app`, and `landing`).
- [ ] **Step 3: Run `docker compose build app landing`** and require both images to build.
- [ ] **Step 4: Start the stack and run `./scripts/smoke.sh`** against both live services, requiring all checks to pass.
- [ ] **Step 5: Review `git diff --check` and `git status --short`**, commit the implementation, and create annotated tag `v0.3.0` on the verified commit.
