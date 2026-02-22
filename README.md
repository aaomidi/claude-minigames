# Claude Minigames

A collection of 19 games written entirely by Claude Code in a single Go binary using [Ebitengine](https://ebitengine.org/). No external assets — all graphics are vector primitives, all audio is procedurally generated 8-bit chiptune.

## Games

| Game | Genre | Highlights |
|------|-------|------------|
| Mine Sweep | Puzzle | Bot solver, screen shake, 3D cells |
| Naval Strike | Strategy | Probability-density AI opponent |
| Snek | Arcade | Gradient body, directional eyes |
| Road Hopper | Arcade | Traffic + water logs, squish animation |
| Traffic Dodge | Arcade | Scrolling road, detailed vehicles |
| Silver Ball | Arcade | Ball physics, flipper motion blur |
| Serpents & Steps | Board | vs AI, animated pieces |
| Tower Defense | Strategy | 3 tower types, projectile trails, 16 waves |
| Deep Miner | Adventure | Dig for minerals, upgrade drill, parallax scrolling |
| Idle Clicker | Idle | Prestige system, pulse rings |
| Idle Evolution | Idle | 10 evolution stages, DNA helix visualization |
| Symbiotic Grid | Puzzle | **Novel game concept** — dual-organism mirrored movement |
| Harvest Valley | RPG | Seasons, crops, NPCs, fishing, day/night cycle |
| Hellshot | FPS | DDA raycaster, humanoid enemies, blood particles |
| Warp Gate | Puzzle | Momentum-conserving portals, 8 levels |
| Hero Summon | Gacha | Pull animations, pity system, battles |
| Realm Quest | RPG | Skills, combat, quests, cooking |
| Crime City | Action | Car physics, wanted stars, missions |
| Epoch | 4X Strategy | Tech tree, fog of war, era-based dynamic music, Gandhi nuclear Easter egg |

## Run

```bash
go run .
```

## Install

```bash
go install github.com/aaomidi/claude-minigames@latest
```

Then run `claude-minigames` from your terminal.

## Build

```bash
go build -o games .
./games
```

## Controls

Each game shows its controls in-game. Common controls:
- **ESC** — return to menu
- **R** — restart game

## Requirements

- Go 1.21+
- On Linux: `libasound2-dev` (for audio)
- On macOS/Windows: no extra dependencies

## How it was made

Every line of code was written by [Claude Code](https://claude.ai/claude-code) (Anthropic's CLI agent) in a single conversation. The games were built iteratively — starting with Minesweeper and expanding to 19 games with procedural audio, visual polish, and AI opponents.

## License

MIT
