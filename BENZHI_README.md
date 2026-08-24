# FnExec

FnExec 是一个轻量级的无服务器函数执行引擎：函数注册后可按事件触发，触发进入调用
队列，执行器调度到运行实例执行，带冷启动管理、超时、重试与按需扩缩容，并提供
一个浏览器控制台页面。

## 构建

依赖已 vendor，构建无需联网：

```bash
go build -mod=vendor ./...
go vet -mod=vendor ./...
go test -mod=vendor ./...
```

## 运行

```bash
go run -mod=vendor ./cmd/fnexecd -addr 127.0.0.1:8080
```

启动后可用浏览器打开 http://127.0.0.1:8080/ 查看控制台；健康检查地址为
`/healthz`，统计接口为 `/api/stats`，触发函数使用 `POST /api/invoke`。

## 内置函数

- `echo`：回显 `text` 字段
- `concat`：拼接 `left` 与 `right`
- `upper`：转大写
- `failif`：载荷包含 `boom` 时返回错误

## Docker

```bash
./build_benzhi_docker.sh
docker run --rm -p 8080:8080 fnexec-benzhi
```
