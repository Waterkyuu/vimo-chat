<div align="center">
  <img src="./assets/logo.png"  />
    <h1>Vimo</h1>
  <p><em>
基于 Go 编写的 AI Chat 交互 TUI，使用 Sqlite 实现 OpenAI Memory 机制</em></p>
</div>

## 特点

1. 内置的skill
- xxx
- xxx
- xxx
- xxx
- xxx

2. 内置了三层上下文压缩
- 工具结果剔除
- 本地 Markdown 摘要替换
- LLM级别摘要

3. 内置工具调用
- Bash 命令
- 技能加载
- 文件读写
- Grep 搜索
- 网络搜索 （配置Key，优先Exa，降级 Parallel）
- 浏览器自动化 （配置Key，需要 Browserbase Cloud 账号）
  
4. 功能
- 新建对话
- 清空对话
- 历史记录保存
- 主动压缩对话
- MCP 配置
- Skill 添加
- 管理工具

## 快速开始
```bash
go run ./cmd
```

## 使用指南
| Action | Shortcut |
| :--- | :--- |
