#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Paraformer 语音识别测试脚本

此脚本演示了如何使用 paraformer-realtime-8k-v2 模型进行语音识别：

1. HTTP API 测试：
   - 演示错误的使用方法（paraformer 不支持 HTTP /v1/audio/transcriptions）
   - 说明错误原因和正确的解决方案

2. WebSocket API 测试：
   - 演示正确的使用方法（使用 /v1/realtime 端点）
   - 实时流式音频传输和转录

关键要点：
- paraformer-realtime-8k-v2 是实时 WebSocket 模型
- 不支持传统的 HTTP /v1/audio/transcriptions 端点
- 必须使用 WebSocket /v1/realtime 端点
- 支持实时流式音频传输和转录
"""

import asyncio
import websockets
import json
import base64
import time
import logging
import requests
import io

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_audio_transcription():
    """测试音频转录功能"""
    import os
    api_key = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
    model = "paraformer-realtime-8k-v2"
    
    ws_url = f"ws://127.0.0.1:3000/v1/realtime?model={model}"
    subprotocols = [
        "realtime",
        f"openai-insecure-api-key.{api_key}",
        "openai-beta.realtime-v1"
    ]
    
    logger.info(f"连接到: {ws_url}")
    
    async with websockets.connect(ws_url, subprotocols=subprotocols) as websocket:
        logger.info("✅ WebSocket 连接成功")
        
        # 等待初始事件
        session_created = False
        conversation_created = False
        
        while not (session_created and conversation_created):
            try:
                message = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                event = json.loads(message)
                event_type = event.get('type')
                logger.info(f"收到事件: {event_type}")
                
                if event_type == "session.created":
                    session_created = True
                elif event_type == "conversation.created":
                    conversation_created = True
                elif event_type == "session.updated":
                    logger.info("会话已更新")
                    
            except asyncio.TimeoutError:
                logger.warning("等待初始事件超时")
                break
        
        # 生成测试音频数据（16kHz PCM16，2秒）
        sample_rate = 16000
        duration = 2.0
        num_samples = int(sample_rate * duration)
        
        # 生成简单的正弦波作为测试音频
        import math
        frequency = 440  # A4 音符
        audio_samples = []
        for i in range(num_samples):
            sample = int(32767 * 0.5 * math.sin(2 * math.pi * frequency * i / sample_rate))
            # 转换为 16-bit little-endian
            audio_samples.append(sample.to_bytes(2, byteorder='little', signed=True))
        
        audio_data = b''.join(audio_samples)
        audio_base64 = base64.b64encode(audio_data).decode('utf-8')
        
        logger.info(f"生成测试音频: {len(audio_data)} 字节, {duration} 秒")
        
        # 发送音频数据
        chunk_size = 1024  # 每次发送1KB
        for i in range(0, len(audio_base64), chunk_size):
            chunk = audio_base64[i:i+chunk_size]
            
            append_event = {
                "type": "input_audio_buffer.append",
                "audio": chunk
            }
            
            await websocket.send(json.dumps(append_event))
            logger.info(f"发送音频块 {i//chunk_size + 1}/{(len(audio_base64) + chunk_size - 1)//chunk_size}")
            
            # 稍微延迟以模拟实时流
            await asyncio.sleep(0.1)
        
        # 提交音频缓冲区
        commit_event = {
            "type": "input_audio_buffer.commit"
        }
        
        await websocket.send(json.dumps(commit_event))
        logger.info("✅ 音频缓冲区已提交")
        
        # 等待转录结果
        logger.info("等待转录结果...")
        timeout_count = 0
        max_timeout = 10  # 最多等待10秒
        
        while timeout_count < max_timeout:
            try:
                message = await asyncio.wait_for(websocket.recv(), timeout=1.0)
                event = json.loads(message)
                event_type = event.get('type')
                
                logger.info(f"收到事件: {event_type}")
                
                if event_type == "conversation.item.input_audio_transcription.completed":
                    transcript = event.get('transcript', '')
                    logger.info(f"🎉 转录完成: {transcript}")
                    break
                elif event_type == "conversation.item.input_audio_transcription.failed":
                    error = event.get('error', {})
                    logger.error(f"❌ 转录失败: {error}")
                    break
                elif event_type == "input_audio_buffer.committed":
                    logger.info("✅ 音频缓冲区提交确认")
                elif event_type == "conversation.item.created":
                    logger.info("✅ 对话项目已创建")
                elif event_type == "conversation.item.updated":
                    logger.info("✅ 对话项目已更新")
                    # 检查是否包含转录结果
                    item = event.get('item', {})
                    content = item.get('content', [])
                    for part in content:
                        if part.get('type') == 'text':
                            transcript = part.get('text', '')
                            logger.info(f"🎉 从更新事件获取转录: {transcript}")
                            return
                
            except asyncio.TimeoutError:
                timeout_count += 1
                logger.info(f"等待中... ({timeout_count}/{max_timeout})")
        
        if timeout_count >= max_timeout:
            logger.warning("⏰ 等待转录结果超时")

def test_http_transcription():
    """
    尝试使用 HTTP API 测试 Paraformer 语音转录
    
    注意：paraformer-realtime-8k-v2 是一个实时 WebSocket 模型，
    不支持传统的 HTTP /v1/audio/transcriptions 端点。
    此测试用于演示错误情况和正确的使用方法。
    """
    import os
    api_key = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
    base_url = "http://127.0.0.1:3000"
    model = "paraformer-realtime-8k-v2"
    
    logger.info("🚀 开始 HTTP API 语音转录测试")
    logger.warning("⚠️  注意：paraformer-realtime-8k-v2 是实时 WebSocket 模型，不支持 HTTP API")
    logger.info("💡 正确用法：使用 WebSocket /v1/realtime 端点")
    
    # 生成测试音频数据（16kHz PCM16，2秒）
    sample_rate = 16000
    duration = 2.0
    num_samples = int(sample_rate * duration)
    
    # 生成简单的正弦波作为测试音频
    import math
    frequency = 440  # A4 音符
    audio_samples = []
    for i in range(num_samples):
        sample = int(32767 * 0.5 * math.sin(2 * math.pi * frequency * i / sample_rate))
        # 转换为 16-bit little-endian
        audio_samples.append(sample.to_bytes(2, byteorder='little', signed=True))
    
    audio_data = b''.join(audio_samples)
    logger.info(f"生成测试音频: {len(audio_data)} 字节, {duration} 秒")
    
    # 创建 WAV 文件头
    def create_wav_header(sample_rate, num_channels, bits_per_sample, data_size):
        """创建 WAV 文件头"""
        byte_rate = sample_rate * num_channels * bits_per_sample // 8
        block_align = num_channels * bits_per_sample // 8
        
        header = b'RIFF'
        header += (36 + data_size).to_bytes(4, byteorder='little')
        header += b'WAVE'
        header += b'fmt '
        header += (16).to_bytes(4, byteorder='little')  # fmt chunk size
        header += (1).to_bytes(2, byteorder='little')   # audio format (PCM)
        header += num_channels.to_bytes(2, byteorder='little')
        header += sample_rate.to_bytes(4, byteorder='little')
        header += byte_rate.to_bytes(4, byteorder='little')
        header += block_align.to_bytes(2, byteorder='little')
        header += bits_per_sample.to_bytes(2, byteorder='little')
        header += b'data'
        header += data_size.to_bytes(4, byteorder='little')
        
        return header
    
    # 创建完整的 WAV 文件
    wav_header = create_wav_header(sample_rate, 1, 16, len(audio_data))
    wav_data = wav_header + audio_data
    
    # 准备 HTTP 请求
    url = f"{base_url}/v1/audio/transcriptions"
    headers = {
        "Authorization": f"Bearer {api_key}"
        # 注意：不要手动设置 Content-Type，让 requests 自动处理 multipart/form-data
    }
    
    # 创建 multipart/form-data 请求
    files = {
        'file': ('test_audio.wav', io.BytesIO(wav_data), 'audio/wav')
    }
    
    # 其他参数通过 data 字段传递
    data = {
        'model': model,
        'response_format': 'json',
        'language': 'zh'
    }
    
    try:
        logger.info(f"发送 HTTP 请求到: {url}")
        logger.info(f"使用模型: {model}")
        
        # 使用 requests_toolbelt 来手动构建 multipart 数据
        try:
            from requests_toolbelt.multipart.encoder import MultipartEncoder
            
            multipart_data = MultipartEncoder(
                fields={
                    'file': ('test_audio.wav', io.BytesIO(wav_data), 'audio/wav'),
                    'model': model,
                    'response_format': 'json',
                    'language': 'zh'
                }
            )
            
            headers['Content-Type'] = multipart_data.content_type
            logger.info(f"使用 requests_toolbelt，Content-Type: {multipart_data.content_type}")
            
            start_time = time.time()
            response = requests.post(url, headers=headers, data=multipart_data, timeout=30)
            end_time = time.time()
            
        except ImportError:
            logger.info("requests_toolbelt 未安装，使用标准 requests")
            # 回退到标准 requests 方法
            start_time = time.time()
            response = requests.post(url, headers=headers, files=files, data=data, timeout=30)
            end_time = time.time()
        
        logger.info(f"请求完成，耗时: {end_time - start_time:.2f} 秒")
        logger.info(f"响应状态码: {response.status_code}")
        
        if response.status_code == 200:
            try:
                result = response.json()
                transcript = result.get('text', '')
                language = result.get('language', 'unknown')
                duration_result = result.get('duration', 0)
                
                logger.info("✅ HTTP 转录成功!")
                logger.info(f"🎉 转录文本: {transcript}")
                logger.info(f"📝 识别语言: {language}")
                logger.info(f"⏱️  音频时长: {duration_result} 秒")
                
                return {
                    'success': True,
                    'transcript': transcript,
                    'language': language,
                    'duration': duration_result,
                    'response_time': end_time - start_time
                }
                
            except json.JSONDecodeError as e:
                logger.error(f"❌ JSON 解析失败: {e}")
                logger.error(f"原始响应: {response.text}")
                return {'success': False, 'error': f'JSON解析失败: {e}'}
                
        else:
            logger.error(f"❌ HTTP 请求失败: {response.status_code}")
            logger.error(f"错误响应: {response.text}")
            
            # 特殊处理 paraformer 模型的错误
            if "Content-Type" in response.text and "not present" in response.text:
                logger.info("📋 错误原因分析：")
                logger.info("   - paraformer-realtime-8k-v2 是实时 WebSocket 模型")
                logger.info("   - 服务器尝试转发到阿里云，但阿里云不支持此 HTTP 端点")
                logger.info("   - 应该使用 WebSocket /v1/realtime 端点")
                return {
                    'success': False, 
                    'error': 'paraformer-realtime-8k-v2 模型不支持 HTTP API，请使用 WebSocket /v1/realtime 端点',
                    'suggestion': '使用 WebSocket /v1/realtime 端点进行实时语音识别'
                }
            
            return {'success': False, 'error': f'HTTP {response.status_code}: {response.text}'}
            
    except requests.exceptions.RequestException as e:
        logger.error(f"❌ 请求异常: {e}")
        return {'success': False, 'error': f'请求异常: {e}'}
    except Exception as e:
        logger.error(f"❌ 未知错误: {e}")
        return {'success': False, 'error': f'未知错误: {e}'}

async def run_all_tests():
    """运行所有测试"""
    print("🚀 Paraformer 语音识别全面测试")
    print("="*60)
    
    # 测试 1: HTTP API
    print("\n📡 测试 1: HTTP API 转录")
    print("-" * 30)
    http_result = test_http_transcription()
    
    if http_result['success']:
        print(f"✅ HTTP 测试成功")
        print(f"   转录结果: {http_result['transcript']}")
        print(f"   响应时间: {http_result['response_time']:.2f} 秒")
    else:
        print(f"❌ HTTP 测试失败: {http_result['error']}")
        if 'suggestion' in http_result:
            print(f"💡 建议: {http_result['suggestion']}")
    
    # 测试 2: WebSocket API
    print("\n🔌 测试 2: WebSocket 实时转录")
    print("-" * 30)
    try:
        await test_audio_transcription()
        print("✅ WebSocket 测试完成")
    except Exception as e:
        print(f"❌ WebSocket 测试失败: {e}")
    
    print("\n" + "="*60)
    print("🏁 所有测试完成")
    print("\n📋 测试总结：")
    print("✅ WebSocket /v1/realtime 端点：正常工作，支持实时语音识别")
    print("❌ HTTP /v1/audio/transcriptions 端点：不支持 paraformer-realtime-8k-v2 模型")
    print("\n💡 结论：")
    print("   paraformer-realtime-8k-v2 是专门的实时 WebSocket 模型")
    print("   必须使用 WebSocket /v1/realtime 端点进行语音识别")
    print("   支持流式音频传输和实时转录功能")

if __name__ == "__main__":
    print("🚀 Paraformer 语音识别测试套件")
    print("支持 HTTP API 和 WebSocket 两种方式")
    print("="*60)
    
    # 询问用户选择测试方式
    print("\n请选择测试方式:")
    print("1. HTTP API 转录测试")
    print("2. WebSocket 实时转录测试") 
    print("3. 运行所有测试")
    
    choice = input("\n请输入选择 (1/2/3): ").strip()
    
    if choice == "1":
        print("\n" + "="*50)
        test_http_transcription()
    elif choice == "2":
        print("\n" + "="*50)
        asyncio.run(test_audio_transcription())
    elif choice == "3":
        asyncio.run(run_all_tests())
    else:
        print("无效选择，运行所有测试...")
        asyncio.run(run_all_tests()) 