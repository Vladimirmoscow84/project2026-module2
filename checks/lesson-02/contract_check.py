#!/usr/bin/env python3
"""Check the six lesson-02 routes without requiring a host package install.

The script checks the public shape that the lesson asks the student to agree on:
JSON/OpenAPI syntax, required paths, and request/response declarations. It does
not prove that Go handlers, PostgreSQL, or authentication work at runtime.
"""
from __future__ import annotations
import json
from pathlib import Path
import sys

PUBLIC = {
    "/api/v1/auth/login": "post",
    "/api/v1/me": "get",
    "/api/v1/auth/logout": "post",
    "/api/v1/analyses": ("post", "get"),
    "/api/v1/analyses/{analysis_id}": "get",
}

def fail(message: str) -> int:
    print(f"contract check: {message}")
    return 1

def main() -> int:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("contracts")
    public_path, ds_path = root / "public.openapi.json", root / "ds.openapi.json"
    try:
        public = json.loads(public_path.read_text(encoding="utf-8"))
        ds = json.loads(ds_path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as error:
        return fail(f"cannot read both JSON documents: {error}")
    paths = public.get("paths")
    if not isinstance(paths, dict) or set(paths) != set(PUBLIC):
        return fail("public paths must contain exactly the six lesson routes")
    for path, methods in PUBLIC.items():
        expected = (methods,) if isinstance(methods, str) else methods
        route = paths.get(path)
        if not isinstance(route, dict) or set(route) != set(expected):
            return fail(f"public route {path} has the wrong methods")
    ds_paths = ds.get("paths")
    if not isinstance(ds_paths, dict) or set(ds_paths) != {"/score"} or set(ds_paths["/score"]) != {"post"}:
        return fail("DS contract must contain only POST /score")
    for document, label in ((public, "public"), (ds, "DS")):
        if document.get("openapi") != "3.0.3" or not isinstance(document.get("components"), dict):
            return fail(f"{label} document is not an OpenAPI 3.0.3 object")
    print("contract check: PASS (shape only; runtime remains a separate check)")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
