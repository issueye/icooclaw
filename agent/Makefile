# Icooclaw Build Script
# Usage: make build

.PHONY: all clean build build-windows build-linux build-darwin build-all install run test

# 项目信息
BINARY_NAME=icooclaw
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d %H:%M:%S')
GO_VERSION=$(shell go version)
LDFLAGS=-ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=$(VERSION) -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.buildTime=$(BUILD_TIME)"

# 默认目标：构建当前平台的二进制
all: build

# 清理构建产物
clean:
	rm -rf bin/ dist/
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe

# 构建当前平台
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/icooclaw
	@echo "Build successful: bin/$(BINARY_NAME)"

# 构建 Windows 版本
build-windows:
	@echo "Building for Windows..."
	@mkdir -p bin/windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/windows/$(BINARY_NAME).exe ./cmd/icooclaw
	@echo "Build successful: bin/windows/$(BINARY_NAME).exe"

# 构建 Linux 版本
build-linux:
	@echo "Building for Linux..."
	@mkdir -p bin/linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/linux/$(BINARY_NAME) ./cmd/icooclaw
	@echo "Build successful: bin/linux/$(BINARY_NAME)"

# 构建 macOS 版本
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p bin/darwin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/darwin/$(BINARY_NAME)_darwin_amd64 ./cmd/icooclaw
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/darwin/$(BINARY_NAME)_darwin_arm64 ./cmd/icooclaw
	@echo "Build successful: bin/darwin/$(BINARY_NAME)_darwin_*

# 构建所有平台版本
build-all: build-windows build-linux build-darwin
	@echo "All platforms built successfully!"

# 安装到 GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/icooclaw

# 运行程序
run:
	go run ./cmd/icooclaw

# 运行测试
test:
	go test -v ./...

# 运行测试（带覆盖率）
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 代码格式化
fmt:
	go fmt ./...
	gofmt -s -w .

# 代码检查
lint:
	go vet ./...

# 依赖管理
deps:
	go mod download
	go mod tidy

# 显示版本信息
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
