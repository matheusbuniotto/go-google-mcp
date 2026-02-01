# Code Review: Google Keep Integration

**Target:** New Keep feature (list, create, get, update, delete notes) and consistency with existing codebase.  
**Reviewer:** Code reviewer skill (correctness, maintainability, readability, efficiency, security, edge cases, testability).  
**TODO/FIXME scan:** No leftover TODO/FIXME/XXX comments found in Go code.

---

## Summary

The Google Keep integration fits the existing patterns: a dedicated `pkg/services/keep` package wrapping `google.golang.org/api/keep/v1`, and five MCP tools in `main.go` (list, create, get, update, delete). The update flow correctly documents the API limitation (no native PATCH) and implements a get → create new → delete old workaround. One edge case (empty note name) was fixed in the service. No TODO/FIXME-style comments remain in the repo.

---

## Findings

### Critical
- **None.**

### Improvements (addressed or suggested)

1. **Empty note name (addressed)**  
   `GetNote` and `DeleteNote` did not reject empty `name`, which could lead to confusing API errors. They now return a clear error: `note name is required`.

2. **UpdateNote failure after create**  
   If `UpdateNote` succeeds in creating the new note but fails on deleting the old one, the error message correctly states that the new note was created and gives its name. No code change required; behavior is documented by the error text.

3. **Duplicate list_items_json parsing**  
   In `main.go`, the same JSON parsing and `[]*keepapi.ListItem` construction appears in both `keep_create_note` and `keep_update_note`. Extracting a small helper (e.g. `parseKeepListItems(json string) ([]*keepapi.ListItem, error)`) would reduce duplication and keep behavior in sync. Optional refactor.

### Nitpicks
- **UpdateNoteInput field alignment**  
   In `UpdateNoteInput`, `Title` and `BodyText` use single-space before the comment, `ListItems` uses two; alignment could be normalized for readability only.

---

## Consistency with Previous Code

| Aspect | Rest of codebase | Keep feature |
|--------|-------------------|-------------|
| Service package | `pkg/services/<name>/` with `New(ctx, opts...)` and wrapped API client | Same: `pkg/services/keep/keep.go`, `keep.NewService` |
| Error wrapping | `fmt.Errorf("...: %w", err)` | Same in keep service |
| MCP tool params | `RequireString` for required, `GetString`/`GetInt` for optional with defaults | Same for all Keep tools |
| Tool result on error | `mcp.NewToolResultError(fmt.Sprintf("...", err))` | Same |
| Auth | Scopes in main + auth login flow | `keepapi.KeepScope` added in both places |
| Naming | `drive_search`, `tasks_list_tasks`, etc. | `keep_list_notes`, `keep_create_note`, etc. |

The new feature is consistent with existing style and structure.

---

## Edge Cases and Error Handling

- **Empty name:** Handled in `GetNote` and `DeleteNote` with explicit validation.
- **Invalid list_items_json:** Handled in create/update handlers; user gets a clear invalid JSON error.
- **UpdateNote: create ok, delete fail:** Error message explains that the new note exists and gives its name; no silent inconsistency.

---

## Testability

- No new tests were added. The keep service is straightforward to unit test (e.g. with a mock or fake for `keep.Service`). Suggested coverage: `GetNote`/`DeleteNote` with empty name, `UpdateNote` merge logic (title-only, body-only, list-only, keep-existing body).

---

## Conclusion

**Approved.** The Keep feature is correct, consistent with the rest of the project, and the only critical edge case (empty name) has been fixed. Optional follow-ups: extract list-items JSON parsing in `main.go` and add unit tests for the keep package.

---

## TODO / FIXME Scan Result

- **Go files:** No comments matching `TODO`, `FIXME`, `XXX`, or `HACK` (case-insensitive).  
- The only match for “to” near “do” was in `drive.go`: “unable to download file”, which is not a to-do comment.
