# wanx2.1-t2i-turbo 模型设置指南

## 🎯 快速设置步骤

### 1. 获取阿里云API密钥
1. 访问 [阿里云DashScope控制台](https://dashscope.console.aliyun.com/)
2. 开通DashScope服务
3. 在"API-KEY管理"页面创建新的API密钥
4. 确保账户已开通图像生成服务权限

### 2. 配置New API渠道
1. 访问 http://localhost:3000 (New API管理界面)
2. 登录管理员账户 (用户名: admin)
3. 进入"渠道管理"页面
4. 编辑现有的阿里云渠道:
   - **API密钥**: 更新为真实的DashScope API密钥
   - **模型**: 确保包含 `wanx2.1-t2i-turbo`
   - **状态**: 启用
5. 保存配置

### 3. 测试验证
```bash
# 方法1: 直接测试阿里云API
export ALI_API_KEY="your-real-dashscope-api-key"
python3 debug_ali_api.py

# 方法2: 通过New API测试
export NEW_API_KEY="sk-WFXP99kKWeu9BhV3UiypR6wj2tb2x5d08TLGWgiLHiDG9r8Q"
python3 test_wanx_model.py
```

## 📋 文件说明

| 文件 | 用途 |
|------|------|
| `test_wanx_model.py` | 完整的wanx模型测试脚本 |
| `verify_api_key.py` | 快速API密钥验证工具 |
| `debug_ali_api.py` | 直接测试阿里云API的调试脚本 |
| `test_final_report.md` | 详细的测试报告 |

## ⚠️ 常见问题

### Q: 403错误 "current user api does not support synchronous calls"
**A**: 这表示API密钥权限不足，请确保：
- 使用真实的阿里云DashScope API密钥
- 账户已开通图像生成服务
- API密钥有足够的权限

### Q: 模型列表中看不到wanx模型
**A**: 检查：
- 渠道配置中的模型列表是否包含wanx模型
- 渠道状态是否启用
- 重启New API服务器

### Q: 如何获取阿里云API密钥？
**A**: 
1. 登录阿里云控制台
2. 搜索"DashScope"服务
3. 开通服务并创建API密钥
4. 确保账户余额充足

## 🎉 成功标志

当配置正确时，你应该看到：
- ✅ 模型列表包含wanx相关模型
- ✅ 图像生成请求返回200状态码
- ✅ 获得图像URL或base64数据
- ✅ 测试脚本显示成功率 > 0%

## 📞 支持

如果遇到问题：
1. 查看 `test_final_report.md` 了解详细分析
2. 运行 `debug_ali_api.py` 诊断API连接
3. 检查New API服务器日志
4. 确认阿里云账户状态和权限 