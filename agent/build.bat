@echo off
setlocal

:: Icooclaw Build Script
:: Usage: build.bat [build|clean|test|install]

set BINARY_NAME=icooclaw
set VERSION=dev

:: 获取 git 版本
for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i

set TARGET=%1
if "%TARGET%"=="" set TARGET=build

if "%TARGET%"=="clean" goto clean
if "%TARGET%"=="test" goto test
if "%TARGET%"=="install" goto install
if "%TARGET%"=="build" goto build
goto build

:clean
echo Cleaning...
if exist bin rmdir /s /q bin
if exist %BINARY_NAME%.exe del /f /q %BINARY_NAME%.exe
echo Done.
goto :eof

:test
echo Running tests...
go test -v .\...
goto :eof

:install
echo Installing...
go install -ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=%VERSION%" .\cmd\icooclaw
echo Done.
goto :eof

:build
echo Building %BINARY_NAME% v%sVERSION%...
if not exist bin mkdir bin
go build -ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=%VERSION%" -o bin\%BINARY_NAME%.exe .\cmd\icooclaw
if errorlevel 1 exit /b 1
echo Build OK: bin\%BINARY_NAME%.exe
