package script

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Engine 脚本引擎
type Engine struct {
	vm      *goja.Runtime
	config  *Config
	logger  *slog.Logger
	context context.Context
}

// Config 脚本引擎配置
type Config struct {
	// 工作目录
	Workspace string
	// 允许文件读取
	AllowFileRead bool
	// 允许文件写入
	AllowFileWrite bool
	// 允许文件删除
	AllowFileDelete bool
	// 允许执行命令
	AllowExec bool
	// 允许网络访问
	AllowNetwork bool
	// 命令执行超时（秒）
	ExecTimeout int
	// HTTP 请求超时（秒）
	HTTPTimeout int
	// 最大内存（字节）
	MaxMemory int64
	// 允许的域名白名单（网络请求）
	AllowedDomains []string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Workspace:       ".",
		AllowFileRead:   true,
		AllowFileWrite:  false,
		AllowFileDelete: false,
		AllowExec:       false,
		AllowNetwork:    true,
		ExecTimeout:     30,
		HTTPTimeout:     30,
		MaxMemory:       10 * 1024 * 1024,
		AllowedDomains:  []string{},
	}
}

// NewEngine 创建新的脚本引擎
func NewEngine(config *Config, logger *slog.Logger) *Engine {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	vm := goja.New()
	vm.SetMaxCallStackSize(100)
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	engine := &Engine{
		vm:      vm,
		config:  config,
		logger:  logger,
		context: context.Background(),
	}

	engine.setupBuiltins()
	return engine
}

// NewEngineWithContext 创建带上下文的脚本引擎
func NewEngineWithContext(ctx context.Context, config *Config, logger *slog.Logger) *Engine {
	engine := NewEngine(config, logger)
	engine.context = ctx
	return engine
}

// Run 执行脚本
func (e *Engine) Run(script string) (goja.Value, error) {
	return e.vm.RunString(script)
}

// RunFile 执行脚本文件
func (e *Engine) RunFile(path string) (goja.Value, error) {
	absPath := e.resolvePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}
	return e.vm.RunString(string(content))
}

// SetGlobal 设置全局变量
func (e *Engine) SetGlobal(name string, value interface{}) error {
	return e.vm.Set(name, value)
}

// GetGlobal 获取全局变量
func (e *Engine) GetGlobal(name string) goja.Value {
	return e.vm.Get(name)
}

// GetVM 获取底层 VM
func (e *Engine) GetVM() *goja.Runtime {
	return e.vm
}

// Call 调用函数
func (e *Engine) Call(name string, args ...interface{}) (goja.Value, error) {
	fn := e.vm.Get(name)
	if fn == nil || goja.IsUndefined(fn) {
		return nil, fmt.Errorf("function '%s' not found", name)
	}

	callable, ok := goja.AssertFunction(fn)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a function", name)
	}

	// 转换参数
	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = e.vm.ToValue(arg)
	}

	return callable(nil, jsArgs...)
}

// SetContext 设置上下文
func (e *Engine) SetContext(ctx context.Context) {
	e.context = ctx
}

// setupBuiltins 设置内置对象和方法
func (e *Engine) setupBuiltins() {
	// console 对象
	consoleObj := &console{logger: e.logger}
	e.SetGlobal("console", map[string]interface{}{
		"log":     consoleObj.Log,
		"info":    consoleObj.Info,
		"debug":   consoleObj.Debug,
		"warn":    consoleObj.Warn,
		"error":   consoleObj.Error,
		"table":   consoleObj.Table,
		"time":    consoleObj.Time,
		"timeEnd": consoleObj.TimeEnd,
	})

	// fs 文件系统对象
	fsObj := &fileSystem{config: e.config, logger: e.logger}
	e.SetGlobal("fs", map[string]interface{}{
		"readFile":       fsObj.ReadFile,
		"readFileBytes":  fsObj.ReadFileBytes,
		"writeFile":      fsObj.WriteFile,
		"writeFileBytes": fsObj.WriteFileBytes,
		"appendFile":     fsObj.AppendFile,
		"deleteFile":     fsObj.DeleteFile,
		"exists":         fsObj.Exists,
		"isDir":          fsObj.IsDir,
		"isFile":         fsObj.IsFile,
		"listDir":        fsObj.ListDir,
		"mkdir":          fsObj.Mkdir,
		"mkdirAll":       fsObj.MkdirAll,
		"rmdir":          fsObj.Rmdir,
		"copyFile":       fsObj.CopyFile,
		"moveFile":       fsObj.MoveFile,
		"getInfo":        fsObj.GetInfo,
	})

	// http HTTP 客户端对象
	httpObj := &httpClient{config: e.config, logger: e.logger}
	e.SetGlobal("http", map[string]interface{}{
		"get":      httpObj.Get,
		"post":     httpObj.Post,
		"postJSON": httpObj.PostJSON,
		"request":  httpObj.Request,
		"download": httpObj.Download,
	})

	// shell 命令执行对象
	shellObj := &shellExec{config: e.config, logger: e.logger, ctx: e.context}
	e.SetGlobal("shell", map[string]interface{}{
		"exec":            shellObj.Exec,
		"execWithTimeout": shellObj.ExecWithTimeout,
		"execInDir":       shellObj.ExecInDir,
		"pipe":            shellObj.Pipe,
	})

	// utils 工具函数对象
	utilsObj := &utils{}
	e.SetGlobal("utils", map[string]interface{}{
		"sleep":      utilsObj.Sleep,
		"now":        utilsObj.Now,
		"timestamp":  utilsObj.Timestamp,
		"formatTime": utilsObj.FormatTime,
		"parseTime":  utilsObj.ParseTime,
		"env":        utilsObj.Env,
		"envOr":      utilsObj.EnvOr,
		"cwd":        utilsObj.Cwd,
		"hostname":   utilsObj.Hostname,
		"uuid":       utilsObj.UUID,
	})

	// 标准库扩展
	e.setupStdLib()
}

