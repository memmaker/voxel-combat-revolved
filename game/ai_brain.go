package game

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

// AICommander is how an AIBrain issues its decisions. Each call must result in exactly one
// follow-up call to AIBrain.Step once the server answered (with the action's result or an error).
type AICommander interface {
	Move(unitID uint64, target voxel.Int3)
	Shoot(unitID uint64, target voxel.Int3)
	Throw(unitID uint64, itemName string, target mgl32.Vec3)
	Overwatch(unitID uint64, targets []voxel.Int3)
	Reload(unitID uint64)
	React(unitID uint64, reaction Reaction)
	Melee(unitID uint64, target voxel.Int3)
	Heal(unitID uint64, target voxel.Int3)
	EndTurn()
}

type AIDifficulty int

const (
	AIEasy AIDifficulty = iota
	AINormal
	AIHard
)

func (d AIDifficulty) String() string {
	switch d {
	case AIEasy:
		return "Easy"
	case AIHard:
		return "Hard"
	}
	return "Normal"
}

// AIBrain decides the moves for one player, one action at a time.
// It only uses information the player would have: its own units, visible enemies and what it remembers.
type AIBrain struct {
	g                *GameInstance
	playerID         uint64
	spawnIndex       int
	difficulty       AIDifficulty
	rng              *rand.Rand
	unitDone         map[uint64]bool
	actionsThisTurn  map[uint64]int
	pendingUnit      uint64
	pendingMove      *voxel.Int3
	rejectedTargets  map[uint64]map[voxel.Int3]bool
	lastKnownEnemies map[uint64]voxel.Int3
	searchTargets    []voxel.Int3
	log              func(string)
	// quietTurns counts our turns without seeing an enemy; impatience grows with it, so the AI
	// never settles into an endless overwatch standoff.
	quietTurns int
	sawEnemy   bool
}

const maxActionsPerUnitAndTurn = 5

// spawnIndex is the player's position in the turn order; it selects the own spawn zone and the defender role.
func NewAIBrain(g *GameInstance, playerID uint64, spawnIndex int, difficulty AIDifficulty) *AIBrain {
	b := &AIBrain{
		g:                g,
		playerID:         playerID,
		spawnIndex:       spawnIndex,
		difficulty:       difficulty,
		rng:              rand.New(rand.NewSource(rand.Int63())),
		unitDone:         make(map[uint64]bool),
		actionsThisTurn:  make(map[uint64]int),
		rejectedTargets:  make(map[uint64]map[voxel.Int3]bool),
		lastKnownEnemies: make(map[uint64]voxel.Int3),
		log:              func(msg string) { util.LogGameInfo(msg) },
	}
	if os.Getenv("BT_AI_LOG") != "" {
		b.log = func(msg string) { println(msg) }
	}
	return b
}

func (b *AIBrain) SetLogger(log func(string)) {
	b.log = log
}

// StartTurn resets the per-turn bookkeeping. Call Step afterwards.
func (b *AIBrain) StartTurn() {
	b.unitDone = make(map[uint64]bool)
	b.actionsThisTurn = make(map[uint64]int)
	b.rejectedTargets = make(map[uint64]map[voxel.Int3]bool)
	b.sawEnemy = false
	b.updateMemory()
	if b.sawEnemy {
		b.quietTurns = 0
	} else {
		b.quietTurns++
	}
}

// OnActionFailed handles a rejected action. A rejected move usually means the destination is blocked by
// something we cannot see, so that target is excluded and the unit may try something else; any other
// rejection finishes the unit for this turn, so nothing is retried in a loop.
func (b *AIBrain) OnActionFailed() {
	if b.pendingMove != nil {
		if b.rejectedTargets[b.pendingUnit] == nil {
			b.rejectedTargets[b.pendingUnit] = make(map[voxel.Int3]bool)
		}
		b.rejectedTargets[b.pendingUnit][*b.pendingMove] = true
		b.pendingMove = nil
		return
	}
	b.unitDone[b.pendingUnit] = true
}

