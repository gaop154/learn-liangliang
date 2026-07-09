# learn-liangliangleang 技术内容归档 📚

原始网站： [learn.lianglianglee.com](https://learn.lianglianglee.com)

推荐访问： [learn-liangliang.wenxuanhe.top](https://learn-liangliang.wenxuanhe.top) (我们对网站进行了性能优化，并且增加了一些趣味功能)

---

## 1. 项目说明

本项目对原始网站内容进行了系统性备份和整理。learn.lianglianglee.com 是一个极具价值的中文技术学习网站，聚合了大量高质量的技术专栏和精选文章，涵盖后端开发、分布式系统、DevOps、架构设计、面试指南等多个领域。由于服务器到期、费用等问题，原站点有随时关闭的风险，宝贵的技术资料可能会丢失。备用站点将长期维护，确保资料持久保存，不会因原站点关闭而丢失。

## 2. 网站内容介绍 📝

**首页 🏠：聚合导航，快速访问各类技术内容。**
<p align="center">
  <img src="img/index.png" />
</p>

**专栏 📖：收录数十个高质量技术专栏，涵盖后端、分布式、架构、前端、AI 等方向。**
<p align="center">
  <img src="img/专栏.png" />
</p>

**文章 📰：精选技术文章，内容包括数据库、微服务、缓存、面试、架构等实战与理论。**
<p align="center">
  <img src="img/文章.png" />
</p>

**极客时间 ⏰：极客时间等平台的部分优质专栏归档，便于系统性学习。**
<p align="center">
  <img src="img/极客时间.png" />
</p>

**PDF 📄：部分资料以 PDF 形式归档，方便离线阅读。**
<p align="center">
  <img src="img/PDF.png" />
</p>

**恋爱必修课 💌：非技术类专栏归档。**
<p align="center">
  <img src="img/恋爱必修课.png" />
</p>

## 3. 项目目录说明 🗃️

- **专栏/**：技术专栏归档
- **文章/**：精选技术文章归档
- **极客时间/**：极客时间专栏归档
- **PDF/**：PDF 资料归档
- **恋爱必修课/**：非技术类专栏
- **static/**、**assets/**：静态资源与前端样式
- **img/**：页面截图与图片资源
- **index.html**：网站首页
- 其他辅助脚本和配置文件

## 4. Docker 部署 🚀

第一版部署使用 Docker Compose 编排：

- `gateway`：Caddy 入口，对外暴露 80 端口。
- `web`：Nginx 静态站点，托管现有 HTML、PDF、图片和前端脚本。
- `api`：Go 后端 API，提供登录和阅读进度同步。
- `db`：PostgreSQL，使用 `postgres_data` volume 持久化数据。

### 4.1 CentOS 安装 Docker

```bash
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable docker
sudo systemctl start docker
```

放行端口：

```bash
sudo firewall-cmd --add-service=http --permanent
sudo firewall-cmd --add-service=https --permanent
sudo firewall-cmd --reload
```

云服务器安全组也需要放行 80/443。生产环境建议使用域名访问，Caddy 会自动申请 HTTPS 证书。

### 4.2 配置环境变量

```bash
cp .env.example .env
```

然后编辑 `.env`，至少替换：

- `SITE_DOMAIN`
- `ACME_EMAIL`
- `POSTGRES_PASSWORD`
- `DATABASE_URL` 中的密码
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`

生产环境保持 `APP_COOKIE_SECURE=true`；只有本地 HTTP 调试时才临时改为 `false`。

### 4.3 启动

```bash
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
docker compose logs -f api
```

浏览器访问：

```text
https://你的域名
```

登录入口：

```text
https://你的域名/login.html
```

阅读记录入口：

```text
https://你的域名/reading-history.html
```

### 4.4 PostgreSQL 备份

```bash
docker compose exec db sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > backup.sql
```

只要不删除 `postgres_data` volume，重建 `web` / `api` 容器不会丢失阅读记录。

## 5. 免责声明 ⚠️

本项目仅用于技术学习与资料备份，所有内容版权归原作者所有，未经许可，请勿用于商业用途。

## 6. 相关信息

[![Star History Chart](https://api.star-history.com/svg?repos=xixiwenxuanhe/learn-liangliang&type=Date)](https://www.star-history.com/#xixiwenxuanhe/learn-liangliang&Date)

## 7. 许可证 📝

本项目采用 MIT License 开源，详见 [LICENSE](./LICENSE) 文件。
