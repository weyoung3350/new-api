# New-API Realtime WebSocket API 使用指南

## 🎯 概述

New-API现在支持两种音频API协议，完全兼容OpenAI的音频API规范：

1. **HTTP REST API** - 传统的单次音频处理
2. **WebSocket Realtime API** - 实时双向音频对话

## 📊 **双协议架构对比**

| 特性 | HTTP REST API | WebSocket Realtime API |
|------|---------------|----------------------|
| **协议** | HTTP/HTTPS | WebSocket |
| **端点** | `/v1/audio/speech` | `/v1/realtime` |
| **交互方式** | 请求-响应 | 双向实时流 |
| **延迟** | 较高 | 极低延迟 |
| **用途** | 单次音频处理 | 实时对话 |
| **音频流** | 完整文件 | 流式处理 |
| **语音检测** | 不支持 | 支持VAD |
| **函数调用** | 不支持 | 支持 |

## 🔧 **HTTP REST API（传统模式）**

### 端点
```
POST /v1/audio/speech
```

### 请求示例
```bash
curl -X POST "http://localhost:3000/v1/audio/speech" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "cosyvoice-v2",
    "input": "你好，欢迎使用New-API的语音合成功能！",
    "voice": "longyingcui",
    "response_format": "mp3",
    "speed": 1.0
  }'
```

### 响应
- 返回二进制音频数据
- Content-Type: `audio/mpeg`

## 🎙️ **WebSocket Realtime API（实时模式）**

### 连接端点
```
WebSocket: ws://localhost:3000/v1/realtime?model=cosyvoice-v2
```

### 连接示例
```javascript
const ws = new WebSocket('ws://localhost:3000/v1/realtime?model=cosyvoice-v2', ['realtime']);
```

### 认证
WebSocket连接需要通过中间件进行Token认证。

## 📡 **事件系统**

Realtime API基于事件驱动模式，支持以下事件类型：

### 客户端事件（Client Events）

#### 1. 会话管理
```javascript
// 更新会话配置
{
  "type": "session.update",
  "session": {
    "instructions": "你是一个友好的AI助手",
    "voice": "longyingcui",
    "modalities": ["text", "audio"],
    "turn_detection": {
      "type": "server_vad",
      "threshold": 0.5,
      "prefix_padding_ms": 300,
      "silence_duration_ms": 200,
      "create_response": true
    }
  }
}
```

#### 2. 音频缓冲区管理
```javascript
// 追加音频数据
{
  "type": "input_audio_buffer.append",
  "audio": "base64_encoded_audio_data"
}

// 提交音频缓冲区
{
  "type": "input_audio_buffer.commit"
}

// 清空音频缓冲区
{
  "type": "input_audio_buffer.clear"
}
```

#### 3. 对话管理
```javascript
// 创建对话项目
{
  "type": "conversation.item.create",
  "item": {
    "type": "message",
    "role": "user",
    "content": [
      {
        "type": "input_text",
        "text": "你好，请介绍一下你自己。"
      }
    ]
  }
}
```

#### 4. 响应控制
```javascript
// 创建响应
{
  "type": "response.create",
  "response": {
    "modalities": ["text", "audio"],
    "instructions": "请用自然、友好的语调回应。"
  }
}

// 取消响应
{
  "type": "response.cancel"
}
```

### 服务端事件（Server Events）

#### 1. 会话事件
- `session.created` - 会话创建
- `session.updated` - 会话更新
- `conversation.created` - 对话创建

#### 2. 音频缓冲区事件
- `input_audio_buffer.committed` - 音频缓冲区已提交
- `input_audio_buffer.cleared` - 音频缓冲区已清空
- `input_audio_buffer.speech_started` - 检测到语音开始
- `input_audio_buffer.speech_stopped` - 检测到语音结束

#### 3. 对话事件
- `conversation.item.created` - 对话项目已创建
- `conversation.item.truncated` - 对话项目已截断
- `conversation.item.deleted` - 对话项目已删除

#### 4. 响应事件
- `response.created` - 响应创建
- `response.done` - 响应完成
- `response.output_item.added` - 输出项目添加
- `response.output_item.done` - 输出项目完成
- `response.content_part.added` - 内容部分添加
- `response.content_part.done` - 内容部分完成
- `response.audio_transcript.delta` - 音频转录增量
- `response.audio_transcript.done` - 音频转录完成
- `response.audio.delta` - 音频数据增量
- `response.audio.done` - 音频数据完成

#### 5. 错误事件
- `error` - 错误事件

## 🎵 **音频格式支持**

### 输入音频格式
- **PCM16** (推荐): 24kHz, 16-bit, 单声道
- **Base64编码**: 用于WebSocket传输

### 输出音频格式
- **MP3**: 默认格式，兼容性好
- **WAV**: 无损格式
- **OPUS**: 高压缩比
- **AAC**: 高质量压缩
- **FLAC**: 无损压缩

### 音频配置示例
```javascript
{
  "input_audio_format": "pcm16",
  "output_audio_format": "mp3"
}
```

## 🔊 **声音支持**

支持40+种CosyVoice官方音色，包括：

### 语音助手类
- `longyingcui` - 龙英翠（知性女声）
- `longxiaochun_v2` - 龙小淳（积极女声）
- `longxiaoxia_v2` - 龙小夏（权威女声）

