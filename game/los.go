package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
	"sort"
)

func (g *GameInstance) SetLOSMatrix(matrix map[uint64]map[uint64]bool) {
	g.losMatrix = matrix
}

func (g *GameInstance) InitLOS() {
	for _, unit := range g.units {
		g.losMatrix[unit.UnitID()] = make(map[uint64]bool)
		for _, other := range g.units {
			if unit.ControlledBy() != other.ControlledBy() {
				g.losMatrix[unit.UnitID()][other.UnitID()] = g.CanSee(unit, other)
			}
		}
	}
}
func (g *GameInstance) GetVisibleUnits(unitID uint64) []*UnitInstance {
	result := make([]*UnitInstance, 0)
	for enemyID, isVisble := range g.losMatrix[unitID] {
		if isVisble {
			result = append(result, g.units[enemyID])
		}
	}
	return result
}
func (g *GameInstance) GetVisibleEnemyUnits(unitID uint64) []*UnitInstance {
	result := make([]*UnitInstance, 0)
	ownInstance := g.units[unitID]
	if !ownInstance.IsActive() {
		return result
	}
	for enemyID, isVisble := range g.losMatrix[unitID] {
		enemy := g.units[enemyID]
		if isVisble && enemy.ControlledBy() != ownInstance.ControlledBy() && enemy.IsActive() {
			result = append(result, enemy)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return ownInstance.GetEyePosition().Sub(result[i].GetEyePosition()).Len() < ownInstance.GetEyePosition().Sub(result[j].GetEyePosition()).Len()
	})
	return result
}
func (g *GameInstance) UnitIsVisibleToPlayer(playerID, unitID uint64) bool {
	unit, isKnown := g.units[unitID]
	if !isKnown {
		return false
	}
	if unit.ControlledBy() == playerID {
		return true
	}
	for _, playerUnit := range g.GetPlayerUnits(playerID) {
		if !playerUnit.IsActive() {
			continue
		}
		if g.losMatrix[playerUnit.UnitID()][unitID] {
			return true
		}
	}
	return false
}

func (g *GameInstance) CanSee(one, another *UnitInstance) bool {
	return g.CanSeeFrom(one, another, one.GetEyePosition())
}

func (g *GameInstance) CanSeeFrom(observer, another *UnitInstance, observerEyePosition mgl32.Vec3) bool {
	return g.CanSeeFromTo(observer, another, observerEyePosition, another.GetPosition())
}

func (g *GameInstance) CanSeeTo(observer, another *UnitInstance, targetFootPosition mgl32.Vec3) bool {
	return g.CanSeeFromTo(observer, another, observer.GetEyePosition(), targetFootPosition)
}

func (g *GameInstance) CanSeeFromTo(observer, another *UnitInstance, observerEye mgl32.Vec3, targetFootPosition mgl32.Vec3) bool {
	if observer == another || observer.ControlledBy() == another.ControlledBy() {
		return true
	}
	if !observer.IsActive() || !another.IsActive() {
		return false
	}

	targetTwo := targetFootPosition
	targetOne := targetFootPosition.Add(another.GetEyeOffset())

	originPositions := []mgl32.Vec3{observerEye.Add(mgl32.Vec3{-0.5, 0, -0.5}), observerEye.Add(mgl32.Vec3{0.5, 0, -0.5}), observerEye.Add(mgl32.Vec3{-0.5, 0, 0.5}), observerEye.Add(mgl32.Vec3{0.5, 0, 0.5})}
	for _, observerEyePosition := range originPositions {
		rayOne := g.RayCastLineOfSight(observerEyePosition, targetOne, another, voxel.PositionToGridInt3(targetFootPosition))

		if rayOne.UnitHit == another {
			return true
		} // fast exit

		rayTwo := g.RayCastLineOfSight(observerEyePosition, targetTwo, another, voxel.PositionToGridInt3(targetFootPosition))

		if rayTwo.UnitHit == another {
			return true
		}
	}
	return false
}

