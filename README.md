## qwikkle-api

Backend API service for Qwikkle.

### Prerequisites

- Go 1.22 or newer installed

### Dev tooling

- [Air](https://github.com/cosmtrek/air) (optional, for auto-reload)
  - Install: `go install github.com/cosmtrek/air@latest`

### Environment variables

Copy `.env.example` to `.env` and adjust values for your local setup:

```bash
cp .env.example .env
```

At minimum you can leave the defaults and only change secrets later.

### Running locally

```bash
go mod tidy
make run
```

The service will start on `http://localhost:8080`. You can verify it with:

```bash
curl http://localhost:8080/healthz
```

### Live reload (optional)

If you installed Air:

```bash
make dev
```

This will rebuild and restart the server when `.go` files change.

### Swagger UI

Swagger UI is available at:

- `http://localhost:8080/swagger/index.html`

As you add more endpoints, you can later introduce Swagger annotations and
regenerate docs using `swag init` if you install the `swag` CLI.


