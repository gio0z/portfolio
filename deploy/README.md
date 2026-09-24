# Deploy — ginapps.my.id

The site runs on **hp-server-linux** as a single Go binary serving the Astro
build on `127.0.0.1:8321`. The Cloudflare tunnel already running on that host
publishes it at `ginapps.my.id`, so Cloudflare terminates TLS and the origin
needs no certificate and no open inbound port.

Deployment is pull-based, matching the `ba-rekon` and `kasirumkm` pattern on the
same host: a systemd timer checks `origin/main` every minute and runs
`deploy/deploy.sh` when it has moved. GitHub never connects to the server.

```
GitHub main ──(timer pulls)──► /srv/projects/portfolio ──► systemd portfolio :8321
                                                                    ▲
                        Cloudflare edge (TLS) ── tunnel ────────────┘
```

## Hostnames

Four origins are required, because the server refuses to start in production
without them. Each needs a Cloudflare DNS record and a tunnel ingress rule.

| Origin | Purpose |
|---|---|
| `https://ginapps.my.id` | public site; also the canonical URL and sitemap origin |
| `https://admin.ginapps.my.id` | admin app; pins the GitHub OAuth redirect URI |
| `https://lab.ginapps.my.id` | published Design Lab artifacts |
| `https://preview.ginapps.my.id` | sandboxed preview bundles |

Only the first two are needed for the public site and admin to work. The last two
are required as *values* even before their content is served.

## 1. Server bootstrap (one time)

```bash
ssh root@hp-server-linux

mkdir -p /etc/portfolio /srv/projects

# Go (go.mod requires 1.23.0 or newer; use the current 1.23.x or later)
curl -fsSL https://go.dev/dl/go1.23.12.linux-amd64.tar.gz -o /tmp/go.tgz
mkdir -p /root/.local/go && tar -C /root/.local/go --strip-components=1 -xzf /tmp/go.tgz

# Node (Astro requires >= 22.12)
curl -fsSL https://nodejs.org/dist/v22.23.3/node-v22.23.3-linux-x64.tar.xz -o /tmp/node.tar.xz
mkdir -p /root/.local/node && tar -C /root/.local/node --strip-components=1 -xJf /tmp/node.tar.xz
```

Verify both resolve, since the deploy script calls them by absolute path:

```bash
/root/.local/go/bin/go version      # go1.23.12 or newer
/root/.local/node/bin/node -v       # v22.23.3 or newer
```

`npm` is a script whose shebang is `env node`, so it only runs with the Node bin
directory on `PATH`. The deploy script exports it; running `npm` by hand on this
host requires the same:

```bash
export PATH=/root/.local/node/bin:$PATH
```

## 2. Repository

```bash
cd /srv/projects
git clone https://github.com/gio0z/portfolio.git
cd portfolio
git checkout main
```

The repository is public, so no deploy key is needed for cloning or fetching.

## 3. Configuration

`/etc/portfolio/portfolio.env`, root-readable only. Start from `.env.example`
in the repository, which documents every variable.

```bash
install -m 600 /dev/null /etc/portfolio/portfolio.env
```

Required values:

```bash
APP_ENV=production
HOST=127.0.0.1
PORT=8321

PORTFOLIO_PUBLIC_ORIGIN=https://ginapps.my.id
ADMIN_PUBLIC_ORIGIN=https://admin.ginapps.my.id
LAB_PUBLIC_ORIGIN=https://lab.ginapps.my.id
PREVIEW_PUBLIC_ORIGIN=https://preview.ginapps.my.id

ADMIN_ALLOWED_GITHUB_LOGIN=gio0z
ADMIN_GITHUB_CLIENT_ID=<from the OAuth app>
ADMIN_GITHUB_CLIENT_SECRET=<from the OAuth app>

ADMIN_SESSION_SIGNING_KEY=<openssl rand -hex 32>
MCP_TOKEN_SECRET=<openssl rand -hex 32>

REGISTRY_PATH=/var/lib/portfolio/registry.sqlite
ARTIFACT_STORE_ROOT=/var/lib/portfolio/artifacts
```

`HOST=127.0.0.1` matters: this host has no firewall, and the API includes the
admin surface. The tunnel reaches the service over loopback, so nothing else
needs to.

Create the data directories, then confirm the server actually starts before
installing any unit:

```bash
mkdir -p /var/lib/portfolio/artifacts
set -a; . /etc/portfolio/portfolio.env; set +a
/srv/projects/portfolio/portfolio-server -host 127.0.0.1 -port 8321 \
  -dist /srv/projects/portfolio/frontend/dist
```

`FRONTEND_DIST` is not used here because the unit passes `-dist` explicitly.

## 4. GitHub OAuth app

Creating an OAuth App is **web-UI only** — GitHub has no API for it, so this step
cannot be automated. At <https://github.com/settings/developers> → New OAuth App:

| Field | Value |
|---|---|
| Application name | Portfolio Admin |
| Homepage URL | `https://ginapps.my.id` |
| Authorization callback URL | `https://admin.ginapps.my.id/api/admin/auth/callback` |

The callback is pinned to `ADMIN_PUBLIC_ORIGIN`, so it must match exactly, and
both values must move together if either changes.

After creating the app, put the two credentials into
`/etc/portfolio/portfolio.env`, replacing the `PLACEHOLDER` values the bootstrap
wrote. They are not read from anywhere else, and the server stores no copy:

