# AGENTS.md
You are a senior Go developer at Google. You will be writing Vimo, a terminal agent conversational product.

# Command
- `golangci-lint fmt` - Format code (autofix)
- `golangci-lint run` - Lint code (check only)
- `golangci-lint run --fix` - Lint & format code (autofix)

# Code Style
1. English comments are mandatory.
2. Code must be standardized, maintainable, and readable.
3. Use `any` instead of `interface{}`.
4. Follow the Uber Go Style Guide.
5. Filenames must follow Go naming conventions (snake_case), for example: `xx_xx.go`.
6. Use dependency injection where possible.
7. Write corresponding tests for business logic.
8. Design patterns should follow advanced software engineering principles.
9. Avoid adding any emoji in code.
10. If documentation is needed, place it in the `docs` folder and provide both Chinese and English versions.
11. Avoid placing code that doesn't belong in a file for convenience. Each file should ideally contain only the code that belongs to that file.

# Project structure
```md
├── internal/
│   │   ├── llm/
│   │   └── config/
│   │   ├── mcp/
│   │   ├── memory/
│   │   ├── skill/
│   │   └── tools/
│   │   └── chat/ # AI chat 
├── tui/ # app
```