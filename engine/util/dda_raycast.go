package util

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type CardinalDirection int

const (
	North CardinalDirection = iota
	East
	South
	West
)

type HitInfo2D struct {
	Distance               float64
	Side                   CardinalDirection
	CollisionX, CollisionY float64
	GridX                  int64
	GridY                  int64
}

func Raycast2D(startX, startY, directionX, directionY float64, shouldStopRay func(x, y int64) bool) HitInfo2D {
	// 2. Die sich wiederholenden Strahlenteile für Schritte in CollisionX und CollisionY Richtung berechnen
	//    - Steigung des Richtungsvektors und ein Schritt in CollisionX bzw. CollisionY Richtung
	//    - Satz des Pythagoras mit a = 1 und b = ray.y / ray.x für einen Schritt in die CollisionX-Richtung
	//    - Satz des Pythagoras mit a = 1 und b = ray.x / ray.y für einen Schritt in die CollisionY-Richtung
	deltaDistX := math.Sqrt(1 + (directionY*directionY)/(directionX*directionX))
	deltaDistY := math.Sqrt(1 + (directionX*directionX)/(directionY*directionY))

	startGridX := int64(math.Floor(startX))
	startGridY := int64(math.Floor(startY))

	mapX := startGridX
	mapY := startGridY

	var intraCellPositionX float64
	var intraCellPositionY float64

	var sideDistX float64
	var sideDistY float64

	var mapStepX int64
	var mapStepY int64

	// 3. Anhand der Richtung des Strahls folgende Werte berechnen
	//     - Die Schrittrichtung auf der Karte (je Quadrant)
	//     - Die position innerhalb der aktuellen Zelle
	//     - Die anteiligen Startteile der beiden Strahlen für CollisionX und CollisionY Schritte
	if directionX < 0 {
		mapStepX = -1
		intraCellPositionX = startX - float64(mapX)
		sideDistX = intraCellPositionX * deltaDistX
	} else {
		mapStepX = 1
		intraCellPositionX = float64(mapX) + 1.0 - startX
		sideDistX = intraCellPositionX * deltaDistX
	}

	if directionY < 0 {
		mapStepY = -1
		intraCellPositionY = startY - float64(mapY)
		sideDistY = intraCellPositionY * deltaDistY
	} else {
		mapStepY = 1
		intraCellPositionY = float64(mapY) + 1.0 - startY
		sideDistY = intraCellPositionY * deltaDistY
	}

	eastWestSide := false
	stopRaycasting := false
	nextCollisionX := 0.0
	nextCollisionY := 0.0

	// 4. Schrittweise die Strahlen verlängern, immer der kürzeste zuerst
	//    - Die Kartenposition wird aktualisiert
	//    - Die Wandrichtung (Nord/Süd vs. Ost/west) wird gesetzt
	//    - Der Strahl wird verlängert
	for !stopRaycasting {
		if sideDistX < sideDistY {
			// move one unit in CollisionX Direction
			nextCollisionX = startX + (directionX * sideDistX)
			nextCollisionY = startY + (directionY * sideDistX)

			mapX += mapStepX
			eastWestSide = true
			sideDistX += deltaDistX
		} else {
			// move one unit in CollisionY Direction
			nextCollisionX = startX + (directionX * sideDistY)
			nextCollisionY = startY + (directionY * sideDistY)

			mapY += mapStepY
			eastWestSide = false
			sideDistY += deltaDistY
		}

		// Optional: Sprites - Prüfen ob Sprites auf diesem Feld sind
		//Sprite visibleSprite = GetSpriteMapAt(mapX, mapY);
		/*
		   if (visibleSprite != null && !mVisibleSprites.Contains(visibleSprite))
		       mVisibleSprites.Add(visibleSprite);
		*/

		// 5. Prüfen wir ob eine Wand getroffen wurde
		stopRaycasting = shouldStopRay(mapX, mapY)
	}

	// 6. Den Abstand zur Wand berechnen (Im Prinzip den letzten Schritt rückgängig machen)
	// Vektoren subtrahieren
	var perpWallDist float64
	if eastWestSide {
		perpWallDist = sideDistX - deltaDistX
	} else {
		perpWallDist = sideDistY - deltaDistY
	}

	// 6.1 Die original Distanz zur Wand für das Texture Mapping verwenden
	var wallX float64 // CollisionX position an der wir die Wand getroffen haben
	// Komponentenweise Vektoraddition und Skalierung
	if eastWestSide {
		wallX = startY + (perpWallDist * directionY)
	} else {
		wallX = startX + (perpWallDist * directionX)
	}

	wallX -= math.Floor(wallX) // Uns interessieren nur die Nachkommastellen
	//textureX := (int)(wallX * mWallTextures[textureIndex].Width);

	var hitSide CardinalDirection
	if eastWestSide {
		if directionX < 0 {
			hitSide = West
		} else {
			hitSide = East
		}
	} else {
		if directionY < 0 {
			hitSide = North
		} else {
			hitSide = South
		}
	}
	return HitInfo2D{
		Distance:   perpWallDist,
		Side:       hitSide,
		CollisionX: nextCollisionX,
		CollisionY: nextCollisionY,
		GridX:      mapX,
		GridY:      mapY,
	}
}

