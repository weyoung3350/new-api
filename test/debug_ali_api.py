#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
阿里云API直接调试脚本
绕过New API直接测试阿里云DashScope API
"""

import os
import requests
import json
import time

def test_ali_direct_api():
    """直接测试阿里云API"""
    
    # 从环境变量获取阿里云API密钥
    ali_api_key = os.getenv("ALI_API_KEY")
    
    if not ali_api_key:
        print("❌ 请设置ALI_API_KEY环境变量")
        print("   export ALI_API_KEY='your-ali-dashscope-api-key'")
        return
    
    print("🧪 阿里云DashScope API直接测试工具")
    print("用于诊断API权限和配置问题")
    print("=" * 60)
    print("🔍 直接测试阿里云DashScope API")
    print("=" * 50)
    print(f"🔑 使用API密钥: {ali_api_key[:20]}...")
    
    # 步骤1: 创建图像生成任务
    print("\n1️⃣ 创建图像生成任务...")
    
    headers = {
        'Authorization': f'Bearer {ali_api_key}',
        'Content-Type': 'application/json'
    }
    
    # 尝试多种模型配置
    test_configs = [
        {
            "name": "wanx-v1 (基础模型)",
            "payload": {
                "model": "wanx-v1",
                "input": {
                    "prompt": "一朵红色的玫瑰花"
                },
                "parameters": {
                    "style": "<auto>",
                    "size": "1024*1024",
                    "n": 1
                }
            }
        },
        {
            "name": "wanx2.1-t2i-turbo",
            "payload": {
                "model": "wanx2.1-t2i-turbo",
                "input": {
                    "prompt": "一朵红色的玫瑰花"
                },
                "parameters": {
                    "style": "<auto>",
                    "size": "1024*1024",
                    "n": 1
                }
            }
        },
        {
            "name": "flux-schnell (如果支持)",
            "payload": {
                "model": "flux-schnell",
                "input": {
                    "prompt": "一朵红色的玫瑰花"
                },
                "parameters": {
                    "style": "<auto>",
                    "size": "1024*1024",
                    "n": 1
                }
            }
        }
    ]
    
    for config in test_configs:
        print(f"\n   🧪 测试模型: {config['name']}")
        
        try:
            response = requests.post(
                "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis",
                headers=headers,
                json=config['payload'],
                timeout=30
            )
            
            print(f"   📊 状态码: {response.status_code}")
            
            if response.status_code == 200:
                result = response.json()
                print(f"   ✅ 任务创建成功!")
                print(f"   📄 响应: {json.dumps(result, indent=2, ensure_ascii=False)}")
                
                # 如果成功，尝试查询任务状态
                if "output" in result and "task_id" in result["output"]:
                    task_id = result["output"]["task_id"]
                    print(f"\n2️⃣ 查询任务状态 (Task ID: {task_id})")
                    
                    # 等待一段时间
                    time.sleep(5)
                    
                    status_response = requests.get(
                        f"https://dashscope.aliyuncs.com/api/v1/tasks/{task_id}",
                        headers=headers,
                        timeout=30
                    )
                    
                    print(f"   📊 状态查询状态码: {status_response.status_code}")
                    if status_response.status_code == 200:
                        status_result = status_response.json()
                        print(f"   📄 任务状态: {json.dumps(status_result, indent=2, ensure_ascii=False)}")
                    else:
                        print(f"   ❌ 状态查询失败: {status_response.text}")
                break  # 找到可用的模型就停止测试
                
            else:
                print(f"   ❌ 任务创建失败: {response.status_code}")
                try:
                    error_detail = response.json()
                    print(f"   📄 错误详情: {json.dumps(error_detail, indent=2, ensure_ascii=False)}")
                except:
                    print(f"   📄 错误详情: {response.text}")
                    
        except Exception as e:
            print(f"   ❌ 请求异常: {e}")
    
    # 步骤3: 测试模型列表
    print(f"\n3️⃣ 测试模型列表...")
    
    try:
        response = requests.get(
            "https://dashscope.aliyuncs.com/api/v1/models",
            headers=headers,
            timeout=30
        )
        
        print(f"   📊 状态码: {response.status_code}")
        if response.status_code == 200:
            models = response.json()
            print(f"   📋 模型列表: {json.dumps(models, indent=2, ensure_ascii=False)}")
        else:
            print(f"   ❌ 获取模型列表失败: {response.text}")
            
    except Exception as e:
        print(f"   ❌ 请求异常: {e}")
    
    print("\n" + "=" * 60)
    print("💡 说明:")
    print("   1. 如果任务创建成功但权限错误，说明API密钥有效但权限不足")
    print("   2. 如果任务创建失败，请检查API密钥或账户状态")
    print("   3. 确保阿里云账户已开通DashScope服务和图像生成功能")
    print("   4. 某些模型可能需要特殊权限或付费开通")

if __name__ == "__main__":
    test_ali_direct_api() 