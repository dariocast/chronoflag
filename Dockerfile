FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/build/ ./internal/httpapi/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /chronograph ./cmd/chronograph

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -S chrono && adduser -S -G chrono chrono
COPY --from=go /chronograph /usr/local/bin/chronograph
USER chrono
EXPOSE 8080
ENTRYPOINT ["chronograph"]
