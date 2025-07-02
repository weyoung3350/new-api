#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
专门测试 CosyVoice WebSocket TTS 功能
"""

import os
import requests
import json
import time

# 配置
API_KEY = "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q"
BASE_URL = "http://127.0.0.1:3000"

def test_short_text_tts():
    """测试短文本TTS（使用WebSocket API）"""
    print("🔍 测试短文本TTS...")
    
    url = f"{BASE_URL}/v1/audio/speech"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "cosyvoice-v2",
        "input": "你好，这是一个短文本测试。",  # 短文本
        "voice": "longyingcui",
        "response_format": "mp3",
        "speed": 1.0
    }
    
    try:
        print(f"   发送请求: {data['input']}")
        start_time = time.time()
        response = requests.post(url, headers=headers, json=data, timeout=60)
        end_time = time.time()
        
        print(f"   状态码: {response.status_code}")
        print(f"   响应时间: {end_time - start_time:.2f}秒")
        
        if response.status_code == 200:
            audio_size = len(response.content)
            print(f"   ✅ 成功！音频大小: {audio_size} 字节")
            
            # 保存音频文件
            output_file = f"test_short_text_{int(time.time())}.mp3"
            with open(output_file, "wb") as f:
                f.write(response.content)
            print(f"   音频保存至: {output_file}")
        else:
            print(f"   ❌ 失败: {response.text}")
            
    except Exception as e:
        print(f"   ❌ 异常: {str(e)}")

def test_long_text_tts():
    """测试长文本TTS（使用WebSocket API）"""
    print("\n🔍 测试长文本TTS...")
    
    url = f"{BASE_URL}/v1/audio/speech"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # 生成长文本（超过500字符）
    long_text = """
    在这个快速发展的数字时代，人工智能技术正在深刻地改变着我们的生活方式和工作模式。
    从智能语音助手到自动驾驶汽车，从医疗诊断到金融风控，AI的应用场景越来越广泛。
    语音合成技术作为人工智能的重要分支，在人机交互、内容创作、教育培训等领域发挥着重要作用。
    CosyVoice作为先进的语音合成模型，能够生成自然流畅、富有表现力的语音内容，
    为用户提供更加智能化和个性化的语音体验。通过WebSocket技术，我们可以实现实时的、
    流式的语音合成服务，大大提升了用户体验和系统性能。这种技术的结合，
    不仅降低了延迟，还提高了系统的并发处理能力，为大规模语音应用奠定了坚实的基础。
    """
    
    data = {
        "model": "cosyvoice-v2",
        "input": long_text.strip(),
        "voice": "longyingcui",
        "response_format": "mp3",
        "speed": 1.0
    }
    
    try:
        print(f"   文本长度: {len(data['input'])} 字符")
        start_time = time.time()
        response = requests.post(url, headers=headers, json=data, timeout=120)
        end_time = time.time()
        
        print(f"   状态码: {response.status_code}")
        print(f"   响应时间: {end_time - start_time:.2f}秒")
        
        if response.status_code == 200:
            audio_size = len(response.content)
            print(f"   ✅ 成功！音频大小: {audio_size} 字节")
            
            # 保存音频文件
            output_file = f"test_long_text_{int(time.time())}.mp3"
            with open(output_file, "wb") as f:
                f.write(response.content)
            print(f"   音频保存至: {output_file}")
        else:
            print(f"   ❌ 失败: {response.text}")
            
    except Exception as e:
        print(f"   ❌ 异常: {str(e)}")


def test_different_voices():
    """测试不同声音"""
    print("\n🔍 测试不同声音...")
    
    # 使用CosyVoice官方支持的音色
    voices = ["longyingcui", "longxiaochun_v2", "longwan_v2", "alloy"]  # alloy会被映射到官方音色
    
    for voice in voices:
        print(f"\n   测试声音: {voice}")
        
        url = f"{BASE_URL}/v1/audio/speech"
        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }
        
        data = {
            "model": "cosyvoice-v2",
            "input": f"这是使用{voice}声音的测试，听起来怎么样呢？",
            "voice": voice,
            "response_format": "mp3",
            "speed": 1.0
        }
        
        try:
            start_time = time.time()
            response = requests.post(url, headers=headers, json=data, timeout=60)
            end_time = time.time()
            
            print(f"     状态码: {response.status_code}")
            print(f"     响应时间: {end_time - start_time:.2f}秒")
            
            if response.status_code == 200:
                audio_size = len(response.content)
                print(f"     ✅ 成功！音频大小: {audio_size} 字节")
                
                # 保存音频文件
                output_file = f"test_voice_{voice}_{int(time.time())}.mp3"
                with open(output_file, "wb") as f:
                    f.write(response.content)
                print(f"     音频保存至: {output_file}")
            else:
                print(f"     ❌ 失败: {response.text}")
                
        except Exception as e:
            print(f"     ❌ 异常: {str(e)}")

def main():
    """主测试函数"""
    print("🚀 开始CosyVoice WebSocket TTS测试")
    print("=" * 60)
    
    # 创建输出目录
    os.makedirs("test_outputs", exist_ok=True)
    os.chdir("test_outputs")
    
    # 运行各种测试
    test_short_text_tts()
    test_long_text_tts()
    test_different_voices()
    
    print("\n" + "=" * 60)
    print("🎉 测试完成！")
    print("\n📝 测试说明:")
    print("- 所有文本请求都使用CosyVoice WebSocket API处理")
    print("- 支持多种官方音色，包括OpenAI格式声音映射")
    print("- SSML标记暂不被CosyVoice官方支持")
    print("- 检查生成的音频文件质量和正确性")

if __name__ == "__main__":
    main() 