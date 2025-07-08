package controller

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"one-api/common"
	"one-api/model"
	"one-api/relay/channel/ali"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket升级器配置
var realtimeUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 在生产环境中应该进行适当的来源检查
	},
	Subprotocols: []string{"realtime"},
}

// Realtime会话管理器
type RealtimeSession struct {
	ID                string               `json:"id"`
	Object            string               `json:"object"`
	Model             string               `json:"model"`
	ExpiresAt         int64                `json:"expires_at"`
	Modalities        []string             `json:"modalities"`
	Instructions      string               `json:"instructions"`
	Voice             string               `json:"voice"`
	InputAudioFormat  string               `json:"input_audio_format"`
	OutputAudioFormat string               `json:"output_audio_format"`
	TurnDetection     *TurnDetectionConfig `json:"turn_detection"`
	Tools             []RealtimeTool       `json:"tools"`
	Temperature       float64              `json:"temperature"`
	MaxTokens         string               `json:"max_response_output_tokens"`
	ToolChoice        string               `json:"tool_choice"`
}

// 语音活动检测配置
type TurnDetectionConfig struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
	CreateResponse    bool    `json:"create_response,omitempty"`
}

// Realtime工具定义
type RealtimeTool struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Realtime WebSocket连接管理器
type RealtimeConnection struct {
	conn          *websocket.Conn
	session       *RealtimeSession
	conversation  *RealtimeConversation
	audioBuffer   []byte
	audioBufferMu sync.Mutex
	cosyClient    *ali.CosyVoiceWSManager
	userID        int
	tokenID       int
	channel       *model.Channel
	isActive      bool
	mu            sync.RWMutex
}

// Realtime对话管理
type RealtimeConversation struct {
	ID    string             `json:"id"`
	Items []ConversationItem `json:"items"`
	mu    sync.RWMutex
}

