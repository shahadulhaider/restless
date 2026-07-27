//go:build ignore

// Command server is a tiny deterministic HTTP stub used only for recording the
// demo GIFs in docs/demo. It is excluded from the module build by the `ignore`
// build tag, so it never joins `go build ./...`.
//
// Every response is a fixed raw string literal rather than a marshalled map so
// that key order — and therefore the recorded pixels — is byte-identical on
// every run.
//
//	go run docs/demo/stub/server.go            # listens on :4010
//	go run docs/demo/stub/server.go -addr :9999
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	addr := flag.String("addr", ":4010", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler(http.StatusOK, health))
	mux.HandleFunc("/auth/login", handler(http.StatusOK, login))
	mux.HandleFunc("/session", sessionHandler)
	mux.HandleFunc("/users/", handler(http.StatusOK, user))
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler(http.StatusCreated, createdUser)(w, r)
			return
		}
		handler(http.StatusOK, users)(w, r)
	})

	log.Printf("restless demo stub listening on %s", *addr)
	srv := &http.Server{Addr: *addr, Handler: mux}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// Echoes the values it was called with, so the env-switch demo can show a variable resolving.
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	body := fmt.Sprintf(sessionTemplate,
		orUnset(q.Get("region")),
		orUnset(q.Get("tier")),
		orUnset(r.Header.Get("X-Api-Version")),
		orUnset(r.Header.Get("X-Api-Key")),
	)
	handler(http.StatusOK, body)(w, r)
}

func orUnset(v string) string {
	if v == "" {
		return "unset"
	}
	return v
}

func handler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload := strings.TrimSpace(body) + "\n"
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Request-Id", "req_8f3c1a94d2e77b05")
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Remaining", "993")
		w.Header().Set("X-Response-Time", "11ms")
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(payload))
	}
}

const sessionTemplate = `
{
  "environment": {
    "region": %q,
    "tier": %q,
    "apiVersion": %q
  },
  "credentials": {
    "apiKey": %q,
    "source": "restless.env.json"
  },
  "quota": {
    "requestsPerMinute": 600,
    "burst": 60,
    "used": 21
  }
}`

const health = `
{
  "status": "ok",
  "version": "2.14.0",
  "uptimeSeconds": 918273,
  "checks": {
    "database": { "status": "ok", "latencyMs": 3 },
    "cache": { "status": "ok", "latencyMs": 1 },
    "queue": { "status": "degraded", "latencyMs": 214, "backlog": 1827 }
  }
}`

const login = `
{
  "token": "eyJhbGciOiJIUzI1NiJ9.demo-token",
  "refreshToken": "rt_4d9a17c0b6e34f28ae51",
  "expiresIn": 3600,
  "tokenType": "Bearer",
  "scope": ["users:read", "users:write", "billing:read"],
  "account": {
    "id": "acct_01HQ8ZK3M",
    "name": "Northwind Labs",
    "plan": {
      "id": "plan_scale",
      "name": "Scale",
      "seats": 25,
      "renewsAt": "2026-09-01T00:00:00Z"
    }
  }
}`

const users = `
{
  "data": [
    {
      "id": "usr_01HQ8ZK3M4",
      "email": "ada@northwind.dev",
      "name": "Ada Okafor",
      "role": "owner",
      "active": true,
      "profile": {
        "title": "Principal Engineer",
        "timezone": "Europe/Lisbon",
        "locale": "en-GB",
        "links": {
          "avatar": "/media/avatars/ada.png",
          "github": "https://github.com/ada"
        }
      },
      "teams": [
        { "id": "team_core", "name": "Core Platform", "role": "lead" },
        { "id": "team_sre", "name": "Reliability", "role": "member" }
      ],
      "lastSeenAt": "2026-07-27T09:14:02Z"
    },
    {
      "id": "usr_01HQ8ZK3M5",
      "email": "milo@northwind.dev",
      "name": "Milo Vasquez",
      "role": "admin",
      "active": true,
      "profile": {
        "title": "Staff Designer",
        "timezone": "America/Denver",
        "locale": "en-US",
        "links": {
          "avatar": "/media/avatars/milo.png",
          "github": "https://github.com/milo"
        }
      },
      "teams": [
        { "id": "team_design", "name": "Design Systems", "role": "lead" }
      ],
      "lastSeenAt": "2026-07-26T18:47:55Z"
    },
    {
      "id": "usr_01HQ8ZK3M6",
      "email": "sena@northwind.dev",
      "name": "Sena Aydin",
      "role": "member",
      "active": false,
      "profile": {
        "title": "Data Engineer",
        "timezone": "Asia/Istanbul",
        "locale": "tr-TR",
        "links": {
          "avatar": "/media/avatars/sena.png",
          "github": "https://github.com/sena"
        }
      },
      "teams": [
        { "id": "team_data", "name": "Data Platform", "role": "member" },
        { "id": "team_core", "name": "Core Platform", "role": "member" }
      ],
      "lastSeenAt": "2026-06-30T11:02:41Z"
    }
  ],
  "meta": {
    "page": 1,
    "perPage": 3,
    "totalPages": 4,
    "totalCount": 11,
    "filters": {
      "role": null,
      "active": true,
      "sort": "-lastSeenAt"
    }
  },
  "links": {
    "self": "/users?page=1",
    "next": "/users?page=2",
    "last": "/users?page=4"
  }
}`

const user = `
{
  "id": "usr_01HQ8ZK3M4",
  "email": "ada@northwind.dev",
  "name": "Ada Okafor",
  "role": "owner",
  "active": true,
  "createdAt": "2023-02-11T08:30:00Z",
  "profile": {
    "title": "Principal Engineer",
    "timezone": "Europe/Lisbon",
    "locale": "en-GB",
    "pronouns": "she/her",
    "links": {
      "avatar": "/media/avatars/ada.png",
      "github": "https://github.com/ada",
      "website": "https://ada.dev"
    }
  },
  "permissions": {
    "users": ["read", "write", "invite", "remove"],
    "billing": ["read", "write"],
    "audit": ["read"]
  },
  "teams": [
    {
      "id": "team_core",
      "name": "Core Platform",
      "role": "lead",
      "members": 9,
      "repositories": ["api", "gateway"]
    },
    {
      "id": "team_sre",
      "name": "Reliability",
      "role": "member",
      "members": 5,
      "repositories": ["runbooks"]
    }
  ],
  "preferences": {
    "theme": "dark",
    "notifications": {
      "email": true,
      "slack": true,
      "digest": "weekly"
    }
  }
}`

const createdUser = `
{
  "id": "usr_01HQ8ZK3N9",
  "email": "juno@northwind.dev",
  "name": "Juno Park",
  "role": "member",
  "active": true,
  "createdAt": "2026-07-27T12:00:00Z",
  "invitation": {
    "id": "inv_7be21a44",
    "status": "pending",
    "expiresAt": "2026-08-03T12:00:00Z",
    "invitedBy": {
      "id": "usr_01HQ8ZK3M4",
      "name": "Ada Okafor"
    }
  },
  "teams": [
    { "id": "team_data", "name": "Data Platform", "role": "member" }
  ]
}`
