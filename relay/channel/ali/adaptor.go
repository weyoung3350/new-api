package ali

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	"one-api/relay/channel"
	"one-api/relay/channel/openai"
	relaycommon "one-api/relay/common"
	"one-api/relay/constant"
	"strings"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	//TODO implement me
	panic("implement me")
	return nil, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	var fullRequestURL string
	switch info.RelayMode {
	case constant.RelayModeEmbeddings:
		fullRequestURL = fmt.Sprintf("%s/api/v1/services/embeddings/text-embedding/text-embedding", info.BaseUrl)
	case constant.RelayModeRerank:
		fullRequestURL = fmt.Sprintf("%s/api/v1/services/rerank/text-rerank/text-rerank", info.BaseUrl)
	case constant.RelayModeImagesGenerations:
		fullRequestURL = fmt.Sprintf("%s/api/v1/services/aigc/text2image/image-synthesis", info.BaseUrl)
	case constant.RelayModeAudioSpeech:
		fullRequestURL = fmt.Sprintf("%s/api/v1/services/audio/tts", info.BaseUrl)
	case constant.RelayModeAudioTranscription:
		fullRequestURL = fmt.Sprintf("%s/api/v1/services/audio/transcription", info.BaseUrl)
	case constant.RelayModeRealtime:
		// 阿里云实时语音识别WebSocket端点
		fullRequestURL = fmt.Sprintf("wss://nls-ws.cn-shanghai.aliyuncs.com/ws/v1", info.BaseUrl)
	case constant.RelayModeCompletions:
		fullRequestURL = fmt.Sprintf("%s/compatible-mode/v1/completions", info.BaseUrl)
	default:
		fullRequestURL = fmt.Sprintf("%s/compatible-mode/v1/chat/completions", info.BaseUrl)
	}
	return fullRequestURL, nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	if info.IsStream {
		req.Set("X-DashScope-SSE", "enable")
	}
	if c.GetString("plugin") != "" {
		req.Set("X-DashScope-Plugin", c.GetString("plugin"))
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	// fix: ali parameter.enable_thinking must be set to false for non-streaming calls
	if !info.IsStream {
		request.EnableThinking = false
	}

	switch info.RelayMode {
	default:
		aliReq := requestOpenAI2Ali(*request)
		return aliReq, nil
	}
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	aliRequest := oaiImage2Ali(request)
	return aliRequest, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return ConvertRerankRequest(request), nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return embeddingRequestOpenAI2Ali(request), nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	// 将音频请求存储到上下文中，供后续WebSocket处理使用
	c.Set("audio_request", request)

	// 由于所有音频请求都通过WebSocket处理，这里返回空的请求体
	// 实际的请求处理将在DoResponse方法中进行
	return bytes.NewBuffer([]byte("{}")), nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// 阿里云支持OpenAI兼容模式的Responses API
	// 处理模型名称的后缀转换 reasoning effort
	if strings.HasSuffix(request.Model, "-high") {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		request.Reasoning.Effort = "high"
		request.Model = strings.TrimSuffix(request.Model, "-high")
	} else if strings.HasSuffix(request.Model, "-low") {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		request.Reasoning.Effort = "low"
		request.Model = strings.TrimSuffix(request.Model, "-low")
	} else if strings.HasSuffix(request.Model, "-medium") {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		request.Reasoning.Effort = "medium"
		request.Model = strings.TrimSuffix(request.Model, "-medium")
	}

	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	// 对于音频请求，直接使用WebSocket处理，不发送HTTP请求
	if info.RelayMode == constant.RelayModeAudioSpeech {
		// 返回一个虚拟的HTTP响应，实际处理将在DoResponse中进行
		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBuffer([]byte("{}"))),
		}, nil
	}

	// 如果启用调试模式，打印详细的请求和响应信息
	if common.DebugEnabled {
		// 读取请求体用于打印
		var bodyBytes []byte
		if requestBody != nil {
			bodyBytes, _ = io.ReadAll(requestBody)
			// 重新创建reader供后续使用
			requestBody = bytes.NewBuffer(bodyBytes)
		}

		fullRequestURL, _ := a.GetRequestURL(info)
		common.LogInfo(c, fmt.Sprintf("[阿里云请求调试] URL: %s", fullRequestURL))
		common.LogInfo(c, fmt.Sprintf("[阿里云请求调试] Method: %s", c.Request.Method))

		// 打印请求头
		if c.Request.Header != nil {
			common.LogInfo(c, "[阿里云请求调试] Request Headers:")
			for key, values := range c.Request.Header {
				for _, value := range values {
					common.LogInfo(c, fmt.Sprintf("  %s: %s", key, value))
				}
			}
		}

		// 打印请求体
		if len(bodyBytes) > 0 {
			common.LogInfo(c, fmt.Sprintf("[阿里云请求调试] Request Body: %s", string(bodyBytes)))
		}
	}

	// 执行实际请求
	resp, err := channel.DoApiRequest(a, c, info, requestBody)
	if err != nil {
		if common.DebugEnabled {
			common.LogError(c, fmt.Sprintf("[阿里云响应调试] Request failed: %v", err))
		}
		return nil, err
	}

	// 如果启用调试模式，打印响应信息
	if common.DebugEnabled && resp != nil {
		common.LogInfo(c, fmt.Sprintf("[阿里云响应调试] Status Code: %d", resp.StatusCode))
		common.LogInfo(c, fmt.Sprintf("[阿里云响应调试] Status: %s", resp.Status))

		// 打印响应头
		if resp.Header != nil {
			common.LogInfo(c, "[阿里云响应调试] Response Headers:")
			for key, values := range resp.Header {
				for _, value := range values {
					common.LogInfo(c, fmt.Sprintf("  %s: %s", key, value))
				}
			}
		}

		// 读取并打印响应体（但需要重新创建reader供后续使用）
		if resp.Body != nil {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				common.LogInfo(c, fmt.Sprintf("[阿里云响应调试] Response Body: %s", string(bodyBytes)))
				// 重新创建响应体reader
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			} else {
				common.LogError(c, fmt.Sprintf("[阿里云响应调试] Failed to read response body: %v", err))
			}
		}
	}

	return resp, nil
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *dto.OpenAIErrorWithStatusCode) {
	switch info.RelayMode {
	case constant.RelayModeImagesGenerations:
		err, usage = aliImageHandler(c, resp, info)
	case constant.RelayModeEmbeddings:
		err, usage = aliEmbeddingHandler(c, resp)
	case constant.RelayModeRerank:
		err, usage = RerankHandler(c, resp, info)
	case constant.RelayModeAudioSpeech:
		// 使用统一音频处理器，所有请求都通过WebSocket API
		if request, ok := c.Get("audio_request"); ok {
			if audioReq, ok := request.(dto.AudioRequest); ok {
				err, usage = UnifiedAudioHandler(c, info, audioReq)
			} else {
				return nil, &dto.OpenAIErrorWithStatusCode{
					Error: dto.OpenAIError{
						Message: "音频请求参数获取失败",
						Type:    "audio_request_error",
					},
					StatusCode: http.StatusBadRequest,
				}
			}
		} else {
			return nil, &dto.OpenAIErrorWithStatusCode{
				Error: dto.OpenAIError{
					Message: "音频请求参数缺失",
					Type:    "audio_request_missing",
				},
				StatusCode: http.StatusBadRequest,
			}
		}
	case constant.RelayModeAudioTranscription:
		err, usage = aliAudioHandler(c, resp)
	case constant.RelayModeRealtime:
		err, usage = aliRealtimeASRHandler(c, resp)
	default:
		if info.IsStream {
			err, usage = openai.OaiStreamHandler(c, resp, info)
		} else {
			err, usage = openai.OpenaiHandler(c, resp, info)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
