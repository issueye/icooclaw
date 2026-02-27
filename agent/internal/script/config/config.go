package config

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
