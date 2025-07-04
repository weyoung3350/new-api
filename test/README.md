# 测试脚本目录

本目录包含了 New API 项目的各种测试脚本和工具。

## 🎨 图像生成测试

### wanx2.1-t2i-turbo 模型测试
- **`test_wanx_model.py`** - 完整的 wanx2.1-t2i-turbo 图像生成模型测试脚本
  - 支持批量测试多个提示词
  - 详细的日志记录和错误处理
  - 自动保存测试结果为JSON格式
  
- **`verify_api_key.py`** - 快速API密钥验证工具
  - 验证API密钥的有效性
  - 检查模型可用性
  - 快速诊断连接问题

- **`debug_ali_api.py`** - 阿里云API直接调试脚本
  - 绕过New API直接测试阿里云DashScope API
  - 支持多种模型配置测试
  - 用于问题诊断和权限验证

### 测试输出
- **`wanx_test_outputs/`** - wanx模型测试的输出目录
  - 包含详细的测试报告JSON文件
  - 测试结果和错误日志
- **`wanx_test_results.log`** - wanx模型测试的日志文件

## 🎵 音频相关测试

- **`test_cosyvoice_websocket.py`** - CosyVoice WebSocket API测试
- **`test_speech_recognition.py`** - 语音识别功能测试
- **`test_realtime_websocket.html`** - 实时WebSocket测试页面

## 🔧 通用测试工具

- **`test_new_api_models.py`** - 通用模型测试脚本
- **`quick_test_example.py`** - 快速测试示例
- **`requirements.txt`** - Python依赖包列表

## 📊 测试数据

- **`test_outputs/`** - 通用测试输出目录
- **`test_results.log`** - 通用测试日志
- **`test_report_*.json`** - 历史测试报告

## 🚀 使用方法

### 运行 wanx2.1-t2i-turbo 测试

```bash
# 设置API密钥
export NEW_API_KEY="your-new-api-key"

# 运行完整测试
python3 test_wanx_model.py

# 快速验证API密钥
python3 verify_api_key.py

# 直接测试阿里云API（需要阿里云API密钥）
export ALI_API_KEY="your-ali-dashscope-api-key"
python3 debug_ali_api.py
```

### 安装依赖

```bash
pip install -r requirements.txt
```

## 📋 测试结果说明

- 测试脚本会自动生成详细的日志文件
- JSON格式的测试报告包含完整的请求/响应数据
- 所有测试输出都保存在对应的输出目录中

## 🔍 故障排除

1. **API密钥问题**: 使用 `verify_api_key.py` 验证密钥有效性
2. **权限问题**: 使用 `debug_ali_api.py` 直接测试阿里云API权限
3. **网络问题**: 检查日志文件中的详细错误信息

## 📚 相关文档

详细的测试报告和设置指南请查看 `docs/` 目录：
- `docs/FINAL_TEST_REPORT.md` - 最终测试报告
- `docs/SETUP_GUIDE.md` - 详细设置指南 