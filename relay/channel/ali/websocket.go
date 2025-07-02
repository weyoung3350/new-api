package ali

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/dto"
	relaycommon "one-api/relay/common"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// CosyVoice WebSocket 管理器
type CosyVoiceWSManager struct {
	connections map[string]*CosyVoiceWSSession
	mutex       sync.RWMutex
	dialer      *websocket.Dialer
}

// 全局WebSocket管理器实例
var cosyVoiceWSManager *CosyVoiceWSManager
var cosyVoiceWSManagerOnce sync.Once

// 获取WebSocket管理器单例
func GetCosyVoiceWSManager() *CosyVoiceWSManager {
	cosyVoiceWSManagerOnce.Do(func() {
		cosyVoiceWSManager = &CosyVoiceWSManager{
			connections: make(map[string]*CosyVoiceWSSession),
			dialer: &websocket.Dialer{
				HandshakeTimeout: 45 * time.Second,
				ReadBufferSize:   1024 * 4,
				WriteBufferSize:  1024 * 4,
			},
		}
	})
	return cosyVoiceWSManager
}

// 创建WebSocket连接
func (m *CosyVoiceWSManager) CreateConnection(c *gin.Context, info *relaycommon.RelayInfo, taskId string) (*CosyVoiceWSSession, error) {
	// 构建正确的WebSocket URL - 根据阿里云CosyVoice WebSocket文档
	wsURL := "wss://dashscope.aliyuncs.com/api-ws/v1/inference"

	// 设置请求头
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+info.ApiKey)
	headers.Set("X-DashScope-DataInspection", "enable")

	common.LogInfo(c, fmt.Sprintf("尝试连接WebSocket: %s", wsURL))

	// 建立WebSocket连接
	conn, resp, err := m.dialer.Dial(wsURL, headers)
	if err != nil {
		if resp != nil {
			common.LogError(c, fmt.Sprintf("WebSocket连接失败，状态码: %d, 错误: %v", resp.StatusCode, err))
		}
		return nil, fmt.Errorf("WebSocket连接失败: %w", err)
	}

	common.LogInfo(c, fmt.Sprintf("WebSocket连接成功建立，任务ID: %s", taskId))

	// 创建会话
	session := &CosyVoiceWSSession{
		TaskId:     taskId,
		Connection: conn,
		AudioChan:  make(chan []byte, 100),
		ErrorChan:  make(chan error, 10),
		DoneChan:   make(chan bool, 1),
		Model:      "cosyvoice-v2",
		Voice:      "longyingcui",
		Format:     "mp3",
		SampleRate: 16000,
		EnableSSML: false,
	}

	// 存储会话
	m.mutex.Lock()
	m.connections[taskId] = session
	m.mutex.Unlock()

	// 启动消息监听
	go m.listenMessages(c, session)

	return session, nil
}

// 监听WebSocket消息
func (m *CosyVoiceWSManager) listenMessages(c *gin.Context, session *CosyVoiceWSSession) {
	defer func() {
		m.CloseConnection(session.TaskId)
		if r := recover(); r != nil {
			common.LogError(c, fmt.Sprintf("WebSocket消息监听发生panic: %v", r))
		}
	}()

	for {
		select {
		case <-session.DoneChan:
			return
		default:
			// 设置读取超时
			session.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))

			messageType, message, err := session.Connection.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					common.LogInfo(c, fmt.Sprintf("WebSocket连接正常关闭，任务ID: %s", session.TaskId))
				} else {
					common.LogError(c, fmt.Sprintf("读取WebSocket消息失败，任务ID: %s, 错误: %v", session.TaskId, err))
					session.ErrorChan <- err
				}
				return
			}

			if messageType == websocket.TextMessage {
				m.handleTextMessage(c, session, message)
			} else if messageType == websocket.BinaryMessage {
				// 二进制消息是音频数据
				select {
				case session.AudioChan <- message:
					common.LogInfo(c, fmt.Sprintf("接收到音频数据，大小: %d 字节", len(message)))
				default:
					common.LogWarn(c, "音频通道已满，丢弃音频数据")
				}
			}
		}
	}
}

