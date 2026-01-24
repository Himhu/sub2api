# Sub2API 服务器部署指南

本文档描述如何从你的 GitHub 仓库部署 Sub2API 到生产服务器，包括蓝绿部署策略。

---

## 目录

1. [服务器环境准备](#1-服务器环境准备)
2. [首次部署](#2-首次部署)
3. [蓝绿部署架构](#3-蓝绿部署架构)
4. [日常更新流程](#4-日常更新流程)
5. [同步官方更新](#5-同步官方更新)
6. [回滚操作](#6-回滚操作)
7. [常用命令速查](#7-常用命令速查)

---

## 1. 服务器环境准备

### 1.1 系统要求

- Linux 服务器 (Ubuntu 20.04+ / Debian 11+ / CentOS 8+)
- 2GB+ 内存
- 10GB+ 磁盘空间

### 1.2 安装依赖

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y git curl wget

# 安装 Go 1.21+
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 安装 Node.js 18+
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# 安装 pnpm
npm install -g pnpm

# 安装 PostgreSQL 15
sudo apt install -y postgresql-15 postgresql-contrib-15

# 安装 Redis
sudo apt install -y redis-server
```

### 1.3 配置 PostgreSQL

```bash
# 启动 PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql << EOF
CREATE USER sub2api WITH PASSWORD 'your_secure_password';
CREATE DATABASE sub2api OWNER sub2api;
GRANT ALL PRIVILEGES ON DATABASE sub2api TO sub2api;
EOF
```

### 1.4 配置 Redis

```bash
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

---

## 2. 首次部署

### 2.1 创建部署目录

```bash
sudo mkdir -p /opt/sub2api
sudo chown $USER:$USER /opt/sub2api
cd /opt/sub2api
```

### 2.2 克隆你的仓库

```bash
# 克隆你的仓库（使用 dev 分支）
git clone -b dev https://github.com/Himhu/sub2api.git current

# 添加官方仓库作为 upstream（用于同步更新）
cd current
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
```

### 2.3 构建应用

```bash
cd /opt/sub2api/current

# 构建前端
cd frontend
pnpm install
pnpm run build

# 构建后端
cd ../backend
go build -tags embed -o sub2api ./cmd/server
```

### 2.4 配置应用

```bash
cd /opt/sub2api/current/backend

# 复制配置模板
cp ../deploy/config.example.yaml config.yaml

# 编辑配置（重点修改以下内容）
nano config.yaml
```

**必须修改的配置项**：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "sub2api"
  password: "your_secure_password"  # 改为你的数据库密码
  dbname: "sub2api"

redis:
  host: "localhost"
  port: 6379
  password: ""  # 如有密码则填写

jwt:
  secret: "生成一个随机字符串"  # openssl rand -hex 32
```

### 2.5 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/sub2api.service << 'EOF'
[Unit]
Description=Sub2API Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/sub2api/current/backend
ExecStart=/opt/sub2api/current/backend/sub2api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl start sub2api
sudo systemctl enable sub2api
```

### 2.6 验证部署

```bash
# 检查服务状态
sudo systemctl status sub2api

# 查看日志
sudo journalctl -u sub2api -f

# 测试访问
curl http://localhost:8080
```

---

## 3. 蓝绿部署架构

### 3.1 目录结构

```
/opt/sub2api/
├── blue/                 # 蓝色环境
│   ├── frontend/
│   └── backend/
├── green/                # 绿色环境
│   ├── frontend/
│   └── backend/
├── current -> blue/      # 符号链接指向当前活跃环境
└── shared/
    └── config.yaml       # 共享配置文件
```

### 3.2 初始化蓝绿环境

```bash
cd /opt/sub2api

# 重命名 current 为 blue
mv current blue

# 创建 green 环境
git clone -b dev https://github.com/Himhu/sub2api.git green
cd green
git remote add upstream https://github.com/Wei-Shaw/sub2api.git

# 创建共享配置目录
mkdir -p /opt/sub2api/shared
mv /opt/sub2api/blue/backend/config.yaml /opt/sub2api/shared/

# 创建配置软链接
ln -sf /opt/sub2api/shared/config.yaml /opt/sub2api/blue/backend/config.yaml
ln -sf /opt/sub2api/shared/config.yaml /opt/sub2api/green/backend/config.yaml

# 创建 current 软链接
ln -sf /opt/sub2api/blue /opt/sub2api/current
```

### 3.3 更新 systemd 服务

```bash
sudo tee /etc/systemd/system/sub2api.service << 'EOF'
[Unit]
Description=Sub2API Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/sub2api/current/backend
ExecStart=/opt/sub2api/current/backend/sub2api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
```

---

## 4. 日常更新流程（SCP 方式）

> **注意**：由于服务器无法直接访问 GitHub，采用本地打包 + SCP 上传的方式部署。

### 4.1 本地开发

```bash
# 切换到 dev 分支
cd /Users/a/Desktop/Usb2api/sub2api
git checkout dev

# 修改代码...

# 提交代码
git add .
git commit -m "feat: 你的功能描述"
git push origin dev
```

### 4.2 本地打包

```bash
cd /Users/a/Desktop/Usb2api/sub2api

# 打包（排除不需要的文件）
tar -czvf sub2api-dev.tar.gz \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='backend/sub2api' \
    --exclude='*.tar.gz' \
    .
```

### 4.3 上传到服务器

```bash
scp sub2api-dev.tar.gz ubuntu@152.136.216.110:/opt/sub2api/
```

### 4.4 服务器部署

```bash
# SSH 登录服务器
ssh ubuntu@152.136.216.110

# 查看当前环境
readlink /opt/sub2api/current
# 输出 /opt/sub2api/blue 或 /opt/sub2api/green

# 确定目标环境（与当前相反）
# 如果当前是 blue，则目标是 green，反之亦然
TARGET=green  # 或 blue

# 清空目标环境并解压
sudo rm -rf /opt/sub2api/$TARGET/*
sudo tar -xzf /opt/sub2api/sub2api-dev.tar.gz -C /opt/sub2api/$TARGET/
sudo chown -R ubuntu:ubuntu /opt/sub2api/$TARGET

# 创建配置软链接
rm -f /opt/sub2api/$TARGET/backend/config.yaml
rm -rf /opt/sub2api/$TARGET/backend/data
ln -sf /opt/sub2api/shared/config.yaml /opt/sub2api/$TARGET/backend/config.yaml
ln -sf /opt/sub2api/shared/data /opt/sub2api/$TARGET/backend/data

# 运行部署脚本
/opt/sub2api/deploy.sh
```

### 4.5 服务器部署脚本

服务器上的 `/opt/sub2api/deploy.sh`：

```bash
#!/bin/bash
set -e

# 确定当前和目标环境
CURRENT=$(readlink /opt/sub2api/current)
if [[ "$CURRENT" == *"blue"* ]]; then
    TARGET="green"
    CURRENT_ENV="blue"
else
    TARGET="blue"
    CURRENT_ENV="green"
fi

echo "当前环境: $CURRENT_ENV"
echo "目标环境: $TARGET"

# 进入目标环境
cd /opt/sub2api/$TARGET

# 删除 macOS 元数据文件
echo "清理 macOS 元数据文件..."
find . -name "._*" -delete

# 构建前端
echo "构建前端..."
cd frontend
npx pnpm install
npx pnpm run build

# 构建后端
echo "构建后端..."
cd ../backend
go build -tags embed -o sub2api ./cmd/server

# 切换环境
echo "切换到 $TARGET 环境..."
sudo ln -sfn /opt/sub2api/$TARGET /opt/sub2api/current

# 重启服务
echo "重启服务..."
sudo systemctl restart sub2api

# 等待启动
sleep 5

# 检查状态
if systemctl is-active --quiet sub2api; then
    echo "部署成功！当前环境: $TARGET"
else
    echo "部署失败，回滚到 $CURRENT_ENV"
    sudo ln -sfn /opt/sub2api/$CURRENT_ENV /opt/sub2api/current
    sudo systemctl restart sub2api
    exit 1
fi
```

---

## 5. 同步官方更新（不破坏二次开发）

> **核心原则**：main 分支跟踪官方，dev 分支用于二次开发。通过 merge 合并官方更新到 dev。

### 5.1 分支策略说明

```
官方仓库 (upstream/main)
        │
        ▼
你的 main 分支 ──────────────── 只用于同步官方，不做修改
        │
        ▼ (merge)
你的 dev 分支 ───────────────── 二次开发在这里
```

### 5.2 同步官方更新步骤

**在本地执行**：

```bash
cd /Users/a/Desktop/Usb2api/sub2api

# 1. 切换到 main 分支
git checkout main

# 2. 拉取官方最新代码
git fetch upstream

# 3. 合并官方更新到 main
git merge upstream/main

# 4. 推送 main 到你的仓库
git push origin main

# 5. 切换到 dev 分支
git checkout dev

# 6. 合并 main 到 dev（这一步可能有冲突）
git merge main
```

### 5.3 解决合并冲突

如果出现冲突，Git 会提示哪些文件有冲突：

```bash
# 查看冲突文件
git status

# 编辑冲突文件，找到类似这样的标记：
# <<<<<<< HEAD
# 你的代码
# =======
# 官方的代码
# >>>>>>> main

# 手动选择保留哪部分代码，或合并两者
# 然后删除冲突标记

# 标记冲突已解决
git add <冲突文件>

# 完成合并
git commit -m "merge: 同步官方更新 vX.X.X"

# 推送到你的仓库
git push origin dev
```

### 5.4 冲突处理建议

| 文件类型 | 建议处理方式 |
|---------|-------------|
| 配置文件 (config.yaml) | 保留你的修改，参考官方新增配置项 |
| 前端组件 | 根据功能决定，通常保留你的 UI 修改 |
| 后端 API | 仔细对比，确保不破坏官方新功能 |
| 数据库迁移 | 保留两者，确保迁移顺序正确 |

### 5.5 同步后部署

```bash
# 本地打包
tar -czvf sub2api-dev.tar.gz \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='backend/sub2api' \
    --exclude='*.tar.gz' \
    .

# 上传到服务器
scp sub2api-dev.tar.gz ubuntu@152.136.216.110:/opt/sub2api/

# 服务器部署（参考 4.4 节）
```

---

## 6. 回滚操作

### 6.1 快速回滚（切换环境）

```bash
#!/bin/bash
# /opt/sub2api/rollback.sh

CURRENT=$(readlink /opt/sub2api/current)
if [[ "$CURRENT" == *"blue"* ]]; then
    TARGET="green"
else
    TARGET="blue"
fi

echo "回滚到 $TARGET 环境..."
ln -sfn /opt/sub2api/$TARGET /opt/sub2api/current
sudo systemctl restart sub2api
echo "回滚完成"
```

### 6.2 回滚到指定版本

```bash
cd /opt/sub2api/current

# 查看提交历史
git log --oneline -10

# 回滚到指定提交
git reset --hard <commit-hash>

# 重新构建
cd frontend && pnpm install && pnpm run build
cd ../backend && go build -tags embed -o sub2api ./cmd/server

# 重启服务
sudo systemctl restart sub2api
```

---

## 7. 常用命令速查

### 服务管理

```bash
# 启动服务
sudo systemctl start sub2api

# 停止服务
sudo systemctl stop sub2api

# 重启服务
sudo systemctl restart sub2api

# 查看状态
sudo systemctl status sub2api

# 查看日志
sudo journalctl -u sub2api -f

# 查看最近100行日志
sudo journalctl -u sub2api -n 100
```

### Git 操作

```bash
# 查看当前分支
git branch

# 查看远程仓库
git remote -v

# 查看提交历史
git log --oneline -10

# 查看当前状态
git status
```

### 环境检查

```bash
# 查看当前活跃环境
readlink /opt/sub2api/current

# 检查端口占用
sudo lsof -i :8080

# 检查进程
ps aux | grep sub2api
```

---

## 附录：完整部署检查清单

- [ ] PostgreSQL 已安装并运行
- [ ] Redis 已安装并运行
- [ ] Go 1.21+ 已安装
- [ ] Node.js 18+ 已安装
- [ ] pnpm 已安装
- [ ] 数据库已创建
- [ ] 配置文件已正确设置
- [ ] systemd 服务已创建
- [ ] 防火墙已开放 8080 端口
- [ ] 蓝绿环境已初始化
- [ ] 部署脚本已创建并测试