// setupStdLib 设置标准库扩展
func (e *Engine) setupStdLib() {
	// JSON 扩展方法
	e.vm.Set("JSON", map[string]interface{}{
		"stringify": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"parse": func(s string) (interface{}, error) {
			var v interface{}
			err := json.Unmarshal([]byte(s), &v)
			return v, err
		},
		"pretty": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
	})

	// Base64 编码/解码
	e.vm.Set("Base64", map[string]interface{}{
		"encode": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"decode": func(s string) (string, error) {
			b, err := base64.StdEncoding.DecodeString(s)
			return string(b), err
		},
	})
}

// resolvePath 解析路径
func (e *Engine) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.config.Workspace, path)
}

// === console 对象 ===

type console struct {
	logger *slog.Logger
}

func (c *console) Log(args ...interface{}) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *console) Info(args ...interface{}) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *console) Debug(args ...interface{}) {
	c.logger.Debug(fmt.Sprint(args...))
}

func (c *console) Warn(args ...interface{}) {
	c.logger.Warn(fmt.Sprint(args...))
}

func (c *console) Error(args ...interface{}) {
	c.logger.Error(fmt.Sprint(args...))
}

func (c *console) Table(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	c.logger.Info(string(b))
}

func (c *console) Time(label string) {
	c.logger.Debug("Timer started", "label", label)
}

func (c *console) TimeEnd(label string) {
	c.logger.Debug("Timer ended", "label", label)
}

// === fs 文件系统对象 ===

type fileSystem struct {
	config *Config
	logger *slog.Logger
}

