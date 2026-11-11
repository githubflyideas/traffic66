[English](../README.md) | **中文** | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow、sFlow 和 IPFIX 采集器与流量分析器

一个程序搞定 sFlow、NetFlow 和 IPFIX 流量分析（IPv4 与 IPv6）：谁在占用带宽、流量去了哪里、统计结果与设备自身的接口计数器是否对得上，在 Web 界面和终端界面中展示。

## 快速开始

1. 从这里下载对应系统的压缩包： [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. 解压后运行程序：

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows：解压 `traffic66-windows-amd64.zip`，运行 `traffic66.exe`（双击即可）。

macOS：先在该目录运行 `xattr -dr com.apple.quarantine .`。

然后：

- Web 界面：`http://<服务器IP>:8066`，监听所有网卡。
- 登录：用户 `admin`，密码 `traffic66`。首次登录必须设置新密码。
- 把设备的流量导出指向服务器 IP：sFlow UDP 6343，NetFlow v5/v9 UDP 2055，IPFIX UDP 4739（监听所有网卡；每个端口都接收所有协议）。
- 刚启动时没数据很正常，稍等一下：设备发来第一个包后几秒内就会出现在 **设定** 里，图表大约晚 1 分钟。NetFlow v9/IPFIX 要先等到设备的模板；设备还没报告采样率时，记录最多暂存 5 分钟（`-sampling-wait`），之后按 1:1 计入。
- 运行要求：程序约 73 MB，不用装任何依赖；内存空闲时约 0.1 GB，每秒 5000 条流时约 0.6–0.8 GB，建议留 1 GB 以上。不要把 Web 界面直接暴露到公网：放在内网，或放在带 HTTPS 的反向代理后面。

## 常用命令

- `./traffic66`  
  按上面的默认设置启动。

- `./traffic66 -addr :9000`  
  Web 界面改用 9000 端口（默认 8066）。

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  修改采集端口（这里把 NetFlow 改为 9995）。

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  只监听一个 IP 地址，而不是所有网卡。

- `./traffic66 -data /var/lib/traffic66`  
  数据放到其他目录（默认是程序旁边的 `traffic66-data`）。

- `./traffic66 passwd`  
  设置或重置 admin 密码（用了 `-data` 就加上 `-data 目录`）。

- `traffic66.exe interfaces`  
  Windows：列出网卡（名称和编号）。

- `traffic66.exe -capture Wi-Fi`  
  Windows：抓取本机这块网卡的流量（填名称或编号）。需要安装 Npcap（npcap.com）。

- `./traffic66 interfaces`  
  Linux、macOS：列出网卡。

- `sudo ./traffic66 -capture eth0`  
  Linux：抓取本机 eth0 的流量（macOS 用 `-capture en0`）。

- `./traffic66 tui`  
  在 traffic66 机器上打开终端界面。

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  终端界面连接远程的 traffic66。

- `./traffic66 demo`  
  用模拟设备试用。首次启动会先生成一天的模拟历史（几秒钟，慢磁盘上更久），完成后才打开 Web 界面。

## 截图

<img src="images/overview.png" alt="概览" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="地理与运营商" width="49%"> <img src="images/interfaces.png" alt="接口对账" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## 完整手册

Web 界面菜单中的 **帮助**；GitHub 上： [MANUAL.zh.md](MANUAL.zh.md)

支持 15 种语言。评估免费，少于 100 人的组织免费（[许可](../LICENSE.md)）。

- **免费**：任何组织都可以无限期评估、测试和演示；少于 100 人（员工加外包合计）的组织和非商业用途可以免费用于生产。更大的组织生产使用 30 天后需要注册。详见 [LICENSE.md](../LICENSE.md) 和 [附加使用授权](../ADDITIONAL-USE-GRANT.md)。
- **未注册**：报表和图上带「评估版」水印，最多 3 个 Web 账号。
- **注册**：去掉水印。联系 route66traffic66@gmail.com。
