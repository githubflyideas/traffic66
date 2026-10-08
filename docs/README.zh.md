[English](../README.md) | **中文** | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66 — NetFlow、sFlow 和 IPFIX 采集器与流量分析器

一个程序搞定 sFlow、NetFlow 和 IPFIX 流量分析：谁在占用带宽、流量去了哪里、统计结果与设备自身的接口计数器是否对得上，在 Web 界面和终端界面中展示。

可自托管的 ntopng、ElastiFlow、pmacct 加 Grafana，或 PRTG 和 SolarWinds NTA 流量分析模块的替代方案，用于网络监控（network monitoring）、带宽监控（bandwidth monitoring）、流量大户排行（top talkers）、DDoS 与扫描检测以及 pcap 分析，无需 Elasticsearch、Kafka 或单独的数据库。

- Windows、Linux、macOS 都只有一个可执行文件；无需安装数据库，可离线使用。
- 任意 UDP 端口均可接收 sFlow v5、NetFlow v5/v9 和 IPFIX，也可在本机网卡上直接抓包。
- 用接口计数器（sFlow 或 SNMP）校验自己的统计结果，并说明对不上的原因。
- 找出扫描、暴力破解、横向移动、异常上传、泛洪和威胁情报流量，采样数据同样适用。
- `traffic66 capture.pcap` 直接分析抓包文件，无需任何配置。
- 支持 13 种语言。评估免费，少于 100 人的组织免费（[许可](#licence)）。

![概览：未处理的发现、按应用划分的带宽与昨天同一时间对比、主要客户端和服务](images/overview.png)

<sub>所有截图均来自 `traffic66 demo`，一个模拟的公司网络。</sub>

<a id="contents"></a>

## 目录

1. [试用演示](#1-try-the-demo)
2. [安装](#2-install)
3. [用户与密码](#3-users-and-passwords)
4. [在设备上配置流导出](#4-send-flows-from-your-devices)
5. [确认流数据已到达](#5-check-that-flows-arrive)
6. [接口与计数器](#6-interfaces-and-counters)
7. [名称、国家与威胁情报列表](#7-names-countries-and-threat-lists)
8. [使用 Web 界面](#8-using-the-web-ui)
9. [离线 pcap、终端界面、本地抓包](#9-offline-pcap-terminal-ui-local-capture)
10. [选项与数据](#10-options-and-data)
11. [安全、容量规划、故障排查](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. 试用演示

从 [Releases 页面](https://github.com/githubflyideas/traffic66/releases) 下载对应系统的压缩包（Windows x64、内核 3.2 及以上的 Linux x86-64/ARM64、macOS 11 及以上），解压后运行：

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

在 macOS 上请先运行 `xattr -dr com.apple.quarantine <folder>`。打开 http://127.0.0.1:8066，用 `admin` / `try66` 登录：可以看到一天的历史数据和四台模拟设备的实时流量，其中包括一次攻击，在 **发现** 中一步步展示。按 Ctrl+C 停止；删除 `traffic66-demo` 即可从头开始。要和正式安装同时运行：`-addr :8067 -listen ""`。

<a id="2-install"></a>

## 2. 安装

traffic66 只有一个文件。端口：UDP 6343（sFlow）、2055（NetFlow）、4739（IPFIX），TCP 8066（Web 界面）；每个 UDP 端口都能接收所有协议。

**Linux**（systemd）：

```
sudo mkdir -p /opt/traffic66 && sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

`/etc/systemd/system/traffic66.service`：

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
MemoryMax=2G
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN   # only for local capture

[Install]
WantedBy=multi-user.target
```

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf && sudo sysctl --system
sudo systemctl daemon-reload && sudo systemctl enable --now traffic66
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload
```

**Windows**（以管理员身份运行 PowerShell）：解压到 `C:\traffic66`，运行 `C:\traffic66\traffic66.exe passwd`，放通端口，并设置开机启动：

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

也可以直接双击 `traffic66.exe`：它会打开 Web 界面，并在窗口中显示首次启动的密码。

**macOS**：解压到 `/usr/local/traffic66`，去掉隔离标记，运行 `traffic66 passwd -data "/Library/Application Support/traffic66"`，然后用一个 LaunchDaemon 启动它：其 `ProgramArguments` 为程序、`-data` 和该目录，并设置 `RunAtLoad` 和 `KeepAlive`。

<a id="3-users-and-passwords"></a>

## 3. 用户与密码

traffic66 首次启动时会创建用户 `admin`，设置一个随机密码并只显示一次（在窗口、终端中，或通过 `journalctl -u traffic66 | grep "first start"` 查看）。用户以加盐哈希的形式保存在数据目录下的 `password` 文件中，在 traffic66 主机上用一条命令管理（如果 traffic66 运行时使用了 `-data …`，命令中也要加上）：

| 操作 | 命令 |
|---|---|
| 修改 `admin` 的密码 | `traffic66 passwd` |
| 添加 `alice`，或修改她的密码 | `traffic66 passwd -user alice` |
| 删除 `alice` | `traffic66 passwd -user alice -delete` |
| 列出用户 | `traffic66 passwd -list` |

更改立即生效。所有用户权限相同。用于脚本和容器时，`TRAFFIC66_PASSWORD=…`（或 `-password`）使本次运行只接受 `-user` 及该密码。同一地址在一分钟内输错五次密码，会被封禁一分钟。

<a id="4-send-flows-from-your-devices"></a>

## 4. 在设备上配置流导出

`192.0.2.50` 为 traffic66，`192.0.2.1` 为设备。活动流超时设为 60 秒，让 NetFlow/IPFIX 设备导出采样器选项，并在**每个接口上做入向采样**（或只在边界接口上）：这样每个报文只计算一次。sFlow 采样率：1 Gb/s 约 1:1000，10 Gb/s 用 1:4096，40/100 Gb/s 用 1:8192。

```
! Cisco IOS-XE, NetFlow v9
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

```
# Huawei CloudEngine, sFlow (H3C Comware is similar)
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

```
# Juniper EX/QFX, sFlow
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow interfaces ge-0/0/0

# Arista EOS, sFlow
sflow sample 4096
sflow destination 192.0.2.50
sflow run

# MikroTik RouterOS 7
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. 确认流数据已到达

只要设备发来任何数据，几秒内就会出现在 **设定** 中：协议、采样率、丢包、它所采样的接口，以及状态不是绿色时需要修正什么。sFlow 的丢包分为途中丢失（如果 `netstat -su` 显示缓冲区错误，请调大 `net.core.rmem_max`）和设备自身丢弃的样本。

![设定：每台设备的协议、采样、丢包以及需要修正的地方](images/sources.png)

某台设备没有出现？运行 `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`：什么都抓不到，说明是路由、防火墙或设备配置的问题；能抓到报文但 **设定** 中没有，说明是本机防火墙或 `-listen` 的问题。在另一台机器上运行 `traffic66 simulate -to 192.0.2.50`，可以用模拟设备测试链路。

<a id="6-interfaces-and-counters"></a>

## 6. 接口与计数器

流统计是估算值（样本数 × 采样率）。**接口对账** 将其与设备的接口计数器（sFlow 计数器，或通过名称中的 `snmp` 行使用 SNMP）比对，并说明差异的原因：有接口未采样、同一流量被采样两次、途中丢包，或采样率未知。每个接口有一张 bits/s 图和一张 packets/s 图，入向为绿色、出向为蓝色，计数器以虚线显示。

每一行上，**✎** 设置名称和简短标识（如 *uplink*），**☆** 把它设为默认接口（★），各页面打开时就看它。

只在部分接口上采样的设备，也会显示这些流另一端的接口。这些 **对端接口** 列在最后，字号更小、呈灰色：它们只包含经过采样接口的流量。采样接口通过 sFlow 数据源或 flowDirection 字段（IPFIX 61）得知；没有该字段时，取承载设备 90% 流量的接口。

![接口对账：每个接口的流量，以及流量估算值与设备计数器并列对比](images/interfaces.png)

traffic66 已经做了这些处理：使用设备实际生效的采样率，等待未知的采样率，补偿导出丢失，把长流分摊到其持续的各分钟，并为 NetFlow/IPFIX 按每包加上 18 字节以太网开销（`-l2-overhead`）。

<a id="7-names-countries-and-threat-lists"></a>

## 7. 名称、国家与威胁情报列表

点击任意地址并选择 **起个名字…**，或使用 **设定 → 名称**。名称保存在数据目录下的 `inventory.txt` 中：

```
net    10.10.0.0/16    Office LAN                  # your networks
net    203.0.113.0/24  Public servers country=JP    # country: lines on the world map
device 192.0.2.1       Core router
device 192.0.2.9       Branch firewall unsampled    # exports every packet
device 192.0.2.20      Edge router sampling=1000    # rate it does not declare
iface  192.0.2.1 3     ISP uplink speed=1000000000 tag=uplink default
host   10.10.3.27      Finance PC
snmp   192.0.2.1       public                       # read counters over SNMPv2c
snmp   192.0.2.9       s3cret  10.99.0.9:161        # other management address
```

私网地址段始终视为自有网段。点击 **保存** 后即生效，无需重启。

国家和网络（AS）开箱即用，使用 DB-IP 的免费 Lite 数据库（[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)，"IP Geolocation by DB-IP"，[db-ip.com](https://db-ip.com)）；**设定** 可以更新它们，或换用 MaxMind GeoLite2、IPinfo Lite 或 IPtoASN 文件。地图轮廓：[Natural Earth](https://www.naturalearthdata.com)。

![地理与运营商：世界地图上按国家显示的外部流量](images/geo.png)

威胁情报列表是文本文件，每行一个地址或网段，保存为 `<data>/threats/<name>.txt`（例如 Spamhaus DROP）；修改后需重启。命中结果显示在 **威胁情报** 中。

![威胁情报：一台内部主机正在向威胁情报列表中的地址发送数据](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. 使用 Web 界面

每个页面上的每个值都可以点击：**只看它** / **排除它**（过滤条件对所有页面生效）、**看它的流记录**、**查看详情**（关于一台主机或一个服务的页面）、**起个名字…**、**在外部网站查询**、**复制**。

| 页面 | 显示什么 |
|---|---|
| 概览 | 所选接口的带宽；按应用划分的流量（合计、入站或出站），与昨天或上周对比；未处理的发现；主要客户端和服务 |
| Top 66 | 前 66 名会话，可按任意列排序，或按应用、网络、网段、设备、封装、VLAN 分组；前 30 名通信方 |
| 流量明细 | 环形图：服务器及其客户端（或反过来），以及服务 |
| 流向 | 主机 → 应用 → 国家，或客户端 → 服务 → 服务器，或按网段 |
| 接口对账 | 每个接口随时间的流量与其计数器对比；名称、标识、默认接口 |
| 流记录 | 单条流记录，每 5 秒实时更新，或任意时间范围 |
| 发现、威胁情报 | 需要处理的问题（[见下文](#findings)）；与列表中地址的流量 |
| 地理与运营商 | 按国家显示的世界地图，网络（AS）随时间的变化 |
| 设定 | 设备、采样、丢包、SNMP、数据库、Logo、名称 |
| 离线 pcap 分析、数据清理 | 抓包文件（[见下文](#9-offline-pcap-terminal-ui-local-capture)）；删除旧数据 |

页面上方有：**接口**（全部，或某一个采样接口；此时流量类页面只显示经过它的流量）、时间范围（15 分钟到 30 天，或自定义）、每 30 秒刷新，以及精确指向当前视图的 **复制链接**。语言和五种颜色主题在菜单底部。图表在数据完整处结束：NetFlow/IPFIX 取决于设备导出的延迟（最多 2 分钟）。超过 6 小时的时间范围从整点开始；单个接口选 7 天或 30 天时读取流明细，因此更慢，且最远只能回溯到明细的保留期限。

![Top 66：前 66 名会话，可按任意列排序](images/topn.png)

![流量明细：以环形图显示服务器及其客户端、服务及其服务器](images/traffic.png)

![单台主机的详情：与它相关的发现、它的流量、在和谁通信、服务、国家和最新流记录](images/detail.png)

![流向：哪台主机在访问哪个国家的哪个应用](images/paths.png)

![中文界面的概览](images/overview-zh.png)

<a id="findings"></a>

### 发现

每 5 分钟检查一次最近 10 分钟的数据；持续一小时的行为是一条不断增长的发现。

| 发现 | 含义 |
|---|---|
| 扫描、端口扫描 | 向许多主机的同一端口，或一台主机的许多端口发送小探测包 |
| 暴力破解 | 对登录服务的大量短连接 |
| 横向移动 | 向此前从未提供过文件共享或远程管理的内部主机发起此类会话 |
| 异常上传 | 10 分钟内向新地址发送 100 MB，且为返回量的三倍 |
| 泛洪 | 每秒 20,000 个以上小包发往同一地址，达到其平时速率的十倍 |
| 威胁情报 | 与列表中地址的流量 |

来自内网的为高，来自互联网的为低。**已处理** 关闭一条发现，**不是问题** 则永久关闭它。横向移动和异常上传需要一天的历史数据。在 1:4096 采样下，演示中的攻击仍被完整发现；非常小规模的扫描可能被采样掩盖。

![发现：攻击的每一步，均通过 1:4096 的 sFlow 采样发现](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. 离线 pcap、终端界面、本地抓包

**离线 pcap 分析** 用相同的页面查看抓包文件（pcap、pcapng），与实时数据分开：`traffic66 a.pcap b.pcapng` 在 127.0.0.1 上启动并打开浏览器（最多 3 个文件，3 GB；按 Ctrl+C 删除导入的数据），也可以在该页面上传最多 3 个 50 MB 的文件。一次分析一个文件，每个文件有自己独立的数据库：在文件所在行点 **分析**，所有页面都显示该文件，顶部的栏可切换到另一个文件。它分析的是流，而不是包内容。

![离线分析：抓包文件及其包数、流数和时间](images/sandbox.png)

**终端界面**：在 traffic66 主机上运行 `traffic66 tui`，或 `traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`。按键：1–8 切换页面，Enter 执行操作，f 只看，x 排除，t 时间范围，w 在浏览器中打开，q 退出；`-lang` 选择语言。

![终端界面：概览](images/tui-overview.png)

![终端界面：Top 66 会话](images/tui-topn.png)

**本地抓包** 从本机网卡生成流，最好是接到交换机镜像端口的网口：`traffic66 interfaces` 列出网卡，`-capture eth1` 开始抓包。在 Windows 上：

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux 需要 root 或 `setcap cap_net_raw,cap_net_admin+ep`，macOS 需要 root，Windows 需要 [Npcap](https://npcap.com)。抓到的流显示为来自设备 `127.0.0.1`。本地抓包没有设备接口和计数器，因此 **接口对账** 对它没有可对比的内容。

<a id="10-options-and-data"></a>

## 10. 选项与数据

`traffic66 -h` 会列出全部选项。最常用的：

| 选项 | 默认值 | |
|---|---|---|
| `-data` | 程序旁边的 `traffic66-data` | 数据目录 |
| `-addr` | `:8066` | Web 界面；`127.0.0.1:8066` 表示仅本机可访问 |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP 采集器；留空则禁用 |
| `-retention-days` | `30` | 流明细保留天数；汇总数据保留 400 天 |
| `-memory` | `0.10` | 数据库缓存可使用的内存比例 |
| `-sampling-wait` | `5m` | 记录等待采样率的时间 |
| `-capture` | | 本机网卡（可重复指定） |
| `-no-dns` | | 不做反向解析 |

数据目录包含 `raw/`（明细，每小时一个文件）、`traffic66.duckdb`（汇总和计数器）、`password`、`inventory.txt`、`license.json`、你的 Logo 和数据库。备份：停止 traffic66 后复制整个目录；升级：替换程序文件。**数据清理** 删除早于 7–120 天的数据，或全部数据。

<a id="licence"></a>

### 许可

以 [PolyForm Noncommercial License 1.0.0](../LICENSE.md) 和 [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) 公开源码（以英文文本为准）：评估免费，少于 100 人的组织免费；更大的组织在生产使用 30 天后需注册；销售、为他人托管或提供竞争性产品需要商业许可。任何功能都不会被关闭。每个页面底部显示 8 位安装编号；把它发给作者，并把收到的 `license.json` 放进数据目录。联系方式：<https://github.com/githubflyideas/traffic66>。

<a id="11-security-sizing-troubleshooting"></a>

## 11. 安全、容量规划、故障排查

Web 界面使用明文 HTTP：在不可信的网络中，请使用 `-addr 127.0.0.1:8066` 并在前面加一层 TLS 代理（`caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`），或使用 SSH 隧道。UDP 端口只对你的设备放行。SNMP community 以明文保存；请使用只读 community。

在 2 核机器上以每秒 5,000 条流计：明细每天约占 12 GB 磁盘（30 天 360 GB），约六分之一个核，内存 0.6–0.8 GB。长时间范围的概览耗时不到 0.2 秒；1 小时内全部会话的 Top 66 约 9 秒。

| 现象 | 解决方法 |
|---|---|
| "waiting for the sampling rate" | 导出采样器选项，或在设备行上使用 `sampling=N` / `unsampled` |
| 低于接口计数器 | 有接口未采样、丢包，或活动超时超过 60 秒 |
| 高于接口计数器 | 同一流量在两个接口或两台设备上被采样 |
| 忘记密码 | 在 traffic66 主机上执行 `traffic66 passwd` |
| `Conflicting lock is held` | 另一个 traffic66 正在使用该数据目录 |
| `address already in use` | 用 `-addr` 或 `-listen` 换用其他端口 |
| Windows："Windows 已保护你的电脑" | **更多信息** → **仍要运行** |

从源码构建：需要 Go 1.24 和 C 编译器，然后运行 `scripts/build.sh 0.1.0 traffic66`。
