#!/usr/bin/env python3
"""PreToolUse(Bash): block installing or executing packages that are not on .claude/approved-deps.txt.
Reused from previalogistic (.claude/hooks/guard-deps.py, docs/11-reuse.md); threat: typosquats and
LLM-hallucinated package names (slopsquatting), docs/07 S15.
Approved list format (one per line):  <ecosystem> <name>   # optional note
  ecosystems: go | npm | pip | cargo     npm scopes may use a wildcard:  npm @radix-ui/*
  go entries approve sub-packages too:  go github.com/aws/aws-sdk-go-v2  approves .../service/s3
"""
import json
import os
import re
import shlex
import sys

SEPARATORS = {"&&", "||", ";", "|", "&"}
VALUE_FLAGS = {"--filter", "-F", "--dir", "-C", "--prefix", "--workspace", "--cwd", "-r", "--requirement",
               "--index-url", "-i", "--extra-index-url", "--target", "-t"}


def project_dir() -> str:
    return os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()


def load_approved():
    path = os.path.join(project_dir(), ".claude", "approved-deps.txt")
    approved = {"go": set(), "npm": set(), "pip": set(), "cargo": set()}
    try:
        with open(path, encoding="utf-8") as fh:
            for line in fh:
                line = line.split("#", 1)[0].strip()
                if not line:
                    continue
                parts = line.split()
                if len(parts) >= 2:
                    eco = "go" if parts[0] == "tool" else parts[0]
                    approved.setdefault(eco, set()).add(parts[1])
    except FileNotFoundError:
        pass
    return approved


def go_ok(mod: str, approved) -> bool:
    mod = mod.split("@", 1)[0]
    if mod in ("", ".", "./...", "all") or mod.startswith(("./", "../")):
        return True
    if "." not in mod.split("/", 1)[0]:  # stdlib-like path (e.g. "fmt")
        return True
    return any(mod == a or mod.startswith(a.rstrip("/") + "/") for a in approved["go"])


def npm_name(spec: str) -> str:
    if spec.startswith("@"):
        parts = spec.split("/")
        if len(parts) < 2:
            return spec
        name = parts[1].split("@", 1)[0]
        return f"{parts[0]}/{name}"
    return spec.split("@", 1)[0]


def npm_ok(spec: str, approved) -> bool:
    if spec.startswith((".", "/", "file:", "link:", "workspace:")):
        return True
    if re.match(r"^(https?:|git\+|github:|git:)", spec):
        return False
    name = npm_name(spec)
    if name in approved["npm"]:
        return True
    if name.startswith("@"):
        scope = name.split("/")[0]
        return f"{scope}/*" in approved["npm"]
    return False


def segments(cmd: str):
    try:
        lex = shlex.shlex(cmd, posix=True, punctuation_chars=";&|")
        lex.whitespace_split = True
        tokens = list(lex)
    except ValueError:
        tokens = cmd.split()
    seg = []
    for t in tokens:
        if t in SEPARATORS:
            if seg:
                yield seg
            seg = []
        else:
            seg.append(t)
    if seg:
        yield seg


def positional(args):
    out, skip = [], False
    for a in args:
        if skip:
            skip = False
            continue
        if a in VALUE_FLAGS:
            skip = True
            continue
        if a.startswith("-"):
            continue
        out.append(a)
    return out


# Flags allowed before an npm-family subcommand, per tool. Anything else there → block (fail closed).
PM_VALUE_FLAGS = {"npm": {"--prefix", "-w", "--workspace"}, "pnpm": {"--filter", "-F", "--dir", "-C"},
                  "yarn": {"--cwd"}, "bun": {"--cwd"}}
PM_BOOL_FLAGS = {"npm": {"-g", "--global", "-ws", "--workspaces", "-s", "--silent", "-q", "--quiet"},
                 "pnpm": {"-r", "--recursive", "-w", "--workspace-root", "-g", "--global", "-s", "--silent"},
                 "yarn": {"-s", "--silent"}, "bun": {"-g", "--global"}}
# Flags allowed before the command of npx / npm exec / pnpm dlx / bun x. Anything else → block (fail closed).
RUNNER_FLAGS = {"-y", "--yes", "--no", "-q", "--quiet", "-s", "--silent"}


def pm_split(exe, args):
    """(subcommand, rest), or (None, None) when an unknown flag precedes the subcommand."""
    i = 0
    while i < len(args) and args[i].startswith("-"):
        flag, has_eq = args[i].split("=", 1)[0], "=" in args[i]
        if flag in PM_VALUE_FLAGS[exe]:
            i += 1 if has_eq else 2
        elif flag in PM_BOOL_FLAGS[exe] and not has_eq:
            i += 1
        else:
            return None, None
    return (args[i], args[i + 1:]) if i < len(args) else ("", [])


def pm_positional(exe, args):
    """Positional args for npm-family tools. Only this tool's value flags take a value; every other flag is
    treated as a boolean, so the token after an unknown flag is still checked (fail closed)."""
    out, skip = [], False
    for a in args:
        if skip:
            skip = False
        elif a.startswith("-"):
            skip = a in PM_VALUE_FLAGS[exe]
        else:
            out.append(a)
    return out


