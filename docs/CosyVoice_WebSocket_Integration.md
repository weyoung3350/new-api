# CosyVoice WebSocket API 集成文档

## 概述

本文档描述了阿里云CosyVoice语音合成模型的WebSocket API集成实现。该实现提供了智能路由功能，能够根据文本特性自动选择最优的API调用方式，同时支持SSML标记和长文本的高效处理。

## 功能特性

### 🚀 智能路由系统
- **自动API选择**: 根据文本长度和特性自动选择HTTP或WebSocket API
- **SSML强制路由**: 包含SSML标记的文本自动使用WebSocket API
- **长文本优化**: 超过500字符的文本使用WebSocket API获得更好性能

### 🎯 WebSocket核心功能
- **实时连接管理**: 自动建立、维护和关闭WebSocket连接
- **流式音频处理**: 支持实时音频数据接收和缓冲
- **会话生命周期管理**: 完整的任务状态跟踪和错误处理
- **连接池优化**: 高效的连接复用和资源管理

### 📝 SSML支持
- **标记检测**: 自动识别SSML标记并启用相应处理
- **语音控制**: 支持语速、音调、音量、停顿等控制
- **多声音支持**: 支持在单个文本中使用多种声音

## 技术架构

### 核心组件

```
CosyVoice WebSocket 集成
├── WebSocket管理器 (CosyVoiceWSManager)
│   ├── 连接池管理
│   ├── 消息监听
│   └── 会话状态管理
├── 智能路由器 (SmartAudioHandler)
│   ├── 文本分析
│   ├── SSML检测
│   └── API选择逻辑
└── 流式处理器 (HandleCosyVoiceWebSocketTTS)
    ├── 音频数据收集
    ├── 实时传输
    └── 错误恢复
```

### 文件结构

```
relay/channel/ali/
├── websocket.go        # WebSocket核心实现
├── text.go            # 智能路由和转换函数
├── dto.go             # 数据结构定义
└── adaptor.go         # 适配器集成
```

## 使用方法

### 基础TTS调用

```bash
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "cosyvoice-v2",
    "input": "你好，这是一个语音合成测试。",
    "voice": "longyingcui",
    "response_format": "mp3",
    "speed": 1.0
  }'
```

### 长文本处理（自动使用WebSocket）

```bash
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "cosyvoice-v2",
    "input": "这是一段很长的文本，包含超过500个字符的内容...",
    "voice": "longyingcui",
    "response_format": "mp3"
  }'
```

### SSML标记支持（强制使用WebSocket）

```bash
curl -X POST http://localhost:3000/v1/audio/speech \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "cosyvoice-v2",
    "input": "<speak><voice name=\"longyingcui\">欢迎使用<emphasis level=\"strong\">CosyVoice</emphasis>！<break time=\"500ms\"/>这是一个<prosody rate=\"slow\">慢速</prosody>示例。</voice></speak>",
    "voice": "longyingcui",
    "response_format": "mp3"
  }'
```

## API参数说明

### 请求参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称，固定为 "cosyvoice-v2" |
| input | string | 是 | 要合成的文本内容，支持SSML标记 |
| voice | string | 否 | 声音类型，默认 "longyingcui" |
| response_format | string | 否 | 音频格式：mp3/wav/opus/aac/flac，默认 mp3 |
| speed | float | 否 | 语速，范围 0.25-4.0，默认 1.0 |

### 支持的声音

| 声音名称 | 描述 | 语言 |
|----------|------|------|
| longyingcui | 龙英翠（女声） | 中文 |
| zhifeng_emo | 智峰（情感男声） | 中文 |
| alloy | 合金（通用） | 多语言 |

## 智能路由规则

### 自动选择逻辑

```go
func ShouldUseWebSocketAPI(text string, enableSSML bool) bool {
    textLength := len([]rune(text))
    
    // 如果启用SSML，必须使用WebSocket
    if enableSSML {
        return true
    }
    
    // 长文本使用WebSocket（超过500字符）
    if textLength > 500 {
        return true
    }
    
    // 短文本使用HTTP API
    return false
}
```

### 路由决策表

