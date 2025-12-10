# 第一步：构建阶段，用 Go 官方镜像编译
FROM golang:1.24 AS builder

WORKDIR /app

# 先拷贝 go.mod / go.sum，利用缓存
COPY go.mod go.sum ./
RUN go mod download

# 再拷贝全部源码
COPY . .

# 编译二进制
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

# 第二步：运行阶段，用更小的基础镜像
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/app .

# 对外暴露 8081 端口（仅文档作用）
EXPOSE 8081

# 容器启动执行的命令
ENTRYPOINT ["./app"]
