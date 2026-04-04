# .http File Format

restless uses the `.http` file format, compatible with JetBrains IDEs and VS Code REST Client.

## Basic Structure

```http
METHOD URL [HTTP/version]
Header-Name: Header-Value

request body
```

## Multiple Requests

Separate requests with `###`:

```http
GET https://api.example.com/users

###

POST https://api.example.com/users
Content-Type: application/json

{"name": "Alice"}

###

DELETE https://api.example.com/users/1
```

## Inline Variables

Define variables at the top of your file:

```http
@baseUrl = https://api.example.com
@token = my-secret-token

GET {{baseUrl}}/users
Authorization: Bearer {{token}}
```

## Metadata Tags

Tags are comments that start with `# @`:

| Tag | Effect | Example |
|-----|--------|---------|
| `# @name <name>` | Name the request | `# @name createUser` |
| `# @no-redirect` | Don't follow redirects | |
| `# @no-cookie-jar` | Don't send/store cookies | |
| `# @timeout <seconds>` | Request timeout | `# @timeout 30` |
| `# @connection-timeout <seconds>` | Connection timeout | `# @connection-timeout 10` |
| `# @insecure` | Skip TLS verification | |
| `# @proxy <url>` | Use HTTP proxy | `# @proxy http://proxy:8080` |
| `# @assert <expression>` | Response assertion | `# @assert status == 200` |

## Request Chaining

Name your requests and reference their responses:

```http
# @name login
POST {{baseUrl}}/auth/login
Content-Type: application/json

{"email": "user@example.com", "password": "secret"}

###

# @name getProfile
GET {{baseUrl}}/me
Authorization: Bearer {{login.response.body.access_token}}
```

Access patterns:
- `{{name.response.body.field}}` — JSON field from response body
- `{{name.response.body.nested.field}}` — nested JSON path
- `{{name.response.headers.Header-Name}}` — response header value

## Dynamic Variables

| Variable | Example Output |
|----------|---------------|
| `{{$uuid}}` | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| `{{$timestamp}}` | `1712234567` |
| `{{$isoTimestamp}}` | `2026-04-04T12:34:56Z` |
| `{{$date}}` | `2026-04-04` |
| `{{$randomInt}}` | `742` |
| `{{$randomFloat}}` | `0.8372` |
| `{{$randomBool}}` | `true` |
| `{{$randomEmail}}` | `user42871@example.com` |
| `{{$randomName}}` | `Alice` |
| `{{$randomSlug}}` | `slug-4821-1234` |
| `{{$datetime "format"}}` | Custom Go time format |

## File Bodies

Reference external files as request body:

```http
POST https://api.example.com/upload
Content-Type: application/json

< ./payload.json
```

## Response Assertions

Add assertions after a request to validate the response:

```http
POST {{baseUrl}}/users
Content-Type: application/json

{"name": "Alice"}

# @assert status == 201
# @assert body.$.id != null
# @assert body.$.name == "Alice"
# @assert header.Content-Type contains json
# @assert duration < 2000
# @assert size < 10240
```

### Assertion Targets

| Target | Description |
|--------|-------------|
| `status` | HTTP status code (integer) |
| `body` | Raw response body as string |
| `body.$.path` | JSON path in response body (via gjson) |
| `header.Name` | Response header value |
| `duration` | Total response time in milliseconds |
| `size` | Response body size in bytes |

### Assertion Operators

| Operator | Description |
|----------|-------------|
| `==` | Equals |
| `!=` | Not equals |
| `<` `>` `<=` `>=` | Numeric comparison |
| `contains` | String contains substring |
| `matches` | Regex match |
| `exists` | Value is not empty/null |
| `!exists` | Value is empty/null |

## Comments

Lines starting with `#` or `//` are comments (unless they're metadata tags):

```http
# This is a comment
// This is also a comment

# @name thisIsMetadata
GET https://example.com
```
