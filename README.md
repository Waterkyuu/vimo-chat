<div align="center">
  <img src="./assets/logo.svg"  width="300"/>
    <h1>Vimo🥁</h1>
  <p><em>
基于 Go 编写的 AI Chat 交互 TUI，快速启动，无模块加载耗时，无需任何环境配置</em></p>
</div>

## 特点

1. 内置的skill
- humanizer - 去除 AI 写作痕迹，让文本更自然
- skill-creator - 创建、改进和评估技能
- mcp-builder - 构建高质量 MCP 服务器
- frontend-design - 独特的 UI 视觉设计指引
- internal-comms - 编写各类内部沟通文案
- karpathy-guidelines - 代码编写简化 skill

2. 内置了三层上下文压缩
- 工具结果剔除
- 本地 Markdown 摘要替换
- LLM级别摘要

3. 记忆机制
- 四层记忆

4. 内置工具调用
- Shell 命令
- Git 命令 （diff、log、status）
- 技能加载
- 文件读写
- Grep 搜索
- sequentialthinking （逐步思考）
- 网络搜索 （配置Key，优先 Exa，降级 Parallel，降级 DuckDuckgo）
- 浏览器自动化 （配置Key，需要 Browserbase Cloud 账号）
  
5. 功能
- 新建对话
- 清空对话
- 历史记录保存
- 主动压缩对话
- MCP 配置
- Skill 添加
- 管理工具
- 主模型和辅助模型设置
- 增强提示词

## 快速开始
无需任何环境，直接运行
```
vimo.exe
```

## 使用指南
| Action | Shortcut |
| :--- | :--- |
