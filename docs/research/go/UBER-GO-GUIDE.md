# Uber Go Style Guide Reference

Source:

https://github.com/uber-go/guide

Role:

A production-oriented Go engineering reference, not an architecture authority.

Useful topics include:

- interfaces and interface compliance;
- pointer/value receivers;
- zero-value mutexes;
- slice/map boundaries;
- `defer`;
- errors;
- panic avoidance;
- globals and `init()`;
- process exit ownership;
- goroutine lifecycle;
- package/naming conventions;
- struct/map initialization;
- table-driven tests;
- functional options;
- linting/tooling.

Usage rule:

```text
real problem
  -> understand Go mechanism
  -> reproduce with smallest code
  -> inspect relevant guide section
  -> understand why the recommendation exists
  -> compare cost/value in current code
  -> adopt / adapt / reject / keep open
  -> prove with code/tests
```
