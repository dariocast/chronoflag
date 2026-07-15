# Landing and App Service Separation

## Goal

Serve the public Chronoflag marketing page and the timing application as independent Docker services. Production will route `chronoflag.com` to the landing container and `app.chronoflag.com` to the application container.

## Architecture

The existing Go/Svelte image remains the `app` service and continues to own the HTTP API, realtime streams, application routes, and PostgreSQL access. A new `landing` service serves a self-contained static HTML/CSS site from an unprivileged Nginx container. The landing has no database or application dependency and links to the canonical application origin through a build-time-independent absolute URL.

Docker Compose publishes the application on `APP_HTTP_PORT` (default `8080`) and the landing on `LANDING_HTTP_PORT` (default `8081`). A production reverse proxy terminates TLS and maps each hostname to its corresponding loopback port. The containers do not perform hostname routing themselves.

## Landing experience

The landing preserves Chronoflag's brutalist visual language while becoming a true product explanation rather than an API client. Its primary action opens `https://app.chronoflag.com`; secondary content explains server-authoritative timing, separate control/view links, multi-clock support, live synchronization, and exports. It is responsive, keyboard accessible, usable without JavaScript, and contains no third-party assets or analytics.

The application root remains the zero-configuration launch surface. Control and view capability routes keep their current URLs and behavior.

## Operations and security

Both public services expose `/healthz` health checks. The landing sends restrictive browser security headers and caches static assets conservatively. PostgreSQL remains private to the Compose network and is required only by `app`.

The example environment documents both published ports. Deployment documentation includes the two reverse-proxy upstreams and preserves the SSE buffering requirement for the application hostname.

## Verification

Automated tests verify that the landing contains the canonical application CTA, core product messaging, accessibility landmarks, and no third-party resource URLs. Deployment tests verify that Compose defines distinct `landing` and `app` services with independent ports and health checks. The existing application suite must remain green, both images must build, and live container probes must return healthy responses with the landing CTA pointing to the application domain.
