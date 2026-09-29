# Go Package Naming

## Core theorem

The package name is caller-facing API context.

A caller imports by package path but normally references exported identifiers as:

```go
http.Server
time.Duration
bytes.Buffer
```

Package naming and exported API naming therefore need to be designed together.

## Practical principles

Prefer package names that are:

- lowercase;
- short but clear;
- usually one word;
- meaningful at call sites;
- focused on a responsibility/capability;
- unlikely to require routine import aliases.

Treat names such as these as questions rather than automatic destinations:

- util / utils;
- common / shared;
- helpers;
- misc;
- types;
- interfaces;
- generic service/model buckets.

The problem is not that these strings are forbidden. The problem is that they
often hide an unclear responsibility.

## Call-site test

Inspect how the API reads for consumers:

```go
http.Server
client.New(...)
payment.Charge(...)
```

Avoid repeating context merely because the identifier is exported:

```go
http.HTTPServer
client.NewClient
```

when the package already supplies the missing context.

This is guidance, not a mechanical rule. A canonical type such as
`client.Client` can still be perfectly reasonable.

## Boundary signal

If a package cannot be named meaningfully, ask whether the package should exist
yet.

Before extracting a package:

1. What responsibility does it own?
2. Who calls it?
3. What belongs here?
4. What clearly does not belong here?
5. What dependencies does it introduce?
6. Does the package name make call sites clearer?
7. Would callers routinely need aliases?
8. Could the code remain unexported in the current package until a real shared
   responsibility emerges?

## Performance

Changing the spelling of a package name does not make runtime execution faster.

The important distinction is:

```text
package-name spelling
  -> readability / API ergonomics

package boundary + dependency graph
  -> compile/build/cache/tooling cost
  -> maintainability
  -> initialization and dependency consequences
```

Generic dependency-magnet packages can indirectly increase build and cognitive
cost. Runtime performance claims still require measurement.

## Evidence

Primary official sources:

- Go Blog: Package names
- Effective Go
- Go Code Review Comments
- Organizing a Go module
- Go specification

Additional engineering evidence:

- Google Go Style
- Uber Go Style Guide
- Go standard library
- Moby
- Terraform
- Prometheus
- Kubernetes
- CockroachDB
