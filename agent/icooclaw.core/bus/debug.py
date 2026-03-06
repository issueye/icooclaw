# -*- coding: utf-8 -*-
with open(r'E:\code\issueye\icooclaw\agent\icooclaw.core\bus\bus.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 找到结构体的位置
i = content.find('type InboundMessage struct {')
j = content.find('}', i)
old_struct = content[i:j+1]

print("原结构体:")
print(repr(old_struct))
print()

# 查找 ID 行
for line in old_struct.split('\n'):
    if 'ID' in line and 'SessionID' not in line and 'ChatID' not in line and 'UserID' not in line:
        print("ID 行:")
        print(repr(line))
        break
