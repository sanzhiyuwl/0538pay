# 0538pay-api

0538pay 支付平台后端（Go 全栈 · B 方案独立重写）。功能对齐彩虹易支付(epay)，
表结构 / 插件契约 / API 约定全部自研。技术选型见 `../docs/后端技术选型.txt`，
架构规划见 `../docs/go后端起步架构.txt`。

## 技术栈

Go 1.22+ · Gin · GORM(MySQL) · JWT · shopspring/decimal · Viper · zap

## 目录

```
cmd/server   服务入口
cmd/seed     初始化首个管理员
internal/    私有代码：config/router/middleware/model/dto/repository/service/handler/channel
pkg/         可复用工具：resp/money/jwtauth
configs/     配置样例
```

## 本机准备

1. 安装 Go 1.22+：https://go.dev/dl/ （装完 `go version` 能输出版本）
2. 确保 MySQL 运行（phpStudy 自带），建库：

   ```sql
   CREATE DATABASE pay0538 DEFAULT CHARSET utf8mb4;
   ```

3. 按本机改 `configs/config.yaml` 里的数据库 DSN 和 jwt.secret。

## 运行

```bash
# 拉依赖
go mod tidy

# 初始化首个管理员（会自动建表）
go run ./cmd/seed -config ./configs -user admin -pass admin888

# 启动服务
go run ./cmd/server -config ./configs
```

## 验证最小闭环

```bash
# 探活
curl http://127.0.0.1:8080/health

# 登录拿 token
curl -X POST http://127.0.0.1:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin888"}'

# 用 token 查订单列表
curl http://127.0.0.1:8080/api/admin/orders?page=1&pageSize=20 \
  -H "Authorization: Bearer <上一步返回的 token>"
```

## 下一步

- 前端 admin-web 新建 `src/lib/api/` 客户端，订单页从 mock 换真接口。
- 按业务域补接口（merchants / channels / settle / risk ...）。
- 支付渠道插件在 `internal/channel/<name>/` 下实现 PaymentChannel 并 init 注册。
