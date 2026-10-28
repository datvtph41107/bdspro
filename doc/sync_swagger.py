#!/usr/bin/env python3
"""Synchronize versioned OpenAPI source with embedded static assets."""
import json
from pathlib import Path
from shutil import copyfile

base = Path(__file__).resolve().parent
spec = base / "swagger" / "openapi.json"
data = json.loads(spec.read_text())
assert data["openapi"].startswith("3.") and data["paths"]
dest = base / "statik" / "swagger"
dest.mkdir(parents=True, exist_ok=True)
for name in ("index.html", "openapi.json"):
    copyfile(base / "swagger" / name, dest / name)
print("OpenAPI operations:", sum(len(v) for v in data["paths"].values()))
