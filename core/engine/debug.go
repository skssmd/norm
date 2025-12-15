package engine

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	debugMode     = true
	debugModeOnce sync.Once
)

// initDebugMode initializes debug mode from environment variable
func initDebugMode() {
	debugModeOnce.Do(func() {
		mode := strings.ToLower(os.Getenv("NORM_DEBUG"))
		debugMode = mode == "true" || mode == "1" || mode == "on"
	})
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	initDebugMode()
	return debugMode
}

// debugLog prints debug information only when debug mode is enabled
func debugLog(format string, args ...interface{}) {
	if IsDebugMode() {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// cacheLog prints cache-related debug information
func cacheLog(format string, args ...interface{}) {
	if IsDebugMode() {
		fmt.Printf("[CACHE] "+format+"\n", args...)
	}
}

// errorLog prints error information (always shown)
func errorLog(context string, err error, details map[string]interface{}) {
	fmt.Printf("[ERROR] %s: %v\n", context, err)
	if len(details) > 0 {
		fmt.Println("  Details:")
		for key, value := range details {
			fmt.Printf("    %s: %v\n", key, value)
		}
	}
}