def runner_packages(args):
    """Packages an exec-style command downloads: each --package/-p before the command, else the command itself.
    None when an unknown flag precedes the command (e.g. `-c '<shell>'`, `--cache <dir>`) → caller blocks."""
    pkgs, i = [], 0
    while i < len(args) and args[i].startswith("-"):
        a = args[i]
        if a == "--":
            i += 1
            break
        if a.startswith("--package="):
            pkgs.append(a.split("=", 1)[1])
        elif a in ("--package", "-p") and i + 1 < len(args):
            pkgs.append(args[i + 1])
            i += 1
        elif a not in RUNNER_FLAGS:
            return None
        i += 1
    if not pkgs and i < len(args):
        pkgs.append(args[i])
    return pkgs


def create_pkg(spec):
    """`npm create x` runs create-x; `@s/x` runs @s/create-x; `@s` runs @s/create (versions stripped)."""
    name = npm_name(spec)
    if not name.startswith("@"):
        return "create-" + name
    if "/" not in name:
        return "@" + name[1:].split("@", 1)[0] + "/create"
    scope, rest = name.split("/", 1)
    return f"{scope}/create-{rest}"


def check(seg, approved):
    # strip env assignments and wrappers like `env`, `time`
    while seg and (re.match(r"^[A-Za-z_][A-Za-z0-9_]*=", seg[0]) or seg[0] in ("env", "time", "command")):
        seg = seg[1:]
    if not seg:
        return []
    exe = os.path.basename(seg[0])
    args = seg[1:]
    bad = []
    if exe == "go" and args:
        sub = args[0]
        if sub in ("get", "install", "run") and len(args) > 1:
            for a in positional(args[1:]):
                if sub == "run" and not "@" in a:
                    continue  # `go run ./cmd/api` or a .go file
                if not go_ok(a, approved):
                    bad.append(("go", a))
    elif exe in ("npm", "pnpm", "yarn", "bun") and args:
        sub, rest = pm_split(exe, args)
        if sub is None:
            return [(exe, "unrecognised flag before the subcommand: " + " ".join(args))]
        adds = {"npm": ("install", "i", "add", "in", "ins"), "pnpm": ("add", "install", "i"),
                "yarn": ("add",), "bun": ("add", "install", "i")}[exe]
        # commands that download and run a package (pnpm/yarn `exec` only run already-installed local bins)
        runners = {"npm": ("exec", "x", "create", "init"), "pnpm": ("dlx", "create"),
                   "yarn": ("dlx", "create"), "bun": ("x", "create")}[exe]
        if sub in adds:
            for a in pm_positional(exe, rest):
                if not npm_ok(a, approved):
                    bad.append(("npm", a))
        if sub in runners:
            if sub in ("create", "init"):  # `npm create vite` runs the package create-vite
                pkgs = [create_pkg(a) for a in pm_positional(exe, rest)[:1]]
            else:
                pkgs = runner_packages(rest)
            if pkgs is None:
                bad.append((exe, "unrecognised flag before the command: " + " ".join(rest)))
            else:
                for a in pkgs:
                    if not npm_ok(a, approved):
                        bad.append(("npm", a))
    elif exe in ("npx", "bunx") and args:
        pkgs = runner_packages(args)
        if pkgs is None:
            return [(exe, "unrecognised flag before the command: " + " ".join(args))]
        for a in pkgs:
            if not npm_ok(a, approved):
                bad.append(("npm", a))
    elif exe in ("pip", "pip3", "uv", "poetry", "pipx") and args:
        if args[0] in ("install", "add") or (exe == "uv" and args[:2] == ["pip", "install"]):
            rest = args[2:] if exe == "uv" and args[0] == "pip" else args[1:]
            for a in positional(rest):
                name = re.split(r"[<>=!~\[ ]", a, 1)[0].lower()
                if name not in approved["pip"]:
                    bad.append(("pip", a))
    elif exe == "cargo" and args and args[0] in ("add", "install"):
        for a in positional(args[1:]):
            if a.split("@", 1)[0] not in approved["cargo"]:
                bad.append(("cargo", a))
    return bad


def main():
    try:
        data = json.load(sys.stdin)
    except Exception:
        sys.exit(0)
    cmd = (data.get("tool_input") or {}).get("command", "")
    if not cmd:
        sys.exit(0)
    approved = load_approved()
    bad = []
    for seg in segments(cmd):
        bad.extend(check(seg, approved))
    if bad:
        listed = ", ".join(f"{eco}:{name}" for eco, name in bad)
        sys.stderr.write(
            f"BLOCKED by .claude/hooks/guard_deps.py (docs/07 S15): not on .claude/approved-deps.txt -> {listed}\n"
            "Package names suggested by AI can be hallucinated and registered by attackers (slopsquatting).\n"
            "Stop. Tell the human which package you want and why, and ask them to verify it (official repo, owner,\n"
            "age, downloads, recent ownership changes) and add it via the /add-dependency checklist.\n"
        )
        sys.exit(2)
    sys.exit(0)


if __name__ == "__main__":
    main()
