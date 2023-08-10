package server

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"math"
)

func (g *GameInstance) CanSee(one, another *game.UnitInstance) bool {
	return g.CanSeeFrom(one, another, one.GetEyePosition())
}

func (g *GameInstance) CanSeeFrom(observer, another *game.UnitInstance, observerEyePosition mgl32.Vec3) bool {
	return g.CanSeeFromTo(observer, another, observerEyePosition, another.GetFootPosition())
}

func (g *GameInstance) CanSeeTo(observer, another *game.UnitInstance, targetFootPosition mgl32.Vec3) bool {
	return g.CanSeeFromTo(observer, another, observer.GetEyePosition(), targetFootPosition)
}

func (g *GameInstance) CanSeeFromTo(observer, another *game.UnitInstance, observerEyePosition mgl32.Vec3, targetFootPosition mgl32.Vec3) bool {
	if observer == another || observer.ControlledBy() == another.ControlledBy() {
		return true
	}

	targetTwo := targetFootPosition
	targetOne := targetFootPosition.Add(another.GetEyeOffset())

	//print(fmt.Sprintf("[GameInstance] Doing expensive LOS check %s -> %s: ", observer.GetName(), another.GetName()))

	rayOne := g.RayCastUnitsProjected(observerEyePosition, targetOne, observer, another, voxel.ToGridInt3(targetFootPosition))
	if rayOne.UnitHit == another { // fast exit
		//println("Line of sight is CLEAR")
		return true
	}
	rayTwo := g.RayCastUnitsProjected(observerEyePosition, targetTwo, observer, another, voxel.ToGridInt3(targetFootPosition))

	hasLos := rayTwo.UnitHit == another
	if hasLos {
		//println("Line of sight is CLEAR")
	} else {
		//println("NO LINE OF SIGHT")
	}

	return hasLos
}

func (g *GameInstance) GetReverseLOSChangesForUser(userID uint64, mover *game.UnitInstance, position voxel.Int3, visibles, invisibles []*game.UnitInstance) ([]uint64, []uint64) {
	seenBy := make([]uint64, 0)
	hiddenTo := make([]uint64, 0)
	for _, observer := range append(visibles, invisibles...) {
		if observer.ControlledBy() != userID {
			continue
		}
		wasVisible, wasKnown := g.currentVisibleEnemies[observer.UnitID()][mover.UnitID()]
		wasVisible = wasKnown && wasVisible
		isVisible := g.CanSeeTo(observer, mover, position.ToBlockCenterVec3())
		if isVisible && !wasVisible {
			seenBy = append(seenBy, observer.UnitID())
		} else if !isVisible && wasVisible {
			hiddenTo = append(hiddenTo, observer.UnitID())
		}
	}
	return seenBy, hiddenTo
}

func (g *GameInstance) GetLOSChanges(unit *game.UnitInstance, pos voxel.Int3) ([]*game.UnitInstance, []*game.UnitInstance) {
	visibleEnemies := g.currentVisibleEnemies[unit.UnitID()]
	newEnemies := make([]*game.UnitInstance, 0)
	lostEnemies := make([]*game.UnitInstance, 0)
	own := g.currentPlayerFaction()
	for _, enemy := range g.units {
		if g.factionMap[enemy] == own {
			continue
		}
		unitWasVisible, wasKnown := visibleEnemies[enemy.UnitID()]
		wasVisible := wasKnown && unitWasVisible

		isVisible := g.CanSeeFrom(unit, enemy, pos.ToBlockCenterVec3().Add(unit.GetEyeOffset()))
		if isVisible && !wasVisible {
			newEnemies = append(newEnemies, enemy)
			//g.currentVisibleEnemies[unit][enemy] = true
		} else if !isVisible && wasVisible {
			lostEnemies = append(lostEnemies, enemy)
			//g.currentVisibleEnemies[unit][enemy] = false
		}
	}
	return newEnemies, lostEnemies
}

func (g *GameInstance) RayCastUnits(rayStart, rayEnd mgl32.Vec3, sourceUnit, targetUnit voxel.MapObject) *game.RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *game.UnitInstance
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				//println(fmt.Sprintf("[GameInstance] Raycast hit block at %d, %d, %d", x, y, z))
				return true
			} else if block.IsOccupied() && (block.GetOccupant().ControlledBy() != sourceUnit.ControlledBy() || block.GetOccupant() == targetUnit) {
				unitHit = block.GetOccupant().(*game.UnitInstance)
				//println(fmt.Sprintf("[GameInstance] Raycast hit unit %s at %d, %d, %d", unitHit.Name, x, y, z))
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

	return &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
}

func (g *GameInstance) RayCastUnitsProjected(rayStart, rayEnd mgl32.Vec3, sourceUnit, targetUnit voxel.MapObject, projectedTargetLocation voxel.Int3) *game.RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *game.UnitInstance
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
				unitHit = targetUnit.(*game.UnitInstance)
				//println(fmt.Sprintf("[GameInstance] Raycast hit unit %s at %d, %d, %d", unitHit.Name, x, y, z))
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

	return &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
}

func (g *GameInstance) RayCastFreeAim(rayStart, rayEnd mgl32.Vec3, sourceUnit *game.UnitInstance) *game.FreeAimHit {
	rayHitObject := false
	var hitPart util.Collider
	var hitPoint mgl32.Vec3
	var hitUnit voxel.MapObject
	var visitedBlocks []voxel.Int3
	checkedCollision := make(map[voxel.MapObject]bool)
	rayHitInfo := util.DDARaycast(rayStart, rayEnd, func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if g.voxelMap.IsSolidBlockAt(x, y, z) || !g.voxelMap.Contains(x, y, z) {
			return true
		}
		block := g.voxelMap.GetGlobalBlock(x, y, z)

		if block != nil && block.IsOccupied() {
			collidingObject := block.GetOccupant().(*game.UnitInstance)
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
	}
	insideMap := g.voxelMap.ContainsGrid(rayHitInfo.CollisionGridPosition) || g.voxelMap.ContainsGrid(rayHitInfo.PreviousGridPosition)
	partName := util.BodyPartNone
	if hitPart != nil {
		partName = hitPart.GetName()
	}
	return &game.FreeAimHit{RayCastHit: game.RayCastHit{HitInfo3D: rayHitInfo, VisitedBlocks: visitedBlocks, UnitHit: hitUnit, InsideMap: insideMap}, BodyPart: partName}
}
