package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExpandPath 展开路径中的环境变量和 ~ 符号
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	// 先展开环境变量
	// Windows 上没有 $HOME，使用 $USERPROFILE
	if _, ok := os.LookupEnv("HOME"); !ok {
		if userProfile, ok := os.LookupEnv("USERPROFILE"); ok {
			os.Setenv("HOME", userProfile)
		}
	}
	path = os.ExpandEnv(path)

	// 处理 ~ 符号
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}
