#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
New API平台模型测试脚本
支持的模型：
- paraformer-realtime-8k-v2 (语音识别)
- cosyvoice-v2 (语音合成) 
- text-embedding-v4 (文本嵌入)

使用OpenAI兼容接口进行测试
"""

import os
import json
import time
import base64
import requests
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from pathlib import Path
import logging

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('test_results.log', encoding='utf-8'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

@dataclass
class APIConfig:
    """API配置类"""
    base_url: str = "http://127.0.0.1:3000"  # 请替换为实际的API地址
    api_key: str = "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q"  # 请设置您的API密钥
    timeout: int = 30
    
    def __post_init__(self):
        if not self.api_key:
            # 尝试从环境变量获取
            self.api_key = os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q")
            if not self.api_key:
                logger.warning("请设置API_KEY环境变量或直接在代码中配置")

class NewAPIClient:
    """New API平台的OpenAI兼容客户端"""
    
    def __init__(self, config: APIConfig):
        self.config = config
        self.headers = {
            "Authorization": f"Bearer {config.api_key}",
            "Content-Type": "application/json"
        }
    
    def _make_request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """发送HTTP请求"""
        url = f"{self.config.base_url.rstrip('/')}/{endpoint.lstrip('/')}"
        kwargs.setdefault('timeout', self.config.timeout)
        kwargs.setdefault('headers', self.headers)
        
        logger.info(f"发送请求: {method} {url}")
        response = requests.request(method, url, **kwargs)
        
        if response.status_code != 200:
            logger.error(f"请求失败: {response.status_code} - {response.text}")
        
        return response

class ParaformerTester:
    """Paraformer语音识别测试器"""
    
    def __init__(self, client: NewAPIClient):
        self.client = client
        self.model = "paraformer-realtime-8k-v2"
    
    def test_transcription(self, audio_file_path: str = None, audio_data: bytes = None) -> Dict[str, Any]:
        """测试语音转文字功能"""
        logger.info(f"开始测试 {self.model} 语音识别功能")
        
        # 准备音频数据
        if audio_file_path and os.path.exists(audio_file_path):
            with open(audio_file_path, 'rb') as f:
                audio_data = f.read()
            logger.info(f"加载音频文件: {audio_file_path}")
        elif not audio_data:
            # 生成测试音频数据（这里用空字节作为示例）
            audio_data = b""
            logger.warning("未提供音频数据，使用空数据进行接口测试")
        
        # 使用OpenAI Whisper API兼容接口
        files = {
            'file': ('audio.wav', audio_data, 'audio/wav'),
            'model': (None, self.model),
            'response_format': (None, 'json'),
            'language': (None, 'zh')  # 中文
        }
        
        try:
            start_time = time.time()
            response = self.client._make_request(
                'POST', 
                '/v1/realtime',
                files=files,
                headers={"Authorization": f"Bearer {self.client.config.api_key}"}  # 只保留Authorization
            )
            end_time = time.time()
            
            result = {
                'success': response.status_code == 200,
                'status_code': response.status_code,
                'response_time': round(end_time - start_time, 2),
                'model': self.model
            }
            
            if response.status_code == 200:
                try:
                    data = response.json()
                    result.update({
                        'text': data.get('text', ''),
                        'language': data.get('language', 'unknown'),
                        'duration': data.get('duration', 0)
                    })
                    logger.info(f"识别成功: {data.get('text', '')[:100]}")
                except json.JSONDecodeError:
                    result['raw_response'] = response.text
                    logger.warning("响应不是有效的JSON格式")
            else:
                result['error'] = response.text
                logger.error(f"识别失败: {response.text}")
            
            return result
            
        except Exception as e:
            logger.error(f"测试 {self.model} 时发生异常: {str(e)}")
            return {
                'success': False,
                'error': str(e),
                'model': self.model
            }

class CosyVoiceTester:
    """CosyVoice语音合成测试器"""
    
    def __init__(self, client: NewAPIClient):
        self.client = client
        self.model = "cosyvoice-v2"
    
    def test_speech_synthesis(self, text: str = "你好，这是一个语音合成测试。", voice: str = "zh-CN-XiaoxiaoNeural") -> Dict[str, Any]:
        """测试文字转语音功能"""
        logger.info(f"开始测试 {self.model} 语音合成功能")
        
        # 使用OpenAI TTS API兼容接口
        payload = {
            "model": self.model,
            "input": text,
            "voice": voice,
            "response_format": "mp3",
            "speed": 1.0
        }
        
        try:
            start_time = time.time()
            response = self.client._make_request(
                'POST',
                '/v1/audio/speech',
                json=payload
            )
            end_time = time.time()
            
            result = {
                'success': response.status_code == 200,
                'status_code': response.status_code,
                'response_time': round(end_time - start_time, 2),
                'model': self.model,
                'input_text': text,
                'voice': voice
            }
            
            if response.status_code == 200:
                # 保存音频文件
                output_dir = Path("test_outputs")
                output_dir.mkdir(exist_ok=True)
                
                output_file = output_dir / f"cosyvoice_output_{int(time.time())}.mp3"
                with open(output_file, 'wb') as f:
                    f.write(response.content)
                
                result.update({
                    'audio_size': len(response.content),
                    'output_file': str(output_file)
                })
                logger.info(f"语音合成成功，音频大小: {len(response.content)} 字节")
                logger.info(f"音频文件保存至: {output_file}")
            else:
                result['error'] = response.text
                logger.error(f"语音合成失败: {response.text}")
            
            return result
            
        except Exception as e:
            logger.error(f"测试 {self.model} 时发生异常: {str(e)}")
            return {
                'success': False,
                'error': str(e),
                'model': self.model
            }

class TextEmbeddingTester:
    """文本嵌入测试器"""
    
    def __init__(self, client: NewAPIClient):
        self.client = client
        self.model = "text-embedding-v4"
    
    def test_embedding(self, texts: List[str] = None) -> Dict[str, Any]:
        """测试文本嵌入功能"""
        logger.info(f"开始测试 {self.model} 文本嵌入功能")
        
        if texts is None:
            texts = [
                "这是一个测试文本。",
                "人工智能技术正在快速发展。",
                "Python是一种流行的编程语言。"
            ]
        
        # 使用OpenAI Embeddings API兼容接口
        payload = {
            "model": self.model,
            "input": texts,
            "encoding_format": "float"
        }
        
        try:
            start_time = time.time()
            response = self.client._make_request(
                'POST',
                '/v1/embeddings',
                json=payload
            )
            end_time = time.time()
            
            result = {
                'success': response.status_code == 200,
                'status_code': response.status_code,
                'response_time': round(end_time - start_time, 2),
                'model': self.model,
                'input_count': len(texts)
            }
            
            if response.status_code == 200:
                try:
                    data = response.json()
                    embeddings = data.get('data', [])
                    
                    result.update({
                        'embedding_count': len(embeddings),
                        'embedding_dimension': len(embeddings[0]['embedding']) if embeddings else 0,
                        'usage': data.get('usage', {}),
                        'embeddings_preview': {
                            f'text_{i}': embedding['embedding'][:5]  # 只显示前5个维度
                            for i, embedding in enumerate(embeddings[:3])  # 最多显示3个样本
                        }
                    })
                    logger.info(f"嵌入生成成功，维度: {result['embedding_dimension']}")
                except (json.JSONDecodeError, KeyError, IndexError) as e:
                    result['raw_response'] = response.text
                    logger.warning(f"解析嵌入响应时出错: {str(e)}")
            else:
                result['error'] = response.text
                logger.error(f"嵌入生成失败: {response.text}")
            
            return result
            
        except Exception as e:
            logger.error(f"测试 {self.model} 时发生异常: {str(e)}")
            return {
                'success': False,
                'error': str(e),
                'model': self.model
            }

class TestRunner:
    """测试运行器"""
    
    def __init__(self, config: APIConfig):
        self.config = config
        self.client = NewAPIClient(config)
        self.results = []
    
    def run_all_tests(self, audio_file: str = None, test_text: str = None, embedding_texts: List[str] = None) -> Dict[str, Any]:
        """运行所有模型测试"""
        logger.info("开始运行全部模型测试")
        
        # 测试Paraformer语音识别
        paraformer_tester = ParaformerTester(self.client)
        paraformer_result = paraformer_tester.test_transcription(audio_file)
        self.results.append(paraformer_result)
        
        # 测试CosyVoice语音合成
        #cosyvoice_tester = CosyVoiceTester(self.client)
        #cosyvoice_result = cosyvoice_tester.test_speech_synthesis(
        #    test_text or "你好，这是CosyVoice语音合成测试。"
        #)
        #self.results.append(cosyvoice_result)
        
        # 测试Text Embedding
        #embedding_tester = TextEmbeddingTester(self.client)
        #embedding_result = embedding_tester.test_embedding(embedding_texts)
        #self.results.append(embedding_result)
        
        # 生成测试报告
        report = self.generate_report()
        return report
    
    def generate_report(self) -> Dict[str, Any]:
        """生成测试报告"""
        successful_tests = sum(1 for r in self.results if r.get('success', False))
        total_tests = len(self.results)
        
        report = {
            'test_summary': {
                'total_tests': total_tests,
                'successful_tests': successful_tests,
                'failed_tests': total_tests - successful_tests,
                'success_rate': f"{(successful_tests/total_tests*100):.1f}%" if total_tests > 0 else "0%"
            },
            'test_results': self.results,
            'test_time': time.strftime("%Y-%m-%d %H:%M:%S")
        }
        
        # 保存测试报告
        report_file = f"test_report_{int(time.time())}.json"
        with open(report_file, 'w', encoding='utf-8') as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
        
        logger.info(f"测试报告已保存至: {report_file}")
        return report

def print_results(report: Dict[str, Any]):
    """打印测试结果"""
    print("\n" + "="*60)
    print("📊 NEW API 平台模型测试报告")
    print("="*60)
    
    summary = report['test_summary']
    print(f"🔍 测试概览:")
    print(f"   总测试数: {summary['total_tests']}")
    print(f"   成功测试: {summary['successful_tests']}")
    print(f"   失败测试: {summary['failed_tests']}")
    print(f"   成功率: {summary['success_rate']}")
    print(f"   测试时间: {report['test_time']}")
    
    print(f"\n📋 详细结果:")
    for i, result in enumerate(report['test_results'], 1):
        status = "✅ 成功" if result.get('success') else "❌ 失败"
        print(f"\n{i}. {result.get('model', 'Unknown')} - {status}")
        print(f"   响应时间: {result.get('response_time', 'N/A')}秒")
        
        if result.get('success'):
            if 'text' in result:  # Paraformer
                print(f"   识别文本: {result['text'][:100]}...")
            elif 'audio_size' in result:  # CosyVoice
                print(f"   音频大小: {result['audio_size']} 字节")
                print(f"   输出文件: {result['output_file']}")
            elif 'embedding_dimension' in result:  # Embedding
                print(f"   嵌入维度: {result['embedding_dimension']}")
                print(f"   处理文本数: {result['input_count']}")
        else:
            print(f"   错误信息: {result.get('error', 'Unknown error')}")

def main():
    """主函数"""
    print("🚀 New API 平台模型功能测试脚本")
    print("支持模型: paraformer-realtime-8k-v2, cosyvoice-v2, text-embedding-v4")
    print("-" * 60)
    
    # 配置API
    config = APIConfig(
        base_url="http://127.0.0.1:3000",  # 请替换为实际的API地址
        api_key=os.getenv("NEW_API_KEY", "sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q"),  # 请设置环境变量或直接填写
        timeout=60
    )
    
    if not config.api_key:
        print("⚠️  请设置NEW_API_KEY环境变量或在代码中配置API密钥")
        print("   export NEW_API_KEY='your-api-key'")
        return
    
    # 创建测试运行器
    runner = TestRunner(config)
    
    # 准备测试数据
    test_params = {
        'audio_file': None,  # 可以指定音频文件路径
        'test_text': '欢迎使用New API平台，这是一个语音合成测试。',
        'embedding_texts': [
            'New API平台提供多种AI模型服务',
            '语音识别和语音合成技术',
            '文本嵌入向量化处理'
        ]
    }
    
    try:
        # 运行测试
        report = runner.run_all_tests(**test_params)
        
        # 显示结果
        print_results(report)
        
    except KeyboardInterrupt:
        print("\n\n⏹️  测试被用户中断")
    except Exception as e:
        logger.error(f"测试过程中发生异常: {str(e)}")
        print(f"\n❌ 测试失败: {str(e)}")

if __name__ == "__main__":
    main() 