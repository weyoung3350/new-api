#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
API密钥验证脚本
用于快速验证不同API密钥对wanx2.1-t2i-turbo模型的支持情况
"""

import os
import requests
import json
import sys

def test_api_key(api_key, base_url="http://localhost:3000"):
    """测试API密钥的可用性"""
    
    print(f"🔑 测试API密钥: {api_key[:20]}...")
    
    headers = {
        'Authorization': f'Bearer {api_key}',
        'Content-Type': 'application/json'
    }
    
    # 1. 测试模型列表
    print("1️⃣ 检查模型列表...")
    try:
        response = requests.get(f"{base_url}/v1/models", headers=headers, timeout=10)
        if response.status_code == 200:
            models = response.json()
            model_names = [m['id'] for m in models.get('data', [])]
            wanx_models = [m for m in model_names if 'wanx' in m.lower()]
            print(f"   ✅ 模型列表获取成功，共 {len(model_names)} 个模型")
            if wanx_models:
                print(f"   🎨 WANX模型: {wanx_models}")
            else:
                print("   ⚠️  未发现WANX模型")
        else:
            print(f"   ❌ 模型列表获取失败: {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ 请求失败: {str(e)}")
        return False
    
    # 2. 测试简单图像生成
    print("2️⃣ 测试图像生成...")
    try:
        payload = {
            "model": "wanx2.1-t2i-turbo",
            "prompt": "一朵红色的玫瑰花",
            "n": 1,
            "size": "1024x1024",
            "response_format": "url"
        }
        
        response = requests.post(
            f"{base_url}/v1/images/generations",
            headers=headers,
            json=payload,
            timeout=30
        )
        
        print(f"   📊 状态码: {response.status_code}")
        
        if response.status_code == 200:
            print("   ✅ 图像生成成功！")
            data = response.json()
            if 'data' in data and len(data['data']) > 0:
                print(f"   🖼️  生成了 {len(data['data'])} 张图像")
                for i, img in enumerate(data['data']):
                    if 'url' in img:
                        print(f"   🔗 图像 {i+1}: {img['url']}")
            return True
        elif response.status_code == 403:
            error_data = response.json()
            error_msg = error_data.get('error', {}).get('message', '未知错误')
            print(f"   ❌ 权限错误 (403): {error_msg}")
            return False
        else:
            print(f"   ❌ 请求失败: {response.status_code}")
            try:
                error_data = response.json()
                print(f"   📄 错误详情: {error_data}")
            except:
                print(f"   📄 响应内容: {response.text}")
            return False
            
    except Exception as e:
        print(f"   ❌ 请求异常: {str(e)}")
        return False

def main():
    """主函数"""
    print("🔍 API密钥验证工具")
    print("=" * 50)
    
    # 从命令行参数或环境变量获取API密钥
    api_key = None
    
    if len(sys.argv) > 1:
        api_key = sys.argv[1]
    else:
        api_key = os.getenv("NEW_API_KEY")
    
    if not api_key:
        print("❌ 请提供API密钥:")
        print("   方法1: export NEW_API_KEY='your-api-key'")
        print("   方法2: python3 verify_api_key.py 'your-api-key'")
        return
    
    # 测试API密钥
    success = test_api_key(api_key)
    
    print("\n" + "=" * 50)
    if success:
        print("🎉 API密钥验证成功！")
        print("✅ 该密钥支持wanx2.1-t2i-turbo模型")
    else:
        print("❌ API密钥验证失败")
        print("💡 建议:")
        print("   1. 检查API密钥是否正确")
        print("   2. 确认账户已开通图像生成服务")
        print("   3. 验证账户权限和配额")

if __name__ == "__main__":
    main() 