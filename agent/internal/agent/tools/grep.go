package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool Grep搜索工具
type GrepTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewGrepTool 创建Grep搜索工具
func NewGrepTool(config *FileToolConfig) *GrepTool {
	tool := NewBaseTool(
		"grep",
		"在文件中搜索匹配的文本。类似于 Unix grep 命令，支持正则表达式、忽略大小写、行号显示等功能。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "要搜索的模式（支持正则表达式）",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "搜索路径（文件或目录）",
				},
				"glob": map[string]interface{}{
					"type":        "string",
					"description": "文件匹配模式，如 *.go, *.txt",
				},
				"ignore_case": map[string]interface{}{
					"type":        "boolean",
					"description": "是否忽略大小写",
					"default":     false,
				},
				"line_number": map[string]interface{}{
					"type":        "boolean",
					"description": "是否显示行号",
					"default":     true,
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归搜索子目录",
					"default":     true,
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "最大结果数量",
					"default":     100,
				},
				"context": map[string]interface{}{
					"type":        "number",
					"description": "显示匹配行的上下文行数",
					"default":     0,
				},
			},
			"required": []string{"pattern", "path"},
		},
		nil,
	)

	return &GrepTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *GrepTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *GrepTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *GrepTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行搜索
func (t *GrepTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("invalid or missing pattern")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	// 可选参数
	glob := ""
	if g, ok := params["glob"].(string); ok {
		glob = g
	}

	ignoreCase := false
	if ic, ok := params["ignore_case"].(bool); ok {
		ignoreCase = ic
	}

	showLineNumber := true
	if ln, ok := params["line_number"].(bool); ok {
		showLineNumber = ln
	}

	recursive := true
	if r, ok := params["recursive"].(bool); ok {
		recursive = r
	}

	maxResults := 100
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	contextLines := 0
	if c, ok := params["context"].(float64); ok {
		contextLines = int(c)
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 编译正则表达式
	var regex *regexp.Regexp
	var err error
	if ignoreCase {
		regex, err = regexp.Compile("(?i)" + pattern)
	} else {
		regex, err = regexp.Compile(pattern)
	}
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	// 搜索
	var matches []map[string]interface{}
	var filesSearched int

	err = t.searchFile(regex, absPath, glob, recursive, showLineNumber, contextLines, maxResults, &matches, &filesSearched)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"pattern":        pattern,
		"path":           absPath,
		"matches":        matches,
		"match_count":    len(matches),
		"files_searched": filesSearched,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// searchFile 搜索文件
func (t *GrepTool) searchFile(
	regex *regexp.Regexp,
	path string,
	glob string,
	recursive bool,
	showLineNumber bool,
	contextLines int,
	maxResults int,
	matches *[]map[string]interface{},
	filesSearched *int,
) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// 目录搜索
		return t.searchDirectory(regex, path, glob, recursive, showLineNumber, contextLines, maxResults, matches, filesSearched)
	}

	// 单文件搜索
	*filesSearched++
	fileMatches, err := t.searchSingleFile(regex, path, showLineNumber, contextLines, maxResults, len(*matches))
	if err != nil {
		return err
	}
	*matches = append(*matches, fileMatches...)

	return nil
}

// searchDirectory 搜索目录
func (t *GrepTool) searchDirectory(
	regex *regexp.Regexp,
	dirPath string,
	glob string,
	recursive bool,
	showLineNumber bool,
	contextLines int,
	maxResults int,
	matches *[]map[string]interface{},
	filesSearched *int,
) error {
	// 获取文件列表
	var files []string

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// 过滤目录
			if strings.HasPrefix(filepath.Base(path), ".") {
				return nil
			}
			// 应用 glob 过滤
			if glob != "" {
				matched, err := filepath.Match(glob, filepath.Base(path))
				if err != nil {
					return err
				}
				if !matched {
					return nil
				}
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dirPath, entry.Name())
			// 应用 glob 过滤
			if glob != "" {
				matched, err := filepath.Match(glob, entry.Name())
				if err != nil {
					return err
				}
				if !matched {
					continue
				}
			}
			files = append(files, path)
		}
	}

	// 搜索每个文件
	for _, file := range files {
		if len(*matches) >= maxResults {
			break
		}

		*filesSearched++
		relPath, _ := filepath.Rel(dirPath, file)
		fileMatches, err := t.searchSingleFile(regex, file, showLineNumber, contextLines, maxResults-len(*matches), len(*matches))
		if err != nil {
			continue
		}

		for _, match := range fileMatches {
			match["file"] = relPath
			*matches = append(*matches, match)
		}
	}

	return nil
}

