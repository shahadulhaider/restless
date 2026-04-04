# Importing Collections

restless can import collections from 5 different sources.

## From Postman

Export your Postman collection as JSON (v2.1), then:

```bash
restless import postman collection.json --output ./my-api
```

To import environments:
```bash
restless import postman collection.json --env environment.json --output ./my-api
```

## From Insomnia

Export your Insomnia workspace (v4 JSON format), then:

```bash
restless import insomnia export.json --output ./my-api
```

Folder structure is preserved. Environment variables are converted to `restless.env.json`.

## From Bruno

Point at your Bruno collection directory:

```bash
restless import bruno ./my-bruno-collection --output ./my-api
```

`.bru` files are converted to `.http` files. Environment files are converted to `restless.env.json`.

## From curl Commands

Import a curl command directly:

```bash
restless import curl "curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\"}'" --output ./my-api
```

Supported curl flags: `-X`, `-H`, `-d`/`--data`, `-u` (basic auth), `-b` (cookies), `-L`, `--url`.

## From OpenAPI / Swagger

Import an OpenAPI 3.x or Swagger 2.0 spec (JSON or YAML):

```bash
restless import openapi spec.yaml --output ./my-api
restless import openapi swagger.json --output ./my-api
```

### FastAPI Users

FastAPI auto-generates an OpenAPI spec:

```bash
# Download the spec from your running server
curl http://localhost:8000/openapi.json -o openapi.json

# Import it
restless import openapi openapi.json --output ./my-api

# Fix base URL (FastAPI uses relative paths)
cd my-api
echo '@baseUrl = http://localhost:8000' | cat - *.http > /tmp/fix && mv /tmp/fix *.http
# Or create restless.env.json with baseUrl
```

Create `restless.env.json`:
```json
{
  "dev": {
    "baseUrl": "http://localhost:8000"
  }
}
```

Then launch: `restless ./my-api`, press `Ctrl+E` to select `dev`.

## Common Options

All import commands support:

| Flag | Description |
|------|-------------|
| `--output <dir>` | Output directory (default: current directory) |
