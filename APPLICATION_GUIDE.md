# 修复应用指南

## 已创建的新文件

1. **internal/util/tool_helpers.go** - 工具调用辅助函数
   - `NormalizeToolResult` - 规范化工具结果类型
   - `NormalizeFunctionCallOutput` - 确保 function call 输出是有效 JSON
   - `ValidateToolCall` - 验证工具调用完整性
   - `FilterValidToolCalls` - 过滤无效工具调用
   - `ShortenClaudeCodexToolID` - 缩短过长的 tool ID

2. **internal/runtime/executor/helps/stream_helpers.go** - 流式响应辅助函数
   - `ShouldSkipStreamChunk` - 判断是否跳过空 chunk
   - `HasUsageMetadata` - 检查是否有 usage 信息
   - `HasSubstantialContent` - 检查是否有实质内容
   - `ExtractSSEDataPayload` - 提取多行 SSE 数据
   - `ValidateAndRepairSSEChunk` - 验证并修复 SSE chunk

---

## 如何在 Executor 中应用

### 示例：在 gemini_executor.go 中应用空 chunk 过滤

```go
// 在 ExecuteStream 方法中，scanner 循环部分：

import (
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
)

for scanner.Scan() {
	line := scanner.Bytes()
	
	// 提取 SSE data payload
	payload := helps.ExtractSSEDataPayload(line)
	
	// 应用修复：检查是否应该跳过
	if skip, reason := helps.ShouldSkipStreamChunk(payload); skip {
		log.Debugf("Skipping chunk: %s", reason)
		continue
	}
	
	// 原有的翻译逻辑
	lines := sdktranslator.TranslateStream(ctx, to, from, req.Model, opts.OriginalRequest, body, bytes.Clone(payload), &param)
	
	for i := range lines {
		select {
		case out <- cliproxyexecutor.StreamChunk{Payload: lines[i]}:
		case <-ctx.Done():
			return
		}
	}
}
```

---

## 如何在 Translator 中应用

### 示例 1：在 Claude → OpenAI translator 中应用工具结果规范化

```go
// 在处理 tool_result 时：

import (
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
)

func processToolResult(content interface{}) string {
	// 应用修复：规范化所有类型
	return util.NormalizeToolResult(content)
}

// 在转换 function call output 时：
func processFunctionCallOutput(output string) string {
	// 应用修复：确保是有效 JSON
	return util.NormalizeFunctionCallOutput(output)
}
```

### 示例 2：在 OpenAI → Claude translator 中应用工具调用验证

```go
import (
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
)

func translateToolCalls(toolCalls []map[string]interface{}) []map[string]interface{} {
	// 应用修复：过滤无效的工具调用
	return util.FilterValidToolCalls(toolCalls)
}
```

### 示例 3：在 Codex → Claude translator 中应用 tool ID 缩短

```go
import (
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
)

func processToolUseBlock(block map[string]interface{}) map[string]interface{} {
	if id, ok := block["id"].(string); ok {
		// 应用修复：缩短过长的 ID
		block["id"] = util.ShortenClaudeCodexToolID(id)
		// 同时应用 Claude ID sanitization
		block["id"] = util.SanitizeClaudeToolID(block["id"].(string))
	}
	return block
}
```

---

## 已有的修复（无需额外应用）

### ✅ SSE Event Framing
- 位置：`sdk/api/handlers/openai/openai_responses_handlers.go`
- 已实现：`responsesSSEFramer` 结构
- 功能完整：事件边界保护、分片处理、output item 修复

### ✅ Tool Name Sanitization
- 位置：`internal/util/util.go`
- 已实现：`SanitizeFunctionName`
- 应用位置：所有 Gemini translators

### ✅ Claude Tool ID Sanitization
- 位置：`internal/util/claude_tool_id.go`
- 已实现：`SanitizeClaudeToolID`
- 符合 Claude API 正则：`^[a-zA-Z0-9_-]+$`

### ✅ JSON Schema Cleanup
- 位置：`internal/util/gemini_schema.go`
- 已实现：`CleanJSONSchemaForGemini`
- 功能：移除 `strict`, `uniqueItems`, `defer_loading` 等字段

### ✅ Goroutine Leak Prevention
- 位置：所有 streaming executors
- 已实现：`select { case out <- ...: case <-ctx.Done(): return }`
- 覆盖：9/9 executors

---

## 建议的应用优先级

### 优先级 1：空 Chunk 处理（影响最大）
```bash
# 应用到所有 streaming executors
- internal/runtime/executor/gemini_executor.go
- internal/runtime/executor/gemini_cli_executor.go
- internal/runtime/executor/gemini_vertex_executor.go
- internal/runtime/executor/claude_executor.go
- internal/runtime/executor/kimi_executor.go
- internal/runtime/executor/openai_compat_executor.go
- internal/runtime/executor/codex_executor.go
```

### 优先级 2：工具结果规范化
```bash
# 应用到工具调用相关的 translators
- internal/translator/claude/openai/*
- internal/translator/openai/claude/*
- internal/translator/gemini/claude/*
- internal/translator/codex/claude/*
```

### 优先级 3：Function Call Output 规范化
```bash
# 应用到 function call 输出处理
- internal/translator/*/openai/chat-completions/*
- internal/translator/*/gemini/*
```

---

## 验证步骤

### 1. 编译测试
```bash
go build -o cli-proxy-api ./cmd/server
```

### 2. 单元测试
```bash
# 测试新的辅助函数
go test ./internal/util -run TestNormalizeToolResult
go test ./internal/runtime/executor/helps -run TestShouldSkipStreamChunk

# 测试 translator 修复
go test ./internal/translator/... -run Tool
```

### 3. 集成测试
```bash
# 启动服务
docker compose up -d --build

# 测试流式响应
curl -X POST http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'

# 测试工具调用
curl -X POST http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "What is the weather?"}],
    "tools": [...]
  }'
```

### 4. 监控日志
```bash
# 检查是否有跳过的 chunk（应该有调试日志）
docker compose logs -f | grep "Skipping chunk"

# 检查是否有工具调用验证失败
docker compose logs -f | grep "invalid tool call"
```

---

## 回滚方案

如果应用修复后出现问题：

```bash
# 1. 切回 self 分支
git checkout self

# 2. 或者只回滚特定文件
git checkout self -- internal/util/tool_helpers.go
git checkout self -- internal/runtime/executor/helps/stream_helpers.go

# 3. 重新构建
docker compose build
docker compose restart
```

---

## 预期改进

### 修复前
- ❌ 偶尔出现 JSON 解析错误（空 chunk）
- ❌ 某些工具调用失败（空 function name）
- ❌ 非 JSON 工具输出导致崩溃

### 修复后
- ✅ 优雅跳过空和无效 chunk
- ✅ 自动过滤无效工具调用
- ✅ 所有工具输出自动规范化为有效 JSON
- ✅ 更好的日志记录便于调试

---

## 下一步

1. **渐进式应用** - 先在一个 executor 中测试
2. **监控日志** - 观察是否有新的跳过或验证失败
3. **逐步推广** - 确认无问题后应用到其他 executors
4. **性能测试** - 确保没有引入性能问题

需要我生成具体某个文件的完整修复补丁吗？