// Step issues the next action, or ends the turn when no unit has anything useful left to do.
func (b *AIBrain) Step(cmd AICommander) {
	b.updateMemory()
	for _, unit := range b.myActiveUnits() {
		if b.unitDone[unit.UnitID()] {
			continue
		}
		if !unit.CanAct() || b.actionsThisTurn[unit.UnitID()] >= maxActionsPerUnitAndTurn {
			b.unitDone[unit.UnitID()] = true
			continue
		}
		b.log(fmt.Sprintf("[AI] considering %s AP=%.1f", unit.GetName(), unit.GetExactAP()))
		b.pendingUnit = unit.UnitID()
		b.pendingMove = nil
		b.actionsThisTurn[unit.UnitID()]++
		if b.decide(unit, cmd) {
			return
		}
		b.unitDone[unit.UnitID()] = true
	}
	b.log(fmt.Sprintf("[AI] Player %d ends turn (%d units)", b.playerID, len(b.g.GetPlayerUnits(b.playerID))))
	cmd.EndTurn()
}

func (b *AIBrain) myActiveUnits() []*UnitInstance {
	var result []*UnitInstance
	for _, unit := range b.g.GetPlayerUnits(b.playerID) {
		if unit.IsActive() {
			result = append(result, unit)
		}
	}
	// act with the units closest to the enemy first, so they open up lines of sight for the others
	sort.SliceStable(result, func(i, j int) bool {
		return b.distanceToNearestThreat(result[i].GetBlockPosition()) < b.distanceToNearestThreat(result[j].GetBlockPosition())
	})
	return result
}

func (b *AIBrain) updateMemory() {
	visibleNow := make(map[uint64]bool)
	var myPositions []voxel.Int3
	for _, unit := range b.g.GetPlayerUnits(b.playerID) {
		if !unit.IsActive() {
			continue
		}
		myPositions = append(myPositions, unit.GetBlockPosition())
		for _, enemy := range b.g.GetVisibleEnemyUnits(unit.UnitID()) {
			b.lastKnownEnemies[enemy.UnitID()] = enemy.GetBlockPosition()
			visibleNow[enemy.UnitID()] = true
			b.sawEnemy = true
		}
	}
	for enemyID, lastPos := range b.lastKnownEnemies {
		enemy, known := b.g.GetUnit(enemyID)
		if known && !enemy.IsActive() {
			delete(b.lastKnownEnemies, enemyID)
			continue
		}
		// we went to look and nobody is there anymore: the memory is stale
		if !visibleNow[enemyID] && anyWithin(myPositions, lastPos, 4) {
			delete(b.lastKnownEnemies, enemyID)
		}
	}
	if b.searchTargets == nil {
		b.searchTargets = b.enemySpawnGuesses()
	}
	// search targets we have reached are explored; keep looking elsewhere
	remaining := b.searchTargets[:0]
	for _, target := range b.searchTargets {
		if !anyWithin(myPositions, target, 5) {
			remaining = append(remaining, target)
		}
	}
	b.searchTargets = remaining
	if len(b.searchTargets) == 0 && len(myPositions) > 0 {
		b.searchTargets = b.randomExplorationTargets(3)
	}
}

func anyWithin(positions []voxel.Int3, target voxel.Int3, distance float64) bool {
	for _, pos := range positions {
		if blockDistance(pos, target) <= distance {
			return true
		}
	}
	return false
}

// randomExplorationTargets picks ground positions spread over the map.
func (b *AIBrain) randomExplorationTargets(count int) []voxel.Int3 {
	voxelMap := b.g.GetVoxelMap()
	chunks := voxelMap.GetSize()
	sizeX := chunks.X * voxelMap.ChunkSizeHorizontal
	sizeY := chunks.Y * voxelMap.ChunkSizeHeight
	sizeZ := chunks.Z * voxelMap.ChunkSizeHorizontal
	var targets []voxel.Int3
	for attempts := 0; attempts < 50 && len(targets) < count; attempts++ {
		start := voxel.Int3{X: b.rng.Int31n(sizeX), Y: sizeY - 1, Z: b.rng.Int31n(sizeZ)}
		ground := voxelMap.GetGroundPosition(start)
		if ground.Y < sizeY-1 && !voxelMap.IsSolidBlockAt(ground.X, ground.Y, ground.Z) {
			targets = append(targets, ground)
		}
	}
	return targets
}

