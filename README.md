<div align="center">

<img src="https://raw.githubusercontent.com/ruskaruma/cuck-code/main/docs/assets/logo.svg" alt="Cuck Code" width="560">

**Your coding agent does the work. You watch from the chair.**

A cinematic ANSI intro that plays before Claude Code, Codex, Cursor, Gemini and every other terminal coding agent.

[![npm](https://img.shields.io/npm/v/cuck-code?color=cb3837)](https://www.npmjs.com/package/cuck-code)
[![CI](https://github.com/ruskaruma/cuck-code/actions/workflows/ci.yml/badge.svg)](https://github.com/ruskaruma/cuck-code/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

![demo](https://raw.githubusercontent.com/ruskaruma/cuck-code/main/docs/assets/demo.gif)

</div>

## The story

You like delegating your work to agents. You hand over the task, lean back, and watch someone else do what you were supposed to do. Cuck Code just makes it honest.

Every time you start your agent, your terminal becomes the view from the chair in the corner of the bedroom. She's already in bed, and she isn't waiting for you. The sign above her says it plainly: *take the back seat, let it do the work.*

The door opens and your agent walks in, name tag and all. She doesn't hide her relief. Finally, someone who does it right. He climbs onto the bed, leans in and kisses her, while you sit there doing exactly nothing. Then he turns around, looks you dead in the eye, smirks, and winks.

The lights go out. Your agent gets to work.

There are no dependencies. It's one small binary you install and forget about.

---

## Install

### npm (recommended)

```bash
npm install -g cuck-code
```

At the end of the install, you'll be asked:

```text
  Do you really want to get cucked? [y/N]
```

If you answer **y**, typing any supported agent's name in your terminal (`claude`, `codex`, `cursor-agent`, `gemini`, ...) plays the intro first. If you answer **n** (or there's no terminal to ask on, e.g. in CI), nothing is changed. You can say yes later with `cuck setup`.

If you install with `sudo npm install -g`, the question applies to **every user on the machine**.

### macOS / Linux, without npm

```bash
curl -fsSL https://raw.githubusercontent.com/ruskaruma/cuck-code/main/packaging/install.sh | sh
```

This installs to `~/.local/bin`, verifies the SHA-256 checksum and asks the same question. You can set `CUCK_INSTALL_DIR`, `CUCK_VERSION` or `CUCK_SKIP_SETUP=1` to change that.

### Windows, without npm

```powershell
irm https://raw.githubusercontent.com/ruskaruma/cuck-code/main/packaging/install.ps1 | iex
```

This installs to `%LOCALAPPDATA%\Programs\cuck`, adds it to your user `PATH` and asks the question.

### Manual

Download a binary for your platform from [Releases](https://github.com/ruskaruma/cuck-code/releases), put it on your `PATH` as `cuck` (`cuck.exe` on Windows), and run `cuck setup`.

### From source

Requires Go 1.26+.

```bash
git clone https://github.com/ruskaruma/cuck-code && cd cuck-code
make install      # builds and installs to ~/.local/bin/cuck
cuck setup
```

---

## Getting cucked

`cuck setup` hooks your shell so that typing an agent's name goes through the chair first:

```text
$ cuck setup

  CUCK CODE

  From now on, when you type any of these, you watch from the chair first:

    claude codex cursor-agent cursor gemini agy copilot opencode aider goose
    amp crush qwen cline kiro-cli droid auggie kimi openhands kilo kilocode
    codebuff jules junie letta qodo plandex grok windsurf

  Installed here:  claude, codex, cursor-agent
  Applies to:      you (alice)
  Shell files:     ~/.bashrc, ~/.zshrc
  Only interactive shells are affected; scripts and IDEs run agents as usual.

  Do you really want to get cucked? [y/N]
  > y
```

What it does:

- It adds a small, clearly marked block to your shell startup files. The block asks `cuck` for a wrapper function for each agent. Supported shells are **bash**, **zsh**, **fish** and **PowerShell** (both 7 and Windows PowerShell 5.1).
- It only affects **interactive** shells. Scripts, cron jobs, CI, IDE integrations and other programs that run `claude` or `codex` are untouched.
- Agents that aren't installed are fine to list. You get the usual `command not found`.
- It is safe to run again. Running `setup` twice never duplicates anything, and npm upgrades refresh the hook without asking again.

| Flag | |
|---|---|
| `--yes`, `-y` | don't ask |
| `--system` | hook every user on the machine: `/etc/bash.bashrc` or `/etc/bashrc`, `/etc/zsh/zshrc` or `/etc/zshrc`, `/etc/fish/conf.d`, and the all-users PowerShell profile. Needs root/Administrator. This is the default when run as root. |
| `--user` | only the current user, even when running as root |

Quick commands, like `claude --version`, `codex exec ...`, `claude -p ...`, `gemini mcp ...`, `update`, `login` and `help`, skip the intro and run immediately.

**Skipping the intro once:**

```bash
command claude                                    # bash, zsh, fish
& (Get-Command claude -CommandType Application)   # PowerShell
```

While the intro is playing, **any key** jumps to the wink, a second key skips the rest, and **Ctrl+C** cancels without launching the agent.

### Supported agents

| Command | Agent | | Command | Agent |
|---|---|---|---|---|
| `claude` | Claude Code | | `kiro-cli` | Kiro CLI (was Amazon Q Developer CLI) |
| `codex` | OpenAI Codex CLI | | `droid` | Factory Droid |
| `cursor-agent` | Cursor CLI | | `auggie` | Augment Code CLI |
| `cursor` | Cursor (editor launcher) | | `kimi` | Kimi Code CLI |
| `gemini` | Gemini CLI | | `openhands` | OpenHands CLI |
| `agy` | Google Antigravity CLI | | `kilo`, `kilocode` | Kilo Code CLI |
| `copilot` | GitHub Copilot CLI | | `codebuff` | Codebuff |
| `opencode` | OpenCode | | `jules` | Google Jules CLI |
| `aider` | Aider | | `junie` | JetBrains Junie CLI |
| `goose` | Block Goose | | `letta` | Letta Code |
| `amp` | Sourcegraph Amp | | `qodo` | Qodo CLI |
| `crush` | Charm Crush | | `plandex` | Plandex |
| `qwen` | Qwen Code | | `grok` | Grok CLI |
| `cline` | Cline CLI | | `windsurf` | Windsurf (editor launcher) |

Run `cuck agents` to see which of these are installed on your machine and where the hook is active. To add your own agents or drop built-in ones, see the [config file](#configuration).

### Uninstall

```bash
cuck uninstall            # removes the hook from every shell file it was added to
npm uninstall -g cuck-code
```

Run `cuck uninstall` **before** removing the package, because npm doesn't run cleanup scripts for global packages. If you forget, nothing breaks: the hook checks that `cuck` exists and does nothing otherwise. Remove the leftover block from your shell files whenever you like; it's the part between `# >>> cuck-code >>>` and `# <<< cuck-code <<<`. System-wide hooks need `sudo cuck uninstall`.

---

## Usage

You don't need the hook to use cuck. You can launch any agent through it directly:

```bash
cuck                              # the default agent (claude)
cuck codex                        # any agent on your PATH
cuck aider --model sonnet         # arguments after the agent name go to the agent
cuck --agent claude -- --resume   # the same thing, using --agent
cuck 'my-agent --flag "quoted"'   # any command line
cuck 'FOO=1 my-agent | tee log'   # shell syntax runs through sh -c / cmd /c
cuck --no-animation claude        # skip the intro
cuck --dry-run yolo               # print what would run
```

| Flag | Default | |
|---|---|---|
| `-a`, `--agent` | `claude` | agent name, config alias or full command line |
| `--no-animation` | off | skip the intro |
| `--duration` | `7500` | intro length in ms (3000–30000); the story speeds up or slows down to fit |
| `--fps` | `30` | frame rate (5–60) |
| `--color` | `auto` | `truecolor`, `256`, `16` or `none` |
| `--dry-run` | | print the resolved command and exit |
| `--version` | | |

| Command | |
|---|---|
| `cuck setup` | get cucked (install the shell hook) |
| `cuck uninstall` | stop getting cucked |
| `cuck agents` | list wrapped agents and where the hook is installed |
| `cuck hook <shell>` | print the hook code for `bash`, `zsh`, `fish` or `powershell` |

cuck checks that the agent exists **before** the intro, so a typo won't cost you the show. On macOS and Linux the agent replaces the cuck process (`exec`), so signals, job control and exit codes behave exactly as if you had run the agent yourself. On Windows, cuck waits for the agent and exits with its exit code.

## Configuration

### Environment

| Variable | |
|---|---|
| `CUCK_AGENT` | default agent when none is given |
| `CUCK_NO_ANIMATION=1` | never play the intro (the hook stays installed but launches instantly) |
| `CUCK_SKIP_SETUP=1` | don't ask the question during `npm install` or the install scripts |
| `CUCK_CONFIG` | config file path |
| `NO_COLOR` | render the intro in plain ASCII |

### Config file

The config file is optional and can contain any of the fields below.

- **Linux:** `~/.config/cuck-code/config.json`
- **macOS:** `~/Library/Application Support/cuck-code/config.json`
- **Windows:** `%AppData%\cuck-code\config.json`

`cuck --help` prints the exact path on your machine.

```json
{
  "agent": "claude",
  "agents": {
    "yolo": "claude --dangerously-skip-permissions",
    "cx": "codex --full-auto"
  },
  "animation": true,
  "duration_ms": 7500,
  "fps": 30,
  "color": "auto",
  "hook_agents": ["my-internal-agent"],
  "hook_exclude": ["cursor", "goose"]
}
```

| Field | |
|---|---|
| `agent` | the agent `cuck` launches with no arguments |
| `agents` | aliases: `cuck yolo` runs the command line on the right |
| `animation` | `false` never plays the intro |
| `duration_ms`, `fps`, `color` | same as the flags |
| `hook_agents` | extra commands for the shell hook to wrap |
| `hook_exclude` | built-in agents the hook should leave alone |

When no agent is given, cuck picks one in this order: `--agent`, then a positional agent, then `$CUCK_AGENT`, then `agent` in the config file, then `claude`.

## License

[MIT](LICENSE) © 2026 Ishaan Sinha ([@ruskaruma](https://github.com/ruskaruma))
