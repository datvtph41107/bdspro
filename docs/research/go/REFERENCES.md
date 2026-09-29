# Go Engineering References

Use sources in this order when practical.

## Official Go

- Go specification: https://go.dev/ref/spec
- Go modules reference: https://go.dev/ref/mod
- go.mod reference: https://go.dev/doc/modules/gomod-ref
- Organizing a Go module: https://go.dev/doc/modules/layout
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
- Package names: https://go.dev/blog/package-names

## Engineering style evidence

- Uber Go Style Guide: https://github.com/uber-go/guide
- Google Go Style Guide: https://google.github.io/styleguide/go/

These are engineering references, not language specifications.

## Production repositories

### Moby

Repository:
https://github.com/moby/moby

Observed module examples:

```text
github.com/moby/moby/v2
github.com/moby/moby/client
```

Useful for studying multi-module repositories, public client modules, nested
module boundaries, `replace`, and package API design.

### Terraform

Repository:
https://github.com/hashicorp/terraform

Module:

`github.com/hashicorp/terraform`

Useful for studying a large application module and extensive use of
Go-enforced `internal/` packages.

### Prometheus

Repository:
https://github.com/prometheus/prometheus

Module:

`github.com/prometheus/prometheus`

Useful for studying capability-oriented packages such as `scrape`,
`discovery`, `rules`, and `storage`.

### Kubernetes

Repository:
https://github.com/kubernetes/kubernetes

Module:

`k8s.io/kubernetes`

Useful for proving that canonical Go module identity does not have to equal the
physical GitHub repository URL.

### CockroachDB

Repository:
https://github.com/cockroachdb/cockroach

Module:

`github.com/cockroachdb/cockroach`

Useful for studying naming constraints and package organization at very large
codebase scale.

## Reading rule

Do not copy a source tree.

For each package/module observed, ask:

```text
what responsibility does it own?
who consumes it?
why are these files together?
why is another boundary separate?
what visibility/release/dependency pressure caused it?
does the same pressure exist in our code?
```