// enemySpawnGuesses returns where the enemy most likely started, used to search when nothing is known.
func (b *AIBrain) enemySpawnGuesses() []voxel.Int3 {
	var guesses []voxel.Int3
	if b.g.mapMeta == nil {
		return guesses
	}
	for index, spawns := range b.g.mapMeta.SpawnPositions {
		if index == b.spawnIndex || len(spawns) == 0 {
			continue
		}
		guesses = append(guesses, centroid(spawns))
	}
	return guesses
}

func centroid(positions []voxel.Int3) voxel.Int3 {
	var x, y, z int32
	for _, p := range positions {
		x += p.X
		y += p.Y
		z += p.Z
	}
	n := int32(len(positions))
	return voxel.Int3{X: x / n, Y: y / n, Z: z / n}
}

// threats are the enemy units we can see or remember, used for cover and approach decisions.
type threat struct {
	unit     *UnitInstance
	position voxel.Int3
	visible  bool
}

func (b *AIBrain) knownThreats() []threat {
	var result []threat
	visible := b.g.GetAllVisibleEnemies(b.playerID)
	for enemyID, pos := range b.lastKnownEnemies {
		enemy, known := b.g.GetUnit(enemyID)
		if !known || !enemy.IsActive() {
			continue
		}
		result = append(result, threat{unit: enemy, position: pos, visible: visible[enemy]})
	}
	return result
}

func (b *AIBrain) distanceToNearestThreat(pos voxel.Int3) float64 {
	best := math.MaxFloat64
	for _, t := range b.knownThreats() {
		best = math.Min(best, blockDistance(pos, t.position))
	}
	if best == math.MaxFloat64 {
		for _, target := range b.objectivesToApproach() {
			best = math.Min(best, blockDistance(pos, target))
		}
	}
	return best
}

func blockDistance(a, b voxel.Int3) float64 {
	return a.Sub(b).Length()
}

// objectivesToApproach returns the places worth moving to when no enemy is known.
func (b *AIBrain) objectivesToApproach() []voxel.Int3 {
	details := b.g.missionDetails
	if details != nil && details.Scenario == MissionScenarioDefend {
		var open []voxel.Int3
		for _, objective := range details.DestroyableObjectives {
			if !details.IsObjectiveDestroyed(objective) {
				open = append(open, objective)
			}
		}
		if len(open) > 0 {
			return open
		}
	}
	return b.searchTargets
}

func (b *AIBrain) isDefender() bool {
	details := b.g.missionDetails
	return details != nil && details.Scenario == MissionScenarioDefend && b.spawnIndex == 0
}

// ---- decisions ----