// ReadFile 读取文件
func (fs *fileSystem) ReadFile(path string) (string, error) {
	if !fs.config.AllowFileRead {
		return "", fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// ReadFileBytes 以字节数组形式读取文件
func (fs *fileSystem) ReadFileBytes(path string) ([]byte, error) {
	if !fs.config.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.ReadFile(absPath)
}

// WriteFile 写入文件
func (fs *fileSystem) WriteFile(path string, content string) error {
	if !fs.config.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)

	// 确保目录存在
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

// WriteFileBytes 写入字节数组到文件
func (fs *fileSystem) WriteFileBytes(path string, content []byte) error {
	if !fs.config.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(absPath, content, 0644)
}

// AppendFile 追加内容到文件
func (fs *fileSystem) AppendFile(path string, content string) error {
	if !fs.config.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// DeleteFile 删除文件
func (fs *fileSystem) DeleteFile(path string) error {
	if !fs.config.AllowFileDelete {
		return fmt.Errorf("file delete is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.Remove(absPath)
}

// Exists 检查文件是否存在
func (fs *fileSystem) Exists(path string) bool {
	absPath := fs.resolvePath(path)
	_, err := os.Stat(absPath)
	return err == nil
}

// IsDir 检查是否是目录
func (fs *fileSystem) IsDir(path string) bool {
	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile 检查是否是文件
func (fs *fileSystem) IsFile(path string) bool {
	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ListDir 列出目录内容
func (fs *fileSystem) ListDir(path string) ([]map[string]interface{}, error) {
	if !fs.config.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		result = append(result, map[string]interface{}{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		})
	}

	return result, nil
}

// Mkdir 创建目录
func (fs *fileSystem) Mkdir(path string) error {
	if !fs.config.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.MkdirAll(absPath, 0755)
}

// MkdirAll 递归创建目录
func (fs *fileSystem) MkdirAll(path string) error {
	return fs.Mkdir(path)
}

// Rmdir 删除目录
func (fs *fileSystem) Rmdir(path string) error {
	if !fs.config.AllowFileDelete {
		return fmt.Errorf("file delete is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.RemoveAll(absPath)
}

// CopyFile 复制文件
func (fs *fileSystem) CopyFile(src, dst string) error {
	if !fs.config.AllowFileRead || !fs.config.AllowFileWrite {
		return fmt.Errorf("file operation not allowed")
	}

	absSrc := fs.resolvePath(src)
	absDst := fs.resolvePath(dst)

	srcFile, err := os.Open(absSrc)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// 确保目标目录存在
	dir := filepath.Dir(absDst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dstFile, err := os.Create(absDst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MoveFile 移动文件
func (fs *fileSystem) MoveFile(src, dst string) error {
	if !fs.config.AllowFileWrite {
		return fmt.Errorf("file operation not allowed")
	}

	absSrc := fs.resolvePath(src)
	absDst := fs.resolvePath(dst)

	// 确保目标目录存在
	dir := filepath.Dir(absDst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.Rename(absSrc, absDst)
}

// GetInfo 获取文件信息
func (fs *fileSystem) GetInfo(path string) (map[string]interface{}, error) {
	if !fs.config.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return map[string]interface{}{
		"name":     info.Name(),
		"size":     info.Size(),
		"is_dir":   info.IsDir(),
		"modified": info.ModTime().Format(time.RFC3339),
		"mode":     info.Mode().String(),
	}, nil
}

func (fs *fileSystem) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fs.config.Workspace, path)
}

// === http 客户端对象 ===

type httpClient struct {
	config *Config
	logger *slog.Logger
}

// Get 发送 GET 请求
func (h *httpClient) Get(url string) (map[string]interface{}, error) {
	return h.Request("GET", url, nil, nil)
}

// Post 发送 POST 请求
func (h *httpClient) Post(url string, body interface{}) (map[string]interface{}, error) {
	return h.Request("POST", url, body, nil)
}

// PostJSON 发送 JSON POST 请求
func (h *httpClient) PostJSON(url string, body interface{}) (map[string]interface{}, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	return h.Request("POST", url, body, headers)
}

// Request 发送 HTTP 请求
func (h *httpClient) Request(method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	if !h.config.AllowNetwork {
		return nil, fmt.Errorf("network access is not allowed")
	}

	// 检查域名白名单
	if len(h.config.AllowedDomains) > 0 {
		allowed := false
		for _, domain := range h.config.AllowedDomains {
			if strings.Contains(url, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("domain not in whitelist")
		}
	}

	timeout := time.Duration(h.config.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
		case []byte:
			reqBody = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			reqBody = bytes.NewReader(jsonBody)
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok && body != nil {
		headers["Content-Type"] = "application/json"
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(respBody),
		"ok":          resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	// 尝试解析 JSON
	var jsonBody interface{}
	if err := json.Unmarshal(respBody, &jsonBody); err == nil {
		result["json"] = jsonBody
	}

	return result, nil
}

// Download 下载文件
func (h *httpClient) Download(url string, savePath string) error {
	if !h.config.AllowNetwork || !h.config.AllowFileWrite {
		return fmt.Errorf("operation not allowed")
	}

	timeout := time.Duration(h.config.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// 确保目录存在
	dir := filepath.Dir(savePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// === shell 命令执行对象 ===

type shellExec struct {
	config *Config
	logger *slog.Logger
	ctx    context.Context
}

// Exec 执行命令
func (s *shellExec) Exec(command string) (map[string]interface{}, error) {
	return s.ExecWithTimeout(command, s.config.ExecTimeout)
}

// ExecWithTimeout 执行命令（带超时）
func (s *shellExec) ExecWithTimeout(command string, timeoutSeconds int) (map[string]interface{}, error) {
	if !s.config.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = s.config.ExecTimeout
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// 确定使用的 shell
	shell := "sh"
	shellArgs := []string{"-c", command}
	if isWindows() {
		shell = "cmd"
		shellArgs = []string{"/c", command}
	}

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = s.config.Workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// ExecInDir 在指定目录执行命令
func (s *shellExec) ExecInDir(command string, workDir string) (map[string]interface{}, error) {
	if !s.config.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.Duration(s.config.ExecTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := "sh"
	shellArgs := []string{"-c", command}
	if isWindows() {
		shell = "cmd"
		shellArgs = []string{"/c", command}
	}

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// Pipe 执行管道命令
func (s *shellExec) Pipe(commands []string) (map[string]interface{}, error) {
	if !s.config.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	// 简化实现：将命令用管道连接
	combinedCommand := strings.Join(commands, " | ")
	return s.Exec(combinedCommand)
}

// isWindows 检查是否是 Windows 系统
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// === utils 工具函数 ===

type utils struct{}

// Sleep 休眠
func (u *utils) Sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// Now 获取当前时间
func (u *utils) Now() string {
	return time.Now().Format(time.RFC3339)
}

// Timestamp 获取时间戳
func (u *utils) Timestamp() int64 {
	return time.Now().Unix()
}

// FormatTime 格式化时间
func (u *utils) FormatTime(timestamp int64, layout string) string {
	if layout == "" {
		layout = time.RFC3339
	}
	return time.Unix(timestamp, 0).Format(layout)
}

// ParseTime 解析时间
func (u *utils) ParseTime(timeStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// Env 获取环境变量
func (u *utils) Env(key string) string {
	return os.Getenv(key)
}

// EnvOr 获取环境变量或默认值
func (u *utils) EnvOr(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// Cwd 获取当前工作目录
func (u *utils) Cwd() string {
	cwd, _ := os.Getwd()
	return cwd
}

// Hostname 获取主机名
func (u *utils) Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// UUID 生成 UUID
func (u *utils) UUID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