// 处理文本消息（事件）
func (m *CosyVoiceWSManager) handleTextMessage(c *gin.Context, session *CosyVoiceWSSession, message []byte) {
	var response CosyVoiceWSResponse
	err := json.Unmarshal(message, &response)
	if err != nil {
		common.LogError(c, fmt.Sprintf("解析WebSocket响应失败: %v, 消息: %s", err, string(message)))
		return
	}

	common.LogInfo(c, fmt.Sprintf("收到WebSocket事件: %s, 任务ID: %s", response.Header.Event, response.Header.TaskID))

	switch response.Header.Event {
	case CosyVoiceWSEventTaskStarted:
		common.LogInfo(c, fmt.Sprintf("任务开始: %s", session.TaskId))

	case CosyVoiceWSEventResultGenerated:
		// 音频结果生成，音频数据在二进制消息中
		common.LogInfo(c, "音频结果生成")

	case CosyVoiceWSEventTaskFinished:
		common.LogInfo(c, fmt.Sprintf("任务完成: %s", session.TaskId))
		session.DoneChan <- true

	case CosyVoiceWSEventTaskFailed:
		errMsg := fmt.Sprintf("任务失败: %s, 错误码: %s, 错误信息: %s",
			session.TaskId, response.Header.ErrorCode, response.Header.ErrorMessage)
		common.LogError(c, errMsg)
		session.ErrorChan <- fmt.Errorf(errMsg)
	}
}

// OpenAI声音到CosyVoice声音的映射表
// 根据阿里云CosyVoice官方音色列表更新，提供更丰富的音色选择
var openAIToCosyVoiceVoiceMap = map[string]string{
	// OpenAI标准声音映射到CosyVoice官方音色（基于音色特质匹配）
	"alloy":   "longxiaochun_v2", // 映射到龙小淳（知性积极女）
	"echo":    "longnan_v2",      // 映射到龙楠（睿智青年男）
	"fable":   "longmiao_v2",     // 映射到龙妙（抑扬顿挫女，适合故事）
	"onyx":    "longsanshu",      // 映射到龙三叔（沉稳质感男）
	"nova":    "longyue_v2",      // 映射到龙悦（温暖磁性女）
	"shimmer": "longyuan_v2",     // 映射到龙媛（温暖治愈女）

	// 语音助手类音色（常用）
	"longyumi_v2":     "longyumi_v2",     // YUMI（正经青年女）
	"longxiaochun_v2": "longxiaochun_v2", // 龙小淳（知性积极女）
	"longxiaoxia_v2":  "longxiaoxia_v2",  // 龙小夏（沉稳权威女）

	// 有声书类音色（高质量）
	"longsanshu":  "longsanshu",  // 龙三叔（沉稳质感男）
	"longxiu_v2":  "longxiu_v2",  // 龙修（博才说书男）
	"longmiao_v2": "longmiao_v2", // 龙妙（抑扬顿挫女）
	"longyue_v2":  "longyue_v2",  // 龙悦（温暖磁性女）
	"longnan_v2":  "longnan_v2",  // 龙楠（睿智青年男）
	"longyuan_v2": "longyuan_v2", // 龙媛（温暖治愈女）

	// 社交陪伴类音色（温暖亲切）
	"longanrou":        "longanrou",        // 龙安柔（温柔闺蜜女）
	"longqiang_v2":     "longqiang_v2",     // 龙嫱（浪漫风情女）
	"longhan_v2":       "longhan_v2",       // 龙寒（温暖痴情男）
	"longxing_v2":      "longxing_v2",      // 龙星（温婉邻家女）
	"longhua_v2":       "longhua_v2",       // 龙华（元气甜美女）
	"longwan_v2":       "longwan_v2",       // 龙婉（积极知性女）
	"longcheng_v2":     "longcheng_v2",     // 龙橙（智慧青年男）
	"longfeifei_v2":    "longfeifei_v2",    // 龙菲菲（甜美娇气女）
	"longxiaocheng_v2": "longxiaocheng_v2", // 龙小诚（磁性低音男）
	"longzhe_v2":       "longzhe_v2",       // 龙哲（呆板大暖男）
	"longyan_v2":       "longyan_v2",       // 龙颜（温暖春风女）
	"longtian_v2":      "longtian_v2",      // 龙天（磁性理智男）
	"longze_v2":        "longze_v2",        // 龙泽（温暖元气男）
	"longshao_v2":      "longshao_v2",      // 龙邵（积极向上男）
	"longhao_v2":       "longhao_v2",       // 龙浩（多情忧郁男）

	// 新闻播报类音色（专业权威）
	"longshu_v2":     "longshu_v2",     // 龙书（沉稳青年男）
	"loongbella_v2":  "loongbella_v2",  // Bella2.0（精准干练女）
	"longshuo_v2":    "longshuo_v2",    // 龙硕（博才干练男）
	"longxiaobai_v2": "longxiaobai_v2", // 龙小白（沉稳播报女）
	"longjing_v2":    "longjing_v2",    // 龙婧（典型播音女）
	"loongstella_v2": "loongstella_v2", // loongstella（飒爽利落女）
}

