# AI SQL 解释器

一个用于分析 MySQL 查询执行计划并提供 AI 优化建议的工具。

## 功能特性

- 解析 MySQL 的 `EXPLAIN` 执行计划
- 自动提取查询涉及的表结构信息
- 通过 OpenAI API 提供智能优化建议
- 支持自定义 OpenAI 配置
- 生成标准 `.env` 配置文件

## 安装

1. 克隆仓库:

```bash
git clone https://github.com/ciclebyte/aiExplain.git
```

2. 安装依赖:

```bash
go mod download
```

3. 生成 `.env` 配置文件:

```bash
go run main.go env
```

## 使用说明

1. 配置 `.env` 文件中的 MySQL 和 OpenAI 参数
2. 运行分析命令:

```bash
go run main.go explain --query "YOUR_SQL_QUERY"
```

## 配置选项

| 环境变量        | 描述                   |
| --------------- | ---------------------- |
| MYSQL_HOST      | MySQL 服务器地址       |
| MYSQL_PORT      | MySQL 服务器端口       |
| MYSQL_USER      | MySQL 用户名           |
| MYSQL_PASSWORD  | MySQL 密码             |
| MYSQL_DATABASE  | MySQL 数据库名         |
| OPENAI_API_KEY  | OpenAI API 密钥        |
| OPENAI_BASE_URL | 自定义 OpenAI API 地址 |
| OPENAI_MODEL    | 使用的 OpenAI 模型     |

## 开发

项目结构:

```
.
├── assets/       # 静态资源
├── cmd/          # 命令行代码
├── resources/    # 嵌入资源
├── go.mod        # Go 模块文件
└── main.go       # 程序入口
```

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

[MIT](LICENSE)
