// 空 Chunk 处理辅助函数
// 位置：internal/runtime/executor/helps/stream_helpers.go (新文件)

package helps

import (
	"bytes"
	"encoding/json"
)

// ShouldSkipStreamChunk 判断是否应该跳过某个流式响应 chunk
// 用于修复空 JSON chunk 问题 (1061354b, c29931e0, b05cfd9f, 8ce22b84)
func ShouldSkipStreamChunk(chunk []byte) (skip bool, reason string) {
	// 1. 跳过空 chunk
	trimmed := bytes.TrimSpace(chunk)
	if len(trimmed) == 0 {
		return true, "empty chunk"
	}
	
	// 2. [DONE] 标记不跳过
	if bytes.Equal(trimmed, []byte("[DONE]")) {
		return false, ""
	}
	
	// 3. 验证 JSON（如果不是 [DONE]）
	if !json.Valid(trimmed) {
		return true, "invalid JSON"
	}
	
	// 4. 检查是否是 usage metadata chunk（即使内容为空也保留）
	if HasUsageMetadata(trimmed) {
		return false, "has usage metadata"
	}
	
	// 5. 检查是否有实质内容
	if !HasSubstantialContent(trimmed) {
		return true, "no substantial content"
	}
	
	return false, ""
}

// HasUsageMetadata 检查 chunk 是否包含 usage 信息
func HasUsageMetadata(chunk []byte) bool {
	// 简单检查是否包含 "usage" 字段
	return bytes.Contains(chunk, []byte(`"usage"`))
}

// HasSubstantialContent 检查 chunk 是否有实质内容
func HasSubstantialContent(chunk []byte) bool {
	// 解析 JSON 检查是否有非空字段
	var data map[string]interface{}
	if err := json.Unmarshal(chunk, &data); err != nil {
		return false
	}
	
	// 空对象 {}
	if len(data) == 0 {
		return false
	}
	
	// 只有 type 字段且为空
	if len(data) == 1 {
		if t, ok := data["type"].(string); ok && t == "" {
			return false
		}
	}
	
	return true
}

// ExtractSSEDataPayload 从 SSE 格式中提取 data payload
// 支持多行 SSE 数据 (80eb0370)
func ExtractSSEDataPayload(sseData []byte) []byte {
	lines := bytes.Split(sseData, []byte("\n"))
	var payload bytes.Buffer
	
	inData := false
	for _, line := range lines {
		line = bytes.TrimRight(line, "\r")
		
		if bytes.HasPrefix(line, []byte("data:")) {
			// 开始 data 部分
			data := bytes.TrimSpace(line[5:]) // 去掉 "data:"
			if payload.Len() > 0 {
				payload.WriteByte('\n')
			}
			payload.Write(data)
			inData = true
		} else if inData && !bytes.HasPrefix(line, []byte("event:")) && !bytes.HasPrefix(line, []byte("id:")) {
			// 继续多行 data
			if len(bytes.TrimSpace(line)) > 0 {
				payload.WriteByte('\n')
				payload.Write(line)
			}
		} else if bytes.HasPrefix(line, []byte("event:")) {
			// 遇到新 event，结束当前 data
			inData = false
		}
	}
	
	return payload.Bytes()
}

// ValidateAndRepairSSEChunk 验证并修复 SSE chunk
func ValidateAndRepairSSEChunk(chunk []byte) ([]byte, error) {
	// 提取 payload
	payload := ExtractSSEDataPayload(chunk)
	
	// 跳过检查
	if skip, _ := ShouldSkipStreamChunk(payload); skip {
		return nil, nil // 返回 nil 表示跳过
	}
	
	return payload, nil
}
