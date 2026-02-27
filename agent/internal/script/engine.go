package script

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/icooclaw/icooclaw/internal/script/builtons"
	"github.com/icooclaw/icooclaw/internal/script/config"
)

// Engine 脚本引擎
type Engine struct {
	vm      *goja.Runtime
	cfg     *config.Config
	logger  *slog.Logger
	context context.Context
}

// NewEngine 创建新的脚本引擎
func NewEngine(cfg *config.Config, logger *slog.Logger) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	vm := goja.New()
	vm.SetMaxCallStackSize(100)
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	engine := &Engine{
		vm:      vm,
		cfg:     cfg,
		logger:  logger,
		context: context.Background(),
	}

	engine.setupBuiltins()
	return engine
}

// NewEngineWithContext 创建带上下文的脚本引擎
func NewEngineWithContext(ctx context.Context, cfg *config.Config, logger *slog.Logger) *Engine {
	engine := NewEngine(cfg, logger)
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
	consoleObj := builtons.NewConsole(e.logger)
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
	fsObj := builtons.NewFileSystem(e.cfg, e.logger)
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
	httpObj := builtons.NewHTTPClient(e.cfg, e.logger)
	e.SetGlobal("http", map[string]interface{}{
		"get":      httpObj.Get,
		"post":     httpObj.Post,
		"postJSON": httpObj.PostJSON,
		"request":  httpObj.Request,
		"download": httpObj.Download,
	})

	// shell 命令执行对象
	shellObj := builtons.NewShellExec(e.context, e.cfg, e.logger)
	e.SetGlobal("shell", map[string]interface{}{
		"exec":            shellObj.Exec,
		"execWithTimeout": shellObj.ExecWithTimeout,
		"execInDir":       shellObj.ExecInDir,
		"pipe":            shellObj.Pipe,
	})

	// utils 工具函数对象
	utilsObj := builtons.NewUtils()
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

	// crypto 加密库
	e.setupCrypto()
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
		"encodeURL": func(s string) string {
			return base64.URLEncoding.EncodeToString([]byte(s))
		},
		"decodeURL": func(s string) (string, error) {
			b, err := base64.URLEncoding.DecodeString(s)
			return string(b), err
		},
	})
}

// setupCrypto 设置加密库
func (e *Engine) setupCrypto() {
	cryptoObj := &crypto{}
	e.SetGlobal("crypto", map[string]interface{}{
		// HMAC
		"hmacSHA1":   cryptoObj.HmacSHA1,
		"hmacSHA256": cryptoObj.HmacSHA256,
		"hmacMD5":    cryptoObj.HmacMD5,

		// 哈希
		"sha1":   cryptoObj.SHA1,
		"sha256": cryptoObj.SHA256,
		"md5":    cryptoObj.MD5,

		// AES
		"aesEncrypt": cryptoObj.AESEncrypt,
		"aesDecrypt": cryptoObj.AESDecrypt,

		// Base64
		"base64Encode": cryptoObj.Base64Encode,
		"base64Decode": cryptoObj.Base64Decode,

		// Hex
		"hexEncode": cryptoObj.HexEncode,
		"hexDecode": cryptoObj.HexDecode,
	})
}

// === crypto 加密对象 ===

type crypto struct{}

// HmacSHA1 生成 HMAC-SHA1
func (c *crypto) HmacSHA1(data, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HmacSHA256 生成 HMAC-SHA256
func (c *crypto) HmacSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HmacMD5 生成 HMAC-MD5
func (c *crypto) HmacMD5(data, key string) string {
	h := hmac.New(md5.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// SHA1 生成 SHA1 哈希
func (c *crypto) SHA1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// SHA256 生成 SHA256 哈希
func (c *crypto) SHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// MD5 生成 MD5 哈希
func (c *crypto) MD5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// AESEncrypt AES 加密
// key 可以是 16, 24, 或 32 字节，分别对应 AES-128, AES-192, AES-256
func (c *crypto) AESEncrypt(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// PKCS7 填充
	blockSize := block.BlockSize()
	plaintextBytes := []byte(plaintext)
	padding := blockSize - len(plaintextBytes)%blockSize
	for i := 0; i < padding; i++ {
		plaintextBytes = append(plaintextBytes, byte(padding))
	}

	// CBC 模式
	ciphertext := make([]byte, len(plaintextBytes))
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintextBytes)

	// 将 IV 放在前面
	result := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// AESDecrypt AES 解密
func (c *crypto) AESDecrypt(ciphertextBase64, key string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 提取 IV
	blockSize := block.BlockSize()
	if len(ciphertext) < blockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]

	// CBC 模式
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除 PKCS7 填充
	padding := int(ciphertext[len(ciphertext)-1])
	if padding > blockSize || padding == 0 {
		return "", fmt.Errorf("invalid padding")
	}
	ciphertext = ciphertext[:len(ciphertext)-padding]

	return string(ciphertext), nil
}

// Base64Encode Base64 编码
func (c *crypto) Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// Base64Decode Base64 解码
func (c *crypto) Base64Decode(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// HexEncode Hex 编码
func (c *crypto) HexEncode(data string) string {
	return hex.EncodeToString([]byte(data))
}

// HexDecode Hex 解码
func (c *crypto) HexDecode(encoded string) (string, error) {
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// resolvePath 解析路径
func (e *Engine) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.cfg.Workspace, path)
}
