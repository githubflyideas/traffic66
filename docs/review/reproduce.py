#!/usr/bin/env python3
"""Replay pure-Go review probes against an archived, exact source commit.

Requires Python 3.12+, Git and an existing Go toolchain. Does not download or
install tools, touch the checkout's production files, or run storage probes.
The final test command is expected to fail on the reviewed baseline. Inspect
the saved log: compilation or environment failures are not bug reproductions,
and an expected invariant failure is not a passing quality gate.
"""
import argparse
import io
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile

BASELINE = "a0c7c54c0cdfed21d96ee6a146fb4bf632889c05"
PACKAGES = ["internal/decode/nf", "internal/decode/sflow", "internal/auth"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default="go")
    parser.add_argument("--work", required=True, type=Path, help="new, empty workspace; never use a source checkout")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--cache", type=Path, help="optional existing Go build cache")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    work = args.work.resolve()
    if work.exists() and any(work.iterdir()):
        parser.error("--work must be empty to avoid overwriting any files")
    work.mkdir(parents=True, exist_ok=True)
    paths = PACKAGES + ["internal/encode", "internal/flow", "internal/packet", "internal/wire", "go.mod", "go.sum"]
    archive = subprocess.run(["git", "-C", str(root), "archive", BASELINE, "--", *paths], check=True, stdout=subprocess.PIPE).stdout
    with tarfile.open(fileobj=io.BytesIO(archive)) as source:
        source.extractall(work, filter="data")
    temp = work / "tmp"
    temp.mkdir()
    env = dict(os.environ, GOTOOLCHAIN="local", CGO_ENABLED="0", TMP=str(temp), TEMP=str(temp), TMPDIR=str(temp))
    env.pop("REVIEW_EXPECT_FIXED", None)
    if args.cache:
        env["GOCACHE"] = str(args.cache.resolve())
    logs = ["Baseline: " + BASELINE, "Platform: " + sys.platform]

    def run(label, arguments):
        command = [args.go, *arguments]
        result = subprocess.run(command, cwd=work, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, encoding="utf-8", errors="replace", timeout=180)
        logs.extend(["\n=== " + label + " ===", "command: " + repr(command), "REVIEW_EXPECT_FIXED=" + env.get("REVIEW_EXPECT_FIXED", ""), result.stdout, "exit=" + str(result.returncode)])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text("\n".join(logs) + "\n", encoding="utf-8")
        print(label + ": exit=" + str(result.returncode), flush=True)
        return result.returncode

    if run("toolchain", ["version"]):
        return 2
    if run("unmodified package baseline", ["test", "-count=1", *["./" + p for p in PACKAGES]]):
        return 2
    for package in PACKAGES:
        for probe in sorted((root / "docs/review/probes" / package).glob("*_test.go.txt")):
            shutil.copyfile(probe, work / package / probe.name.removesuffix(".txt"))
    if run("NF observed characterization", ["test", "-count=1", "-v", "-run", "^TestReview", "./internal/decode/nf"]):
        return 2
    env["REVIEW_EXPECT_FIXED"] = "1"
    return run("desired invariants (failure expected on reviewed baseline)", ["test", "-count=1", "-v", "-run", "^TestReview", *["./" + p for p in PACKAGES]])


if __name__ == "__main__":
    sys.exit(main())
