package utils

import "runtime"

// isWindows 检查是否是 Windows 系统
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