// 发送run-task指令（按照官方格式）
func (m *CosyVoiceWSManager) SendRunTask(c *gin.Context, session *CosyVoiceWSSession, request dto.AudioRequest) error {
	// 设置默认声音并进行映射
	voice := "longxiaochun_v2" // 默认声音
	if request.Voice != "" {
		if mappedVoice, exists := openAIToCosyVoiceVoiceMap[request.Voice]; exists {
			voice = mappedVoice
			common.LogInfo(c, fmt.Sprintf("声音映射: %s -> %s", request.Voice, voice))
		} else {
			voice = request.Voice // 如果没有映射，直接使用原声音
			common.LogInfo(c, fmt.Sprintf("使用原始声音: %s", voice))
		}
	}

	// 设置默认格式
	format := "mp3"
	if request.ResponseFormat != "" {
		format = request.ResponseFormat
	}

	wsEvent := CosyVoiceWSEvent{
		Header: CosyVoiceWSHeader{
			Action:    CosyVoiceWSCommandRunTask,
			TaskID:    session.TaskId,
			Streaming: "duplex",
		},
		Payload: CosyVoiceWSPayload{
			TaskGroup: "audio",
			Task:      "tts",
			Function:  "SpeechSynthesizer",
			Model:     "cosyvoice-v2",
			Parameters: CosyVoiceWSParams{
				TextType:   "PlainText",
				Voice:      voice,
				Format:     format,
				SampleRate: 22050, // 官方示例中的采样率
				Volume:     50,
				Rate:       1,
				Pitch:      1,
			},
			Input: CosyVoiceWSInput{}, // 空的input，在continue-task中发送文本
		},
	}

	return m.sendMessage(c, session, wsEvent)
}

// 发送continue-task指令（按照官方格式）
func (m *CosyVoiceWSManager) SendContinueTask(c *gin.Context, session *CosyVoiceWSSession, text string) error {
	wsEvent := CosyVoiceWSEvent{
		Header: CosyVoiceWSHeader{
			Action:    CosyVoiceWSCommandContinueTask,
			TaskID:    session.TaskId,
			Streaming: "duplex",
		},
		Payload: CosyVoiceWSPayload{
			Input: CosyVoiceWSInput{
				Text: text,
			},
		},
	}

	return m.sendMessage(c, session, wsEvent)
}

// 发送finish-task指令（按照官方格式）
func (m *CosyVoiceWSManager) SendFinishTask(c *gin.Context, session *CosyVoiceWSSession) error {
	wsEvent := CosyVoiceWSEvent{
		Header: CosyVoiceWSHeader{
			Action:    CosyVoiceWSCommandFinishTask,
			TaskID:    session.TaskId,
			Streaming: "duplex",
		},
		Payload: CosyVoiceWSPayload{
			Input: CosyVoiceWSInput{}, // 空的input
		},
	}

	return m.sendMessage(c, session, wsEvent)
}

// 发送WebSocket消息
func (m *CosyVoiceWSManager) sendMessage(c *gin.Context, session *CosyVoiceWSSession, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 设置写入超时
	session.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))

	err = session.Connection.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		common.LogError(c, fmt.Sprintf("发送WebSocket消息失败: %v", err))
		return fmt.Errorf("发送WebSocket消息失败: %w", err)
	}

	common.LogInfo(c, fmt.Sprintf("WebSocket消息发送成功，大小: %d 字节", len(data)))
	return nil
}

