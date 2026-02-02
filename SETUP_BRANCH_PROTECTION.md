# GitHub Web UI Setup Guide - Step by Step

## PART 1: BRANCH PROTECTION RULES

### Step 1: Open Repository Settings
1. Go to: https://github.com/matheusbuniotto/go-google-mcp
2. Click **Settings** tab (top right)
3. Left sidebar: Click **Branches**

### Step 2: Add Branch Protection Rule
1. Click **Add rule** button
2. In "Branch name pattern" field: Type **main**
3. Leave other options as default for now

### Step 3: Configure Protection Options

Check these boxes in order:

#### Require a pull request before merging
- ✓ Check: "Require a pull request before merging"
- ✓ Check: "Require approvals" → Set to **1**
- ✓ Check: "Dismiss stale pull request approvals when new commits are pushed"
- ✓ Check: "Require review from Code Owners"
- ✓ Check: "Require approval of the most recent reviewable push"

#### Require status checks to pass
- ✓ Check: "Require status checks to pass before merging"
- ✓ Check: "Require branches to be up to date before merging"
- Leave "Status checks that must pass" empty for now (we'll add after CI is running)

#### Other protections
- ✓ Check: "Require conversation resolution before merging"
- ✓ Check: "Require signed commits"
- Skip: "Require deployments to succeed before merging"
- Skip: "Lock branch"

#### Rules enforcement
- ✓ Check: "Enforce all the above rules for administrators too"
- Leave unchecked: "Allow force pushes"
- Leave unchecked: "Allow deletions"

### Step 4: Save
1. Scroll to bottom
2. Click **Create** button

---

## PART 2: CODE SECURITY & ANALYSIS

### Step 1: Go to Security Settings
1. Settings → Left sidebar → **Code security and analysis**

### Step 2: Enable Security Features

#### Dependabot
- "Dependabot alerts" → Click **Enable** (if not already on)
- "Dependabot security updates" → Click **Enable**
- "Dependabot version updates" → Click **Enable**

#### Secret Scanning
- "Secret scanning" → Should be enabled (public repos)
- "Push protection" → Click **Enable** (prevents secrets from being pushed)

#### Code Scanning (Optional for now)
- "Code scanning" → Can set up later with CodeQL

---

## PART 3: PULL REQUEST SETTINGS

### Step 1: Go to PR Settings
1. Settings → Left sidebar → **Pull Requests**

### Step 2: Configure
- ✓ Check: "Allow auto-merge"
- ✓ Check: "Automatically delete head branches"
- ✓ Check: "Always suggest updating branch when behind"

---

## PART 4: CREATE CI/CD WORKFLOW

### Step 1: Create Workflow File
1. Go to your repo main page: https://github.com/matheusbuniotto/go-google-mcp
2. Click **Actions** tab
3. Click **set up a workflow yourself** or **New workflow**
4. Name the file: `.github/workflows/ci.yml`

### Step 2: Copy This CI Configuration

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.24']

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}

    - name: Run go fmt
      run: |
        if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
          echo "Code formatting issues found:"
          gofmt -s -d .
          exit 1
        fi

    - name: Run go vet
      run: go vet ./...

    - name: Run tests
      run: go test -v -race -coverprofile=coverage.txt ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.txt
        flags: unittests
        name: codecov-umbrella

    - name: Install golangci-lint
      run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

    - name: Run golangci-lint
      run: golangci-lint run ./...
```

### Step 3: Save/Commit
1. Scroll to bottom
2. Click **Commit new file**
3. Commit message: `ci: add GitHub Actions workflow`
4. Click **Commit new file** button

---

## PART 5: SETUP DEPENDABOT

### Step 1: Create Dependabot Config
1. Go to repo main page: https://github.com/matheusbuniotto/go-google-mcp
2. Click **Add file** dropdown → **Create new file**
3. File path: `.github/dependabot.yml`

### Step 2: Copy This Configuration

```yaml
version: 2
updates:
  # Go dependencies
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "daily"
      time: "09:00"
      timezone: "UTC"
    open-pull-requests-limit: 10
    reviewers:
      - "matheusbuniotto"
    labels:
      - "chore"
      - "dependencies"
    commit-message:
      prefix: "chore(deps):"
      include: "scope"
    allow:
      - dependency-type: "production"
      - dependency-type: "development"
```

### Step 3: Save/Commit
1. Scroll to bottom
2. Click **Commit new file**
3. Commit message: `chore: add Dependabot configuration`

---

## PART 6: COMMIT CODEOWNERS & CONTRIBUTING

The files have been created locally:
- `.github/CODEOWNERS`
- `CONTRIBUTING.md`

Commit and push them:

```bash
git add .github/CODEOWNERS CONTRIBUTING.md
git commit -m "docs: add CODEOWNERS and CONTRIBUTING guidelines"
git push origin feat/sheets-tabs-and-batch-update
```

Then open a Pull Request to merge these files into main.

---

## VERIFICATION CHECKLIST

After completing all steps, verify:

- ☐ Branch protection rule shows on Settings > Branches > main
- ☐ CI workflow appears in Actions tab and runs on PRs
- ☐ Dependabot configuration saved in Settings > Code security
- ☐ Secret scanning enabled in Code security and analysis
- ☐ CODEOWNERS file merged into main
- ☐ CONTRIBUTING.md visible in repo root
- ☐ Next PR requires: 1 approval + CI pass + resolved conversations

---

## Testing the Setup

1. Create a test branch
2. Make a small change
3. Open a Pull Request
4. Verify:
   - CI runs automatically
   - Requires approval
   - Cannot merge until CI passes
   - Cannot merge without approval

---

## Troubleshooting

**CI workflow not running?**
- Check Actions tab > CI workflow > Logs
- Verify `.github/workflows/ci.yml` syntax is correct

**Branch protection not enforcing?**
- Refresh the page
- Check Settings > Branches > main rule is enabled

**Dependabot not creating PRs?**
- Give it 24-48 hours for first run
- Check Settings > Code security and analysis > Dependabot is enabled
- Check `.github/dependabot.yml` syntax

---

Last Updated: 2026-02-01
