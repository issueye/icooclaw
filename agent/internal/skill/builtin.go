package skill

func GetBuiltInSkills() []Skill {
	return []Skill{
		newIdentitySkill(),
		newFileSkill(),
		newShellSkill(),
		newWebSkill(),
	}
}

func newIdentitySkill() Skill {
	return Skill{
		Name:        "identity",
		Description: "身份设定技能",
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
		AlwaysLoad: true,
		Enabled:    true,
	}
}

func newFileSkill() Skill {
	return Skill{
		Name:        "file",
		Description: "文件操作技能",
		Content: `## 文件操作技能
你可以帮助用户读取、写入和管理文件。
- 读取文件：使用file_read工具
- 写入文件：使用file_write工具
- 列出目录：使用file_list工具`,
		AlwaysLoad: true,
		Enabled:    true,
	}
}

func newShellSkill() Skill {
	return Skill{
		Name:        "shell",
		Description: "Shell命令执行技能",
		Content: `## Shell命令执行技能
你可以帮助用户执行Shell命令。
- 执行命令：使用shell_exec工具`,
		AlwaysLoad: true,
		Enabled:    true,
	}
}

func newWebSkill() Skill {
	return Skill{
		Name:        "web",
		Description: "网页搜索和抓取技能",
		Content: `## 网页搜索和抓取技能
你可以帮助用户搜索互联网和抓取网页内容。
- 搜索网页：使用web_search工具
- 抓取网页：使用web_fetch工具`,
		AlwaysLoad: false,
		Enabled:    true,
	}
}
