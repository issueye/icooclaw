@echo off
setlocal enabledelayedexpansion

:: Icooclaw Build Script for Windows
:: Usage: build.bat [target]

set BINARY_NAME=icooclaw
set VERSION=dev
set BUILD_TIME=%date% %time%

:: 获取 git 版本信息
for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i

echo ========================================
echo  Icooclaw Build Script (Windows)
echo ========================================
echo Version: %VERSION%
echo Build Time: %BUILD_TIME%
echo.

:: 解析命令参数
set TARGET=%1
if "%TARGET%"=="" set TARGET=build

if "%TARGET%"=="clean" goto clean
if "%TARGET%"=="build" goto build
if "%TARGET%"=="build-all" goto build-all
if "%TARGET%"=="test" goto test
if "%TARGET%"=="install" goto install
if "%TARGET%"=="deps" goto deps

echo Unknown target: %TARGET%
echo.
echo Usage: build.bat [target]
echo.
echo Targets:
echo   build       - Build for current platform (default)
echo   build-all   - Build for all platforms
echo   clean       - Clean build artifacts
echo   test        - Run tests
echo   install     - Install to GOPATH\bin
echo   deps        - Download dependencies
goto :eof

:clean
echo Cleaning...
if exist bin rmdir /s /q bin
if exist %BINARY_NAME%.exe del /f /q %BINARY_NAME%.exe
if exist coverage.out del /f /q coverage.out
if exist coverage.html del /f /q coverage.html
echo Clean complete.
goto :eof

:build
echo Building %BINARY_NAME%...
if not exist bin mkdir bin

set LDFLAGS=-ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=%VERSION% -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.buildTime=%BUILD_TIME%"

go build %LDFLAGS% -o bin\%BINARY_NAME%.exe .\cmd\icooclaw
if errorlevel 1 (
    echo Build failed!
    exit /b 1
)
echo.
echo Build successful: bin\%BINARY_NAME%.exe
goto :eof

:build-all
echo Building all platforms...
if not exist bin mkdir bin
if not exist bin\windows mkdir bin\windows
if not exist bin\linux mkdir bin\linux
if not exist bin\darwin mkdir bin\darwin

set LDFLAGS=-ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=%VERSION%"

echo Building for Windows...
go build %LDFLAGS% -o bin\windows\%BINARY_NAME%.exe .\cmd\icooclaw

echo Building for Linux...
set GOOS=linux
set GOARCH=amd64
go build %LDFLAGS% -o bin\linux\%BINARY_NAME% .\cmd\icooclaw

echo Building for macOS...
set GOOS=darwin
set GOARCH=amd64
go build %LDFLAGS% -o bin\darwin\%BINARY_NAME%_darwin_amd64 .\cmd\icooclaw
set GOOS=darwin
set GOARCH=arm64
go build %LDFLAGS% -o bin\darwin\%BINARY_NAME%_darwin_arm64 .\cmd\icooclaw

echo.
echo All platforms built successfully!
goto :eof

:test
echo Running tests...
go test -v .\...
goto :eof

:install
echo Installing to GOPATH\bin...
set LDFLAGS=-ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=%VERSION%"
go install %LDFLAGS% .\cmd\icooclaw
echo Install complete.
goto :eof

:deps
echo Downloading dependencies...
go mod download
go mod tidy
echo Dependencies ready.
goto :eof
