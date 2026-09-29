# Cuck Code: the docs

Back to the [README](../README.md).

## Troubleshooting

**Typing `claude` doesn't play the intro after setup.** Open a new terminal, because existing ones haven't re-read their startup files. Then run `type claude`, which should say it's a function. If you define your own `alias claude=...`, the alias wins; remove it or wrap it with `cuck`.

**npm never asked me the question.** npm can't prompt when there's no terminal (CI, some IDE terminals). pnpm and Bun don't run install scripts unless you approve them. In either case, run `cuck setup` yourself.

**`the cuck-code-<platform> package is missing`.** The native binary was skipped by `--omit=optional` or `--no-optional`. Reinstall without it.

**The colours look wrong.** Force a mode with `--color 256`, `--color 16` or `--color none`, or set `"color"` in the config file.

**There's a short pause after the wink.** That's the agent starting up. Claude Code, for example, takes a second to boot. cuck prints *"claude is taking it from here..."* while you wait.

**My terminal is tiny.** Below 40×14 you get the same story as text captions.

## How it works

The scene isn't made of pre-drawn frames. cuck casts one ray per character cell from a fixed camera at seated eye height, into a small 3D bedroom:

- **Room.** A hinged door with a lit hallway behind it, a bed, a nightstand lamp, a curtained window and a rug. The lighting comes from the lamp, the hallway (through the doorway only), moonlight and the glow of your own screen.
- **Cast.** She's built from animated ellipsoids under the blanket; she blinks, bounces a foot, props herself up, follows him with her eyes, draws her knees up and squirms. He's a billboard standing in the world, so perspective and occlusion by the door frame and bed come for free, with limbs driven by a walk cycle and a lean for climbing onto the bed. His head turns independently of his body for the look back, and his face is drawn feature by feature (brows, eyes, blush, mouth) so the stare, smirk and wink stay crisp at any size.
- **Camera.** It never moves. The chair's red leather armrests sit in the bottom corners.
- **Output.** Frames are quantised per terminal to 24-bit, 256-colour, 16-colour or plain ASCII, and only changed cells are redrawn. No Unicode is required.
- **Terminal handling.** The intro runs on the alternate screen with the cursor hidden and keyboard echo off. Keys typed during the show never leak into the agent. The terminal is restored on normal exit, Ctrl+C, SIGTERM, SIGHUP and panics.

```text
cmd/cuck/            CLI, setup/uninstall/hook/agents commands
internal/animation/  scene, cast, timeline, transition, encoder
internal/shellhook/  agent list, shell wrappers, startup-file editing
internal/tty/        terminal modes, size, key polling, Windows VT mode
internal/runner/     command parsing and exec
internal/config/     config file
npm/                 npm packages (launcher + per-platform binaries)
tools/preview/       renders frames/GIFs without a terminal
```

## Releasing

Releases are cut by pushing a tag:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

[`.github/workflows/release.yml`](../.github/workflows/release.yml) then does the following:

1. Runs the tests.
2. Builds all six binaries.
3. Creates the GitHub release with checksums and the install scripts.
4. Publishes `cuck-code` and its `cuck-code-<os>-<cpu>` packages to npm. This needs an `NPM_TOKEN` repository secret.

Tags with a hyphen (`v0.2.0-rc.1`) become prereleases published under the `next` npm tag.

For a local build:

```bash
make test                     # vet + tests
make dist VERSION=v0.1.0      # dist/cuck-<os>-<arch>
make npm VERSION=v0.1.0       # npm/out/, ready for npm publish
make demo                     # regenerate the GIF and screenshots
go run ./tools/preview -t 6.9 -text   # print a frame as ASCII
```
