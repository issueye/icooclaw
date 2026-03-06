# -*- coding: utf-8 -*-
with open(r'E:\code\issueye\icooclaw\agent\icooclaw.core\bus\bus.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 找到结构体的位置
i = content.find('type InboundMessage struct {')
j = content.find('}', i)
old_struct = content[i:j+1]

# 从文件中提取 ID 行
for line in old_struct.split('\n'):
    if line.startswith('\tID') and 'SessionID' not in line:
        old_id_line = line
        break

# 构建新的 ID 行（调整空格对齐）
new_id_line = old_id_line + '  '  # 添加两个空格以对齐
session_line = '\tSessionID uint           `json:"session_id,omitempty"`  // 会话 ID'

# 构建新的结构体
lines = old_struct.split('\n')
new_lines = []
for line in lines:
    if line == old_id_line:
        new_lines.append(new_id_line)
        new_lines.append(session_line)
    else:
        new_lines.append(line)

new_struct = '\n'.join(new_lines)

# 替换原内容
new_content = content.replace(old_struct, new_struct)

with open(r'E:\code\issueye\icooclaw\agent\icooclaw.core\bus\bus.go', 'w', encoding='utf-8') as f:
    f.write(new_content)

print("修改成功！")
print("新结构体:")
print(new_struct)
