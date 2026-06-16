// 综合修复补丁 - 应用于缺失的功能
// 位置：internal/util/tool_helpers.go (新文件)

package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NormalizeToolResult 确保工具结果是字符串格式
// 用于修复非字符串 tool response 问题 (17be6442, a1487b09)
func NormalizeToolResult(result interface{}) string {
	if result == nil {
		return ""
	}
	
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		// 复杂类型，序列化为 JSON
		data, err := json.Marshal(v)
		if err != nil {
			// 失败时使用 fmt
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}

// NormalizeFunctionCallOutput 确保 function call 输出是有效 JSON
// 用于修复非 JSON 工具输出问题 (346b6630, 0bcae68c, 72c7ef76)
func NormalizeFunctionCallOutput(output string) string {
	output = strings.TrimSpace(output)
	
	if output == "" {
		return `""`
	}
	
	// 如果已经是有效 JSON，直接返回
	if json.Valid([]byte(output)) {
		return output
	}
	
	// 尝试解析为 JSON（可能只是格式问题）
	var parsed interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err == nil {
		// 成功解析，重新编码
		if data, err := json.Marshal(parsed); err == nil {
			return string(data)
		}
	}
	
	// 不是 JSON，包装为 JSON 字符串
	escaped, _ := json.Marshal(output)
	return string(escaped)
}

// ValidateToolCall 验证工具调用的完整性
// 用于修复空 function name 问题 (c1caa454)
func ValidateToolCall(toolCall map[string]interface{}) error {
	if toolCall == nil {
		return fmt.Errorf("nil tool call")
	}
	
	// 检查 function 字段
	function, ok := toolCall["function"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid function field")
	}
	
	// 检查 function name
	name, ok := function["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("empty or missing function name")
	}
	
	return nil
}

// FilterValidToolCalls 过滤掉无效的工具调用
func FilterValidToolCalls(toolCalls []map[string]interface{}) []map[string]interface{} {
	valid := make([]map[string]interface{}, 0, len(toolCalls))
	
	for _, tc := range toolCalls {
		if err := ValidateToolCall(tc); err != nil {
			// 记录但跳过无效的工具调用
			continue
		}
		valid = append(valid, tc)
	}
	
	return valid
}

// ShortenClaudeCodexToolID 缩短过长的 tool call ID
// 用于修复 Claude Codex tool call ID 过长问题 (8bc2eff5)
func ShortenClaudeCodexToolID(id string) string {
	// Claude 的 tool_use.id 限制约 64 字符
	maxLen := 64
	
	if len(id) <= maxLen {
		return id
	}
	
	// 保留前缀和后缀，中间用哈希
	prefix := id[:20]
	suffix := id[len(id)-20:]
	
	return fmt.Sprintf("%s_%s", prefix, suffix)
}
