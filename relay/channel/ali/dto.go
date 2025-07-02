package ali

import "one-api/dto"

type AliMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type AliInput struct {
	Prompt string `json:"prompt,omitempty"`
	//History []AliMessage `json:"history,omitempty"`
	Messages []AliMessage `json:"messages"`
}

type AliParameters struct {
	TopP              float64 `json:"top_p,omitempty"`
	TopK              int     `json:"top_k,omitempty"`
	Seed              uint64  `json:"seed,omitempty"`
	EnableSearch      bool    `json:"enable_search,omitempty"`
	IncrementalOutput bool    `json:"incremental_output,omitempty"`
}

type AliChatRequest struct {
	Model      string        `json:"model"`
	Input      AliInput      `json:"input,omitempty"`
	Parameters AliParameters `json:"parameters,omitempty"`
}

type AliEmbeddingRequest struct {
	Model string `json:"model"`
	Input struct {
		Texts []string `json:"texts"`
	} `json:"input"`
	Parameters *struct {
		TextType string `json:"text_type,omitempty"`
	} `json:"parameters,omitempty"`
}

type AliEmbedding struct {
	Embedding []float64 `json:"embedding"`
	TextIndex int       `json:"text_index"`
}

type AliEmbeddingResponse struct {
	Output struct {
		Embeddings []AliEmbedding `json:"embeddings"`
	} `json:"output"`
	Usage AliUsage `json:"usage"`
	AliError
}

type AliError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"request_id"`
}

type AliUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type TaskResult struct {
	B64Image string `json:"b64_image,omitempty"`
	Url      string `json:"url,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

type AliOutput struct {
	TaskId       string       `json:"task_id,omitempty"`
	TaskStatus   string       `json:"task_status,omitempty"`
	Text         string       `json:"text"`
	FinishReason string       `json:"finish_reason"`
	Message      string       `json:"message,omitempty"`
	Code         string       `json:"code,omitempty"`
	Results      []TaskResult `json:"results,omitempty"`
}

type AliResponse struct {
	Output AliOutput `json:"output"`
	Usage  AliUsage  `json:"usage"`
	AliError
}

type AliImageRequest struct {
	Model string `json:"model"`
	Input struct {
		Prompt         string `json:"prompt"`
		NegativePrompt string `json:"negative_prompt,omitempty"`
	} `json:"input"`
	Parameters struct {
		Size  string `json:"size,omitempty"`
		N     int    `json:"n,omitempty"`
		Steps string `json:"steps,omitempty"`
		Scale string `json:"scale,omitempty"`
	} `json:"parameters,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type AliRerankParameters struct {
	TopN            *int  `json:"top_n,omitempty"`
	ReturnDocuments *bool `json:"return_documents,omitempty"`
}

type AliRerankInput struct {
	Query     string `json:"query"`
	Documents []any  `json:"documents"`
}

type AliRerankRequest struct {
	Model      string              `json:"model"`
	Input      AliRerankInput      `json:"input"`
	Parameters AliRerankParameters `json:"parameters,omitempty"`
}

type AliRerankResponse struct {
	Output struct {
		Results []dto.RerankResponseResult `json:"results"`
	} `json:"output"`
	Usage     AliUsage `json:"usage"`
	RequestId string   `json:"request_id"`
	AliError
}

// 阿里云TTS音频请求结构
type AliAudioRequest struct {
	Model string `json:"model"`
	Input struct {
		Text string `json:"text"`
	} `json:"input"`
	Parameters struct {
		Voice      string  `json:"voice,omitempty"`
		Rate       string  `json:"rate,omitempty"`
		Pitch      string  `json:"pitch,omitempty"`
		Volume     string  `json:"volume,omitempty"`
		Format     string  `json:"format,omitempty"`
		SampleRate int     `json:"sample_rate,omitempty"`
		Speed      float64 `json:"speed,omitempty"`
	} `json:"parameters,omitempty"`
}

// 阿里云音频响应结构
type AliAudioResponse struct {
	Output struct {
		AudioUrl  string `json:"audio_url,omitempty"`
		AudioData string `json:"audio_data,omitempty"` // base64编码的音频数据
	} `json:"output"`
	Usage     AliUsage `json:"usage"`
	RequestId string   `json:"request_id"`
	AliError
}

// 阿里云实时语音识别WebSocket请求结构
type AliRealtimeASRRequest struct {
	Header struct {
		MessageId string `json:"message_id"`
		TaskId    string `json:"task_id"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		AppKey    string `json:"appkey"`
		Token     string `json:"token"`
	} `json:"header"`
	Payload struct {
		Model      string `json:"model"`
		SampleRate int    `json:"sample_rate"`
		Format     string `json:"format"`
		AudioData  string `json:"audio_data"` // base64编码的音频数据
	} `json:"payload"`
}

// 阿里云实时语音识别WebSocket响应结构
type AliRealtimeASRResponse struct {
	Header struct {
		MessageId  string `json:"message_id"`
		TaskId     string `json:"task_id"`
		Namespace  string `json:"namespace"`
		Name       string `json:"name"`
		Status     int    `json:"status"`
		StatusText string `json:"status_text"`
	} `json:"header"`
	Payload struct {
		Result string `json:"result"`
		Time   int64  `json:"time"`
		Index  int    `json:"index"`
		IsEnd  bool   `json:"is_end"`
	} `json:"payload"`
}

// 阿里云实时语音识别会话配置
type AliRealtimeASRSession struct {
	Model              string `json:"model"`
	SampleRate         int    `json:"sample_rate"`
	Format             string `json:"format"`
	EnablePunctuate    bool   `json:"enable_punctuate"`
	EnableITN          bool   `json:"enable_itn"`
	EnableWordsLevel   bool   `json:"enable_words_level"`
	MaxSentenceSilence int    `json:"max_sentence_silence"`
}
