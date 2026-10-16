[English](../README.md) | **中文** | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow、sFlow 和 IPFIX 采集器与流量分析器

一个程序搞定 sFlow、NetFlow 和 IPFIX 流量分析：谁在占用带宽、流量去了哪里、统计结果与设备自身的接口计数器是否对得上，在 Web 界面和终端界面中展示。

## 快速开始

```
# Linux、macOS：下载到 ./traffic66 并开始采集
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows（PowerShell）：同上
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

终端会打印 Web 界面地址，以及设备要指向的 IP 和 UDP 端口（[设备配置](MANUAL.zh.md#device-configuration)）。用 `admin` / `traffic66` 登录并设置新密码。只想先看看？用模拟设备跑演示：

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

端口：TCP 8066 Web 界面，UDP 6343 sFlow、2055 NetFlow、4739 IPFIX（每个 UDP 端口都接收所有协议）。终端会打印设备应指向的地址；首次登录会要求设置新密码。

macOS 手动安装时先运行 `xattr -dr com.apple.quarantine traffic66-darwin-*`。Windows 提示「已保护你的电脑」：**更多信息** → **仍要运行**。

## 命令速查

| 用途 | 命令 |
|---|---|
| 采集 sFlow 6343、NetFlow v5/v9 2055、IPFIX 4739（默认） | `./traffic66` |
| 只收 NetFlow v5/v9（2055）和 IPFIX（4739） | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| 只收 sFlow，UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Web 界面绑定指定 IP 和端口 | `./traffic66 -addr 10.0.0.5:8080` |
| 抓本机网卡（Linux） | `sudo ./traffic66 -capture eth0 -listen ""` |
| 设置或重置 admin 密码（用了 -data 就加上） | `./traffic66 passwd` |
| 终端界面连接远程服务器 | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![概览：未处理的发现、按应用划分的带宽与昨天同一时间对比、主要客户端和服务](images/overview.png)

## 完整手册

Web 界面菜单中的 **帮助**；GitHub 上： [MANUAL.zh.md](MANUAL.zh.md)

支持 15 种语言。评估免费，少于 100 人的组织免费（[许可](../LICENSE.md)）。