func (b *AIBrain) decide(unit *UnitInstance, cmd AICommander) bool {
	weapon := unit.GetWeapon()
	visibleEnemies := b.g.GetVisibleEnemyUnits(unit.UnitID())

	// 1. an empty gun is useless: reload if possible
	if !weapon.IsReady() {
		if unit.CanReload() {
			b.log(fmt.Sprintf("[AI] %s reloads", unit.GetName()))
			cmd.Reload(unit.UnitID())
			return true
		}
		return b.tryMove(unit, cmd, false)
	}

	// 1b. medics patch up badly wounded allies within reach
	if unit.CanHeal() {
		for _, target := range NewActionHeal(b.g, unit).GetValidTargets() {
			if ally, ok := b.g.GetVoxelMap().GetMapObjectAt(target).(*UnitInstance); ok && ally.Health*2 <= ally.Definition.CoreStats.Health {
				b.log(fmt.Sprintf("[AI] %s heals %s", unit.GetName(), ally.GetName()))
				cmd.Heal(unit.UnitID(), target)
				return true
			}
		}
	}

	shotCost := int(weapon.Definition.BaseAPForShot)
	canShootNow := unit.CanSnapshot()
	bestShot, bestShotScore := b.bestShotFrom(unit, unit.GetBlockPosition(), visibleEnemies)

	// 2. grenades when they hit more than a rifle would
	if grenadeTarget, grenadeItem, grenadeScore := b.bestGrenade(unit, visibleEnemies); grenadeItem != "" && grenadeScore > bestShotScore*1.2 && unit.GetIntegerAP() >= unit.Definition.CoreStats.BaseAPForThrow {
		b.log(fmt.Sprintf("[AI] %s throws %s (score %.2f vs shot %.2f)", unit.GetName(), grenadeItem, grenadeScore, bestShotScore))
		cmd.Throw(unit.UnitID(), grenadeItem, grenadeTarget)
		return true
	}

	// 3. reposition first, if the unit can still shoot afterwards and the new spot is clearly better
	apAfterShot := unit.GetIntegerAP() - shotCost
	if canShootNow && len(visibleEnemies) > 0 && apAfterShot >= 1 && b.difficulty != AIEasy {
		currentValue := b.positionValue(unit, unit.GetBlockPosition(), true)
		if target, value, found := b.bestPosition(unit, float64(apAfterShot), true); found && value > currentValue+1.0 {
			b.log(fmt.Sprintf("[AI] %s repositions before shooting to %s (%.2f > %.2f)", unit.GetName(), target.ToString(), value, currentValue))
			b.move(unit, target, cmd)
			return true
		}
	}

	// 3b. an enemy right next to us: strike if that beats shooting
	if unit.CanMelee() && b.difficulty != AIEasy {
		for _, enemy := range visibleEnemies {
			if !IsInMeleeRange(unit.GetBlockPosition(), enemy.GetBlockPosition()) {
				continue
			}
			meleeValue := MeleeHitChance * MeleeDamage * b.noise()
			if enemy.Health <= MeleeDamage {
				meleeValue *= 1.5
			}
			if meleeValue > bestShotScore || !canShootNow {
				b.log(fmt.Sprintf("[AI] %s strikes %s in melee", unit.GetName(), enemy.GetName()))
				cmd.Melee(unit.UnitID(), enemy.GetBlockPosition())
				return true
			}
		}
	}

	// 4. take the shot
	if canShootNow && bestShot != nil && bestShotScore >= b.minimumShotScore() {
		b.log(fmt.Sprintf("[AI] %s shoots at %s (score %.2f)", unit.GetName(), bestShot.GetName(), bestShotScore))
		cmd.Shoot(unit.UnitID(), bestShot.GetBlockPosition())
		return true
	}
	if canShootNow && b.shootObjective(unit, cmd) {
		return true
	}

	// 5. move into a better position (keeping enough AP to shoot when possible)
	if b.tryMove(unit, cmd, true) {
		return true
	}

	// 6. nothing to shoot and nowhere better to go: watch the approaches or take a defensive stance
	if b.difficulty != AIEasy && b.chooseDefense(unit, visibleEnemies, cmd) {
		return true
	}

	// 7. low on ammo and nothing else to do
	if !weapon.IsMagazineFull() && unit.CanReload() {
		cmd.Reload(unit.UnitID())
		return true
	}
	return false
}

func (b *AIBrain) minimumShotScore() float64 {
	switch b.difficulty {
	case AIEasy:
		return 0.05
	case AIHard:
		return 0.6
	}
	return 0.4
}

// hitChance estimates the chance of one projectile hitting a unit at the given distance.
// Shots deviate in a cone; the deviation radius grows with distance and with (1 - accuracy).
func hitChance(accuracy float64, distance float64) float64 {
	const tanHalfFOV = 0.414 // the server simulates shots with a 45 degree camera
	const targetRadius = 0.45
	spread := distance * (1 - accuracy) * tanHalfFOV
	if spread <= targetRadius {
		return 0.95
	}
	return util.Clamp(targetRadius/spread, 0.02, 0.95)
}

func (b *AIBrain) estimatedShotAccuracy(unit *UnitInstance) float64 {
	pressure := 1.0
	if unit.GetWeapon().Definition.WeaponType == WeaponSniper && !unit.Definition.CoreStats.IgnoresPressure {
		pressure = math.Max(0.5, 1.0-b.g.GetTotalPressure(unit.UnitID())*0.25)
	}
	return unit.GetFreeAimAccuracy() * pressure
}

