package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type CollidingObject interface {
	GetExtents() mgl32.Vec3
	GetAABB() AABB
	GetPosition() mgl32.Vec3
	SetPosition(pos mgl32.Vec3)
	GetVelocity() mgl32.Vec3
	GetColliders() []Collider
	GetName() string
	SetVelocity(newVelocity mgl32.Vec3)
	HitWithProjectile(projectile CollidingObject, part Collider)
	IsDead() bool
}
type CollisionSolver struct {
	allObjects              []CollidingObject
	allProjectiles          []CollidingObject
	isSolid                 func(x int32, y int32, z int32) bool
	corrections             map[CollidingObject]mgl32.Vec3
	objectsOnGrid           map[voxel.Int3]map[CollidingObject]bool
	accelerationFromGravity mgl32.Vec3
}

func NewCollisionSolver(isSolid func(x int32, y int32, z int32) bool) *CollisionSolver {
	return &CollisionSolver{
		isSolid:                 isSolid,
		corrections:             make(map[CollidingObject]mgl32.Vec3),
		objectsOnGrid:           make(map[voxel.Int3]map[CollidingObject]bool),
		accelerationFromGravity: mgl32.Vec3{0, -9.8, 0},
	}
}

func (c *CollisionSolver) moveObjectOnGrid(obj CollidingObject, oldPos mgl32.Vec3, newPos mgl32.Vec3) {
	c.removeFromGrid(oldPos, obj)
	c.addToGrid(newPos, obj)
}

const epsilon = 0.001

// can lead to stuff leaving the map and goign to infinity
func (c *CollisionSolver) Update(deltaTime float64) {
	var collidingObjects []CollidingObject
	// 0. move stuff (player, actors, projectiles)
	for _, obj := range c.allObjects {
		previousPos := obj.GetPosition()
		rawVelocity := obj.GetVelocity()
		appliedVelocity := rawVelocity.Add(c.accelerationFromGravity).Mul(float32(deltaTime))
		newPos := obj.GetPosition().Add(appliedVelocity)
		obj.SetPosition(newPos)
		// 2. everything that moves will mark the blocks it touches as dirty for collision detection
		c.moveObjectOnGrid(obj, previousPos, newPos)
	}
	// 7. (Optional) For objects that are marked as "projectile" do a raycast against all relevant colliders
	for _, obj := range c.allObjects {
		objVelocity := obj.GetVelocity().Add(c.accelerationFromGravity).Mul(float32(deltaTime))
		previousPos := obj.GetPosition().Sub(objVelocity)
		newPos := obj.GetPosition()

		// 1. check for collisions with world, but don't correct, only save the correction vector
		correctedPos, isCollidingWithMap := AABBVoxelMapIntersection(previousPos, newPos, obj.GetExtents(), c.isSolid)
		if isCollidingWithMap {
			c.corrections[obj] = correctedPos.Sub(newPos)
		}

		// 3. Do broad phase by iterating over all dirty blocks
		// 4. Return all AABBs that are in the same block as the dirty block
		isColliding, sumOfCorrections, _ := c.DoAABBNarrowPhase(deltaTime, obj, c.getGridObjectsNear(newPos, obj), collidingObjects)
		if isColliding {
			c.corrections[obj] = c.corrections[obj].Add(sumOfCorrections)
		}
		// 6. Resolve collisions by moving the objects back by the sum of all correction vectors
		if correction, ok := c.corrections[obj]; ok {
			finalPos := newPos.Add(correction)
			obj.SetPosition(finalPos)
			c.moveObjectOnGrid(obj, newPos, finalPos)
			delete(c.corrections, obj)
		}
	}

	c.updateProjectiles(deltaTime)
}

