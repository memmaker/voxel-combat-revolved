# Handover

Branch `roadmap-progress`, 11 commits on top of `main`. Nothing pushed. The live checklist is `TODO.md`.

## State

All tests pass: `go test ./...`. The desktop client (`-tags client`), the server (`-tags server`) and the WASM build (`build_web.sh`, which uses `go.web.mod`) all compile. Check the WASM build separately: `go.web.mod` and `web/glfw` (a stub of glfw for the browser) must keep up with new dependencies and keys. The game starts at a main menu: play against the AI, host, join, map editor, settings.

## What exists now (beyond the original PoC)

| Area | Where |
|---|---|
| Main menu, squad picker, settings, lobby | `client/menu.go`, `app.go` |
| HUD, unit list, turn banner and timer | `client/hud.go` |
| Pause, help (F1) and end-of-match overlays | `client/overlay.go` |
| Immediate-mode UI toolkit | `client/ui.go`, `engine/gui/rects.go` |
| AI (cover, line of sight, grenades, overwatch, reactions, melee, heal) | `game/ai_brain.go`, driven by `game/ai.go` (headless) and `client/autoplay.go` |
| Reactions: Dodge, Brace, Parry | `game/reactions.go`, `server/srv_reaction.go` |
| Melee, Heal | `game/act_melee.go`, `game/act_heal.go`, `server/srv_melee.go`, `server/srv_heal.go` |
| Classes (Scout, Marksman, Heavy, Medic) | `assets/content/factions.json` |
| Game content (maps, factions, weapons, items) | `assets/content/*.json`, `game/content.go` |
| Map generator (`gen:<style>:<seed>`) | `game/mapgen.go` |
| Lighting (sun and lamps) | `engine/voxel/lighting.go`, `client/shader/chunk.frag` |
| Cutaway (Z/C), silhouettes behind walls | `engine/voxel/map.go`, `client/battleclient.go` |
| Map editor | `client/gs_edit_map.go`, `client/editor.go` |
| Sound and music (synthesized, no asset files) | `client/audio.go`, `client/music.go` |
| Reconnect (seat held 60 s), turn timer | `server/reconnect.go`, `server/server.go` |

## Key architecture decisions

- **The server is authoritative for all damage.** Explosion damage is sent to clients as `UnitDamages` / `EffectDamages` lists. Clients never recompute it; it used to desync.
- **Server message handling is serialized** by one mutex (`BattleServer.mu`).
- **Generated maps are deterministic per name**, so the server and every client build the same map independently.
- **Reactions are hidden from opponents**: the `UnitInstance.Reaction` field has `json:"-"`.
- **Singleplayer** runs the server in-process over in-memory pipes (`game.MemPipe`).

## Testing tools

- `BT_MATCHES=40 go test ./server/ -run TestAIMatchFinishes -v` plays headless AI-vs-AI matches on all maps. This is the best regression and balance check.
- Environment variables:
  - `BT_QUICKSTART=1` skips the menu.
  - `BT_AUTOPLAY=1` lets the AI play your side.
  - `BT_SHOT_DIR=<dir> BT_SHOT_EVERY=<s>` makes the game save its own frames.
  - `BT_EDITOR=<map>` opens the editor directly.
  - `BT_MENU_SCREEN`, `BT_OVERLAY=help` open a specific screen.
  - `BT_AI_LOG=1` logs AI decisions.
  - `BT_AI_DIFFICULTY=easy|normal|hard` sets the AI difficulty.
- `BT_DUMP_SOUNDS=<dir> go test ./client -run 'TestSounds|TestMusic'` writes all sounds and the music as WAV files.

## Open issues

- **Own unit not dying in first-person view:** not reproduced. Enemy shots are only shown in first person with the action camera, and an autoplay run with it enabled saw no deaths.
- **Not built yet:**
  - Deflect, the counter to special attacks (Brace covers explosives for now).
  - Feign death.
  - Fire and oil.
  - A wall-building action for players.
  - More objective types.
  - The campaign and progression layer.
  - Rebindable keys.
  - An interactive tutorial.
  - Async multiplayer.
- **Balance:** headshots are frequent, so fights are lethal. AI-vs-AI wins split roughly 50/50 between factions. No deliberate balance pass has been done yet.
- **Not verified by hand:** I couldn't click in the window myself, so menu clicks, editor tools and free aim were only checked through tests and screenshots, not played.
- **The Minecraft `.construction` maps are still unused.** They need a mapping from Minecraft block names to this game's tileset.
- **The web build can't save** (no filesystem). The editor's Save button shows an error there.
