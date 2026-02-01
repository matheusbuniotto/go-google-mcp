# GitHub Repo Best Practices – What Should(n’t) Be There

## What’s currently tracked (and is fine)

Everything currently committed is appropriate to have on GitHub:

- **Source:** `cmd/`, `pkg/` – application and library code  
- **Config:** `go.mod`, `go.sum` – dependencies (go.sum should stay for reproducible builds)  
- **CI:** `.github/workflows/ci.yml` – build and test  
- **Docs:** `README.md`, `LICENSE`, `docs/*.md` – no secrets, only instructions  
- **Ignore rules:** `.gitignore` – no secrets listed, only patterns  

No binaries, no `token.json` or `client_secrets.json`, no planning-only files (task_plan, findings, progress, briefing) are tracked. That matches best practice.

---

## What should never be on GitHub

Already covered by `.gitignore` (and not committed):

- **Secrets:** `token.json`, `client_secrets.json`, `.go-google-mcp/`, `.gogo-mcp/`  
- **Binaries:** `/go-google-mcp`, `/gogo-mcp`, `/mcp-test`  
- **Planning-only files:** `task_plan.md`, `findings.md`, `progress.md`, `briefing.md`  

Code does not hardcode API keys or passwords; auth uses OAuth and config paths only.

---

## .gitignore updates (done)

`.gitignore` was tightened so these stay untracked by default:

| Category              | Added patterns                                      |
|-----------------------|-----------------------------------------------------|
| Binaries              | `*.exe`, `*.dll`, `*.so`, `*.dylib`                 |
| Env / secrets         | `*.env`, `.env`, `.env.*` (optional `!.env.example`) |
| IDE / local tooling   | `.cursor/`, `.claude/`, `.gemini/`, `.opencode/`    |
| Go test artifacts     | `*.test`, `coverage.out`, `coverage.html`, `*.coverprofile` |
| OS junk               | `.DS_Store`, `Thumbs.db`                            |

- **`.agents/`** – Left **not** ignored so you can commit it if you share skills; add `.agents/` to `.gitignore` only if you want that folder to stay local.

---

## CI

- `.github/workflows/ci.yml` only runs `go build` and `go test`; it does not log or expose secrets.  
- Using `go-version: '1.24'` with Go 1.24.x in `go.mod` is fine.

---

## Summary

- **On GitHub:** Only what’s already tracked (code, go.mod/go.sum, CI, docs, .gitignore) – all appropriate.  
- **Never commit:** Secrets, binaries, env files with secrets, local-only planning files – all ignored.  
- **Going forward:** Stronger .gitignore keeps IDE/tooling dirs, env files, test/coverage output, and OS junk out of the repo by default.