### 有声书类
- `longsanshu` - 龙三叔（沉稳男声）
- `longmiao_v2` - 龙妙（抑扬顿挫女声）
- `longyue_v2` - 龙悦（温暖磁性女声）

### OpenAI兼容映射
- `alloy` → `longxiaochun_v2`
- `echo` → `longnan_v2`
- `fable` → `longmiao_v2`
- `onyx` → `longsanshu`
- `nova` → `longyue_v2`
- `shimmer` → `longyuan_v2`

## 🛠️ **语音活动检测（VAD）**

### 配置选项
```javascript
{
  "turn_detection": {
    "type": "server_vad",        // 服务端VAD
    "threshold": 0.5,            // 检测阈值
    "prefix_padding_ms": 300,    // 前缀填充时间
    "silence_duration_ms": 200,  // 静音持续时间
    "create_response": true      // 自动创建响应
  }
}
```

### VAD类型
- `server_vad` - 服务端语音活动检测
- `none` - 禁用VAD，手动控制

## 🧪 **测试和调试**

### 1. 使用测试客户端
```bash
# 在浏览器中打开
open test/test_realtime_websocket.html
```

### 2. 命令行测试
```bash
# 测试HTTP REST API
curl -X POST "http://localhost:3000/v1/audio/speech" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"cosyvoice-v2","input":"测试文本","voice":"longyingcui"}' \
  --output test_audio.mp3
```

### 3. WebSocket连接测试
```javascript
const ws = new WebSocket('ws://localhost:3000/v1/realtime?model=cosyvoice-v2');

ws.onopen = () => {
  console.log('连接已建立');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到事件:', data.type);
};
```

## 🚀 **使用场景**

### HTTP REST API适用场景
- 单次文本转语音
- 批量音频生成
- 简单的TTS应用
- 文档朗读

### WebSocket Realtime API适用场景
- 实时语音对话
- 语音助手应用
- 客服机器人
- 实时翻译
- 语音游戏
- 教育互动应用

## 🔧 **集成示例**

### 简单TTS应用
```javascript
// HTTP方式
async function textToSpeech(text) {
  const response = await fetch('/v1/audio/speech', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model: 'cosyvoice-v2',
      input: text,
      voice: 'longyingcui'
    })
  });
  
  const audioBlob = await response.blob();
  const audio = new Audio(URL.createObjectURL(audioBlob));
  audio.play();
}
```

### 实时对话应用
```javascript
// WebSocket方式
class RealtimeChat {
  constructor() {
    this.ws = new WebSocket('ws://localhost:3000/v1/realtime?model=cosyvoice-v2');
    this.setupEventHandlers();
  }
  
  setupEventHandlers() {
    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'response.audio.delta') {
        this.playAudioDelta(data.delta);
      }
    };
  }
  
  sendText(text) {
    this.ws.send(JSON.stringify({
      type: 'conversation.item.create',
      item: {
        type: 'message',
        role: 'user',
        content: [{ type: 'input_text', text }]
      }
    }));
    
    this.ws.send(JSON.stringify({
      type: 'response.create'
    }));
  }
}
```

## 📋 **配置和部署**

### 服务器配置
```yaml
# docker-compose.yml
version: '3'
services:
  new-api:
    image: new-api:latest
    ports:
      - "3000:3000"
    environment:
      - REALTIME_ENABLED=true
      - COSYVOICE_ENDPOINT=wss://dashscope.aliyuncs.com/api-ws/v1/inference
```

### 环境变量
- `REALTIME_ENABLED` - 启用Realtime API
- `COSYVOICE_ENDPOINT` - CosyVoice WebSocket端点
- `MAX_WEBSOCKET_CONNECTIONS` - 最大WebSocket连接数

## 🛡️ **安全考虑**

### 认证和授权
- 使用Token认证
- 支持用户级别权限控制
- WebSocket连接自动过期（30分钟）

### 资源限制
- 连接数限制
- 音频缓冲区大小限制
- 会话时长限制

## 📈 **性能优化**

### 延迟优化
- WebSocket连接池
- 音频流式传输
- 服务端VAD减少网络传输

### 资源管理
- 自动清理过期会话
- 音频缓冲区管理
- 连接状态监控

## 🔍 **故障排查**

### 常见问题

1. **WebSocket连接失败**
   - 检查Token是否有效
   - 确认服务器端口开放
   - 验证模型参数

2. **音频质量问题**
   - 检查音频格式配置
   - 验证采样率设置
   - 确认声音参数

3. **延迟过高**
   - 使用WebSocket Realtime API
   - 启用服务端VAD
   - 优化网络连接

### 调试方法
- 查看WebSocket事件日志
- 监控音频缓冲区状态
- 检查服务器资源使用

## 📚 **更多资源**

- [OpenAI Realtime API文档](https://platform.openai.com/docs/guides/realtime)
- [CosyVoice官方文档](https://help.aliyun.com/zh/model-studio/cosyvoice-websocket-api)
- [WebSocket API规范](https://tools.ietf.org/html/rfc6455)

---

## 🎉 **总结**

New-API的双协议支持为不同应用场景提供了灵活的选择：

- **HTTP REST API** - 简单、可靠、适合批量处理
- **WebSocket Realtime API** - 实时、低延迟、适合交互应用

通过这种设计，开发者可以根据具体需求选择最适合的API模式，同时保持与OpenAI API的完全兼容性。 