func (g *GameInstance) CanSeePos(observer *UnitInstance, targetBlockPosition voxel.Int3) bool {
	if !observer.IsActive() {
		return false
	}

	rayOne := g.RayCastToPos(observer.GetEyePosition(), targetBlockPosition)
	var wasLastBlock bool
	if len(rayOne.VisitedBlocks) > 0 {
		lastBlock := rayOne.VisitedBlocks[len(rayOne.VisitedBlocks)-1]
		wasLastBlock = lastBlock == targetBlockPosition
	}
	return wasLastBlock || rayOne.CollisionGridPosition == targetBlockPosition || rayOne.PreviousGridPosition == targetBlockPosition
}

func (g *GameInstance) GetReverseLOSChangesForUser(userID uint64, mover *UnitInstance) ([]uint64, []uint64) {
	seenBy := make([]uint64, 0)
	hiddenTo := make([]uint64, 0)
	for _, observer := range g.GetPlayerUnits(userID) {
		wasVisible, wasKnown := g.losMatrix[observer.UnitID()][mover.UnitID()]
		wasVisible = wasKnown && wasVisible
		isVisible := g.CanSee(observer, mover)
		if isVisible && !wasVisible {
			seenBy = append(seenBy, observer.UnitID())
		} else if !isVisible && wasVisible {
			hiddenTo = append(hiddenTo, observer.UnitID())
		}
	}
	return seenBy, hiddenTo
}

func (g *GameInstance) GetLOSChanges(unit *UnitInstance, pos voxel.Int3) ([]*UnitInstance, []*UnitInstance, bool) {
	allVisibleEnemies := g.GetAllVisibleEnemies(unit.ControlledBy())
	visibleEnemiesForUnit := g.losMatrix[unit.UnitID()]
	newEnemies := make([]*UnitInstance, 0)
	lostEnemies := make([]*UnitInstance, 0)
	newContact := false
	for _, enemy := range g.units {
		if enemy.ControlledBy() == unit.ControlledBy() {
			continue
		}
		unitWasVisible, wasKnown := visibleEnemiesForUnit[enemy.UnitID()]
		wasVisible := wasKnown && unitWasVisible

		isVisible := g.CanSeeFrom(unit, enemy, pos.ToBlockCenterVec3().Add(unit.GetEyeOffset()))
		if isVisible && !wasVisible {
			newEnemies = append(newEnemies, enemy)
			if !allVisibleEnemies[enemy] {
				newContact = true
			}
			//g.losMatrix[unit][enemy] = true
		} else if !isVisible && wasVisible {
			lostEnemies = append(lostEnemies, enemy)
			//g.losMatrix[unit][enemy] = false
		}
	}
	return newEnemies, lostEnemies, newContact
}

func (g *GameInstance) RayCastLineOfSight(rayStart, rayEnd mgl32.Vec3, targetUnit voxel.MapObject, projectedTargetLocation voxel.Int3) *RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *UnitInstance
	occupiedByTarget := map[voxel.Int3]bool{}
	for _, offset := range targetUnit.GetOccupiedBlockOffsets() {
		occupiedByTarget[projectedTargetLocation.Add(offset)] = true
	}
	stopRay := func(x, y, z int32) bool {
		currentBlockPos := voxel.Int3{X: x, Y: y, Z: z}
		visitedBlocks = append(visitedBlocks, currentBlockPos)
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				//println(fmt.Sprintf("[GameInstance] Raycast hit block at %d, %d, %d", x, y, z))
				return true
			} else if occupiedByTarget[currentBlockPos] {
				unitHit = targetUnit.(*UnitInstance)
				//println(fmt.Sprintf("[GameInstance] Raycast hit unit %s at %d, %d, %d", unitHit.name, x, y, z))
				return true
			}
		} else {
			//println(fmt.Sprintf("[GameInstance] Raycast hit out of bounds at %d, %d, %d", x, y, z))
			return true
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	insideMap := voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)

	return &RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
}

