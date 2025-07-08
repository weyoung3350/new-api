#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
简单的 WebSocket 连接调试脚本
"""

import asyncio
import websockets
import json
import os

async def test_model_connection(model_name):
    """测试特定模型的连接"""
    api_key = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
    
    print(f"\n🔍 测试模型: {model_name}")
    print("-" * 50)
    
    # 使用子协议认证
    try:
        ws_url = f"ws://127.0.0.1:3000/v1/realtime?model={model_name}"
        subprotocols = [
            "realtime",
            f"openai-insecure-api-key.{api_key}",
            "openai-beta.realtime-v1"
        ]
        print(f"连接到: {ws_url}")
        
        async with websockets.connect(ws_url, subprotocols=subprotocols) as websocket:
            print("✅ WebSocket 连接成功")
            
            # 等待服务器消息
            try:
                response = await asyncio.wait_for(websocket.recv(), timeout=3.0)
                print(f"收到响应: {response}")
                
                # 尝试发送会话更新
                session_update = {
                    "type": "session.update",
                    "session": {
                        "modalities": ["text", "audio"],
                        "instructions": "你是一个语音助手。",
                        "voice": "alloy",
                        "input_audio_format": "pcm16",
                        "output_audio_format": "pcm16"
                    }
                }
                
                await websocket.send(json.dumps(session_update))
                print("✅ 会话更新消息已发送")
                
                # 等待更多响应
                try:
                    response2 = await asyncio.wait_for(websocket.recv(), timeout=3.0)
                    print(f"收到会话更新响应: {response2}")
                except asyncio.TimeoutError:
                    print("⏰ 等待会话更新响应超时")
                    
            except asyncio.TimeoutError:
                print("⏰ 等待初始响应超时")
            
    except Exception as e:
        print(f"❌ 连接失败: {e}")

async def test_websocket_connection():
    """测试 WebSocket 连接"""
    
    # 首先测试已知工作的模型
    await test_model_connection("cosyvoice-v2")
    
    # 然后测试 Paraformer 模型
    await test_model_connection("paraformer-realtime-8k-v2")

if __name__ == "__main__":
    print("🚀 WebSocket 连接调试工具")
    print("="*50)
    asyncio.run(test_websocket_connection()) 