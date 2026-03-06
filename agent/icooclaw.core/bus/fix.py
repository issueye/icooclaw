# -*- coding: utf-8 -*-
with open(r'E:\code\issueye\icooclaw\agent\icooclaw.core\bus\bus.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 找到结构体的位置
i = content.find('type InboundMessage struct {')
j = content.find('}', i)
old_struct = content[i:j+1]

# ID 行的确切内容（从文件中提取）
old_id_line = '\tID        string         `json:"id,omitempty"`        // 消息 ID'

# 新的 ID 行和 SessionID 行
new_id_line = '\tID        string         `json:"id,omitempty"`          // 消息 ID'
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
