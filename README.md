# BuilderHub

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/builder-hub)](https://goreportcard.com/report/github.com/flashbots/builder-hub)
[![Test status](https://github.com/flashbots/builder-hub/actions/workflows/checks.yml/badge.svg?branch=main)](https://github.com/flashbots/builder-hub/actions?query=workflow%3A%22Checks%22)

Contains code for the builder hub service that acts as a data source for builders registration and configuration.

Docs here: https://buildernet.org

---

## Getting started

```bash
# start the server
go run cmd/httpserver/main.go

# public endpoints
curl localhost:8080/api/l1-builder/v1/measurements

# client-aTLS secured endpoints
curl localhost:8080/api/l1-builder/v1/builders
curl localhost:8080/api/l1-builder/v1/configuration
curl -X POST localhost:8080/api/l1-builder/v1/register_credentials/rbuilder
```

### Development

**Install dev dependencies**

```bash
go install mvdan.cc/gofumpt@v0.4.0
go install honnef.co/go/tools/cmd/staticcheck@v0.4.2
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
go install go.uber.org/nilaway/cmd/nilaway@v0.0.0-20240821220108-c91e71c080b7
go install github.com/daixiang0/gci@v0.11.2
```

**Lint, test, format**

```bash
make lint
make test
make fmt
```

### Database tests

```bash
# start the database
docker run -d --name postgres-test -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres postgres

for file in schema/*.sql; do psql "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" -f $file; done

# run the tests
RUN_DB_TESTS=1 make test

# stop the database
docker rm -f postgres-test
```
