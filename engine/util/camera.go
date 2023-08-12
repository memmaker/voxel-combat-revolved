package util

import (
    "github.com/go-gl/mathgl/mgl32"
)

type Camera interface {
    GetViewMatrix() mgl32.Mat4
    GetProjectionMatrix() mgl32.Mat4
    GetFront() mgl32.Vec3
    GetFrustumPlanes(matrix mgl32.Mat4) []mgl32.Vec4
    GetPosition() mgl32.Vec3
    ChangePosition(dir [2]int, delta float32)
    GetNearPlaneDist() float32
}


func AdjustForAspectRatio(x,y float64, screenWidth, screenHeight int) (float64, float64) {
    width := float64(screenHeight) / float64(screenWidth) // eg. 600/800 = 0.75
    //centerOffsetXStart := (1 - width) / 2                 // eg. (1-0.75)/2 = 0.125
    //centerOffsetXEnd := 1 - centerOffsetXStart            // eg. 1-0.125 = 0.875

    x = x * width
    return x, y
}
func GetRayFromCameraPlane(cam Camera, normalizedX float32, normalizedY float32) (mgl32.Vec3, mgl32.Vec3) {
    rayLength := float32(100)

    normalizedNearPos := mgl32.Vec4{normalizedX, normalizedY, cam.GetNearPlaneDist(), 1}
    normalizedFarPos := mgl32.Vec4{normalizedX, normalizedY, cam.GetNearPlaneDist() + rayLength, 1}

    proj := cam.GetProjectionMatrix()
    view := cam.GetViewMatrix()
    projViewInverted := proj.Mul4(view).Inv()

    // project point from camera space to world space
    nearWorldPos := projViewInverted.Mul4x1(normalizedNearPos)
    farWorldPos := projViewInverted.Mul4x1(normalizedFarPos)
    // perspective divide
    rayStart := nearWorldPos.Vec3().Mul(1 / nearWorldPos.W())
    farPosCorrected := farWorldPos.Vec3().Mul(1 / farWorldPos.W())
    dir := rayStart.Sub(farPosCorrected).Normalize()
    rayEnd := rayStart.Add(dir.Mul(rayLength))
    return rayStart, rayEnd
}
