package util

import (
    "fmt"
    "github.com/go-gl/gl/v3.3-core/gl"
)

func CheckForGLError() {
    errorCodeOfGL := gl.GetError()

    if errorCodeOfGL != gl.NO_ERROR {
        fmt.Printf("GL error: %v\n", errorCodeOfGL)
    }
}
