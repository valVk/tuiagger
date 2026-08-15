#!/usr/bin/env python3
"""Automated smoke test for tuiagger.

`task smoke` launches the real TUI for a human to click through manually
(bubbletea needs a real terminal — this sandboxed environment can't drive
one directly). This script does the same thing unattended: it builds the
binary, drives it through a real pty (stdlib `pty`, no third-party
dependency — pip installs are blocked in this environment anyway), and
asserts on screen content after each keystroke.

It is a smoke test, not a full behavioral suite (that's what `task test`
is for): it walks the keyboard-shortcut surface documented in CLAUDE.md
one flow at a time and checks the UI actually got where a keypress should
take it. No live HTTP calls are made (petstore3.swagger.io may not be
reachable from CI/sandboxed environments) — execute-flow correctness is
covered by internal/tui's own unit tests via a stub HTTP client.

Usage:
    python3 scripts/smoke_test.py
    task smoke-auto
"""

from __future__ import annotations

import fcntl
import os
import re
import select
import shutil
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PETSTORE_SPEC = os.path.join(ROOT, "internal/openapi/testdata/petstore.json")

ANSI_RE = re.compile(r"\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[=>]")


def strip_ansi(s: str) -> str:
    return ANSI_RE.sub("", s)


class Failure(AssertionError):
    pass


# bubbletea/lipgloss probe terminal capabilities on startup — an OSC 11
# query ("what's your background color?") and a DSR query ("where's the
# cursor?") — and block waiting for a reply for several seconds before
# giving up and proceeding with defaults. A real terminal answers these
# instantly; our pty harness is the "terminal" here, so it has to too, or
# every launch eats that multi-second stall.
OSC11_QUERY = b"\x1b]11;?\x1b\\"
OSC11_REPLY = b"\x1b]11;rgb:0000/0000/0000\x1b\\"  # claim a dark background
DSR_QUERY = b"\x1b[6n"
DSR_REPLY = b"\x1b[1;1R"  # claim cursor at row 1, col 1


