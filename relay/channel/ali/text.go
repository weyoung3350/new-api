package ali

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	relaycommon "one-api/relay/common"
	"one-api/relay/helper"
	"one-api/service"
	"strings"

	"github.com/gin-gonic/gin"
)

// https://help.aliyun.com/document_detail/613695.html?spm=a2c4g.2399480.0.0.1adb778fAdzP9w#341800c0f8w0r

const EnableSearchModelSuffix = "-internet"

func requestOpenAI2Ali(request dto.GeneralOpenAIRequest) *dto.GeneralOpenAIRequest {
	if request.TopP >= 1 {
		request.TopP = 0.999
	} else if request.TopP <= 0 {
		request.TopP = 0.001
	}
	return &request
}

func embeddingRequestOpenAI2Ali(request dto.EmbeddingRequest) *AliEmbeddingRequest {
	return &AliEmbeddingRequest{
		Model: request.Model,
		Input: struct {
			Texts []string `json:"texts"`
		}{
			Texts: request.ParseInput(),
		},
	}
}

func aliEmbeddingHandler(c *gin.Context, resp *http.Response) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	var aliResponse AliEmbeddingResponse
	err := json.NewDecoder(resp.Body).Decode(&aliResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	common.CloseResponseBodyGracefully(resp)

	if aliResponse.Code != "" {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: aliResponse.Message,
				Type:    aliResponse.Code,
				Param:   aliResponse.RequestId,
				Code:    aliResponse.Code,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}

	model := c.GetString("model")
	if model == "" {
		model = "text-embedding-v4"
	}
	fullTextResponse := embeddingResponseAli2OpenAI(&aliResponse, model)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, &fullTextResponse.Usage
}

func embeddingResponseAli2OpenAI(response *AliEmbeddingResponse, model string) *dto.OpenAIEmbeddingResponse {
	openAIEmbeddingResponse := dto.OpenAIEmbeddingResponse{
		Object: "list",
		Data:   make([]dto.OpenAIEmbeddingResponseItem, 0, len(response.Output.Embeddings)),
		Model:  model,
		Usage: dto.Usage{
			PromptTokens: response.Usage.TotalTokens,
			TotalTokens:  response.Usage.TotalTokens,
		},
	}

	for _, item := range response.Output.Embeddings {
		openAIEmbeddingResponse.Data = append(openAIEmbeddingResponse.Data, dto.OpenAIEmbeddingResponseItem{
			Object:    `embedding`,
			Index:     item.TextIndex,
			Embedding: item.Embedding,
		})
	}
	return &openAIEmbeddingResponse
}

func responseAli2OpenAI(response *AliResponse) *dto.OpenAITextResponse {
	choice := dto.OpenAITextResponseChoice{
		Index: 0,
		Message: dto.Message{
			Role:    "assistant",
			Content: response.Output.Text,
		},
		FinishReason: response.Output.FinishReason,
	}
	fullTextResponse := dto.OpenAITextResponse{
		Id:      response.RequestId,
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Choices: []dto.OpenAITextResponseChoice{choice},
		Usage: dto.Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}
	return &fullTextResponse
}

func streamResponseAli2OpenAI(aliResponse *AliResponse) *dto.ChatCompletionsStreamResponse {
	var choice dto.ChatCompletionsStreamResponseChoice
	choice.Delta.SetContentString(aliResponse.Output.Text)
	if aliResponse.Output.FinishReason != "null" {
		finishReason := aliResponse.Output.FinishReason
		choice.FinishReason = &finishReason
	}
	response := dto.ChatCompletionsStreamResponse{
		Id:      aliResponse.RequestId,
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   "ernie-bot",
		Choices: []dto.ChatCompletionsStreamResponseChoice{choice},
	}
	return &response
}

func aliStreamHandler(c *gin.Context, resp *http.Response) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	var usage dto.Usage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	dataChan := make(chan string)
	stopChan := make(chan bool)
	go func() {
		for scanner.Scan() {
			data := scanner.Text()
			if len(data) < 5 { // ignore blank line or wrong format
				continue
			}
			if data[:5] != "data:" {
				continue
			}
			data = data[5:]
			dataChan <- data
		}
		stopChan <- true
	}()
	helper.SetEventStreamHeaders(c)
	lastResponseText := ""
	c.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			var aliResponse AliResponse
			err := json.Unmarshal([]byte(data), &aliResponse)
			if err != nil {
				common.SysError("error unmarshalling stream response: " + err.Error())
				return true
			}
			if aliResponse.Usage.OutputTokens != 0 {
				usage.PromptTokens = aliResponse.Usage.InputTokens
				usage.CompletionTokens = aliResponse.Usage.OutputTokens
				usage.TotalTokens = aliResponse.Usage.InputTokens + aliResponse.Usage.OutputTokens
			}
			response := streamResponseAli2OpenAI(&aliResponse)
			response.Choices[0].Delta.SetContentString(strings.TrimPrefix(response.Choices[0].Delta.GetContentString(), lastResponseText))
			lastResponseText = aliResponse.Output.Text
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				common.SysError("error marshalling stream response: " + err.Error())
				return true
			}
			c.Render(-1, common.CustomEvent{Data: "data: " + string(jsonResponse)})
			return true
		case <-stopChan:
			c.Render(-1, common.CustomEvent{Data: "data: [DONE]"})
			return false
		}
	})
	common.CloseResponseBodyGracefully(resp)
	return nil, &usage
}

