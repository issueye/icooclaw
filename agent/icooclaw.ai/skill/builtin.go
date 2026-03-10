package skill

// GetBuiltInSkills 获取所有内置技能
// 返回系统预定义的技能列表
func GetBuiltInSkills() []*Skill {
	return []*Skill{
		newIdentitySkill(),
		newFileSkill(),
		newShellSkill(),
		newWebSkill(),
	}
}

// newIdentitySkill 创建身份设定技能
// 帮助用户设定 AI 的身份、名字以及用户的称呼
func newIdentitySkill() *Skill {
	return &Skill{
		ID:          "builtin_identity",
		Name:        "identity",
		Description: "身份设定技能 - 管理 AI 和用户的身份信息",
		Content: `## 身份设定技能

你可以帮助用户设定 AI 的身份、名字以及用户的称呼。

### 记忆文件位置
- SOUL.md: 存储 AI 身份和人格设定（workspace/SOUL.md）
- USER.md: 存储用户信息和称呼（workspace/USER.md）

### 设定 AI 身份/名字
当用户告诉你 AI 的名字时，你应该：
1. 使用 file_read 工具读取 SOUL.md 文件内容
2. 使用 file_write 工具更新 SOUL.md 文件的 "身份" 部分
3. 格式：设置我的名字为 [名字]

### 设定用户称呼
当用户告诉你希望如何称呼他时，你应该：
1. 使用 file_read 工具读取 USER.md 文件内容
2. 使用 file_write 工具更新 USER.md 文件的 "用户称呼" 部分
3. 格式：叫我 [称呼] 即可 / 请叫我 [称呼]

### 重要提示
- 如果用户还没有设置称呼，第一次对话时你应该主动询问用户希望如何称呼
- 如果 AI 还没有名字，你应该主动询问用户希望给你起什么名字
- 记住用户的称呼并在后续对话中使用
- 更新文件时保持原有内容不变，只修改对应部分`,
		Prompt: `你是一个身份管理助手。帮助用户设置 AI 的名字和用户的称呼。
当用户要求设置名字或称呼时，使用 file_read 和 file_write 工具操作 SOUL.md 和 USER.md 文件。`,
		Tools:       []string{"file_read", "file_write"},
		Tags:        []string{"identity", "settings", "memory"},
		Source:      "builtin",
		AlwaysLoad:  true,
		Enabled:     true,
		Version:     "1.0.0",
	}
}

// newFileSkill 创建文件操作技能
// 提供文件读写、目录管理等功能
func newFileSkill() *Skill {
	return &Skill{
		ID:          "builtin_file",
		Name:        "file",
		Description: "文件操作技能 - 读写文件、管理目录",
		Content: `## 文件操作技能

你可以帮助用户进行文件和目录操作。

### 支持的文件操作
- 读取文件内容
- 写入文件内容
- 创建目录
- 列出目录内容
- 删除文件或目录
- 搜索文件

### 使用规范
1. 读取文件前先确认文件存在
2. 写入文件时注意备份重要数据
3. 删除操作需要用户确认
4. 支持多种文件格式：文本、JSON、Markdown 等`,
		Prompt: `你是一个文件操作助手。帮助用户管理文件和目录。
使用 file_read、file_write、directory_list 等工具完成操作。`,
		Tools:       []string{"file_read", "file_write", "directory_list", "file_delete"},
		Tags:        []string{"file", "filesystem", "io"},
		Source:      "builtin",
		AlwaysLoad:  false,
		Enabled:     true,
		Version:     "1.0.0",
	}
}

// newShellSkill 创建 Shell 命令技能
// 提供命令执行、系统管理等功能
func newShellSkill() *Skill {
	return &Skill{
		ID:          "builtin_shell",
		Name:        "shell",
		Description: "Shell 命令技能 - 执行命令、管理系统",
		Content: `## Shell 命令技能

你可以帮助用户执行 Shell 命令和系统管理任务。

### 支持的命令类型
- 文件操作命令 (ls, cat, mkdir, rm 等)
- 系统信息命令 (uname, df, top 等)
- 开发工具命令 (git, npm, go 等)
- 网络命令 (curl, ping, netstat 等)

### 安全规范
1. 危险命令需要用户确认 (rm -rf, format 等)
2. 限制对系统关键目录的访问
3. 命令超时时间为 60 秒
4. 记录所有执行的命令`,
		Prompt: `你是一个 Shell 命令助手。帮助用户执行命令行操作。
使用 execute_command 工具执行命令，注意安全和权限控制。`,
		Tools:       []string{"execute_command"},
		Tags:        []string{"shell", "command", "system"},
		Source:      "builtin",
		AlwaysLoad:  false,
		Enabled:     true,
		Version:     "1.0.0",
	}
}

// newWebSkill 创建 Web 操作技能
// 提供网页访问、数据抓取等功能
func newWebSkill() *Skill {
	return &Skill{
		ID:          "builtin_web",
		Name:        "web",
		Description: "Web 操作技能 - 访问网页、抓取数据",
		Content: `## Web 操作技能

你可以帮助用户访问网页和获取网络数据。

### 支持的 Web 操作
- 访问网页内容
- 搜索信息
- 下载文件
- API 调用
- 数据抓取

### 使用规范
1. 遵守网站的 robots.txt 规则
2. 注意请求频率限制
3. 处理网络超时和错误
4. 支持多种数据格式：HTML、JSON、XML 等`,
		Prompt: `你是一个 Web 操作助手。帮助用户访问网络资源。
使用 web_search、fetch_url 等工具完成操作。`,
		Tools:       []string{"web_search", "fetch_url"},
		Tags:        []string{"web", "network", "http"},
		Source:      "builtin",
		AlwaysLoad:  false,
		Enabled:     true,
		Version:     "1.0.0",
	}
}
