#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
调试音频转录请求的脚本
"""

import requests
import os

# 配置
API_KEY = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
BASE_URL = "http://127.0.0.1:3000"

def test_audio_transcription():
    """测试音频转录端点"""
    print("🔍 测试 /v1/audio/transcriptions 端点...")
    
    url = f"{BASE_URL}/v1/audio/transcriptions"
    
    # 准备测试音频数据（空文件用于测试）
    audio_data = b""
    
    # 方式1: 使用 files 参数（推荐）
    print("\n1. 使用 files 参数:")
    try:
        files = {
            'file': ('test.wav', audio_data, 'audio/wav'),
            'model': (None, 'paraformer-realtime-8k-v2'),
            'response_format': (None, 'json'),
            'language': (None, 'zh')
        }
        
        headers = {
            'Authorization': f'Bearer {API_KEY}'
            # 注意：不设置 Content-Type，让 requests 自动处理
        }
        
        print(f"   URL: {url}")
        print(f"   Headers: {headers}")
        print(f"   Files: {files}")
        
        response = requests.post(url, headers=headers, files=files, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应头: {dict(response.headers)}")
        print(f"   响应: {response.text}")
        
    except Exception as e:
        print(f"   异常: {str(e)}")

    # 方式2: 手动构建 multipart 数据
    print("\n2. 手动构建 multipart 数据:")
    try:
        from requests_toolbelt.multipart.encoder import MultipartEncoder
        
        multipart_data = MultipartEncoder(
            fields={
                'file': ('test.wav', audio_data, 'audio/wav'),
                'model': 'paraformer-realtime-8k-v2',
                'response_format': 'json',
                'language': 'zh'
            }
        )
        
        headers = {
            'Authorization': f'Bearer {API_KEY}',
            'Content-Type': multipart_data.content_type
        }
        
        print(f"   URL: {url}")
        print(f"   Headers: {headers}")
        print(f"   Content-Type: {multipart_data.content_type}")
        
        response = requests.post(url, headers=headers, data=multipart_data, timeout=30)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
        
    except ImportError:
        print("   需要安装 requests-toolbelt: pip install requests-toolbelt")
    except Exception as e:
        print(f"   异常: {str(e)}")

if __name__ == "__main__":
    test_audio_transcription() 