func (g *GameInstance) RayCastToPos(rayStart mgl32.Vec3, targetBlockPos voxel.Int3) *RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *UnitInstance
	stopRay := func(x, y, z int32) bool {
		currentBlockPos := voxel.Int3{X: x, Y: y, Z: z}
		visitedBlocks = append(visitedBlocks, currentBlockPos)
		if currentBlockPos == targetBlockPos {
			return true
		}
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				//println(fmt.Sprintf("[GameInstance] Raycast hit block at %d, %d, %d", x, y, z))
				return true
			}
		} else {
			//println(fmt.Sprintf("[GameInstance] Raycast hit out of bounds at %d, %d, %d", x, y, z))
			return true
		}
		return false
	}
	rayEnd := targetBlockPos.ToBlockCenterVec3D()
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	insideMap := voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)

	return &RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
}

func (g *GameInstance) RayCastFreeAim(rayStart, rayEnd mgl32.Vec3, sourceUnit *UnitInstance) *FreeAimHit {
	rayHitObject := false
	var hitPart util.Collider
	var hitPoint mgl32.Vec3
	var hitUnit *UnitInstance
	var visitedBlocks []voxel.Int3
	checkedCollision := make(map[voxel.MapObject]bool)
	rayHitInfo := util.DDARaycast(rayStart, rayEnd, func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if g.voxelMap.IsSolidBlockAt(x, y, z) || !g.voxelMap.Contains(x, y, z) {
			return true
		}
		block := g.voxelMap.GetGlobalBlock(x, y, z)

		if block != nil && block.IsOccupied() {
			collidingObject := block.GetOccupant().(*UnitInstance)
			if collidingObject == sourceUnit {
				return false
			}
			var rayPoint mgl32.Vec3
			rayHit := false
			if _, checkedBefore := checkedCollision[collidingObject]; checkedBefore {
				return false
			}
			minDistance := float32(math.MaxFloat32)
			//println(fmt.Sprintf("Checking %s against %s", obj.GetName(), collidingObject.GetName()))
			for _, meshPartCollider := range collidingObject.GetColliders() {
				//meshsCollided, _ = util.GJK(projectile.GetCollider(), meshPartCollider) // we made this sweeping for the projectiles only for now
				rayHit, rayPoint = meshPartCollider.IntersectsRay(rayStart, rayEnd)
				if rayHit {
					rayHitObject = true
					dist := rayPoint.Sub(rayStart).Len()
					if dist < minDistance {
						minDistance = dist
						hitPart = meshPartCollider
						hitPoint = rayPoint
						hitUnit = collidingObject
					}
				}
				checkedCollision[collidingObject] = true
			}
			if rayHitObject {
				return true
			}
		}
		return false
	})
	if rayHitObject {
		rayHitInfo = rayHitInfo.WithCollisionWorldPosition(hitPoint)
		rayHitInfo = rayHitInfo.WithDistance(float64(rayHitInfo.CollisionWorldPosition.Sub(rayStart).Len()))
	}
	insideMap := g.voxelMap.ContainsGrid(rayHitInfo.CollisionGridPosition) || g.voxelMap.ContainsGrid(rayHitInfo.PreviousGridPosition)
	partName := util.ZoneNone
	if hitPart != nil {
		colliderName := hitPart.GetName()
		if colliderName == hitUnit.GetWeapon().Definition.Model {
			partName = util.ZoneWeapon
		} else {
			partName = util.DamageZone(colliderName)
		}
	}
	return &FreeAimHit{RayCastHit: RayCastHit{HitInfo3D: rayHitInfo, VisitedBlocks: visitedBlocks, UnitHit: hitUnit, InsideMap: insideMap}, BodyPart: partName, Origin: rayStart}
}