// 关闭连接
func (m *CosyVoiceWSManager) CloseConnection(taskId string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if session, exists := m.connections[taskId]; exists {
		session.Connection.Close()
		close(session.AudioChan)
		close(session.ErrorChan)
		close(session.DoneChan)
		delete(m.connections, taskId)
	}
}

// 获取连接
func (m *CosyVoiceWSManager) GetConnection(taskId string) (*CosyVoiceWSSession, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.connections[taskId]
	return session, exists
}

// 处理CosyVoice WebSocket TTS请求
func HandleCosyVoiceWebSocketTTS(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	// 生成任务ID
	taskId := fmt.Sprintf("tts_%d_%s", time.Now().UnixNano(), common.GetRandomString(8))

	// 获取WebSocket管理器
	manager := GetCosyVoiceWSManager()

	// 创建WebSocket连接
	session, err := manager.CreateConnection(c, info, taskId)
	if err != nil {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: fmt.Sprintf("创建WebSocket连接失败: %v", err),
				Type:    "websocket_connection_error",
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	defer manager.CloseConnection(taskId)

	// 发送run-task指令
	err = manager.SendRunTask(c, session, request)
	if err != nil {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: fmt.Sprintf("发送run-task指令失败: %v", err),
				Type:    "websocket_send_error",
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// 等待任务开始
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 发送文本内容
	err = manager.SendContinueTask(c, session, request.Input)
	if err != nil {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: fmt.Sprintf("发送continue-task指令失败: %v", err),
				Type:    "websocket_send_error",
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// 发送完成指令
	err = manager.SendFinishTask(c, session)
	if err != nil {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: fmt.Sprintf("发送finish-task指令失败: %v", err),
				Type:    "websocket_send_error",
			},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// 收集音频数据
	var audioData []byte
	var totalCharacters int

	for {
		select {
		case <-ctx.Done():
			return &dto.OpenAIErrorWithStatusCode{
				Error: dto.OpenAIError{
					Message: "WebSocket处理超时",
					Type:    "websocket_timeout",
				},
				StatusCode: http.StatusRequestTimeout,
			}, nil

		case err := <-session.ErrorChan:
			return &dto.OpenAIErrorWithStatusCode{
				Error: dto.OpenAIError{
					Message: fmt.Sprintf("WebSocket处理错误: %v", err),
					Type:    "websocket_processing_error",
				},
				StatusCode: http.StatusInternalServerError,
			}, nil

		case audio := <-session.AudioChan:
			audioData = append(audioData, audio...)

		case <-session.DoneChan:
			// 任务完成，返回音频数据
			if len(audioData) == 0 {
				return &dto.OpenAIErrorWithStatusCode{
					Error: dto.OpenAIError{
						Message: "未接收到音频数据",
						Type:    "no_audio_data",
					},
					StatusCode: http.StatusInternalServerError,
				}, nil
			}

			// 设置响应头
			c.Writer.Header().Set("Content-Type", getAudioContentType(session.Format))
			c.Writer.WriteHeader(http.StatusOK)

			// 写入音频数据
			_, err = c.Writer.Write(audioData)
			if err != nil {
				return &dto.OpenAIErrorWithStatusCode{
					Error: dto.OpenAIError{
						Message: fmt.Sprintf("写入音频响应失败: %v", err),
						Type:    "response_write_error",
					},
					StatusCode: http.StatusInternalServerError,
				}, nil
			}

			// 计算使用量（基于字符数）
			totalCharacters = len([]rune(request.Input))
			usage := &dto.Usage{
				TotalTokens: totalCharacters,
			}

			common.LogInfo(c, fmt.Sprintf("WebSocket TTS完成，音频大小: %d 字节, 字符数: %d",
				len(audioData), totalCharacters))

			return nil, usage
		}
	}
}

// 获取音频内容类型
func getAudioContentType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "opus":
		return "audio/opus"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}

// 注意：已移除智能路由逻辑，所有音频请求都使用WebSocket API
