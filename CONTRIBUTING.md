# Contributing to go-google-mcp

Thank you for your interest in contributing! This document provides guidelines and best practices for contributing to the project.

## Code of Conduct

Be respectful and inclusive. We follow the Contributor Covenant code of conduct.

## Getting Started

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/YOUR_USERNAME/go-google-mcp.git`
3. **Create a branch**: `git checkout -b feat/your-feature-name`
4. **Make changes** and commit with clear messages
5. **Push** to your fork
6. **Open a Pull Request** to the main branch

## Branch Naming Conventions

Use semantic branch names:
- `feat/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test additions or fixes
- `chore/` - Dependency updates, CI/CD changes
- `perf/` - Performance improvements

Example: `feat/add-gmail-search-filters`

## Commit Message Guidelines

Write clear, descriptive commit messages:

```
Type: Brief description (50 chars max)

Longer explanation of the change if needed. Wrap at 72 characters.
Explain WHAT and WHY, not HOW.

Fixes #123
```

Types:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `refactor:` Code refactoring
- `test:` Test additions
- `chore:` Build/dependency updates
- `perf:` Performance improvement

## Pull Request Guidelines

### Before Submitting:
1. ✓ Run `go fmt ./...` for code formatting
2. ✓ Run `go vet ./...` for code analysis
3. ✓ Run `go test ./...` to ensure tests pass
4. ✓ Update documentation if needed

### PR Description Template:

```markdown
## Description
Brief summary of what this PR does

## Related Issues
Fixes #123

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe how to test the changes

## Checklist
- [ ] Code follows project style
- [ ] Tests pass locally
- [ ] Documentation is updated
- [ ] Commit messages are descriptive
```

## Code Style

- Follow Go conventions (gofmt, golint)
- Use clear variable names
- Add comments for non-obvious logic
- Keep functions focused and small
- Maximum line length: 100 characters

## Testing Requirements

- Write tests for new features
- Maintain or improve code coverage
- Tests must pass before merge
- Use table-driven tests for multiple cases

## Pull Request Review Process

All PRs require:
1. ✓ At least 1 approval from code owner
2. ✓ All CI checks passing
3. ✓ All conversations resolved
4. ✓ Branch up to date with main

## Merge Strategy

- **Squash merge** preferred for feature branches (keeps history clean)
- **Rebase merge** for multi-commit work that should be preserved
- Branches are auto-deleted after merge

## Questions?

Open an issue or discussion! We're here to help.

---

Happy coding! 🚀
