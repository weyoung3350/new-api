#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
New API 平台快速测试示例
简化版本，用于快速验证模型可用性
"""

import os
import requests
import json

# 配置信息
API_KEY = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
BASE_URL = "http://127.0.0.1:3000"  # 请替换为实际的API地址

def test_text_embedding():
    """快速测试文本嵌入模型"""
    url = f"{BASE_URL}/v1/embeddings"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    data = {
        "model": "text-embedding-v4",
        "input": ["测试文本嵌入功能"],
        "encoding_format": "float"
    }
    
    try:
        response = requests.post(url, headers=headers, json=data, timeout=30)
        print(f"文本嵌入测试 - 状态码: {response.status_code}")
        if response.status_code == 200:
            result = response.json()
            embeddings = result.get('data', [])
            if embeddings:
                dimension = len(embeddings[0]['embedding'])
                print(f"✅ 成功！嵌入维度: {dimension}")
            else:
                print("❌ 响应格式异常")
        else:
            print(f"❌ 失败: {response.text}")
    except Exception as e:
        print(f"❌ 异常: {str(e)}")

def test_speech_synthesis():
    """快速测试语音合成模型"""
    url = f"{BASE_URL}/v1/audio/speech"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    data = {
        "model": "cosyvoice-v2",
        "input": "这是一个语音合成测试",
        "voice": "longyingcui",
        "response_format": "mp3"
    }
    
    try:
        response = requests.post(url, headers=headers, json=data, timeout=60)
        print(f"语音合成测试 - 状态码: {response.status_code}")
        if response.status_code == 200:
            audio_size = len(response.content)
            print(f"✅ 成功！音频大小: {audio_size} 字节")
            # 可选：保存音频文件
            # with open("test_output.mp3", "wb") as f:
            #     f.write(response.content)
        else:
            print(f"❌ 失败: {response.text}")
    except Exception as e:
        print(f"❌ 异常: {str(e)}")

def test_speech_recognition():
    """快速测试语音识别模型（需要音频文件）"""
    # 注意：这个测试需要实际的音频文件
    print("语音识别测试 - 需要提供音频文件")
    audio_file_path = "test_audio.wav"  # 请替换为实际的音频文件路径
    
    if not os.path.exists(audio_file_path):
        print("❌ 未找到音频文件，跳过语音识别测试")
        print(f"   请将音频文件命名为 {audio_file_path} 或修改代码中的路径")
        return
    
    url = f"{BASE_URL}/v1/realtime"
    headers = {"Authorization": f"Bearer {API_KEY}"}
    
    try:
        with open(audio_file_path, 'rb') as audio_file:
            files = {
                'file': ('audio.wav', audio_file, 'audio/wav'),
                'model': (None, 'paraformer-realtime-8k-v2'),
                'response_format': (None, 'json'),
                'language': (None, 'zh')
            }
            
            response = requests.post(url, headers=headers, files=files, timeout=60)
            print(f"语音识别测试 - 状态码: {response.status_code}")
            if response.status_code == 200:
                result = response.json()
                text = result.get('text', '')
                print(f"✅ 成功！识别结果: {text}")
            else:
                print(f"❌ 失败: {response.text}")
    except Exception as e:
        print(f"❌ 异常: {str(e)}")

def main():
    """主函数"""
    print("🚀 New API 平台快速测试")
    print("=" * 40)
    
    if API_KEY == "your-api-key-here" or not API_KEY:
        print("⚠️  请先设置 NEW_API_KEY 环境变量")
        print("   export NEW_API_KEY='your-actual-api-key'")
        return
    
    print(f"📡 API地址: {BASE_URL}")
    print(f"🔑 API密钥: {API_KEY[:8]}...")
    print("-" * 40)
    
    # 依次测试三个模型
    print("\n1. 测试文本嵌入模型...")
    test_text_embedding()
    
    print("\n2. 测试语音合成模型...")
    test_speech_synthesis()
    
    print("\n3. 测试语音识别模型...")
    test_speech_recognition()
    
    print("\n📊 快速测试完成！")

if __name__ == "__main__":
    main() 