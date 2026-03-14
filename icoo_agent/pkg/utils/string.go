package utils

// SplitProviderModel 分割模型字符串，格式为 "provider/model"。
func SplitProviderModel(modelStr string) []string {
	idx := -1
	for i := 0; i < len(modelStr); i++ {
		if modelStr[i] == '/' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	return []string{modelStr[:idx], modelStr[idx+1:]}
}