func aliHandler(c *gin.Context, resp *http.Response) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	var aliResponse AliResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	common.CloseResponseBodyGracefully(resp)
	err = json.Unmarshal(responseBody, &aliResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	if aliResponse.Code != "" {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: aliResponse.Message,
				Type:    aliResponse.Code,
				Param:   aliResponse.RequestId,
				Code:    aliResponse.Code,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}
	fullTextResponse := responseAli2OpenAI(&aliResponse)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, &fullTextResponse.Usage
}

// 统一音频处理器 - 所有音频请求都使用WebSocket API
func UnifiedAudioHandler(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	common.LogInfo(c, fmt.Sprintf("处理音频请求，模型: %s, 文本长度: %d 字符, 声音: %s",
		request.Model, len([]rune(request.Input)), request.Voice))

	// 所有音频请求都通过WebSocket API处理
	return HandleCosyVoiceWebSocketTTS(c, info, request)
}

// 阿里云音频响应处理
func aliAudioHandler(c *gin.Context, resp *http.Response) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	var aliResponse AliAudioResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	common.CloseResponseBodyGracefully(resp)

	err = json.Unmarshal(responseBody, &aliResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	if aliResponse.Code != "" {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: aliResponse.Message,
				Type:    aliResponse.Code,
				Param:   aliResponse.RequestId,
				Code:    aliResponse.Code,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}

	// 如果有音频数据，直接返回
	if aliResponse.Output.AudioData != "" {
		// 将base64音频数据写入响应
		c.Writer.Header().Set("Content-Type", "audio/mpeg")
		c.Writer.WriteHeader(resp.StatusCode)

		audioData, err := service.DecodeBase64AudioData(aliResponse.Output.AudioData)
		if err != nil {
			return service.OpenAIErrorWrapper(err, "decode_audio_data_failed", http.StatusInternalServerError), nil
		}

		_, err = c.Writer.Write([]byte(audioData))
		if err != nil {
			return service.OpenAIErrorWrapper(err, "write_audio_response_failed", http.StatusInternalServerError), nil
		}
	}

	// 返回使用情况
	usage := &dto.Usage{
		TotalTokens: aliResponse.Usage.TotalTokens,
	}

	return nil, usage
}

// 实时语音识别请求转换：Realtime -> 阿里云格式
func realtimeASRRequestOpenAI2Ali(event dto.RealtimeEvent, taskId string) *AliRealtimeASRRequest {
	aliRequest := &AliRealtimeASRRequest{}

	// 设置请求头
	aliRequest.Header.MessageId = event.EventId
	aliRequest.Header.TaskId = taskId
	aliRequest.Header.Namespace = "SpeechTranscriber"
	aliRequest.Header.Name = "StartTranscription"
	// AppKey和Token需要从配置中获取

	// 设置负载
	aliRequest.Payload.Model = "paraformer-realtime-8k-v2"
	aliRequest.Payload.SampleRate = 16000
	aliRequest.Payload.Format = "pcm"
	aliRequest.Payload.AudioData = event.Audio

	return aliRequest
}

// 阿里云实时语音识别响应转换：阿里云格式 -> OpenAI格式
func realtimeASRResponseAli2OpenAI(aliResponse *AliRealtimeASRResponse) *dto.RealtimeEvent {
	event := &dto.RealtimeEvent{
		EventId: aliResponse.Header.MessageId,
		Type:    dto.RealtimeEventResponseAudioTranscriptionDelta,
		Delta:   aliResponse.Payload.Result,
	}

	// 如果是结束标志，设置为完成事件
	if aliResponse.Payload.IsEnd {
		event.Type = dto.RealtimeEventTypeResponseDone
	}

	return event
}

// 阿里云实时语音识别WebSocket处理器
func aliRealtimeASRHandler(c *gin.Context, resp *http.Response) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	// 注意：这是一个简化的实现，实际的WebSocket处理会更复杂
	// 真实的实现需要：
	// 1. 建立WebSocket连接到阿里云
	// 2. 处理音频流的实时传输
	// 3. 管理会话状态和错误处理
	// 4. 实现心跳和重连机制

	// 这里返回一个基础的Usage结构
	usage := &dto.Usage{
		TotalTokens: 1, // 实时语音识别通常按时长计费
	}

	// 在实际实现中，这里会：
	// - 解析WebSocket消息
	// - 转换为OpenAI Realtime API格式
	// - 流式返回转录结果

	return nil, usage
}

// 创建阿里云实时语音识别会话配置
func createAliRealtimeASRSession() *AliRealtimeASRSession {
	return &AliRealtimeASRSession{
		Model:              "paraformer-realtime-8k-v2",
		SampleRate:         16000,
		Format:             "pcm",
		EnablePunctuate:    true,
		EnableITN:          true,
		EnableWordsLevel:   false,
		MaxSentenceSilence: 800, // 毫秒
	}
}