// shotValue is the expected damage of shooting target from position (with kill and focus fire bonuses).
func (b *AIBrain) shotValue(unit *UnitInstance, from voxel.Int3, target *UnitInstance) float64 {
	weapon := unit.GetWeapon()
	distance := blockDistance(from, target.GetBlockPosition())
	if distance > float64(weapon.Definition.MaxRange) {
		return 0
	}
	if weapon.Definition.InsteadOfDamage != TargetedEffectNone {
		return b.explosiveShotValue(unit, target)
	}
	visibility := b.visibleFraction(unit, from, target)
	if visibility <= 0 {
		return 0
	}
	perBullet := float64(weapon.AdjustDamageForDistance(float32(distance), weapon.Definition.BaseDamagePerBullet))
	expectedHits := hitChance(b.estimatedShotAccuracy(unit), distance) * float64(weapon.Definition.BulletsPerShot) * visibility
	expectedDamage := expectedHits * perBullet
	value := expectedDamage
	if expectedDamage >= float64(target.Health)*0.8 {
		value *= 1.5 // likely kill
	}
	if target.Health < target.Definition.CoreStats.Health {
		value *= 1.2 // focus fire on wounded units
	}
	return value
}

func (b *AIBrain) explosiveShotValue(unit *UnitInstance, target *UnitInstance) float64 {
	radius := unit.GetWeapon().Definition.Radius
	value := 0.0
	for _, other := range b.g.GetAllUnits() {
		if !other.IsActive() || blockDistance(other.GetBlockPosition(), target.GetBlockPosition()) > radius+1 {
			continue
		}
		if other.ControlledBy() == b.playerID {
			value -= float64(b.g.rules.ExplosionDamage) * 2
		} else {
			value += float64(b.g.rules.ExplosionDamage)
		}
	}
	if blockDistance(unit.GetBlockPosition(), target.GetBlockPosition()) <= radius+1 {
		value -= 100
	}
	return value
}

// visibleFraction checks several points of the target from the shooter's eye at position from (0..1).
func (b *AIBrain) visibleFraction(unit *UnitInstance, from voxel.Int3, target *UnitInstance) float64 {
	eye := from.ToBlockCenterVec3().Add(unit.GetEyeOffset())
	targetFoot := target.GetPosition()
	points := []mgl32.Vec3{
		targetFoot.Add(mgl32.Vec3{0, 0.4, 0}),
		targetFoot.Add(mgl32.Vec3{0, 1.1, 0}),
		targetFoot.Add(target.GetEyeOffset()),
	}
	visible := 0
	for _, point := range points {
		hit := b.g.RayCastLineOfSight(eye, point, target, target.GetBlockPosition())
		if hit.UnitHit == target {
			visible++
		}
	}
	return float64(visible) / float64(len(points))
}

func (b *AIBrain) bestShotFrom(unit *UnitInstance, from voxel.Int3, candidates []*UnitInstance) (*UnitInstance, float64) {
	var best *UnitInstance
	bestScore := 0.0
	for _, enemy := range candidates {
		if !enemy.IsActive() {
			continue
		}
		score := b.shotValue(unit, from, enemy) * b.noise()
		if score > bestScore {
			best, bestScore = enemy, score
		}
	}
	return best, bestScore
}

func (b *AIBrain) noise() float64 {
	switch b.difficulty {
	case AIEasy:
		return 0.5 + b.rng.Float64()
	case AIHard:
		return 0.95 + b.rng.Float64()*0.1
	}
	return 0.85 + b.rng.Float64()*0.3
}

