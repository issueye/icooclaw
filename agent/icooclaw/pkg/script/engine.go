// Package script provides JavaScript scripting engine for icooclaw.
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
)

// Config contains script engine configuration.
type Config struct {
	// Workspace is the base directory for file operations.
	Workspace string
	// AllowFileRead enables file reading.
	AllowFileRead bool
	// AllowFileWrite enables file writing.
	AllowFileWrite bool
	// AllowFileDelete enables file deletion.
	AllowFileDelete bool
	// AllowExec enables shell command execution.
	AllowExec bool
	// AllowNetwork enables HTTP requests.
	AllowNetwork bool
	// ExecTimeout is the timeout for shell commands in seconds.
	ExecTimeout int
	// HTTPTimeout is the timeout for HTTP requests in seconds.
	HTTPTimeout int
	// MaxMemory is the maximum memory in bytes.
	MaxMemory int64
	// AllowedDomains is the whitelist for network requests.
	AllowedDomains []string
}

// DefaultConfig returns the default configuration.
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
	}
}

// Engine is a JavaScript script engine.
type Engine struct {
	vm       *goja.Runtime
	cfg      *Config
	logger   *slog.Logger
	ctx      context.Context
	builtins []Builtin
}

// Builtin is the interface for builtin objects.
type Builtin interface {
	Name() string
	Object() map[string]any
}

// NewEngine creates a new script engine.
func NewEngine(cfg *Config, logger *slog.Logger) *Engine {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	vm := goja.New()
	vm.SetMaxCallStackSize(100)
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	engine := &Engine{
		vm:       vm,
		cfg:      cfg,
		logger:   logger,
		ctx:      context.Background(),
		builtins: []Builtin{},
	}

	engine.setupBuiltins()
	return engine
}

// NewEngineWithContext creates a script engine with context.
func NewEngineWithContext(ctx context.Context, cfg *Config, logger *slog.Logger) *Engine {
	engine := NewEngine(cfg, logger)
	engine.ctx = ctx
	return engine
}

// Run executes a script string.
func (e *Engine) Run(script string) (goja.Value, error) {
	return e.vm.RunString(script)
}

// RunFile executes a script file.
func (e *Engine) RunFile(path string) (goja.Value, error) {
	absPath := e.resolvePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}
	return e.vm.RunString(string(content))
}

// SetGlobal sets a global variable.
func (e *Engine) SetGlobal(name string, value any) error {
	return e.vm.Set(name, value)
}

// GetGlobal gets a global variable.
func (e *Engine) GetGlobal(name string) goja.Value {
	return e.vm.Get(name)
}

// VM returns the underlying VM.
func (e *Engine) VM() *goja.Runtime {
	return e.vm
}

// Call calls a function by name.
func (e *Engine) Call(name string, args ...any) (goja.Value, error) {
	fn := e.vm.Get(name)
	if fn == nil || goja.IsUndefined(fn) {
		return nil, fmt.Errorf("function '%s' not found", name)
	}

	callable, ok := goja.AssertFunction(fn)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a function", name)
	}

	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = e.vm.ToValue(arg)
	}

	return callable(nil, jsArgs...)
}

// SetContext sets the context.
func (e *Engine) SetContext(ctx context.Context) {
	e.ctx = ctx
}

// RegisterBuiltin registers a builtin object.
func (e *Engine) RegisterBuiltin(b Builtin) {
	e.builtins = append(e.builtins, b)
	e.SetGlobal(b.Name(), b.Object())
}

// setupBuiltins sets up builtin objects and methods.
func (e *Engine) setupBuiltins() {
	// Register default builtins
	e.RegisterBuiltin(NewConsole(e.logger))
	e.RegisterBuiltin(NewFileSystem(e.cfg, e.logger))
	e.RegisterBuiltin(NewHTTPClient(e.cfg, e.logger))
	e.RegisterBuiltin(NewShellExec(e.ctx, e.cfg, e.logger))
	e.RegisterBuiltin(NewUtils())

	// Standard library extensions
	e.setupStdLib()

	// Crypto library
	e.setupCrypto()
}

// setupStdLib sets up standard library extensions.
func (e *Engine) setupStdLib() {
	// JSON extensions
	e.vm.Set("JSON", map[string]any{
		"stringify": func(v any) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"parse": func(s string) (any, error) {
			var v any
			err := json.Unmarshal([]byte(s), &v)
			return v, err
		},
		"pretty": func(v any) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
	})

	// Base64 encoding/decoding
	e.vm.Set("Base64", map[string]any{
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

// setupCrypto sets up crypto library.
func (e *Engine) setupCrypto() {
	c := &crypto{}
	e.SetGlobal("crypto", map[string]any{
		// HMAC
		"hmacSHA1":   c.HmacSHA1,
		"hmacSHA256": c.HmacSHA256,
		"hmacMD5":    c.HmacMD5,

		// Hash
		"sha1":   c.SHA1,
		"sha256": c.SHA256,
		"md5":    c.MD5,

		// AES
		"aesEncrypt": c.AESEncrypt,
		"aesDecrypt": c.AESDecrypt,

		// Base64
		"base64Encode": c.Base64Encode,
		"base64Decode": c.Base64Decode,

		// Hex
		"hexEncode": c.HexEncode,
		"hexDecode": c.HexDecode,
	})
}

// resolvePath resolves a path relative to workspace.
func (e *Engine) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.cfg.Workspace, path)
}

// crypto provides crypto functions.
type crypto struct{}

func (c *crypto) HmacSHA1(data, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) HmacSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) HmacMD5(data, key string) string {
	h := hmac.New(md5.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) SHA1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) SHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) MD5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *crypto) AESEncrypt(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	plaintextBytes := []byte(plaintext)
	padding := blockSize - len(plaintextBytes)%blockSize
	for i := 0; i < padding; i++ {
		plaintextBytes = append(plaintextBytes, byte(padding))
	}

	ciphertext := make([]byte, len(plaintextBytes))
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintextBytes)

	result := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

func (c *crypto) AESDecrypt(ciphertextBase64, key string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	if len(ciphertext) < blockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	padding := int(ciphertext[len(ciphertext)-1])
	if padding > blockSize || padding == 0 {
		return "", fmt.Errorf("invalid padding")
	}
	ciphertext = ciphertext[:len(ciphertext)-padding]

	return string(ciphertext), nil
}

func (c *crypto) Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func (c *crypto) Base64Decode(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (c *crypto) HexEncode(data string) string {
	return hex.EncodeToString([]byte(data))
}

func (c *crypto) HexDecode(encoded string) (string, error) {
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}