# BuilderConfigHub

[![Test status](https://github.com/flashbots/builder-config-hub/actions/workflows/checks.yml/badge.svg?branch=main)](https://github.com/flashbots/builder-config-hub/actions?query=workflow%3A%22Checks%22)

Endpoint for TDX builders to talk to.

https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87

---

## Getting started

```bash
# start the server
go run cmd/httpserver/main.go

# public endpoints
curl localhost:8080/api/v1/measurements

# client-aTLS secured endpoints
curl localhost:8080/api/v1/auth-client-atls/builders
curl localhost:8080/api/v1/auth-client-atls/configuration
curl -X POST localhost:8080/api/v1/auth-client-atls/register_credentials?service=abc
```

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