func (b *AIBrain) bestGrenade(unit *UnitInstance, visibleEnemies []*UnitInstance) (mgl32.Vec3, string, float64) {
	if b.difficulty == AIEasy {
		return mgl32.Vec3{}, "", 0
	}
	var bestItem string
	var bestTarget mgl32.Vec3
	bestScore := 0.0
	for _, item := range unit.GetItems() {
		if item.Definition.ItemType != ItemTypeGrenade || item.Definition.Effect != TargetedEffectExplosion {
			continue
		}
		maxThrowDistance := unit.Definition.CoreStats.ThrowVelocity * unit.Definition.CoreStats.ThrowVelocity / 9.8 * 0.8
		for _, enemy := range visibleEnemies {
			distance := blockDistance(unit.GetBlockPosition(), enemy.GetBlockPosition())
			if distance > maxThrowDistance || distance <= item.Definition.Radius+1.5 {
				continue
			}
			score := 0.0
			for _, other := range b.g.GetAllUnits() {
				if !other.IsActive() || blockDistance(other.GetBlockPosition(), enemy.GetBlockPosition()) > item.Definition.Radius {
					continue
				}
				if other.ControlledBy() == b.playerID {
					score -= float64(b.g.rules.ExplosionDamage) * 2
				} else {
					score += float64(b.g.rules.ExplosionDamage)
					if other.Health <= b.g.rules.ExplosionDamage {
						score += 3
					}
				}
			}
			// grenades are worth more against units in cover, where bullets would be wasted
			score += (1 - b.visibleFraction(unit, unit.GetBlockPosition(), enemy)) * 2
			if score > bestScore {
				bestScore, bestItem, bestTarget = score, item.Definition.UniqueName, enemy.GetPosition().Add(mgl32.Vec3{0, 0.1, 0})
			}
		}
	}
	return bestTarget, bestItem, bestScore
}

// coverAt counts solid blocks directly between pos and the threat's direction at leg and torso height.
func (b *AIBrain) coverAt(pos voxel.Int3, threatPos voxel.Int3) float64 {
	dir := threatPos.Sub(pos)
	var steps []voxel.Int3
	if dir.X != 0 {
		steps = append(steps, voxel.Int3{X: sign(dir.X)})
	}
	if dir.Z != 0 {
		steps = append(steps, voxel.Int3{Z: sign(dir.Z)})
	}
	voxelMap := b.g.GetVoxelMap()
	cover := 0.0
	for _, step := range steps {
		weight := math.Abs(float64(dir.X))
		if step.Z != 0 {
			weight = math.Abs(float64(dir.Z))
		}
		weight /= math.Abs(float64(dir.X)) + math.Abs(float64(dir.Z)) + 0.001
		for height := int32(0); height <= 1; height++ {
			p := pos.Add(step).Add(voxel.Int3{Y: height})
			if voxelMap.IsSolidBlockAt(p.X, p.Y, p.Z) {
				cover += 0.5 * weight
			}
		}
	}
	return cover
}