// 对话项目
type ConversationItem struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	Type      string        `json:"type"`
	Status    string        `json:"status"`
	Role      string        `json:"role,omitempty"`
	Content   []ContentPart `json:"content,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	Output    string        `json:"output,omitempty"`
}

// 内容部分
type ContentPart struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	Audio      string `json:"audio,omitempty"`
	Transcript string `json:"transcript,omitempty"`
}

// Realtime事件基础结构
type RealtimeEvent struct {
	Type    string      `json:"type"`
	EventID string      `json:"event_id,omitempty"`
	Data    interface{} `json:",inline"`
}

// 客户端事件类型
type ClientEvent struct {
	Type string `json:"type"`
}

// 服务端事件类型
type ServerEvent struct {
	Type    string `json:"type"`
	EventID string `json:"event_id"`
}

// RealtimeHandler 处理WebSocket连接的Realtime API
func RealtimeHandler(c *gin.Context) {
	// 验证用户认证
	userID := c.GetInt("id")
	tokenID := c.GetInt("token_id")

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 获取模型参数
	model := c.Query("model")
	if model == "" {
		model = "cosyvoice-v2" // 默认模型
	}

	// 验证模型支持
	if !isRealtimeModelSupported(model) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的实时模型"})
		return
	}

	// 升级为WebSocket连接
	conn, err := realtimeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		common.SysError("WebSocket升级失败: " + err.Error())
		return
	}
	defer conn.Close()

	// 创建Realtime连接管理器
	realtimeConn := &RealtimeConnection{
		conn:     conn,
		userID:   userID,
		tokenID:  tokenID,
		isActive: true,
	}

	// 初始化会话
	if err := realtimeConn.initializeSession(model); err != nil {
		common.SysError("初始化Realtime会话失败: " + err.Error())
		return
	}

	// 开始处理WebSocket消息
	realtimeConn.handleConnection()
}

// 初始化Realtime会话
func (rc *RealtimeConnection) initializeSession(model string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// 创建会话ID
	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	conversationID := fmt.Sprintf("conv_%d", time.Now().UnixNano())

	// 初始化会话配置
	rc.session = &RealtimeSession{
		ID:                sessionID,
		Object:            "realtime.session",
		Model:             model,
		ExpiresAt:         time.Now().Add(30 * time.Minute).Unix(), // 30分钟过期
		Modalities:        []string{"text", "audio"},
		Instructions:      "你是一个有用的AI助手。请用自然、友好的语调回应用户。",
		Voice:             "longyingcui", // 默认声音
		InputAudioFormat:  "pcm16",
		OutputAudioFormat: "pcm16",
		TurnDetection: &TurnDetectionConfig{
			Type:              "server_vad",
			Threshold:         0.5,
			PrefixPaddingMs:   300,
			SilenceDurationMs: 200,
			CreateResponse:    true,
		},
		Tools:       []RealtimeTool{},
		Temperature: 0.8,
		MaxTokens:   "inf",
		ToolChoice:  "auto",
	}

	// 初始化对话
	rc.conversation = &RealtimeConversation{
		ID:    conversationID,
		Items: []ConversationItem{},
	}

	// 发送session.created事件
	return rc.sendServerEvent("session.created", map[string]interface{}{
		"session": rc.session,
	})
}

// 处理WebSocket连接
func (rc *RealtimeConnection) handleConnection() {
	defer func() {
		rc.mu.Lock()
		rc.isActive = false
		rc.mu.Unlock()

		if rc.cosyClient != nil {
			// TODO: 实现CosyVoice客户端关闭方法
			// rc.cosyClient.Close()
		}
	}()

	// 发送conversation.created事件
	rc.sendServerEvent("conversation.created", map[string]interface{}{
		"conversation": map[string]string{
			"id":     rc.conversation.ID,
			"object": "realtime.conversation",
		},
	})

	// 消息处理循环
	for {
		rc.mu.RLock()
		if !rc.isActive {
			rc.mu.RUnlock()
			break
		}
		rc.mu.RUnlock()

		// 读取客户端消息
		var clientEvent map[string]interface{}
		err := rc.conn.ReadJSON(&clientEvent)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理客户端事件
		if err := rc.handleClientEvent(clientEvent); err != nil {
			common.SysError(fmt.Sprintf("处理客户端事件失败: %v", err))
			rc.sendErrorEvent("invalid_event", err.Error(), "")
		}
	}
}

// 处理客户端事件
func (rc *RealtimeConnection) handleClientEvent(event map[string]interface{}) error {
	eventType, ok := event["type"].(string)
	if !ok {
		return fmt.Errorf("无效的事件类型")
	}

	switch eventType {
	case "session.update":
		return rc.handleSessionUpdate(event)
	case "input_audio_buffer.append":
		return rc.handleAudioBufferAppend(event)
	case "input_audio_buffer.commit":
		return rc.handleAudioBufferCommit(event)
	case "input_audio_buffer.clear":
		return rc.handleAudioBufferClear(event)
	case "conversation.item.create":
		return rc.handleConversationItemCreate(event)
	case "conversation.item.truncate":
		return rc.handleConversationItemTruncate(event)
	case "conversation.item.delete":
		return rc.handleConversationItemDelete(event)
	case "response.create":
		return rc.handleResponseCreate(event)
	case "response.cancel":
		return rc.handleResponseCancel(event)
	default:
		return fmt.Errorf("不支持的事件类型: %s", eventType)
	}
}

// 处理会话更新
func (rc *RealtimeConnection) handleSessionUpdate(event map[string]interface{}) error {
	sessionData, ok := event["session"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的会话数据")
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// 更新会话配置
	if instructions, ok := sessionData["instructions"].(string); ok {
		rc.session.Instructions = instructions
	}
	if voice, ok := sessionData["voice"].(string); ok {
		rc.session.Voice = voice
	}
	if modalities, ok := sessionData["modalities"].([]interface{}); ok {
		rc.session.Modalities = make([]string, len(modalities))
		for i, m := range modalities {
			if s, ok := m.(string); ok {
				rc.session.Modalities[i] = s
			}
		}
	}
	if turnDetection, ok := sessionData["turn_detection"].(map[string]interface{}); ok {
		rc.updateTurnDetection(turnDetection)
	}
	if tools, ok := sessionData["tools"].([]interface{}); ok {
		rc.updateTools(tools)
	}

	// 发送session.updated事件
	return rc.sendServerEvent("session.updated", map[string]interface{}{
		"session": rc.session,
	})
}

// 更新语音活动检测配置
func (rc *RealtimeConnection) updateTurnDetection(data map[string]interface{}) {
	if rc.session.TurnDetection == nil {
		rc.session.TurnDetection = &TurnDetectionConfig{}
	}

	if typ, ok := data["type"].(string); ok {
		rc.session.TurnDetection.Type = typ
	}
	if threshold, ok := data["threshold"].(float64); ok {
		rc.session.TurnDetection.Threshold = threshold
	}
	if padding, ok := data["prefix_padding_ms"].(float64); ok {
		rc.session.TurnDetection.PrefixPaddingMs = int(padding)
	}
	if silence, ok := data["silence_duration_ms"].(float64); ok {
		rc.session.TurnDetection.SilenceDurationMs = int(silence)
	}
	if create, ok := data["create_response"].(bool); ok {
		rc.session.TurnDetection.CreateResponse = create
	}
}

// 更新工具配置
func (rc *RealtimeConnection) updateTools(tools []interface{}) {
	rc.session.Tools = make([]RealtimeTool, 0, len(tools))
	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			rt := RealtimeTool{}
			if typ, ok := toolMap["type"].(string); ok {
				rt.Type = typ
			}
			if name, ok := toolMap["name"].(string); ok {
				rt.Name = name
			}
			if desc, ok := toolMap["description"].(string); ok {
				rt.Description = desc
			}
			if params, ok := toolMap["parameters"]; ok {
				rt.Parameters = params
			}
			rc.session.Tools = append(rc.session.Tools, rt)
		}
	}
}

// 处理音频缓冲区追加
func (rc *RealtimeConnection) handleAudioBufferAppend(event map[string]interface{}) error {
	audioBase64, ok := event["audio"].(string)
	if !ok {
		return fmt.Errorf("无效的音频数据")
	}

	// 解码base64音频数据并添加到缓冲区
	rc.audioBufferMu.Lock()
	defer rc.audioBufferMu.Unlock()

	// 解码base64音频数据
	audioData, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return fmt.Errorf("音频数据解码失败: %v", err)
	}

	// 添加到音频缓冲区
	rc.audioBuffer = append(rc.audioBuffer, audioData...)

	return nil
}

// 处理音频缓冲区提交
func (rc *RealtimeConnection) handleAudioBufferCommit(event map[string]interface{}) error {
	rc.audioBufferMu.Lock()
	defer rc.audioBufferMu.Unlock()

	if len(rc.audioBuffer) == 0 {
		return nil
	}

	// 创建用户消息项目
	itemID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	item := ConversationItem{
		ID:     itemID,
		Object: "realtime.item",
		Type:   "message",
		Status: "in_progress",
		Role:   "user",
		Content: []ContentPart{
			{
				Type:  "input_audio",
				Audio: "", // 这里应该是base64编码的音频
			},
		},
	}

	// 添加到对话
	rc.conversation.mu.Lock()
	rc.conversation.Items = append(rc.conversation.Items, item)
	rc.conversation.mu.Unlock()

	// 发送事件
	rc.sendServerEvent("input_audio_buffer.committed", map[string]interface{}{
		"item_id": itemID,
	})

	rc.sendServerEvent("conversation.item.created", map[string]interface{}{
		"item": item,
	})

	// 如果是Paraformer模型，启动语音识别处理
	if rc.session.Model == "paraformer-realtime-8k-v2" {
		go rc.processParaformerTranscription(itemID)
	}

	// 清空音频缓冲区
	rc.audioBuffer = rc.audioBuffer[:0]

	return nil
}

// 处理音频缓冲区清空
func (rc *RealtimeConnection) handleAudioBufferClear(event map[string]interface{}) error {
	rc.audioBufferMu.Lock()
	rc.audioBuffer = rc.audioBuffer[:0]
	rc.audioBufferMu.Unlock()

	return rc.sendServerEvent("input_audio_buffer.cleared", map[string]interface{}{})
}

// 处理对话项目创建
func (rc *RealtimeConnection) handleConversationItemCreate(event map[string]interface{}) error {
	itemData, ok := event["item"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的项目数据")
	}

	// 解析项目数据
	item := ConversationItem{
		ID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Object: "realtime.item",
		Status: "completed",
	}

	if typ, ok := itemData["type"].(string); ok {
		item.Type = typ
	}
	if role, ok := itemData["role"].(string); ok {
		item.Role = role
	}

	// 添加到对话
	rc.conversation.mu.Lock()
	rc.conversation.Items = append(rc.conversation.Items, item)
	rc.conversation.mu.Unlock()

	return rc.sendServerEvent("conversation.item.created", map[string]interface{}{
		"item": item,
	})
}

// 处理对话项目截断
func (rc *RealtimeConnection) handleConversationItemTruncate(event map[string]interface{}) error {
	// TODO: 实现项目截断逻辑
	return rc.sendServerEvent("conversation.item.truncated", map[string]interface{}{})
}

// 处理对话项目删除
func (rc *RealtimeConnection) handleConversationItemDelete(event map[string]interface{}) error {
	// TODO: 实现项目删除逻辑
	return rc.sendServerEvent("conversation.item.deleted", map[string]interface{}{})
}

// 处理响应创建
func (rc *RealtimeConnection) handleResponseCreate(event map[string]interface{}) error {
	// 创建响应ID
	responseID := fmt.Sprintf("resp_%d", time.Now().UnixNano())

	// 发送response.created事件
	rc.sendServerEvent("response.created", map[string]interface{}{
		"response": map[string]interface{}{
			"id":     responseID,
			"object": "realtime.response",
			"status": "in_progress",
			"output": []interface{}{},
		},
	})

	// TODO: 这里应该调用CosyVoice生成音频响应
	// 目前先发送一个简单的文本响应
	go rc.generateMockResponse(responseID)

	return nil
}

// 处理响应取消
func (rc *RealtimeConnection) handleResponseCancel(event map[string]interface{}) error {
	// TODO: 实现响应取消逻辑
	return nil
}

// 生成模拟响应（临时实现）
func (rc *RealtimeConnection) generateMockResponse(responseID string) {
	time.Sleep(100 * time.Millisecond) // 模拟处理延迟

	// 创建输出项目
	itemID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// 发送response.output_item.added事件
	rc.sendServerEvent("response.output_item.added", map[string]interface{}{
		"response_id":  responseID,
		"output_index": 0,
		"item": map[string]interface{}{
			"id":      itemID,
			"object":  "realtime.item",
			"type":    "message",
			"status":  "in_progress",
			"role":    "assistant",
			"content": []interface{}{},
		},
	})

	// 发送conversation.item.created事件
	rc.sendServerEvent("conversation.item.created", map[string]interface{}{
		"item": map[string]interface{}{
			"id":      itemID,
			"object":  "realtime.item",
			"type":    "message",
			"status":  "in_progress",
			"role":    "assistant",
			"content": []interface{}{},
		},
	})

	// 发送response.content_part.added事件
	rc.sendServerEvent("response.content_part.added", map[string]interface{}{
		"response_id":   responseID,
		"item_id":       itemID,
		"output_index":  0,
		"content_index": 0,
		"part": map[string]interface{}{
			"type": "audio",
		},
	})

	// 模拟音频转录增量
	transcript := "Hello! How can I help you today?"
	for _, char := range transcript {
		rc.sendServerEvent("response.audio_transcript.delta", map[string]interface{}{
			"response_id":   responseID,
			"item_id":       itemID,
			"output_index":  0,
			"content_index": 0,
			"delta":         string(char),
		})
		time.Sleep(50 * time.Millisecond)
	}

	// 发送完成事件
	rc.sendServerEvent("response.audio_transcript.done", map[string]interface{}{
		"response_id":   responseID,
		"item_id":       itemID,
		"output_index":  0,
		"content_index": 0,
		"transcript":    transcript,
	})

	rc.sendServerEvent("response.content_part.done", map[string]interface{}{
		"response_id":   responseID,
		"item_id":       itemID,
		"output_index":  0,
		"content_index": 0,
		"part": map[string]interface{}{
			"type":       "audio",
			"transcript": transcript,
		},
	})

	rc.sendServerEvent("response.output_item.done", map[string]interface{}{
		"response_id":  responseID,
		"output_index": 0,
		"item": map[string]interface{}{
			"id":     itemID,
			"object": "realtime.item",
			"type":   "message",
			"status": "completed",
			"role":   "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type":       "audio",
					"transcript": transcript,
				},
			},
		},
	})

	rc.sendServerEvent("response.done", map[string]interface{}{
		"response": map[string]interface{}{
			"id":     responseID,
			"object": "realtime.response",
			"status": "completed",
			"output": []interface{}{
				map[string]interface{}{
					"id":     itemID,
					"object": "realtime.item",
					"type":   "message",
					"status": "completed",
					"role":   "assistant",
					"content": []interface{}{
						map[string]interface{}{
							"type":       "audio",
							"transcript": transcript,
						},
					},
				},
			},
		},
	})
}

// 发送服务端事件
func (rc *RealtimeConnection) sendServerEvent(eventType string, data map[string]interface{}) error {
	event := map[string]interface{}{
		"type":     eventType,
		"event_id": fmt.Sprintf("event_%d", time.Now().UnixNano()),
	}

	// 合并数据
	for k, v := range data {
		event[k] = v
	}

	return rc.conn.WriteJSON(event)
}

// 发送错误事件
func (rc *RealtimeConnection) sendErrorEvent(code, message, eventID string) error {
	return rc.sendServerEvent("error", map[string]interface{}{
		"error": map[string]interface{}{
			"type":     code,
			"code":     code,
			"message":  message,
			"param":    nil,
			"event_id": eventID,
		},
	})
}

// 处理Paraformer语音识别转录
func (rc *RealtimeConnection) processParaformerTranscription(itemID string) {
	// 模拟语音识别处理延迟
	time.Sleep(500 * time.Millisecond)

	// 模拟语音识别结果（实际应该调用阿里云Paraformer API）
	transcript := "这是一段测试语音识别的内容"

	// 发送转录完成事件
	rc.sendServerEvent("conversation.item.input_audio_transcription.completed", map[string]interface{}{
		"item_id":       itemID,
		"content_index": 0,
		"transcript":    transcript,
	})

	// 更新对话项目状态
	rc.conversation.mu.Lock()
	for i := range rc.conversation.Items {
		if rc.conversation.Items[i].ID == itemID {
			rc.conversation.Items[i].Status = "completed"
			// 添加转录内容
			rc.conversation.Items[i].Content = append(rc.conversation.Items[i].Content, ContentPart{
				Type:       "text",
				Text:       transcript,
				Transcript: transcript,
			})
			break
		}
	}
	rc.conversation.mu.Unlock()

	// 发送项目更新事件
	rc.sendServerEvent("conversation.item.updated", map[string]interface{}{
		"item_id": itemID,
		"item": map[string]interface{}{
			"id":     itemID,
			"object": "realtime.item",
			"type":   "message",
			"status": "completed",
			"role":   "user",
			"content": []interface{}{
				map[string]interface{}{
					"type":       "input_audio",
					"audio":      "",
					"transcript": transcript,
				},
				map[string]interface{}{
					"type": "text",
					"text": transcript,
				},
			},
		},
	})
}

// 检查模型是否支持实时API
func isRealtimeModelSupported(model string) bool {
	supportedModels := []string{
		"cosyvoice-v2",
		"paraformer-realtime-8k-v2", // 添加Paraformer语音识别支持
		"gpt-4o-realtime-preview",
		"gpt-4o-mini-realtime-preview",
	}

	log.Printf("检查模型支持: %s", model)
	for _, supported := range supportedModels {
		if model == supported {
			log.Printf("模型 %s 受支持", model)
			return true
		}
	}
	log.Printf("模型 %s 不受支持", model)
	return false
}
