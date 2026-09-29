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

## Install

```bash
npm install -g cuck-code
```

At the end of the install it asks **"Do you really want to get cucked?"** If you say yes, typing `claude`, `codex`, `cursor-agent`, `gemini` or any other supported agent plays the intro first.

No npm:

```bash
curl -fsSL https://raw.githubusercontent.com/ruskaruma/cuck-code/main/packaging/install.sh | sh     # macOS / Linux
irm https://raw.githubusercontent.com/ruskaruma/cuck-code/main/packaging/install.ps1 | iex          # Windows PowerShell
```

## Usage

```bash
claude                          # after setup: the intro plays, then Claude Code opens
cuck codex --full-auto          # or launch any agent through cuck directly
cuck setup                      # get cucked later / cuck uninstall to undo it
command claude                  # skip the intro once
```

Any key skips ahead during the intro, and Ctrl+C cancels. Config, every flag, the full list of supported agents and troubleshooting are in the [docs](docs/README.md).

## License

[MIT](LICENSE) © 2026 Ishaan Sinha ([@ruskaruma](https://github.com/ruskaruma))