type CubeSide int

const (
	Front CubeSide = iota
	Back
	Left
	Right
	Top
	Bottom
)

type HitInfo3D struct {
	Distance               float64
	Side                   CubeSide
	CollisionWorldPosition mgl32.Vec3
	PreviousGridPosition   voxel.Int3
	CollisionGridPosition  voxel.Int3
	Hit                    bool
}

func (d HitInfo3D) WithCollisionWorldPosition(point mgl32.Vec3) HitInfo3D {
	d.CollisionWorldPosition = point
	return d
}

func (d HitInfo3D) WithDistance(newDist float64) HitInfo3D {
	d.Distance = newDist
	return d
}

func DDARaycast(rayStart, rayEnd mgl32.Vec3, stopRay func(x, y, z int32) bool) HitInfo3D {
	// adapted from: https://github.com/fenomas/fast-voxel-raycast/blob/master/index.js
	t := 0.0
	ix := int32(math.Floor(float64(rayStart.X())))
	iy := int32(math.Floor(float64(rayStart.Y())))
	iz := int32(math.Floor(float64(rayStart.Z())))

	ray := rayEnd.Sub(rayStart)
	maxRayLength := ray.Len()
	rayDir := ray.Normalize()

	stepx := int32(-1)
	if rayDir.X() > 0 {
		stepx = 1
	}
	stepy := int32(-1)
	if rayDir.Y() > 0 {
		stepy = 1
	}
	stepz := int32(-1)
	if rayDir.Z() > 0 {
		stepz = 1
	}

	txDelta := math.Abs(float64(1.0 / rayDir.X()))
	tyDelta := math.Abs(float64(1.0 / rayDir.Y()))
	tzDelta := math.Abs(float64(1.0 / rayDir.Z()))

	xdist := float64(rayStart.X()) - float64(ix)
	if stepx > 0 {
		xdist = float64(ix+1) - float64(rayStart.X())
	}
	ydist := float64(rayStart.Y()) - float64(iy)
	if stepy > 0 {
		ydist = float64(iy+1) - float64(rayStart.Y())
	}
	zdist := float64(rayStart.Z()) - float64(iz)
	if stepz > 0 {
		zdist = float64(iz+1) - float64(rayStart.Z())
	}

	txMax := math.Inf(1)
	if txDelta < math.Inf(1) {
		txMax = txDelta * xdist
	}
	tyMax := math.Inf(1)
	if tyDelta < math.Inf(1) {
		tyMax = tyDelta * ydist
	}
	tzMax := math.Inf(1)
	if tzDelta < math.Inf(1) {
		tzMax = tzDelta * zdist
	}

	steppedIndex := -1

	for t <= float64(maxRayLength) {
		if stopRay(ix, iy, iz) {
			var previousGridPosition voxel.Int3
			var side CubeSide
			if steppedIndex == 0 {
				if stepx > 0 {
					side = Left
					previousGridPosition = voxel.Int3{ix - 1, iy, iz}
				} else {
					side = Right
					previousGridPosition = voxel.Int3{ix + 1, iy, iz}
				}
			}
			if steppedIndex == 1 {
				if stepy > 0 {
					side = Bottom
					previousGridPosition = voxel.Int3{ix, iy - 1, iz}
				} else {
					side = Top
					previousGridPosition = voxel.Int3{ix, iy + 1, iz}
				}
			}
			if steppedIndex == 2 {
				if stepz > 0 {
					side = Back
					previousGridPosition = voxel.Int3{ix, iy, iz - 1}
				} else {
					side = Front
					previousGridPosition = voxel.Int3{ix, iy, iz + 1}
				}
			}

			return HitInfo3D{
				Hit:                    true,
				Distance:               t,
				Side:                   side,
				CollisionWorldPosition: rayStart.Add(rayDir.Mul(float32(t))),
				PreviousGridPosition:   previousGridPosition,
				CollisionGridPosition:  voxel.Int3{X: ix, Y: iy, Z: iz},
			}
		}

		if txMax < tyMax {
			if txMax < tzMax {
				ix += stepx
				t = txMax
				txMax += txDelta
				steppedIndex = 0
			} else {
				iz += stepz
				t = tzMax
				tzMax += tzDelta
				steppedIndex = 2
			}
		} else {
			if tyMax < tzMax {
				iy += stepy
				t = tyMax
				tyMax += tyDelta
				steppedIndex = 1
			} else {
				iz += stepz
				t = tzMax
				tzMax += tzDelta
				steppedIndex = 2
			}
		}
	}

	return HitInfo3D{Hit: false}
}
