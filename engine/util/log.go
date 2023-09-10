package util

import (
    "fmt"
    termColor "github.com/fatih/color"
)

var GLOBAL_LOG_LEVEL = LogLevelInfo
var GLOBAL_LOG_CATEGORIES = LogVoxel
var GLOBAL_LOG_ENVIRONMENT = LogEnvironmentServer | LogEnvironmentGraphicalClient

type LogEnvironment int

func (e LogEnvironment) ToString() string {
    switch e {
    case LogEnvironmentServer:
        return "Server"
    case LogEnvironmentGraphicalClient:
        return "GL-Client"
    case LogEnvironmentAiClient:
        return "AI-Client"
    }
    return fmt.Sprintf("Unknown(%d)", e)
}

const (
    LogEnvironmentServer LogEnvironment = 1 << iota
    LogEnvironmentGraphicalClient
    LogEnvironmentAiClient
)

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

var allEnv = LogEnvironmentServer | LogEnvironmentGraphicalClient | LogEnvironmentAiClient

func logGlobal(cat LogCategory, lvl LogLevel, txt string) {
    log(allEnv, cat, lvl, txt)
}

func log(env LogEnvironment, cat LogCategory, lvl LogLevel, txt string) {
    if lvl > GLOBAL_LOG_LEVEL {
        return
    }
    if GLOBAL_LOG_CATEGORIES&cat == 0 {
        return
    }
    if GLOBAL_LOG_ENVIRONMENT&env == 0 {
        return
    }
    if env != allEnv {
        txt = fmt.Sprintf("[%s] %s", env.ToString(), txt)
    }
    println(txt)
}

func LogVoxelInfo(txt string) {
    logGlobal(LogVoxel, LogLevelInfo, txt)
}

func LogVoxelDebug(txt string) {
    logGlobal(LogVoxel, LogLevelDebug, txt)
}
func LogVoxelError(txt string) {
    logGlobal(LogVoxel, LogLevelError, txt)
}
func LogNetworkInfo(txt string) {
    logGlobal(LogNetwork, LogLevelInfo, txt)
}

func LogNetworkDebug(txt string) {
    logGlobal(LogNetwork, LogLevelDebug, txt)
}

func LogNetworkWarning(txt string) {
    logGlobal(LogNetwork, LogLevelWarning, txt)
}

func LogNetworkError(txt string) {
    logGlobal(LogNetwork, LogLevelError, txt)
}

func LogSystemInfo(txt string) {
    logGlobal(LogSystem, LogLevelInfo, txt)
}

func LogIOError(txt string) {
    logGlobal(LogIO, LogLevelError, txt)
}

func LogGameInfo(txt string) {
    logGlobal(LogGameStateGlobal, LogLevelInfo, txt)
}

func LogServerGameInfo(txt string) {
    log(LogEnvironmentServer, LogGameStateGlobal, LogLevelInfo, txt)
}

func LogGraphicalClientGameInfo(txt string) {
    log(LogEnvironmentGraphicalClient, LogGameStateGlobal, LogLevelInfo, txt)
}

func LogGraphicalClientGameDebug(txt string) {
    log(LogEnvironmentGraphicalClient, LogGameStateGlobal, LogLevelDebug, txt)
}
func getGreenLogger() func(format string, a ...interface{}) {
    return termColor.New(termColor.FgGreen).PrintfFunc()
}
func LogGreen(txt string) {
    greenLogger := getGreenLogger()
    greenLogger(txt + "\n")
}

func LogAiClientGameInfo(txt string) {
    log(LogEnvironmentAiClient, LogGameStateGlobal, LogLevelInfo, txt)
}

func LogServerGameError(txt string) {
    log(LogEnvironmentServer, LogGameStateGlobal, LogLevelError, txt)
}

func LogGraphicalClientGameError(txt string) {
    log(LogEnvironmentGraphicalClient, LogGameStateGlobal, LogLevelError, txt)
}

func LogAiClientGameError(txt string) {
    log(LogEnvironmentAiClient, LogGameStateGlobal, LogLevelError, txt)
}

func LogGameError(txt string) {
    logGlobal(LogGameStateGlobal, LogLevelError, txt)
}

func LogGlobalUnitDebug(txt string) {
    logGlobal(LogUnitState, LogLevelDebug, txt)
}

func LogServerUnitDebug(txt string) {
    log(LogEnvironmentServer, LogUnitState, LogLevelDebug, txt)
}

func LogTextureDebug(txt string) {
    logGlobal(LogTextures, LogLevelDebug, txt)
}

func LogTextureError(txt string) {
    logGlobal(LogTextures, LogLevelError, txt)
}

func LogAnimationDebug(txt string) {
    logGlobal(LogAnimation, LogLevelDebug, txt)
}

func LogAnimationError(txt string) {
    logGlobal(LogAnimation, LogLevelError, txt)
}

func LogAnimationInfo(txt string) {
    logGlobal(LogAnimation, LogLevelInfo, txt)
}

func LogGlInfo(txt string) {
    logGlobal(LogOpenGL, LogLevelInfo, txt)
}

func LogGlDebug(txt string) {
    logGlobal(LogOpenGL, LogLevelDebug, txt)
}

func LogGlError(txt string) {
    logGlobal(LogOpenGL, LogLevelError, txt)
}

func LogGlWarning(txt string) {
    logGlobal(LogOpenGL, LogLevelWarning, txt)
}
