#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
专门测试 paraformer-realtime-8k-v2 语音识别功能
"""

import os
import requests
import json
import time

# 配置
API_KEY = "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q"
BASE_URL = "http://127.0.0.1:3000"

def test_realtime_endpoint():
    """测试 /v1/realtime 端点"""
    print("🔍 测试 /v1/realtime 端点...")
    
    url = f"{BASE_URL}/v1/realtime"
    headers = {"Authorization": f"Bearer {API_KEY}"}
    
    # 尝试不同的请求格式
    
    # 方式1: multipart/form-data (类似Whisper API)
    print("\n1. 尝试 multipart/form-data 格式:")
    try:
        files = {
            'file': ('audio.wav', b'', 'audio/wav'),  # 空音频数据用于测试
            'model': (None, 'paraformer-realtime-8k-v2'),
            'response_format': (None, 'json'),
            'language': (None, 'zh')
        }
        
        response = requests.post(url, headers=headers, files=files, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
    except Exception as e:
        print(f"   异常: {str(e)}")
    
    # 方式2: JSON格式
    print("\n2. 尝试 JSON 格式:")
    try:
        headers_json = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }
        
        data = {
            "model": "paraformer-realtime-8k-v2",
            "audio": "",  # 空音频数据用于测试
            "language": "zh",
            "response_format": "json"
        }
        
        response = requests.post(url, headers=headers_json, json=data, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
    except Exception as e:
        print(f"   异常: {str(e)}")

def test_audio_transcriptions_endpoint():
    """测试 /v1/audio/transcriptions 端点"""
    print("\n🔍 测试 /v1/audio/transcriptions 端点...")
    
    url = f"{BASE_URL}/v1/audio/transcriptions"
    headers = {"Authorization": f"Bearer {API_KEY}"}
    
    try:
        files = {
            'file': ('audio.wav', b'', 'audio/wav'),
            'model': (None, 'paraformer-realtime-8k-v2'),
            'response_format': (None, 'json'),
            'language': (None, 'zh')
        }
        
        response = requests.post(url, headers=headers, files=files, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
    except Exception as e:
        print(f"   异常: {str(e)}")

def test_completions_endpoint():
    """测试 /v1/completions 端点 (可能用于音频处理)"""
    print("\n🔍 测试 /v1/completions 端点...")
    
    url = f"{BASE_URL}/v1/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    try:
        data = {
            "model": "paraformer-realtime-8k-v2",
            "prompt": "转录音频",
            "max_tokens": 100
        }
        
        response = requests.post(url, headers=headers, json=data, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
    except Exception as e:
        print(f"   异常: {str(e)}")

def test_chat_completions_endpoint():
    """测试 /v1/chat/completions 端点"""
    print("\n🔍 测试 /v1/chat/completions 端点...")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    try:
        data = {
            "model": "paraformer-realtime-8k-v2",
            "messages": [
                {"role": "user", "content": "请转录这段音频"}
            ],
            "max_tokens": 100
        }
        
        response = requests.post(url, headers=headers, json=data, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
    except Exception as e:
        print(f"   异常: {str(e)}")

def list_available_endpoints():
    """尝试获取可用端点"""
    print("\n🔍 探测可用端点...")
    
    common_endpoints = [
        "/v1/models",
        "/v1/audio",
        "/v1/audio/transcriptions", 
        "/v1/audio/speech",
        "/v1/realtime",
        "/v1/completions",
        "/v1/chat/completions",
        "/v1/embeddings"
    ]
    
    for endpoint in common_endpoints:
        try:
            url = f"{BASE_URL}{endpoint}"
            response = requests.get(url, headers={"Authorization": f"Bearer {API_KEY}"}, timeout=10)
            print(f"   GET {endpoint}: {response.status_code}")
            if response.status_code != 404:
                print(f"      响应预览: {response.text[:100]}...")
        except Exception as e:
            print(f"   GET {endpoint}: 连接异常")

def main():
    """主函数"""
    print("🚀 paraformer-realtime-8k-v2 语音识别专项测试")
    print("=" * 60)
    
    # 首先探测可用端点
    list_available_endpoints()
    
    print("\n" + "=" * 60)
    print("开始测试各种可能的调用方式...")
    
    # 测试不同的端点和请求格式
    test_realtime_endpoint()
    test_audio_transcriptions_endpoint()
    test_completions_endpoint()
    test_chat_completions_endpoint()
    
    print("\n" + "=" * 60)
    print("🏁 测试完成！")

if __name__ == "__main__":
    main() 