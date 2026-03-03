## qwikkle-api

Backend API service for Qwikkle.

### Prerequisites

- Go 1.22 or newer installed

### Environment variables

Copy `.env.example` to `.env` and adjust values for your local setup:

```bash
cp .env.example .env
```

At minimum you can leave the defaults and only change secrets later.

### Running locally

```bash
make run
```

The service will start on `http://localhost:8080`. You can verify it with:

```bash
curl http://localhost:8080/healthz
```

