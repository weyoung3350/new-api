#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
wanx2.1-t2i-turbo 模型测试脚本
测试阿里云通义万相图像生成模型的API接口
"""

import os
import json
import time
import requests
from typing import Dict, Any, Optional
from dataclasses import dataclass
import logging
from pathlib import Path

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('wanx_test_results.log', encoding='utf-8'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

@dataclass
class APIConfig:
    """API配置类"""
    base_url: str = "http://localhost:3000"
    api_key: str = ""
    timeout: int = 60
    
    def __post_init__(self):
        if not self.api_key:
            # 尝试从环境变量获取
            self.api_key = os.getenv("NEW_API_KEY", "sk-test-key")
            if not self.api_key or self.api_key == "sk-test-key":
                logger.warning("请设置NEW_API_KEY环境变量或在代码中配置API密钥")

class WanxModelTester:
    """wanx2.1-t2i-turbo 模型测试器"""
    
    def __init__(self, config: APIConfig):
        self.config = config
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {config.api_key}',
            'Content-Type': 'application/json',
            'User-Agent': 'wanx-model-tester/1.0'
        })
        
        # 确保输出目录存在
        self.output_dir = Path("test_outputs")
        self.output_dir.mkdir(exist_ok=True)
    
    def test_model_availability(self) -> bool:
        """测试模型是否可用"""
        try:
            url = f"{self.config.base_url}/v1/models"
            response = self.session.get(url, timeout=self.config.timeout)
            
            if response.status_code == 200:
                models = response.json()
                model_names = [model['id'] for model in models.get('data', [])]
                
                # 检查是否包含wanx相关模型
                wanx_models = [m for m in model_names if 'wanx' in m.lower()]
                
                logger.info(f"✅ 模型列表获取成功，共发现 {len(model_names)} 个模型")
                if wanx_models:
                    logger.info(f"🎨 发现WANX相关模型: {wanx_models}")
                else:
                    logger.info("📝 未发现WANX相关模型，将使用通用图像生成接口")
                
                return True
            else:
                logger.error(f"❌ 获取模型列表失败: {response.status_code} - {response.text}")
                return False
                
        except Exception as e:
            logger.error(f"❌ 测试模型可用性失败: {str(e)}")
            return False
    
    def test_image_generation(self, prompt: str, model: str = "wanx2.1-t2i-turbo") -> Dict[str, Any]:
        """测试图像生成功能"""
        logger.info(f"🎨 开始测试图像生成 - 模型: {model}")
        logger.info(f"📝 提示词: {prompt}")
        
        try:
            url = f"{self.config.base_url}/v1/images/generations"
            
            payload = {
                "model": model,
                "prompt": prompt,
                "n": 1,
                "size": "1024x1024",
                "quality": "standard",
                "response_format": "url"
            }
            
            start_time = time.time()
            response = self.session.post(
                url, 
                json=payload, 
                timeout=self.config.timeout
            )
            end_time = time.time()
            
            response_time = end_time - start_time
            
            result = {
                "success": False,
                "response_time": response_time,
                "status_code": response.status_code,
                "model": model,
                "prompt": prompt
            }
            
            if response.status_code == 200:
                data = response.json()
                result.update({
                    "success": True,
                    "response_data": data,
                    "image_urls": [img.get('url') for img in data.get('data', [])],
                    "revised_prompt": data.get('data', [{}])[0].get('revised_prompt', '')
                })
                
                logger.info(f"✅ 图像生成成功！")
                logger.info(f"⏱️  响应时间: {response_time:.2f}秒")
                logger.info(f"🖼️  生成图像数量: {len(result['image_urls'])}")
                
                if result['revised_prompt']:
                    logger.info(f"📝 优化后提示词: {result['revised_prompt']}")
                
                # 保存结果
                self._save_test_result(result)
                
            else:
                result.update({
                    "error": response.text,
                    "headers": dict(response.headers)
                })
                logger.error(f"❌ 图像生成失败: {response.status_code}")
                logger.error(f"📄 错误信息: {response.text}")
            
            return result
            
        except Exception as e:
            logger.error(f"❌ 图像生成测试异常: {str(e)}")
            return {
                "success": False,
                "error": str(e),
                "model": model,
                "prompt": prompt
            }
    
    def test_multiple_prompts(self, prompts: list, model: str = "wanx2.1-t2i-turbo") -> Dict[str, Any]:
        """测试多个提示词"""
        logger.info(f"🔄 开始批量测试 - 共 {len(prompts)} 个提示词")
        
        results = []
        successful_tests = 0
        
        for i, prompt in enumerate(prompts, 1):
            logger.info(f"\n--- 测试 {i}/{len(prompts)} ---")
            result = self.test_image_generation(prompt, model)
            results.append(result)
            
            if result.get('success'):
                successful_tests += 1
            
            # 避免请求过于频繁
            time.sleep(1)
        
        summary = {
            "total_tests": len(prompts),
            "successful_tests": successful_tests,
            "failed_tests": len(prompts) - successful_tests,
            "success_rate": (successful_tests / len(prompts)) * 100,
            "results": results
        }
        
        logger.info(f"\n📊 批量测试完成:")
        logger.info(f"✅ 成功: {successful_tests}/{len(prompts)} ({summary['success_rate']:.1f}%)")
        logger.info(f"❌ 失败: {summary['failed_tests']}/{len(prompts)}")
        
        return summary
    
    def _save_test_result(self, result: Dict[str, Any]):
        """保存测试结果到文件"""
        timestamp = time.strftime("%Y%m%d_%H%M%S")
        filename = f"wanx_test_{timestamp}.json"
        filepath = self.output_dir / filename
        
        try:
            with open(filepath, 'w', encoding='utf-8') as f:
                json.dump(result, f, ensure_ascii=False, indent=2)
            logger.info(f"💾 测试结果已保存: {filepath}")
        except Exception as e:
            logger.error(f"❌ 保存测试结果失败: {str(e)}")
    
    def run_comprehensive_test(self) -> Dict[str, Any]:
        """运行综合测试"""
        logger.info("🚀 开始 wanx2.1-t2i-turbo 模型综合测试")
        logger.info("=" * 60)
        
        # 1. 测试模型可用性
        logger.info("1️⃣ 测试模型可用性...")
        if not self.test_model_availability():
            return {"success": False, "error": "模型不可用"}
        
        # 2. 准备测试提示词
        test_prompts = [
            "一只可爱的小猫咪坐在阳光下的花园里",
            "现代城市夜景，霓虹灯闪烁，高楼大厦林立",
            "古代中国山水画风格，山峦叠嶂，云雾缭绕",
            "科幻未来世界，飞行汽车在空中穿梭",
            "温馨的咖啡厅内部，暖黄色灯光，书架和植物"
        ]
        
        # 3. 执行图像生成测试
        logger.info(f"\n2️⃣ 开始图像生成测试...")
        test_results = self.test_multiple_prompts(test_prompts)
        
        # 4. 生成最终报告
        final_report = {
            "test_type": "wanx2.1-t2i-turbo 模型综合测试",
            "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
            "config": {
                "base_url": self.config.base_url,
                "model": "wanx2.1-t2i-turbo",
                "timeout": self.config.timeout
            },
            "results": test_results,
            "summary": {
                "total_prompts": len(test_prompts),
                "success_rate": test_results.get('success_rate', 0),
                "average_response_time": self._calculate_average_response_time(test_results.get('results', []))
            }
        }
        
        # 保存最终报告
        self._save_final_report(final_report)
        
        return final_report
    
    def _calculate_average_response_time(self, results: list) -> float:
        """计算平均响应时间"""
        if not results:
            return 0.0
        
        successful_results = [r for r in results if r.get('success') and 'response_time' in r]
        if not successful_results:
            return 0.0
        
        total_time = sum(r['response_time'] for r in successful_results)
        return total_time / len(successful_results)
    
    def _save_final_report(self, report: Dict[str, Any]):
        """保存最终测试报告"""
        timestamp = time.strftime("%Y%m%d_%H%M%S")
        filename = f"wanx_final_report_{timestamp}.json"
        filepath = self.output_dir / filename
        
        try:
            with open(filepath, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            logger.info(f"📋 最终测试报告已保存: {filepath}")
        except Exception as e:
            logger.error(f"❌ 保存最终报告失败: {str(e)}")

def main():
    """主函数"""
    print("🎨 wanx2.1-t2i-turbo 模型测试脚本")
    print("测试阿里云通义万相图像生成模型")
    print("-" * 60)
    
    # 配置API
    config = APIConfig(
        base_url="http://localhost:3000",
        api_key=os.getenv("NEW_API_KEY", "sk-test-key"),
        timeout=60
    )
    
    if not config.api_key or config.api_key == "sk-test-key":
        print("⚠️  请设置NEW_API_KEY环境变量或在代码中配置API密钥")
        print("   export NEW_API_KEY='your-api-key'")
        return
    
    # 创建测试器
    tester = WanxModelTester(config)
    
    try:
        # 运行综合测试
        report = tester.run_comprehensive_test()
        
        # 显示结果摘要
        print("\n" + "=" * 60)
        print("📊 测试结果摘要:")
        print(f"✅ 成功率: {report['summary']['success_rate']:.1f}%")
        print(f"⏱️  平均响应时间: {report['summary']['average_response_time']:.2f}秒")
        print(f"📝 总测试数: {report['summary']['total_prompts']}")
        print("=" * 60)
        
        if report['results']['success_rate'] > 0:
            print("🎉 测试完成！部分或全部测试成功")
        else:
            print("❌ 测试失败！请检查配置和网络连接")
            
    except KeyboardInterrupt:
        print("\n\n⏹️  测试被用户中断")
    except Exception as e:
        logger.error(f"测试过程中发生异常: {str(e)}")
        print(f"\n❌ 测试失败: {str(e)}")

if __name__ == "__main__":
    main() 