func sign(v int32) int32 {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

// positionValue scores standing at pos at the end of a move.
func (b *AIBrain) positionValue(unit *UnitInstance, pos voxel.Int3, wantsToShoot bool) float64 {
	threats := b.knownThreats()
	weapon := unit.GetWeapon()
	value := 0.0

	if len(threats) == 0 {
		// nothing known: advance towards objectives / the likely enemy location, keep some cover
		best := math.MaxFloat64
		for _, target := range b.objectivesToApproach() {
			best = math.Min(best, blockDistance(pos, target))
		}
		if best != math.MaxFloat64 {
			value -= best * 0.3
		}
		return value + b.allySpacing(unit, pos)
	}

	exposure := 0.0
	var shootableNow, shootableRemembered []*UnitInstance
	for _, t := range threats {
		cover := b.coverAt(pos, t.position)
		value += cover * 0.8
		eye := pos.ToBlockCenterVec3().Add(unit.GetEyeOffset())
		if !b.g.CanSeeFrom(unit, t.unit, eye) {
			continue
		}
		if t.visible {
			shootableNow = append(shootableNow, t.unit)
			exposure += 1.0 - math.Min(cover, 0.8)
		} else {
			shootableRemembered = append(shootableRemembered, t.unit)
		}
	}

	if wantsToShoot {
		_, shotScore := b.bestShotFrom(unit, pos, shootableNow)
		_, rememberedScore := b.bestShotFrom(unit, pos, shootableRemembered)
		value += math.Max(shotScore*1.5, rememberedScore*0.8)
	}
	if len(shootableNow)+len(shootableRemembered) == 0 {
		value -= 1.5 // a position without any line of sight wins nothing: keep pushing
	}
	value -= exposure * 0.6

	// keep to the weapon's effective range
	nearest := math.MaxFloat64
	for _, t := range threats {
		nearest = math.Min(nearest, blockDistance(pos, t.position))
	}
	// the longer nothing happens, the more we want to close in
	value -= nearest * 0.05 * float64(b.quietTurns)
	effectiveRange := float64(weapon.Definition.EffectiveRange)
	if nearest > effectiveRange {
		value -= (nearest - effectiveRange) * 0.25
	} else if nearest < 3 && weapon.Definition.WeaponType != WeaponShotgun {
		value -= (3 - nearest) * 0.5
	}

	// defenders stay close to what they defend
	if b.isDefender() {
		for _, objective := range b.objectivesToApproach() {
			if d := blockDistance(pos, objective); d > 8 {
				value -= (d - 8) * 0.3
			}
		}
	}
	return value + b.allySpacing(unit, pos)
}

// allySpacing discourages clumping together, which makes grenades too effective.
func (b *AIBrain) allySpacing(unit *UnitInstance, pos voxel.Int3) float64 {
	penalty := 0.0
	for _, ally := range b.g.GetPlayerUnits(b.playerID) {
		if ally == unit || !ally.IsActive() {
			continue
		}
		if d := blockDistance(pos, ally.GetBlockPosition()); d < 3 {
			penalty -= (3 - d) * 0.4
		}
	}
	return penalty
}

// bestPosition finds the best reachable destination spending at most apBudget on movement.
func (b *AIBrain) bestPosition(unit *UnitInstance, apBudget float64, wantsToShoot bool) (voxel.Int3, float64, bool) {
	moveAction := NewActionMove(b.g.GetVoxelMap(), unit)
	maxMoves := apBudget / unit.APPerMovement()
	bestValue := math.Inf(-1)
	var best voxel.Int3
	found := false
	for _, target := range moveAction.GetValidTargets() {
		cost := moveAction.GetCost(target)
		if cost > maxMoves || cost <= 0 || b.rejectedTargets[unit.UnitID()][target] {
			continue
		}
		value := b.positionValue(unit, target, wantsToShoot)*b.noise() - cost*0.02
		if value > bestValue {
			bestValue, best, found = value, target, true
		}
	}
	return best, bestValue, found
}

func (b *AIBrain) tryMove(unit *UnitInstance, cmd AICommander, keepAPForShot bool) bool {
	if !unit.CanMove() {
		return false
	}
	budget := unit.GetExactAP()
	shotCost := float64(unit.GetWeapon().Definition.BaseAPForShot)
	threatsKnown := len(b.knownThreats()) > 0
	if keepAPForShot && threatsKnown && budget-shotCost >= unit.APPerMovement() {
		budget -= shotCost
	}
	currentValue := b.positionValue(unit, unit.GetBlockPosition(), keepAPForShot)
	target, value, found := b.bestPosition(unit, budget, keepAPForShot)
	if !found || value <= currentValue+0.3 {
		return false
	}
	b.log(fmt.Sprintf("[AI] %s moves to %s (%.2f > %.2f)", unit.GetName(), target.ToString(), value, currentValue))
	b.move(unit, target, cmd)
	return true
}

func (b *AIBrain) move(unit *UnitInstance, target voxel.Int3, cmd AICommander) {
	b.log(fmt.Sprintf("[AI] move request %s: from %s with %d moves left to %s", unit.GetName(), unit.GetBlockPosition().ToString(), unit.MovesLeft(), target.ToString()))
	b.pendingMove = &target
	cmd.Move(unit.UnitID(), target)
}

func (b *AIBrain) shootObjective(unit *UnitInstance, cmd AICommander) bool {
	details := b.g.missionDetails
	if details == nil || details.Scenario != MissionScenarioDefend || b.isDefender() {
		return false
	}
	for _, objective := range b.objectivesToApproach() {
		distance := blockDistance(unit.GetBlockPosition(), objective)
		if distance > float64(unit.GetWeapon().Definition.EffectiveRange) {
			continue
		}
		if b.g.CanSeePos(unit, objective) {
			b.log(fmt.Sprintf("[AI] %s shoots at objective %s", unit.GetName(), objective.ToString()))
			cmd.Shoot(unit.UnitID(), objective)
			return true
		}
	}
	return false
}

// overwatchTargets picks up to a handful of visible blocks on the way towards known threats.
func (b *AIBrain) overwatchTargets(unit *UnitInstance) []voxel.Int3 {
	action := NewActionOverwatch(b.g, unit)
	candidates := action.GetValidTargets()
	threats := b.knownThreats()
	type scored struct {
		pos   voxel.Int3
		score float64
	}
	var options []scored
	for _, pos := range candidates {
		ownDistance := blockDistance(unit.GetBlockPosition(), pos)
		if ownDistance < 3 || ownDistance > float64(unit.GetWeapon().Definition.EffectiveRange)*1.5 {
			continue
		}
		nearestThreat := math.MaxFloat64
		for _, t := range threats {
			nearestThreat = math.Min(nearestThreat, blockDistance(pos, t.position))
		}
		// blocks between us and the enemy, close to the enemy, are the ones they will walk over
		options = append(options, scored{pos: pos, score: -nearestThreat - ownDistance*0.2})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].score > options[j].score })
	var result []voxel.Int3
	for i := 0; i < len(options) && i < 6; i++ {
		result = append(result, options[i].pos)
	}
	return result
}

