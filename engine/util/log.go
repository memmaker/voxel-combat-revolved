package util

var GLOBAL_LOG_LEVEL = LogLevelInfo
var GLOBAL_LOG_CATEGORIES = LogNetwork

type LogLevel int

const (
    LogLevelError LogLevel = 1 << iota
    LogLevelWarning
    LogLevelDebug
    LogLevelInfo
)

type LogCategory int

const (
    LogVoxel LogCategory = 1 << iota
    LogNetwork
    LogSystem
    LogOpenGL
    LogIO
    LogUnitState
    LogGameStateGlobal
    LogTextures
    LogAnimation
)

func log(cat LogCategory, lvl LogLevel, txt string) {
    if lvl > GLOBAL_LOG_LEVEL {
        return
    }
    if GLOBAL_LOG_CATEGORIES&cat == 0 {
        return
    }
    println(txt)
}

func LogVoxelInfo(txt string) {
    log(LogVoxel, LogLevelInfo, txt)
}

func LogVoxelDebug(txt string) {
    log(LogVoxel, LogLevelDebug, txt)
}
func LogVoxelError(txt string) {
    log(LogVoxel, LogLevelError, txt)
}
func LogNetworkInfo(txt string) {
    log(LogNetwork, LogLevelInfo, txt)
}

func LogNetworkDebug(txt string) {
    log(LogNetwork, LogLevelDebug, txt)
}

func LogNetworkWarning(txt string) {
    log(LogNetwork, LogLevelWarning, txt)
}

func LogNetworkError(txt string) {
    log(LogNetwork, LogLevelError, txt)
}

func LogSystemInfo(txt string) {
    log(LogSystem, LogLevelInfo, txt)
}

func LogIOError(txt string) {
    log(LogIO, LogLevelError, txt)
}

func LogGameInfo(txt string) {
    log(LogGameStateGlobal, LogLevelInfo, txt)
}

func LogGameError(txt string) {
    log(LogGameStateGlobal, LogLevelError, txt)
}

func LogUnitDebug(txt string) {
    log(LogUnitState, LogLevelDebug, txt)
}

func LogTextureDebug(txt string) {
    log(LogTextures, LogLevelDebug, txt)
}

func LogTextureError(txt string) {
    log(LogTextures, LogLevelError, txt)
}

func LogAnimationDebug(txt string) {
    log(LogAnimation, LogLevelDebug, txt)
}

func LogAnimationError(txt string) {
    log(LogAnimation, LogLevelError, txt)
}

func LogAnimationInfo(txt string) {
    log(LogAnimation, LogLevelInfo, txt)
}

func LogGlInfo(txt string) {
    log(LogOpenGL, LogLevelInfo, txt)
}

func LogGlDebug(txt string) {
    log(LogOpenGL, LogLevelDebug, txt)
}

func LogGlError(txt string) {
    log(LogOpenGL, LogLevelError, txt)
}

func LogGlWarning(txt string) {
    log(LogOpenGL, LogLevelWarning, txt)
}