| 文本特征 | API选择 | 原因 |
|----------|---------|------|
| 包含SSML标记 | WebSocket | SSML仅WebSocket支持 |
| 文本 > 500字符 | WebSocket | 长文本性能优化 |
| 文本 ≤ 500字符 | HTTP | 短文本快速响应 |

## WebSocket交互流程

### 连接建立

```
1. 客户端请求 → 智能路由分析
2. 创建WebSocket连接 → 阿里云端点
3. 发送认证头 → Bearer Token
4. 连接成功 → 开始任务流程
```

### 任务执行

```
1. 发送 run-task 指令 → 初始化任务
2. 接收 task-started 事件 → 任务开始确认
3. 发送 continue-task 指令 → 提交文本内容
4. 接收 result-generated 事件 → 音频数据流
5. 发送 finish-task 指令 → 完成任务
6. 接收 task-finished 事件 → 任务结束
```

### 错误处理

```
- 连接超时 → 自动重试
- 认证失败 → 返回错误信息
- 任务失败 → task-failed 事件处理
- 网络中断 → 连接重建
```

## 性能优化

### 连接管理

- **连接池**: 复用WebSocket连接减少建立开销
- **超时控制**: 读写操作设置合理超时时间
- **资源清理**: 自动清理无效连接和会话

### 音频处理

- **流式缓冲**: 实时接收和缓存音频数据
- **内存优化**: 高效的音频数据管理
- **并发处理**: 支持多任务并行执行

### 错误恢复

- **重连机制**: 自动检测和重建连接
- **状态恢复**: 会话状态的持久化和恢复
- **优雅降级**: 失败时的备用处理方案

## 监控和日志

### 关键指标

- WebSocket连接数量和状态
- 任务执行时间和成功率
- 音频数据传输量和速度
- 错误频率和类型统计

### 日志记录

```go
// 连接建立
common.LogInfo(c, fmt.Sprintf("WebSocket连接成功建立，任务ID: %s", taskId))

// 消息处理
common.LogInfo(c, fmt.Sprintf("收到WebSocket事件: %s, 任务ID: %s", event, taskId))

// 错误处理
common.LogError(c, fmt.Sprintf("WebSocket连接失败: %v", err))
```

## 故障排查

### 常见问题

1. **连接失败**
   - 检查API Key有效性
   - 验证网络连接
   - 确认阿里云服务状态

2. **音频质量问题**
   - 检查文本编码格式
   - 验证SSML标记语法
   - 确认声音参数设置

3. **性能问题**
   - 监控连接池使用情况
   - 检查文本长度和复杂度
   - 分析网络延迟

### 调试模式

启用调试模式查看详细日志：

```go
common.DebugEnabled = true
```

## 测试指南

### 运行测试

```bash
cd test/
python test_cosyvoice_websocket.py
```

### 测试场景

1. **短文本TTS**: 验证HTTP API路由
2. **长文本TTS**: 验证WebSocket API使用
3. **SSML标记**: 验证SSML处理功能
4. **多声音测试**: 验证不同声音支持
5. **错误处理**: 验证异常情况处理

## 配置选项

### 环境变量

```bash
# 阿里云API密钥
DASHSCOPE_API_KEY=your_api_key_here

# WebSocket超时设置（秒）
COSYVOICE_WS_TIMEOUT=30

# 音频缓冲区大小
COSYVOICE_AUDIO_BUFFER_SIZE=100
```

### 高级配置

```go
// WebSocket连接配置
dialer := &websocket.Dialer{
    HandshakeTimeout: 45 * time.Second,
    ReadBufferSize:   1024 * 4,
    WriteBufferSize:  1024 * 4,
}

// 智能路由阈值
const LONG_TEXT_THRESHOLD = 500 // 字符数
```

## 版本历史

### v1.0.0 (2025-01-02)
- ✅ 实现WebSocket连接管理
- ✅ 添加智能路由功能
- ✅ 支持SSML标记处理
- ✅ 完成流式音频处理
- ✅ 集成错误处理机制

## 贡献指南

### 开发环境

1. 确保Go版本 >= 1.19
2. 安装依赖: `go mod tidy`
3. 运行测试: `go test ./...`

### 代码规范

- 遵循Go官方代码规范
- 添加详细的中文注释
- 编写对应的单元测试
- 更新相关文档

## 许可证

本项目遵循原项目的开源许可证。 