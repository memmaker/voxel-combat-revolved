# Roadmap to a complete, enjoyable game

Legend: `[x]` done · `[~]` partial / proof of concept · `[ ]` missing.
Design goals live in `readme.md` and `notes.txt`.

## 1. Core loop: start a match, play it, see it end
- [x] In-game main menu: play vs AI, host, join, settings (`client/menu.go`); terminal flow removed (`bgclient create|join host:port` still works for scripts)
- [~] Squad & loadout picker: 1-4 soldiers, unit type, weapon, item per soldier — no draft/point budget yet
- [x] End-of-match screen (victory/defeat, kills, losses, damage, hit rate; rematch / main menu / quit) and Esc pause menu with surrender (`client/overlay.go`)
- [x] Turn banner, turn counter, turn limit display (`client/hud.go`)
- [x] HUD: health bars, AP pips, selected-unit panel, floating damage numbers, clickable unit list
- [x] Sound: 17 synthesized effects plus a generated looping soundtrack, separate volume settings (`client/audio.go`, `client/music.go`)
- [~] Onboarding: F1 help overlay explaining controls, shooting, wounds, pressure, overwatch; no interactive tutorial

## 2. Singleplayer opponent
- [x] Real AI: cover, line-of-sight seeking, focus fire, grenades, overwatch, reloads, memory of last-known enemies, exploration, anti-stalemate impatience (`game/ai_brain.go`) — deployment still random
- [x] Headless AI-vs-AI match test (`server/match_test.go`, `BT_MATCHES=40 go test ./server/`) and graphical autoplay (`BT_AUTOPLAY=1`)
- [x] Difficulty levels: easy/normal/hard, chosen in the menu

## 3. Multiplayer
- [x] Lobby: host on port 9999 (shows LAN IPs), join by address with a list of open games (`ListGames`)
- [x] Disconnects: seat held 60 s, rejoin from the lobby (state + terrain replayed); optional turn timer (host setting)
- [~] Server-side validation: actions now rejected outside your turn, reload checks ownership, match ends as soon as a side is wiped out; still open: snap-shot target LOS check, free-aim angle sanity checks
- [ ] Chat / quick signals, activity during the enemy turn
- [ ] Async multiplayer (optional)

## 4. Yomi: every move has a counter
- [~] Overwatch (exists; designed rules not implemented: chosen blocks, escalating AP cost, once per encounter)
- [~] Hidden reactions: Dodge (50% evasion vs bullets, not explosions) and Brace (-1 per bullet, half explosion damage), secret until triggered; AI picks overwatch/dodge/brace with weighted randomness (`game/reactions.go`). Parry/Deflect need melee and special attacks first
- [x] Melee attacks (2 AP, 85% for 4) with the Parry reaction as counter
- [ ] Shields & armor (localized, toggleable)
- [x] Stances (`game/posture.go`): Crouch (1 AP, lower hitbox and eyes, 25% less spread, moving stands up; AI ducks behind low cover) and Prone (1 AP, lies across its own and the forward block, eyes at 0.75, 50% less spread, cannot move until it stands up for 1 AP; AI marksmen lie down for shots at 12+ blocks)
- [~] Hidden information: the server only sends enemies a player can see; overwatch is secret, heals are only shown when seen, LOS rows omit hidden enemy IDs. Shots and throws still reveal the shooter's position (intended). Designated leader missing
- [ ] Feign death / retreat

## 5. Deception: smoke & mirrors
- [~] Smoke grenades (needs server-side smoke that blocks LOS and interacts with bullets)
- [ ] Build / remove walls as a player action
- [ ] Fire & oil spreading

## 6. Combat depth & balance
- [~] Per-body-part damage: model part names now map to zones (headshots x2, weapon hits 0 HP, arm/leg wounds apply penalties); explosion damage still flat; no weapon destruction
- [~] Classes: Scout, Marksman (ignores pressure), Heavy (cheap grenades), Medic (Heal skill); more skills open
- [ ] Distinct stat blocks; animations for 3x3 walker/monster units
- [ ] Distinct faction identities
- [ ] Ammo variants, more items, loadout crates
- [ ] Balance pass against the "On Balance" rules
- [ ] Reward aggression over slow play

## 7. Missions & maps
- [~] Scenarios: Deathmatch, Defend
- [ ] More objectives: VIP, capture, carry item, survive, king of the hill
- [ ] Mid-match objective changes, asymmetric win conditions
- [x] Playable maps: seeded battlefield generator with 3 styles (urban, ruins, outpost): 1-2 storey buildings with windows, doors, stairs and roofs, low/high walls, crates, TNT, spawn zones, Defend objectives (`game/mapgen.go`, tested for connectivity). The Minecraft `.construction` files are still unused (they need a block-name mapping to the game's tileset)
- [ ] Doors, chests, buttons, deco items
- [ ] Start in contact / speed up early movement

## 8. Camera, visibility & UX
- [x] Ceiling removal (Z / C cutaway) and team-colored silhouettes of units behind walls
- [ ] Unit & body-part outlines when targeting
- [x] Fast-forward: Space toggles 3x speed
- [ ] Undo / confirm before committing a move
- [x] In-game settings screen (resolution, fullscreen, mouse, cameras, volume) saved to settings.json; default 1280x720
- [~] Hotkeys fixed: action bar uses number keys 1-9 (shown on the bar), Enter ends turn; Reload/Fire no longer share R; keys not yet rebindable

## 9. Known bugs
- [x] Faction skins scrambled (Minecraft-layout skin on an atlas-UV model)
- [x] Walk cycle stutter (keyframe sampler skipped keys; slerp NaN on near-identical rotations)
- [x] Units vanished near explosions (NaN rotation from vertical hit force)
- [x] Server never applied grenade effects (no damage/destruction/smoke server-side → units dead on screen but alive on server)
- [x] Explosions hit a unit once per occupied block (5 dmg grenade did 10-15)
- [x] Explosion damage resolved independently per client → desyncs; now server-authoritative (`UnitDamages` in messages)
- [x] AI/headless clients didn't update map occupancy for their own moves
- [x] Data races in the server (one goroutine per client, no locking) and connection handler swap
- [x] Standalone mode raced the local TCP listener on startup (now in-memory)
- [x] Units could keep drifting with a leftover velocity / block the client's event queue forever
- [x] Multiline text in the bitmap font was offset by the label position
- [ ] Own unit doesn't visually die when hit during FPS view — could not reproduce (code paths reviewed, autoplay with action cam saw no death)
- [x] Block removal at a chunk edge doesn't re-mesh the neighbor chunk
- [ ] Choppy diagonal climb/drop animations
- [x] Hard-coded -1 Y offset for Defend objectives (now the documented `MapMetadata.ObjectivePositions` convention)
- [ ] Unit firing not animated for remote shots (`battleclient.go` TODO); missing invisibility delay

## 10. Polish
- [ ] Better shots, explosions, blood decals on units
- [x] Lighting: sunlight + lamp propagation, face shading, lit units, updates on destruction; no bloom
- [~] Action cam & bullet cam (exist, disabled by default)
- [x] Data-driven content: maps, factions, units, weapons, items in `assets/content/*.json`, validated at startup
- [x] Map editor (main menu): tools, palette, undo, save/register, test-play; no online sharing

## 11. Meta & progression
- [ ] Campaign layer: base, roster, parallel-universe structure, unlocks, levelling
- [ ] Permadeath decision