func (c *CollisionSolver) updateProjectiles(deltaTime float64) {
	checkedCollision := make(map[CollidingObject]bool, 10)
	for pIndex := len(c.allProjectiles) - 1; pIndex >= 0; pIndex-- {
		obj := c.allProjectiles[pIndex]
		if obj.IsDead() {
			c.allProjectiles = append(c.allProjectiles[:pIndex], c.allProjectiles[pIndex+1:]...)
			continue
		}
		for checkedCollider := range checkedCollision {
			delete(checkedCollision, checkedCollider)
		}
		objVelocity := obj.GetVelocity().Mul(float32(deltaTime))
		if objVelocity.Len() <= 0 {
			continue
		}
		previousPos := obj.GetPosition()
		newPos := obj.GetPosition().Add(objVelocity)
		rayHitObject := false
		var nearestPart Collider
		var nearestPoint mgl32.Vec3
		var nearestObject CollidingObject
		firstGridCell := true
		var lastGridCell voxel.Int3
		var gridCell voxel.Int3
		rayHitInfo := DDARaycast(previousPos, newPos, func(x, y, z int32) bool {
			if c.isSolid(int32(x), int32(y), int32(z)) {
				return true
			}
			gridCell = voxel.Int3{x, y, z}.Div(2)
			if objectsInCell, ok := c.objectsOnGrid[gridCell]; (firstGridCell || lastGridCell != gridCell) && ok && len(objectsInCell) > 0 {
				//isColliding, _, collisions := c.DoAABBNarrowPhase(deltaTime, obj, objectsInCell, collidingObjects) // WONT WORK FOR PROJECTILES
				//println(fmt.Sprintf("Checking %s against %d objects in cell %v", obj.GetName(), len(objectsInCell), gridCell))
				var rayPoint mgl32.Vec3
				rayHit := false
				for collidingObject := range objectsInCell {
					if _, checkedBefore := checkedCollision[collidingObject]; checkedBefore {
						continue
					}
					minDistance := float32(math.MaxFloat32)
					//println(fmt.Sprintf("Checking %s against %s", obj.GetName(), collidingObject.GetName()))
					for _, meshPartCollider := range collidingObject.GetColliders() {
						//meshsCollided, _ = util.GJK(projectile.GetCollider(), meshPartCollider) // we made this sweeping for the projectiles only for now
						rayHit, rayPoint = meshPartCollider.IntersectsRay(previousPos, newPos)
						if rayHit {
							rayHitObject = true
							dist := rayPoint.Sub(previousPos).Len()
							if dist < minDistance {
								minDistance = dist
								nearestPart = meshPartCollider
								nearestPoint = rayPoint
								nearestObject = collidingObject
							}
						}
						checkedCollision[collidingObject] = true
					}
				}
			}
			lastGridCell = gridCell
			firstGridCell = false
			return false
		})
		if rayHitObject {
			println(fmt.Sprintf("Hit: %s of %s", nearestPart.GetName(), nearestObject.GetName()))
			c.onProjectileObjectCollision(obj, nearestObject, nearestPart, nearestPoint)
		} else if rayHitInfo.Hit {
			println(fmt.Sprintf("Hit Block: %v", rayHitInfo.CollisionWorldPosition))
			c.onProjectileMapCollision(obj, rayHitInfo)
		} else {
			obj.SetPosition(newPos)
		}
	}
}

func (c *CollisionSolver) DoAABBNarrowPhase(deltaTime float64, obj CollidingObject, other map[CollidingObject]bool, buffer []CollidingObject) (bool, mgl32.Vec3, []CollidingObject) {
	buffer = buffer[:0]
	var isColliding bool
	var resultingCorrectionVector mgl32.Vec3
	objVelocity := obj.GetVelocity().Add(c.accelerationFromGravity).Mul(float32(deltaTime))
	previousPos := obj.GetPosition().Sub(objVelocity)
	newPos := obj.GetPosition()
	for otherObj, _ := range other {
		if otherObj == obj {
			continue
		}
		//println(fmt.Sprintf("Checking %s against %s", obj.GetName(), otherObj.GetName()))
		otherVelocity := otherObj.GetVelocity().Add(c.accelerationFromGravity).Mul(float32(deltaTime))
		otherPrevPos := otherObj.GetPosition().Sub(otherVelocity)
		combinedRelativeVelocity := objVelocity.Sub(otherVelocity)
		prevMin := previousPos.Sub(obj.GetExtents().Mul(0.5))
		otherPrevMin := otherPrevPos.Sub(otherObj.GetExtents().Mul(0.5))

		mMin, mExtents := AABBMinkowskiDifference(prevMin, obj.GetExtents(), otherPrevMin, otherObj.GetExtents())
		minkowskiAABB := NewAABBFromMin(mMin, mExtents)

		// 5. Do narrow phase by checking all AABBs against each other
		colInfo := SweepAABBFromMinkowski(minkowskiAABB, combinedRelativeVelocity)
		AABBsCollided := colInfo.Result < 1.0 || colInfo.MinkowskiDifferenceContainsOrigin
		if AABBsCollided { // AABBs collided / are colliding
			isColliding = true
			buffer = append(buffer, otherObj)
			correctedPos := previousPos.Add(objVelocity.Mul(colInfo.Result).Add(colInfo.Normal.Mul(epsilon)))
			correctionVector := correctedPos.Sub(newPos)
			resultingCorrectionVector = resultingCorrectionVector.Add(correctionVector)
		}
	}
	return isColliding, resultingCorrectionVector, buffer
}

func (c *CollisionSolver) AddObject(obj CollidingObject) {
	c.allObjects = append(c.allObjects, obj)
	c.addToGrid(obj.GetPosition(), obj)
}

func (c *CollisionSolver) AddProjectile(obj CollidingObject) {
	c.allProjectiles = append(c.allProjectiles, obj)
}