```bash
install -m 600 /dev/null /tmp/oauth.env   # edit, then:
# ADMIN_GITHUB_CLIENT_ID=<id>
# ADMIN_GITHUB_CLIENT_SECRET=<secret>
grep -q ADMIN_GITHUB_CLIENT_ID /etc/portfolio/portfolio.env \
  && sed -i "s|^ADMIN_GITHUB_CLIENT_ID=.*|ADMIN_GITHUB_CLIENT_ID=<id>|" /etc/portfolio/portfolio.env
systemctl restart portfolio
```

While the placeholders are in place the server still starts, because the values
are non-empty; sign-in simply fails at GitHub. That is deliberate — it keeps the
public site up while the OAuth app is being registered — but it means a
placeholder is indistinguishable from a typo until someone tries to sign in.

## 5. Cloudflare

Tunnel ingress is managed in the Cloudflare dashboard (the tunnel token is
remote-configured, so there is no local `config.yml`).

**Already done:** the ingress rule for `ginapps.my.id` exists and already points
at `http://127.0.0.1:8321` — that rule is what produced the 502s while nothing
was listening, and it began serving correctly the moment the service started.
No Cloudflare change is needed for the public site.

**Still to add**, for the admin and Lab origins:

| Public hostname | Service |
|---|---|
| `admin.ginapps.my.id` | `http://127.0.0.1:8321` |
| `lab.ginapps.my.id` | `http://127.0.0.1:8321` |
| `preview.ginapps.my.id` | `http://127.0.0.1:8321` |

Add a proxied `CNAME` for each to the tunnel target, and a Public Hostname entry
pointing at the same port. The Go handler decides what each origin serves from
the request path; the application distinguishes them by configuration, not by
listener.

Tunnel and account identifiers, if the dashboard asks for them:

```
account tag: e6eadeb051f54dac0fcc9a4ac9a3d45a
tunnel id:   4fee6d91-3b10-4fda-8cea-5c02ddcc3530
```

Set SSL/TLS mode to **Full**, not Flexible: the tunnel origin speaks plain HTTP
over loopback, and Flexible would send users over plain HTTP at the edge.

## 6. Install the units

```bash
cd /srv/projects/portfolio
install -m 644 deploy/portfolio.service /etc/systemd/system/portfolio.service
install -m 644 deploy/portfolio-deploy.service /etc/systemd/system/portfolio-deploy.service
install -m 644 deploy/portfolio-deploy.timer /etc/systemd/system/portfolio-deploy.timer
chmod +x deploy/deploy.sh

systemctl daemon-reload
systemctl enable --now portfolio
systemctl enable --now portfolio-deploy.timer
```

The first `deploy.sh` run performs the initial build (frontend + binary) and
restarts the unit.

## 7. Verify

```bash
systemctl status portfolio --no-pager
curl -s http://127.0.0.1:8321/api/health
curl -sI https://ginapps.my.id/about/ | head -3
curl -s https://ginapps.my.id/robots.txt
curl -s https://ginapps.my.id/sitemap-0.xml | head -c 200
```

Expected: health returns `{"status":"ok"}`, `/about/` returns `200`, and the
sitemap lists `https://ginapps.my.id/...` URLs — not `localhost`, which would
mean `PORTFOLIO_PUBLIC_ORIGIN` was missing at build time.

## Operations

```bash
journalctl -u portfolio -f              # application logs
journalctl -u portfolio-deploy -n 50    # deploy history
systemctl start portfolio-deploy        # deploy now, without waiting for the timer
```

**Rollback.** The deploy script never leaves a half-built state: the binary is
built as `portfolio-server.new` and moved into place only on success, and a
failed health check aborts the run. To go back to a previous revision:

```bash
cd /srv/projects/portfolio
git reset --hard <previous-sha>
( cd frontend && /root/.local/node/bin/npm ci && /root/.local/node/bin/npm run build )
/root/.local/go/bin/go build -o portfolio-server .
systemctl restart portfolio
```

The timer will redeploy `main` on its next tick, so pause it first with
`systemctl stop portfolio-deploy.timer` if the rollback must hold.

## Known gap: Lab and preview artifact serving

`LAB_PUBLIC_ORIGIN` and `PREVIEW_PUBLIC_ORIGIN` are configured, but nothing
serves their files yet. Publishing writes immutable versions to
`var/portfolio/lab/versions/<hash>/` and flips a pointer file
(`var/portfolio/lab/pointers/<slug>.json`) that names the live hash, and the
public URL is `<LAB_PUBLIC_ORIGIN>/<slug>` — so the pointer must be resolved at
request time. There is no code for that resolution yet, and the threat model
assigns CSP, frame isolation, and `noindex` on preview and Lab routes to the
edge.

Consequence today: the public site, the admin review flow's API, and publishing
all work, but published Lab case studies are not reachable at their public URL
and preview bundles cannot be rendered in an iframe.

## Security notes

- `portfolio.service` binds loopback and runs with `NoNewPrivileges` and
  `PrivateTmp`.
- `/etc/portfolio/portfolio.env` holds the OAuth secret and both HMAC keys;
  keep it mode `600` and out of git. `.env` is gitignored.
- `APP_ENV` must be `production` on this host. Any other value relaxes admin
  auth, and would additionally send session cookies over plain HTTP.
- Rate limiting on login, review, and publication endpoints is spec-mandated but
  not implemented in code (`docs/security/admin-mcp-lab-threat-model.md` §4).
  Until it lands, it belongs at the Cloudflare edge.