// searchSingleFile 搜索单个文件
func (t *GrepTool) searchSingleFile(
	regex *regexp.Regexp,
	filePath string,
	showLineNumber bool,
	contextLines int,
	maxResults int,
	currentCount int,
) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []map[string]interface{}
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		loc := regex.FindStringIndex(line)
		if loc == nil {
			continue
		}

		match := map[string]interface{}{
			"line":     line,
			"line_num": lineNumber,
		}

		// 获取匹配的子字符串
		matchedStr := line[loc[0]:loc[1]]
		match["match"] = matchedStr
		match["column"] = loc[0] + 1

		if !showLineNumber {
			delete(match, "line_num")
		}

		matches = append(matches, match)

		if len(matches) >= maxResults {
			break
		}
	}

	return matches, scanner.Err()
}

// ToDefinition 转换为工具定义
func (t *GrepTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// resolvePath 解析路径
func (t *GrepTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *GrepTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}

// FindTool 文件查找工具
type FindTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFindTool 创建文件查找工具
func NewFindTool(config *FileToolConfig) *FindTool {
	tool := NewBaseTool(
		"find",
		"查找文件和目录。类似于 Unix find 命令，支持按名称、类型、大小、时间等条件查找。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "搜索路径",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "文件名匹配模式（支持 * 和 ? 通配符）",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "文件类型：f(文件), d(目录)",
				},
				"max_depth": map[string]interface{}{
					"type":        "number",
					"description": "最大搜索深度",
					"default":     10,
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "最大结果数量",
					"default":     100,
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &FindTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FindTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FindTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FindTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行查找
func (t *FindTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	name := ""
	if n, ok := params["name"].(string); ok {
		name = n
	}

	fileType := ""
	if ft, ok := params["type"].(string); ok {
		fileType = ft
	}

	maxDepth := 10
	if md, ok := params["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	maxResults := 100
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 查找
	var results []map[string]interface{}
	var searchErr error

	t.search(absPath, name, fileType, maxDepth, 0, maxResults, &results, &searchErr)
	if searchErr != nil {
		return "", searchErr
	}

	// 构建结果
	result := map[string]interface{}{
		"path":    absPath,
		"name":    name,
		"type":    fileType,
		"results": results,
		"count":   len(results),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// search 递归搜索
func (t *FindTool) search(
	path string,
	name string,
	fileType string,
	maxDepth int,
	currentDepth int,
	maxResults int,
	results *[]map[string]interface{},
	err *error,
) {
	if currentDepth > maxDepth || len(*results) >= maxResults {
		return
	}

	entries, readErr := os.ReadDir(path)
	if readErr != nil {
		*err = readErr
		return
	}

	for _, entry := range entries {
		if len(*results) >= maxResults {
			break
		}

		entryPath := filepath.Join(path, entry.Name())

		// 名称过滤
		if name != "" {
			matched, matchErr := filepath.Match(name, entry.Name())
			if matchErr != nil {
				*err = matchErr
				return
			}
			if !matched {
				if entry.IsDir() {
					t.search(entryPath, name, fileType, maxDepth, currentDepth+1, maxResults, results, err)
				}
				continue
			}
		}

		// 类型过滤
		if fileType != "" {
			if fileType == "f" && !entry.IsDir() {
				info, statErr := os.Stat(entryPath)
				if statErr == nil {
					*results = append(*results, map[string]interface{}{
						"name": entry.Name(),
						"path": entryPath,
						"type": "file",
						"size": info.Size(),
					})
				}
			} else if fileType == "d" && entry.IsDir() {
				*results = append(*results, map[string]interface{}{
					"name": entry.Name(),
					"path": entryPath,
					"type": "directory",
				})
			}
		} else {
			info, statErr := os.Stat(entryPath)
			if statErr == nil {
				result := map[string]interface{}{
					"name": entry.Name(),
					"path": entryPath,
				}
				if info.IsDir() {
					result["type"] = "directory"
				} else {
					result["type"] = "file"
					result["size"] = info.Size()
				}
				*results = append(*results, result)
			}
		}

		// 递归子目录
		if entry.IsDir() && currentDepth < maxDepth {
			// 跳过隐藏目录
			if !strings.HasPrefix(entry.Name(), ".") {
				t.search(entryPath, name, fileType, maxDepth, currentDepth+1, maxResults, results, err)
			}
		}
	}
}

// ToDefinition 转换为工具定义
func (t *FindTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// resolvePath 解析路径
func (t *FindTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FindTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}

// TreeTool 目录树工具
type TreeTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewTreeTool 创建目录树工具
func NewTreeTool(config *FileToolConfig) *TreeTool {
	tool := NewBaseTool(
		"tree",
		"显示目录树结构。类似于 Unix tree 命令，以树形结构显示目录和文件。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
				"max_depth": map[string]interface{}{
					"type":        "number",
					"description": "最大显示深度",
					"default":     3,
				},
				"include_files": map[string]interface{}{
					"type":        "boolean",
					"description": "是否显示文件",
					"default":     true,
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &TreeTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *TreeTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *TreeTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *TreeTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 显示目录树
func (t *TreeTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	maxDepth := 3
	if md, ok := params["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	includeFiles := true
	if ifv, ok := params["include_files"].(bool); ok {
		includeFiles = ifv
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 检查路径是否为目录
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	// 生成树结构
	tree := t.buildTree(absPath, 0, maxDepth, includeFiles)

	// 构建结果
	result := map[string]interface{}{
		"path": absPath,
		"tree": tree,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// buildTree 递归构建树
func (t *TreeTool) buildTree(path string, currentDepth int, maxDepth int, includeFiles bool) []interface{} {
	if currentDepth > maxDepth {
		return nil
	}

	var items []interface{}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		// 跳过隐藏文件/目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// 目录
			children := t.buildTree(entryPath, currentDepth+1, maxDepth, includeFiles)
			item := map[string]interface{}{
				"name":     entry.Name(),
				"type":     "directory",
				"children": children,
			}
			items = append(items, item)
		} else if includeFiles {
			// 文件
			info, _ := os.Stat(entryPath)
			var size int64
			if info != nil {
				size = info.Size()
			}
			item := map[string]interface{}{
				"name": entry.Name(),
				"type": "file",
				"size": size,
			}
			items = append(items, item)
		}
	}

	return items
}

// ToDefinition 转换为工具定义
func (t *TreeTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// resolvePath 解析路径
func (t *TreeTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *TreeTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}

// ReadPartTool 部分读取文件工具
type ReadPartTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewReadPartTool 创建部分读取文件工具
func NewReadPartTool(config *FileToolConfig) *ReadPartTool {
	tool := NewBaseTool(
		"read_part",
		"部分读取文件内容。支持指定起始行、结束行，类似于 Unix sed 和 head/tail 命令的功能。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"start_line": map[string]interface{}{
					"type":        "number",
					"description": "起始行号（1-based）",
					"default":     1,
				},
				"end_line": map[string]interface{}{
					"type":        "number",
					"description": "结束行号",
				},
				"lines": map[string]interface{}{
					"type":        "number",
					"description": "读取的行数（从 start_line 开始）",
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &ReadPartTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *ReadPartTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *ReadPartTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *ReadPartTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 部分读取文件
func (t *ReadPartTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedRead {
		return "", fmt.Errorf("file read is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	startLine := 1
	if sl, ok := params["start_line"].(float64); ok {
		startLine = int(sl)
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 打开文件
	file, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 读取行
	var lines []string
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		if lineNumber < startLine {
			continue
		}

		// 检查结束行
		if el, ok := params["end_line"].(float64); ok && lineNumber > int(el) {
			break
		}

		// 检查行数限制
		if l, ok := params["lines"].(float64); ok && lineNumber-startLine+1 > int(l) {
			break
		}

		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"path":        absPath,
		"start_line":  startLine,
		"end_line":    startLine + len(lines) - 1,
		"total_lines": len(lines),
		"content":     strings.Join(lines, "\n"),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *ReadPartTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// resolvePath 解析路径
func (t *ReadPartTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *ReadPartTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}

// LineCountTool 行数统计工具
type LineCountTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewLineCountTool 创建行数统计工具
func NewLineCountTool(config *FileToolConfig) *LineCountTool {
	tool := NewBaseTool(
		"wc",
		"统计文件的行数、单词数、字符数。类似于 Unix wc 命令。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"count_lines": map[string]interface{}{
					"type":        "boolean",
					"description": "统计行数",
					"default":     true,
				},
				"count_words": map[string]interface{}{
					"type":        "boolean",
					"description": "统计单词数",
					"default":     true,
				},
				"count_chars": map[string]interface{}{
					"type":        "boolean",
					"description": "统计字符数",
					"default":     true,
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &LineCountTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *LineCountTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *LineCountTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *LineCountTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 统计行数
func (t *LineCountTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedRead {
		return "", fmt.Errorf("file read is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	countLines := true
	if cl, ok := params["count_lines"].(bool); ok {
		countLines = cl
	}

	countWords := true
	if cw, ok := params["count_words"].(bool); ok {
		countWords = cw
	}

	countChars := true
	if cc, ok := params["count_chars"].(bool); ok {
		countChars = cc
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 读取文件
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 统计
	result := map[string]interface{}{
		"path":  absPath,
		"bytes": len(content),
	}

	if countLines {
		lines := strings.Split(string(content), "\n")
		result["lines"] = len(lines)
	}

	if countWords {
		words := strings.Fields(string(content))
		result["words"] = len(words)
	}

	if countChars {
		result["chars"] = len(string(content))
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *LineCountTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// resolvePath 解析路径
func (t *LineCountTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *LineCountTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}