func (c *CollisionSolver) addToGrid(atPos mgl32.Vec3, obj CollidingObject) {
	// check all 8 corners of the AABB
	extents := obj.GetExtents()

	// top face corners (ttl, ttr, tbl, tbr)
	ttl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	ttr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	tbl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)
	tbr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)

	// bottom face corners (btl, btr, bbl, bbr)
	btl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	btr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	bbl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)
	bbr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)

	if c.objectsOnGrid[ttl] == nil {
		c.objectsOnGrid[ttl] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[ttr] == nil {
		c.objectsOnGrid[ttr] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[tbl] == nil {
		c.objectsOnGrid[tbl] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[tbr] == nil {
		c.objectsOnGrid[tbr] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[btl] == nil {
		c.objectsOnGrid[btl] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[btr] == nil {
		c.objectsOnGrid[btr] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[bbl] == nil {
		c.objectsOnGrid[bbl] = make(map[CollidingObject]bool)
	}
	if c.objectsOnGrid[bbr] == nil {
		c.objectsOnGrid[bbr] = make(map[CollidingObject]bool)
	}

	c.objectsOnGrid[ttl][obj] = true
	c.objectsOnGrid[ttr][obj] = true
	c.objectsOnGrid[tbl][obj] = true
	c.objectsOnGrid[tbr][obj] = true
	c.objectsOnGrid[btl][obj] = true
	c.objectsOnGrid[btr][obj] = true
	c.objectsOnGrid[bbl][obj] = true
	c.objectsOnGrid[bbr][obj] = true
}

func (c *CollisionSolver) removeFromGrid(atPos mgl32.Vec3, obj CollidingObject) {
	extents := obj.GetExtents()

	// top face corners (ttl, ttr, tbl, tbr)
	ttl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	ttr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	tbl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)
	tbr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() + extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)

	// bottom face corners (btl, btr, bbl, bbr)
	btl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	btr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() - extents.Z())}.Div(2)
	bbl := voxel.Int3{int32(atPos.X() - extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)
	bbr := voxel.Int3{int32(atPos.X() + extents.X()), int32(atPos.Y() - extents.Y()), int32(atPos.Z() + extents.Z())}.Div(2)

	delete(c.objectsOnGrid[ttl], obj)
	delete(c.objectsOnGrid[ttr], obj)
	delete(c.objectsOnGrid[tbl], obj)
	delete(c.objectsOnGrid[tbr], obj)
	delete(c.objectsOnGrid[btl], obj)
	delete(c.objectsOnGrid[btr], obj)
	delete(c.objectsOnGrid[bbl], obj)
	delete(c.objectsOnGrid[bbr], obj)
}

func (c *CollisionSolver) getGridObjectsNear(pos mgl32.Vec3, collidingObject CollidingObject) map[CollidingObject]bool {

	extents := collidingObject.GetExtents()
	// top face corners (ttl, ttr, tbl, tbr)
	ttl := voxel.Int3{int32(pos.X() - extents.X()), int32(pos.Y() + extents.Y()), int32(pos.Z() - extents.Z())}.Div(2)
	ttr := voxel.Int3{int32(pos.X() + extents.X()), int32(pos.Y() + extents.Y()), int32(pos.Z() - extents.Z())}.Div(2)
	tbl := voxel.Int3{int32(pos.X() - extents.X()), int32(pos.Y() + extents.Y()), int32(pos.Z() + extents.Z())}.Div(2)
	tbr := voxel.Int3{int32(pos.X() + extents.X()), int32(pos.Y() + extents.Y()), int32(pos.Z() + extents.Z())}.Div(2)

	// bottom face corners (btl, btr, bbl, bbr)
	btl := voxel.Int3{int32(pos.X() - extents.X()), int32(pos.Y() - extents.Y()), int32(pos.Z() - extents.Z())}.Div(2)
	btr := voxel.Int3{int32(pos.X() + extents.X()), int32(pos.Y() - extents.Y()), int32(pos.Z() - extents.Z())}.Div(2)
	bbl := voxel.Int3{int32(pos.X() - extents.X()), int32(pos.Y() - extents.Y()), int32(pos.Z() + extents.Z())}.Div(2)
	bbr := voxel.Int3{int32(pos.X() + extents.X()), int32(pos.Y() - extents.Y()), int32(pos.Z() + extents.Z())}.Div(2)

	objects := make(map[CollidingObject]bool)
	for obj := range c.objectsOnGrid[ttl] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[ttr] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[tbl] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[tbr] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[btl] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[btr] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[bbl] {
		objects[obj] = true
	}
	for obj := range c.objectsOnGrid[bbr] {
		objects[obj] = true
	}
	return objects

}

func (c *CollisionSolver) onProjectileObjectCollision(projectile CollidingObject, collidingObject CollidingObject, part Collider, worldPosOfCollision mgl32.Vec3) {
	collidingObject.HitWithProjectile(projectile, part)
	projectile.SetVelocity(mgl32.Vec3{0, 0, 0})
	projectile.SetPosition(worldPosOfCollision)
}

func (c *CollisionSolver) onProjectileMapCollision(projectile CollidingObject, info HitInfo3D) {
	projectile.SetVelocity(mgl32.Vec3{0, 0, 0})
	projectile.SetPosition(info.CollisionWorldPosition)
}
