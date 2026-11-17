# R07 — Router composition, scopes, and mounted child registration

Gitea reference: https://github.com/go-gitea/gitea/blob/main/modules/web/router.go

Gitea's `Group` temporarily extends the current route prefix and
middleware scope, then restores the parent's scope. `Mount` attaches the
sub-router to the Chi matcher using the parent group and AfterRouting chain.

BDSPro keeps the same route-declaration vocabulary but uses Gin as a
single matcher. Its `Mount` copies compiled route descriptors from
the child into the parent at **registration time**. A child route
registered after that copy would not exist under the parent path.
Such mutations now panic: `Methods`, `Group`, `Mount`,
`BeforeRouting`, and `AfterRouting` are closed on a mounted child.

The child is still reusable at multiple mount points; freezing only
prohibits later edits to its route tree. Parent route declarations
may continue after a mount, so sibling groups retain independent
permission guard scopes.

Expected matched-route guard sequence:

```text
parent AfterRouting
  -> parent Group guards
  -> child AfterRouting
  -> child Group guards
  -> endpoint actions
```

A sibling route outside the parent Group must not inherit those
resource/permission guards. All of these contracts are tested with
HTTP requests using the same Gin matcher.

This additional freeze is intentionally BDSPro-specific: Chi/Gitea
mounts a sub-router directly rather than copying flattened Gin
route descriptors. The goal is to ensure route declarations cannot
silently misrepresent production behavior.

Real authentication and resource policies are not introduced here.