class Session:
    """A tuiagger process attached to a real pty, with expect()/send()."""

    def __init__(self, argv: list[str], env: dict[str, str], cols=120, rows=40):
        self.pid, self.fd = os.forkpty()
        if self.pid == 0:  # child
            os.execvpe(argv[0], argv, env)
            os._exit(127)  # execvpe only returns on failure
        self._set_winsize(cols, rows)
        self._buf = ""

    def _set_winsize(self, cols: int, rows: int) -> None:
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))

    def send(self, s: str) -> None:
        os.write(self.fd, s.encode())

    def _answer_terminal_queries(self, chunk: bytes) -> None:
        if OSC11_QUERY in chunk:
            os.write(self.fd, OSC11_REPLY)
        if DSR_QUERY in chunk:
            os.write(self.fd, DSR_REPLY)

    def _drain(self, timeout: float) -> str:
        """Read whatever arrives within `timeout`, resetting the clock each
        time new bytes show up (so a slow-but-steady stream of redraw
        frames doesn't get cut off mid-render)."""
        out = b""
        deadline = time.time() + timeout
        while time.time() < deadline:
            r, _, _ = select.select([self.fd], [], [], 0.05)
            if self.fd not in r:
                continue
            try:
                chunk = os.read(self.fd, 65536)
            except OSError:
                break
            if not chunk:
                break
            self._answer_terminal_queries(chunk)
            out += chunk
            deadline = time.time() + min(timeout, 0.3)
        return out.decode(errors="ignore")

    def expect(self, pattern: str, timeout: float = 4.0) -> str:
        """Wait for `pattern` (regex, applied to ANSI-stripped output) to
        appear in newly-arrived output since the last expect() call."""
        self._buf = ""
        deadline = time.time() + timeout
        while time.time() < deadline:
            self._buf += self._drain(0.3)
            screen = strip_ansi(self._buf)
            if re.search(pattern, screen):
                return screen
        raise Failure(
            f"timed out after {timeout}s waiting for {pattern!r}\n"
            f"--- last screen ---\n{strip_ansi(self._buf)}\n--- end ---"
        )

    def close(self, timeout: float = 3.0) -> int | None:
        try:
            os.kill(self.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        deadline = time.time() + timeout
        while time.time() < deadline:
            pid, status = os.waitpid(self.pid, os.WNOHANG)
            if pid != 0:
                return status
            time.sleep(0.05)
        os.kill(self.pid, signal.SIGKILL)
        os.waitpid(self.pid, 0)
        return None


def build_binary(tmp_bin_dir: str) -> str:
    bin_path = os.path.join(tmp_bin_dir, "tuiagger-smoke")
    subprocess.run(
        ["go", "build", "-o", bin_path, "./cmd/tuiagger"],
        cwd=ROOT,
        check=True,
    )
    return bin_path


def setup_fake_home(tmp_home: str) -> None:
    collection_dir = os.path.join(tmp_home, ".tuiagger", "PetStore")
    os.makedirs(collection_dir, exist_ok=True)
    shutil.copy(PETSTORE_SPEC, os.path.join(collection_dir, "petstore.json"))


Step = tuple[str, str, str]  # (description, keys-to-send, pattern-to-expect)

SPEC_TITLE = "Swagger Petstore"  # internal/openapi/testdata/petstore.json's info.title prefix

STEPS: list[Step] = [
    ("launch shows the left panel and header", "", SPEC_TITLE),
    ("Enter expands the first tag", "\r", r"GET|POST|PUT|DELETE"),
    ("j moves onto the newly-expanded endpoint row", "j", SPEC_TITLE),
    ("l focuses the right panel on the selected endpoint", "l", r"Try it out \(t\)"),
    (
        "t enters try-it-out mode, BODY lists declared content types as tabs",
        "t",
        r"PARAMETERS(?:.|\n)*application/x-www-form-urlencoded",
    ),
    ("Esc exits try-it-out back to browse", "\x1b", r"Try it out \(t\)"),
    ("m opens the manual request builder", "m", r"MANUAL REQUEST"),
    ("Esc closes the manual request builder", "\x1b", SPEC_TITLE),
    ("i opens the info popup", "i", r"SERVERS"),
    ("Tab switches info popup section", "\t", r"AUTH|ENVIRONMENTS"),
    ("Esc closes the info popup", "\x1b", SPEC_TITLE),
    ("? opens the help cheatsheet", "?", r"KEYBOARD SHORTCUTS"),
    ("? closes the help cheatsheet", "?", SPEC_TITLE),
]


def run_smoke() -> None:
    with tempfile.TemporaryDirectory(prefix="tuiagger-smoke-bin-") as tmp_bin_dir, \
         tempfile.TemporaryDirectory(prefix="tuiagger-smoke-home-") as tmp_home:
        print("Building tuiagger...")
        bin_path = build_binary(tmp_bin_dir)

        print("Seeding an isolated collection (HOME redirected, real ~/.tuiagger untouched)...")
        setup_fake_home(tmp_home)

        env = dict(os.environ)
        env["HOME"] = tmp_home
        env["TERM"] = "xterm-256color"

        session = Session([bin_path, "PetStore"], env)
        passed = 0
        try:
            for desc, keys, pattern in STEPS:
                if keys:
                    session.send(keys)
                session.expect(pattern)
                passed += 1
                print(f"  ok  {desc}")

            print("  ..  q quits cleanly")
            session.send("q")
            status = session.close()
            if status is not None and os.WIFEXITED(status) and os.WEXITSTATUS(status) != 0:
                raise Failure(f"tuiagger exited with status {os.WEXITSTATUS(status)}")
            print(f"  ok  q quits cleanly")
            passed += 1
        except Failure as e:
            session.close()
            print(f"  FAIL {e}", file=sys.stderr)
            print(f"\n{passed}/{len(STEPS) + 1} steps passed before failure", file=sys.stderr)
            sys.exit(1)

        print(f"\nAll {passed}/{len(STEPS) + 1} smoke steps passed.")


if __name__ == "__main__":
    run_smoke()