// chooseDefense picks between overwatch, dodge and brace with weighted randomness, so the
// opponent cannot reliably predict which counter to use (the core of the yomi design).
func (b *AIBrain) chooseDefense(unit *UnitInstance, visibleEnemies []*UnitInstance, cmd AICommander) bool {
	type option struct {
		weight float64
		apply  func()
		name   string
	}
	var options []option
	if unit.CanFreeAim() && len(b.knownThreats()) > 0 && b.quietTurns < 2 {
		if targets := b.overwatchTargets(unit); len(targets) > 0 {
			weight := 0.5
			nearest := b.distanceToNearestThreat(unit.GetBlockPosition())
			if nearest > 8 { // enemies must move to reach us: overwatch is strongest
				weight = 1.2
			}
			options = append(options, option{weight, func() { cmd.Overwatch(unit.UnitID(), targets) }, fmt.Sprintf("overwatch (%d blocks)", len(targets))})
		}
	}
	if unit.CanReact() && len(visibleEnemies) > 0 {
		grenadeThreat, shotgunThreat := 0.0, 0.0
		for _, enemy := range visibleEnemies {
			distance := blockDistance(unit.GetBlockPosition(), enemy.GetBlockPosition())
			for _, item := range enemy.GetItems() {
				if item.Definition.Effect == TargetedEffectExplosion && distance < 14 {
					grenadeThreat += 0.5
				}
			}
			if enemy.GetWeapon() != nil && enemy.GetWeapon().Definition.WeaponType == WeaponShotgun && distance < 8 {
				shotgunThreat += 0.5
			}
		}
		for _, ally := range b.g.GetPlayerUnits(b.playerID) {
			if ally != unit && ally.IsActive() && blockDistance(ally.GetBlockPosition(), unit.GetBlockPosition()) < 4 {
				grenadeThreat += 0.3 // grouped up: one grenade hits several of us
			}
		}
		meleeThreat := 0.0
		for _, enemy := range visibleEnemies {
			if blockDistance(unit.GetBlockPosition(), enemy.GetBlockPosition()) <= 4 {
				meleeThreat += 0.6 // close enough to run up and strike
			}
		}
		options = append(options,
			option{1.0, func() { cmd.React(unit.UnitID(), ReactionDodge) }, "dodge"},
			option{0.4 + grenadeThreat + shotgunThreat, func() { cmd.React(unit.UnitID(), ReactionBrace) }, "brace"},
		)
		if meleeThreat > 0 {
			options = append(options, option{meleeThreat, func() { cmd.React(unit.UnitID(), ReactionParry) }, "parry"})
		}
	}
	if len(options) == 0 {
		return false
	}
	total := 0.0
	for _, o := range options {
		total += o.weight
	}
	roll := b.rng.Float64() * total
	for _, o := range options {
		roll -= o.weight
		if roll <= 0 {
			b.log(fmt.Sprintf("[AI] %s takes a defensive stance: %s", unit.GetName(), o.name))
			o.apply()
			return true
		}
	}
	return false
}
