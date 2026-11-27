#!/usr/bin/env python3
"""Local, read-only HTTP/log correlation smoke test. No credentials or providers."""
import argparse
import json
import time
import uuid
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener


CASES = (
    ("GET", "/api/v1/health", 200, "info", ".Health"),
    ("GET", "/api/v1/me", 401, "warn", ".APIAuth"),
    ("POST", "/api/v1/auth/login", 403, "warn", ".BrowserMutation"),
    ("GET", "/api/v1/__logging_probe_missing__", 404, "warn", None),
)


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def local_base(value):
    parsed = urlsplit(value)
    if (parsed.scheme != "http" or parsed.hostname not in ("127.0.0.1", "::1")
            or parsed.username is not None or parsed.password is not None
            or parsed.path not in ("", "/") or parsed.query or parsed.fragment
            or parsed.port is None):
        raise ValueError("base must be http://127.0.0.1:PORT or http://[::1]:PORT")
    return value.rstrip("/")


def probe(base, case):
    method, path, expected, _, _ = case
    marker = "logging-probe-" + uuid.uuid4().hex
    req = Request(base + path + "?probe=" + marker, method=method,
                  data=b"{}" if method == "POST" else None,
                  headers={"Content-Type": "application/json", "X-Request-ID": marker})
    opener = build_opener(ProxyHandler({}), NoRedirect())
    try:
        response = opener.open(req, timeout=10)
    except HTTPError as exc:
        response = exc
    with response:
        if response.code != expected:
            raise ValueError(f"{method} {path}: expected {expected}, got {response.code}")
        rid = response.headers.get("X-Request-ID", "")
        if str(uuid.UUID(rid)) != rid or rid == marker:
            raise ValueError(f"{path}: missing canonical server UUID")
    return rid, marker


def events_for(path, rid):
    events = []
    with path.open(encoding="utf-8", errors="replace") as stream:
        for line in stream:
            try:
                event = json.loads(line)
            except ValueError:
                continue  # old text or a partially written final line
            if isinstance(event, dict) and event.get("request_id") == rid:
                events.append(event)
    return events


def verify_events(events, case, marker):
    method, path, status, level, suffix = case
    started = [e for e in events if e.get("message") == "http.started"]
    completed = [e for e in events if e.get("message") == "http.request"]
    if len(started) != 1 or len(completed) != 1:
        raise ValueError(f"{path}: expected exactly one started and completed event")
    event = completed[0]
    if (event.get("method"), event.get("path"), event.get("status"), event.get("level")) != (method, path, status, level):
        raise ValueError(f"{path}: completion metadata mismatch")
    if started[0].get("method") != method or started[0].get("path") != path or events.index(started[0]) >= events.index(event):
        raise ValueError(f"{path}: started metadata or ordering mismatch")
    handler = event.get("handler")
    if suffix is not None:
        if not isinstance(handler, str) or not handler.endswith(suffix) or not event.get("handler_source"):
            raise ValueError(f"{path}: expected executed handler {suffix}")
    elif handler is not None:
        raise ValueError(f"{path}: unmatched route must not invent a handler")
    if marker in json.dumps(events):
        raise ValueError(f"{path}: query or untrusted request ID leaked")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default="http://127.0.0.1:8080")
    parser.add_argument("--log-file", type=Path, default=Path("data/log/bdspro.log"))
    args = parser.parse_args()
    try:
        base = local_base(args.base)
        if not args.log_file.is_file():
            raise ValueError("JSON log file not found; enable file DEBUG logging first")
        for case in CASES:
            rid, marker = probe(base, case)
            deadline = time.monotonic() + 5
            while True:
                events = events_for(args.log_file, rid)
                if any(e.get("message") == "http.request" for e in events):
                    break
                if time.monotonic() >= deadline:
                    raise ValueError(f"{case[1]}: no completed JSON event; check file level/path or rotation")
                time.sleep(0.1)
            verify_events(events, case, marker)
            print(f"PASS {case[0]} {case[1]} {case[2]} rid={rid[:8]}")
        print("PASS HTTP/log correlation (4 cases)")
    except (ValueError, OSError, URLError) as exc:
        parser.exit(1, f"FAIL: {exc}\n")


if __name__ == "__main__":
    main()
