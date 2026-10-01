[English](../README.md) | **中文** | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

一个程序搞定 sFlow、NetFlow 和 IPFIX 流量分析。它接收交换机、路由器和防火墙导出的流数据，存入内嵌数据库，并在 Web 界面和终端界面中展示：谁在占用带宽、流量去了哪里、统计结果与设备自身的接口计数器是否对得上。

- Windows、Linux、macOS 都只有一个可执行文件。无需安装数据库，无需运行时，可离线使用。
- 任意 UDP 端口均可接收 sFlow v5、NetFlow v5、NetFlow v9 和 IPFIX；也可选择在本机网卡或镜像端口上直接抓包。
- 用接口计数器（sFlow 计数器或 SNMP）校验自己的统计结果，对不上时会说明原因。
- Top 66 排行、流向、国家与运营商网络、威胁情报命中、流记录、封装（GRE、IPIP、VXLAN、GENEVE、MPLS）。
- Web 界面和终端界面均支持 13 种语言。

![概览：按应用划分的带宽与上周对比、增长情况、主要客户端和服务](images/overview.png)

<sub>所有截图均来自 `traffic66 demo`，这是一个你可以自己运行的模拟公司网络（见 [试用演示](#1-try-the-demo)）。</sub>

<a id="contents"></a>

## 目录

1. [试用演示](#1-try-the-demo)
2. [安装](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [用户与密码](#3-users-and-passwords)
4. [在设备上配置流导出](#4-send-flows-from-your-devices)
5. [确认流数据已到达](#5-check-that-flows-arrive)
6. [让统计结果与接口计数器对上](#6-make-the-numbers-match-the-interface-counters)
7. [名称、SNMP 与自有网段](#7-names-snmp-and-your-own-networks)
8. [国家、运营商网络与威胁情报列表](#8-countries-networks-and-threat-lists)
9. [使用 Web 界面](#9-using-the-web-ui)
10. [终端界面](#10-terminal-ui)
11. [本地抓包](#11-local-capture)
12. [选项](#12-options)
13. [数据、备份、升级、卸载](#13-data-backup-upgrade-uninstall)
14. [安全](#14-security)
15. [容量规划](#15-sizing)
16. [故障排查](#16-troubleshooting)
17. [从源码构建](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. 试用演示

从 [Releases 页面](https://github.com/githubflyideas/traffic66/releases) 下载对应系统的压缩包：

| 系统 | 压缩包 |
|---|---|
| Windows 10/11、Server 2016 及以上（x64） | `traffic66-windows-amd64.zip` |
| Linux x86-64：内核 3.2 及以上的任意发行版，包括 CentOS 7 和 Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64：相同的发行版 | `traffic66-linux-arm64.tar.gz` |
| macOS 11 及以上，Apple 芯片 | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 及以上，Intel | `traffic66-darwin-amd64.tar.gz` |

Linux：

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS（第二行用于允许运行从网上下载、非 App Store 来源的程序）：

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows（PowerShell）：

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

打开 http://127.0.0.1:8066，用 `admin` / `try66` 登录。演示模式会搭建一个小型公司网络，带一天的历史数据，外加四台模拟设备的实时流量，其中埋了两起异常事件等你去找：从 **概览** 开始，看 **比平时多了谁**，然后一路点下去即可。按 Ctrl+C 停止。演示数据保存在程序旁边的 `traffic66-demo` 目录中，删除该目录即可从头开始。

演示模式使用与正式安装相同的端口（8066，以及 UDP 6343、2055、4739）。如果要和正式实例同时运行，请换用其他端口：`traffic66 demo -password try66 -addr :8067 -listen ""`。

在 Windows 上也可以直接双击 `traffic66.exe`。这会以正式模式（而非演示模式）启动 traffic66，并在浏览器中打开 Web 界面；首次启动的密码显示在黑色窗口中，关闭该窗口即停止 traffic66。如果 Windows 提示 "Windows 已保护你的电脑"，请点击 **更多信息** → **仍要运行**。

<a id="2-install"></a>

## 2. 安装

traffic66 只有一个文件。所谓安装，就是把它放到某个位置、指定数据目录、设置密码、放通防火墙并配置开机自启。示例中 traffic66 主机地址为 `192.0.2.50`，路由器地址为 `192.0.2.1`，请替换成你自己的地址。

端口：

| 端口 | 用途 |
|---|---|
| UDP 6343 | sFlow（默认） |
| UDP 2055 | NetFlow（默认） |
| UDP 4739 | IPFIX（默认） |
| TCP 8066 | Web 界面和 API |

每个 UDP 端口都能接收所有协议，所以如果方便，设备也可以把 NetFlow 发到 6343。用 `-listen` 修改或增加端口。

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

最后一条命令会提示输入用户 `admin` 的密码。

创建 `/etc/systemd/system/traffic66.service`：

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target
Wants=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
# only needed for local capture (-capture):
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

启动服务，并调大 UDP 接收缓冲区，避免突发流量时丢包：

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

防火墙，使用 firewalld（RHEL、Rocky、Alma、Fedora）：

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

或使用 ufw（Ubuntu、Debian）：

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

解压到 `C:\traffic66` 并设置密码（以管理员身份运行 PowerShell）：

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

数据保存在程序旁边的 `C:\traffic66\traffic66-data`。

放通防火墙：

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

想先在前台试运行，直接执行 `C:\traffic66\traffic66.exe`，按 Ctrl+C 停止。要让它开机后在后台运行、无需任何人登录，请注册为开机启动任务：

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` 不能省：否则 Windows 会在三天后停止该任务。停止任务用 `Stop-ScheduledTask -TaskName
traffic66`，删除任务用 `Unregister-ScheduledTask -TaskName traffic66`。

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

创建 `/Library/LaunchDaemons/traffic66.plist`：

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>traffic66</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/traffic66/traffic66</string>
    <string>-data</string>
    <string>/Library/Application Support/traffic66</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardErrorPath</key><string>/Library/Logs/traffic66.log</string>
</dict>
</plist>
```

启动与停止：

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

如果开启了 macOS 防火墙，请在“系统设置 → 网络 → 防火墙 → 选项”中允许 traffic66 的传入连接。

<a id="3-users-and-passwords"></a>

## 3. 用户与密码

**简而言之**：用户和密码都保存在数据目录下的一个文件 `password` 中。你不需要手动编辑它：用 `traffic66 passwd` 命令即可添加、修改、列出和删除用户。打开 `http://<traffic66 machine>:8066`，用其中任一用户登录。

<a id="the-first-sign-in"></a>

### 首次登录

traffic66 首次启动时会创建用户 `admin`，设置一个随机密码，并只显示一次：

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- 在 Windows 上双击启动：显示在黑色窗口中。
- 在终端中启动：显示在该终端中。
- Linux 服务：`journalctl -u traffic66 | grep "first start"`
- macOS 服务：`grep "first start" /Library/Logs/traffic66.log`

错过了？用 `traffic66 passwd`（见下文）设置一个新密码即可。如果像上面的安装步骤那样，在首次启动前已用 `traffic66 passwd` 设置了密码，则不会生成随机密码。

<a id="where-the-users-are-stored"></a>

### 用户保存在哪里

保存在数据目录下的 `password` 文件中：

| traffic66 的运行方式 | 文件 |
|---|---|
| 解压后从所在文件夹启动（默认） | 程序旁边的 `traffic66-data/password` |
| Linux 服务（第 2 节） | `/var/lib/traffic66/password` |
| Windows 开机启动任务（第 2 节） | `C:\traffic66\traffic66-data\password` |
| macOS 服务（第 2 节） | `/Library/Application Support/traffic66/password` |
| 演示 | 程序旁边的 `traffic66-demo/password` |

每个用户占一行。密码以加盐哈希的形式保存，任何人都无法从文件中读回密码，包括你自己；忘记密码时，重新设置一个即可。该文件只有其所有者可读。

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### 管理用户

在 traffic66 主机上执行以下命令：

| 操作 | 命令 |
|---|---|
| 修改 `admin` 的密码 | `traffic66 passwd` |
| 添加用户 `alice`，或修改她的密码 | `traffic66 passwd -user alice` |
| 删除用户 `alice` | `traffic66 passwd -user alice -delete` |
| 列出所有用户 | `traffic66 passwd -list` |
| 设置随机密码并打印出来 | `traffic66 passwd -generate`（其他用户加上 `-user`） |

- 命令会要求输入两次新密码，输入内容不会显示。密码至少 8 个字符。
- 如果 traffic66 运行时使用了 `-data`，命令中也要加上相同的 `-data`。以第 2 节中的 Linux 服务为例：

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  在 Windows 上（以管理员身份运行 PowerShell）：

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- 更改立即生效，无需重启：新密码在下次登录时即可使用，被删除的用户会在已打开的浏览器中被登出。
- 最后一个用户无法删除；请先添加另一个用户。
- 所有用户看到和能修改的内容完全相同，没有角色之分。

<a id="passwords-for-scripts-and-containers"></a>

### 用于脚本和容器的密码

在环境变量中设置 `TRAFFIC66_PASSWORD=…`，或在命令行中使用 `-password …`，traffic66 在本次运行中就只接受一个用户：由 `-user` 指定的用户（默认 `admin`），密码为该值。此时 `password` 文件会被忽略，也不会被修改。建议使用环境变量：命令行对本机其他用户是可见的。

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

同一地址在一分钟内输错五次密码，会被封禁一分钟。

<a id="4-send-flows-from-your-devices"></a>

## 4. 在设备上配置流导出

把每台设备的导出目标指向 traffic66 主机。不同型号和软件版本的命令有差异，请以设备手册为准。所有示例中，`192.0.2.50` 为 traffic66，`192.0.2.1` 为设备自身地址。

通用建议：

- 活动流超时（active timeout）设为 60 秒。超时越长，流量到达得越晚、越成块。
- 如果设备对 NetFlow/IPFIX 做了采样，请让它导出采样器选项（sampler options），这样才能知道采样率。在采样率到达之前，traffic66 会暂存记录，而不是按 1:1 计数。
- 要么所有接口都采样，要么只采样边界接口，并且只采一个方向。同一份流量在入方向和出方向各采一次会被重复计算；**接口对账** 会指出这种情况。
- sFlow 采样率参考：1 Gb/s 链路约 1:1000，10 Gb/s 用 1:4096，40/100 Gb/s 用 1:8192。

Cisco IOS / IOS-XE（Flexible NetFlow）：

```
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 export-protocol netflow-v9
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

Cisco NX-OS（sFlow）：

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS（sFlow）：

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX（sFlow）：

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

华为 CloudEngine（sFlow）：

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

H3C Comware（sFlow）：

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7（NetFlow v9 / IPFIX）：

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 及以上（NetFlow v9）：

```
config system netflow
    config collectors
        edit 1
            set collector-ip 192.0.2.50
            set collector-port 2055
        next
    end
end
config system interface
    edit port1
        set netflow-sampler both
    next
end
```

Linux 服务器和主机，使用 softflowd（NetFlow v9）：

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. 确认流数据已到达

打开 **接入**。只要设备发来了数据，几秒内就会出现在这里，并显示协议、采样率、丢包、最后收包时间和状态。状态不是绿色时，旁边的文字会说明问题所在以及该改什么。

![接入：每台设备的协议、采样、丢包以及需要修正的地方](images/sources.png)

如果某台设备没有出现：

1. 在 traffic66 主机上抓包看看（Linux、macOS）：`sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`。什么都抓不到，说明报文没有到达本机：检查设备配置、路由以及沿途的防火墙。
2. 报文能抓到，但 **接入** 仍然是空的：说明本机防火墙丢弃了报文（见 [安装](#2-install)），或者 traffic66 监听的是其他端口（`-listen`）。
3. 想在不动设备的情况下从另一台机器测试链路，可以在那台机器上运行`traffic66 simulate -to 192.0.2.50` 几秒钟。它会以模拟设备的身份发送sFlow、NetFlow 和 IPFIX；这些设备随后会出现在 **接入** 和数据中，所以最好在测试环境里做。

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. 让统计结果与接口计数器对上

流统计是估算值：采样到的报文数乘以采样率。traffic66 会将其与设备自身的接口计数器比对，在 **接口对账** 中显示差值；差值超出采样本身所能解释的范围时，还会给出可能的原因。

![接口对账：每个接口的流量估算值与设备计数器并列对比](images/interfaces.png)

获取用于比对的计数器：

- sFlow 设备只要配置了计数器上报间隔（`sflow counter interval 30` 之类），就会自动发送计数器。
- 对于 NetFlow 和 IPFIX 设备，在 **接入 → 名称** 中添加一行 `snmp`（见 [名称](#7-names-snmp-and-your-own-networks)）。traffic66 会每分钟读取一次接口计数器。

为了让数字对得上，traffic66 已经做了这些处理：使用设备实际生效的采样率；在采样率确定之前暂存 NetFlow/IPFIX 记录；补偿途中丢失的导出报文；把长流按其持续的各分钟分摊；并为 NetFlow/IPFIX 字节数按每包加上 18 字节以太网开销（接口计数器包含这部分，IP 层的流统计不包含；可用 `-l2-overhead` 修改）。

仍有差异的常见原因（都会在 **接口对账** 中报告）：部分接口没有采样；同一流量在两个接口上被采样；导出报文在到达 traffic66 之前丢失；或者采样率尚未获知。

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. 名称、SNMP 与自有网段

Web 界面中的 **接入 → 名称** 每行一条。内容保存为数据目录下的 `inventory.txt`，因此也可以直接编辑该文件（参考 `inventory.txt.example`）。每一行都是可选的。

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers

# device names; "unsampled" if it exports every packet (1:1)
device 192.0.2.1     Core router
device 192.0.2.9     Branch firewall unsampled

# interface names, by device address and ifIndex; speed in bits per second
iface  192.0.2.1 3   ISP uplink speed=1000000000

# host names shown instead of addresses
host   10.10.3.27    Finance PC

# read interface counters over SNMPv2c (IF-MIB 64-bit counters)
snmp   192.0.2.1     public
snmp   192.0.2.9     s3cret  10.99.0.9:161
```

- `net`：私网地址段（10/8、172.16/12、192.168/16、100.64/10）始终视为自有网段。把你的公网地址段也加进来，进出这些地址的流量才会算作内部流量；名称会显示在 **Top-N → 网段** 和流向图中。
- `snmp <device> <community> [<management address>[:port]]`：device 是流数据的来源地址。如果设备在另一个地址上响应 SNMP，请加上管理地址。通过 SNMP 读到的接口描述会用作接口名，除非你已用 `iface` 为该接口命名。记得在设备的 SNMP 访问控制列表中放行 traffic66 主机。
- 点击 **保存** 后即生效，无需重启。

<a id="8-countries-networks-and-threat-lists"></a>

## 8. 国家、运营商网络与威胁情报列表

显示国家和网络（AS）名称需要一张 IP 到 ASN 的映射表。可从[iptoasn.com](https://iptoasn.com) 免费下载：

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

任何相同格式的文件都可以（Tab 分隔：起始地址、结束地址、AS 号、国家代码、AS 名称；纯文本或 gzip 均可）。替换后需重启 traffic66；建议每月左右更新一次。

威胁情报列表是纯文本文件，每行一个地址或网段（`#` 或 `;` 之后的内容会被忽略），保存为 `<data directory>/threats/<name>.txt`，例如：

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

添加或修改列表后需重启 traffic66。命中结果会按列表名显示在 **威胁情报** 中。

![威胁情报：一台内部主机正在向威胁情报列表中的地址发送数据](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. 使用 Web 界面

基本不需要手动输入。每个页面上的每个值——地址、端口、应用、国家、设备——都可以点击：

- **只看它** / **排除它**：添加过滤条件。过滤条件显示在顶栏下方，在你移除之前对所有页面生效。
- **看它的流记录**：打开匹配的单条流记录。
- **在外部网站查询**：在公共查询网站中打开该地址或 AS。
- **复制**：复制该值。

页面：

| 页面 | 回答什么问题 |
|---|---|
| 概览 | 当前流量多大、与上周相比如何，按应用拆分；什么在增长；主要客户端和服务 |
| Top-N | 客户端、服务器、会话、应用、端口、国家、网络、网段、设备、封装或 VLAN 的前 66 名 |
| 会话 | 谁在和谁通信：前 66 个客户端–服务器对及其服务 |
| 流向 | 哪个网段在访问哪个国家的哪个应用 |
| 地理与运营商 | 按国家和按网络（AS）统计的流量 |
| 威胁情报 | 与威胁情报列表中地址有通信的主机，以及它们发送了多少流量 |
| 流记录 | 单条流记录，最新的在前，可选择显示列 |
| 接口对账 | 流统计与接口计数器并列对比，差异最大的在前，附原因 |
| 接入 | 设备、采样、丢包、采集器、SNMP，以及 **名称** |

页面上方有：时间范围（15 分钟到 30 天）、可选的搜索框、每 30 秒自动刷新，以及**复制链接**——复制一个精确指向当前视图（页面、时间范围和过滤条件）的链接，方便发给同事。界面语言跟随浏览器设置，可在菜单底部切换。

超过 6 小时的时间范围从整点开始，因此页面上的每个数字统计的都是完全相同的时间段：“24 小时”涵盖最近 24 个完整小时加上当前这一小时。这些范围的 Top-N 来自小时汇总数据，无法使用过滤条件，页面上也会有提示。需要过滤时请选择较短的时间范围。**会话** 页面始终读取流明细，因此在流量速率高时选择长时间范围可能需要等一会儿；1 小时最快。

![Top-N：最近一小时的前 66 个客户端](images/topn.png)

![会话：谁在和谁通信，以及所用服务](images/conv.png)

![流向：哪个网段在访问哪个国家的哪个应用](images/paths.png)

同一个概览的中文界面；所有页面都支持 13 种语言：

![中文界面的概览](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. 终端界面

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

在 traffic66 主机上，只要能读取数据目录，`traffic66 tui` 就会自动登录（数据目录不是默认位置时请指定 `-data`）。如果 traffic66 以其他用户身份运行（作为服务运行时就是这样），请改用 `-user` 和 `-password`。`-lang` 选择语言（`en`、`zh`、`hi`、`es`、`ar`、`fr`、`bn`、`pt`、`ru`、`id`、`ur`、`ja`、`ko`）。

按键：1–9 切换页面，↑↓ 选择，Enter 对选中的值执行操作，f 只看，x 排除，/ 搜索，t 时间范围，c 清除过滤条件，w 在浏览器中打开相同视图，q 退出。

![终端界面：概览](images/tui-overview.png)

![终端界面：Top-N 客户端](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. 本地抓包

除了接收流导出，traffic66 还能直接从运行它的机器的网卡上抓取报文并自行生成流。能看到什么取决于所用的网卡：

| 网卡 | traffic66 能看到什么 |
|---|---|
| 一个空闲网口，接到交换机的镜像（SPAN）端口 | 交换机镜像过来的全部流量：整个网络或一条上联链路 |
| 本机自己的以太网或 Wi-Fi | 只有本机自己的流量 |

Wi-Fi 网卡看不到其他设备的流量。要看整个 Wi-Fi 网络，请让路由器或无线接入点导出流（第 4 节），或者镜像接入点所连接的那个交换机端口。

<a id="windows-1"></a>

### Windows

1. 使用默认选项安装 [Npcap](https://npcap.com)。如果勾选了 "Restrict Npcap driver's access to Administrators only"，请以管理员身份运行 traffic66。
2. 列出网卡（PowerShell）：

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Name 列是 Windows 网络设置中的连接名称；正在使用的网卡带有地址。
3. 按名称或编号在 Wi-Fi 上抓包：

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   名称中含空格时请加引号：`-capture "Ethernet 2"`。重复 `-capture` 即可在多个网卡上抓包。如果只想抓包、不需要流采集器，请加上 `-listen=`。对于第 2 节中的开机启动任务，把该选项加到 `-Argument` 中：`-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`。

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

抓包需要 root，或具备 `CAP_NET_RAW` 和 `CAP_NET_ADMIN` 能力：即上面的 `setcap` 那一行，或第 2 节 systemd unit 中的 `AmbientCapabilities` 那一行。Wi-Fi 网卡通常名为 `wlan0` 或 `wlp…`。

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

抓包需要 root，无需安装其他软件。在 MacBook 上 `en0` 就是 Wi-Fi。

<a id="checking-that-it-works"></a>

### 确认是否生效

**接入** 中会列出每个抓包网卡，以及抓包方式和已看到的报文数。这些流在所有页面上都显示为来自设备 `127.0.0.1`（即本机），和其他设备的流一样。同一报文被看到两次（例如在两个镜像端口上）会被计算两次。

<a id="12-options"></a>

## 12. 选项

`traffic66 -h` 和 `traffic66 <command> -h` 会列出全部选项。

命令：

| 命令 | |
|---|---|
| `traffic66` | 采集流数据并提供 Web 界面 |
| `traffic66 demo` | 同上，但使用模拟网络 |
| `traffic66 tui` | 连接正在运行的 traffic66 的终端界面 |
| `traffic66 passwd` | 添加、修改、列出或删除用户（见 [用户与密码](#3-users-and-passwords)） |
| `traffic66 simulate -to HOST` | 向采集器发送模拟的导出数据 |
| `traffic66 interfaces` | 列出可用于本地抓包的网卡 |
| `traffic66 version` | 显示版本号 |

`traffic66` 和 `traffic66 demo` 的选项：

| 选项 | 默认值 | |
|---|---|---|
| `-addr` | `:8066` | Web 界面监听地址；`127.0.0.1:8066` 表示仅本机可访问 |
| `-data` | 程序旁边的 `traffic66-data` | 数据目录 |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP 采集器，格式为 `name=address`，逗号分隔；留空则禁用 |
| `-user` | `admin` | 首次启动时创建的用户名，也是 `-password` 所对应的用户 |
| `-password` | 未设置 | 本次运行只接受 `-user` 和此密码，忽略 `password` 文件（也可用 `TRAFFIC66_PASSWORD`） |
| `-retention-days` | `30` | 流明细保留天数；汇总数据保留 400 天 |
| `-memory` | `0.10` | 数据库缓存可使用的物理内存比例；程序其余部分另有同样大小的软上限（各自至少 256 MB） |
| `-l2-overhead` | `18` | NetFlow/IPFIX 字节数中每包追加的字节数 |
| `-sampling-wait` | `5m` | 记录等待采样率的最长时间 |
| `-capture` | | 在本机网卡上抓包（可重复指定） |
| `-inventory` | `<data>/inventory.txt` | 名称文件 |
| `-asn` | `<data>/asn.tsv.gz` | IP 到 ASN 映射表 |
| `-threat` | `<data>/threats/*.txt` | 额外的威胁情报列表，格式为 `name=path`（可重复指定） |
| `-dns-upstream` | 系统解析器 | 用于显示主机名的 DNS 服务器 |
| `-dns-rate` | `20` | 每秒反向解析次数上限 |
| `-dns-cache` | `2m` | 主机名缓存时间 |
| `-no-dns` | | 不做反向解析 |
| `-tui` | | 同时打开终端界面 |

示例：增加一个采集端口、明细保留一年、Web 界面仅限本机访问：

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. 数据、备份、升级、卸载

所有数据都在数据目录中：

| | |
|---|---|
| `raw/` | 流明细，每小时一个压缩文件 |
| `traffic66.duckdb` | 汇总数据、接口计数器和当前小时的数据 |
| `password` | 登录密码（哈希） |
| `inventory.txt` | 名称（**接入 → 名称**） |
| `asn.tsv.gz`, `threats/` | 你添加的查询表 |

- **备份**：停止 traffic66，复制整个目录。如果不停服务，就复制 `raw/`、`password`和 `inventory.txt`；这样会缺少当前小时的数据和汇总数据。
- **迁移**：停止 traffic66，移动目录，启动时用 `-data` 指向新位置。
- **升级**：停止 traffic66，替换程序文件，再重新启动。数据不受影响。以 Linux 为例：

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **卸载**：停止并删除服务或开机启动任务（见 [安装](#2-install)），然后删除程序目录和数据目录。

<a id="14-security"></a>

## 14. 安全

- Web 界面使用明文 HTTP：密码和数据在网络上不加密传输。在不完全可信的网络中，请只监听本机（`-addr 127.0.0.1:8066`），并在前面加一层 TLS 反向代理，例如用 [Caddy](https://caddyserver.com)：`caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`。或者通过 VPN 或 SSH 隧道访问：`ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`，然后打开http://127.0.0.1:8066。
- UDP 采集端口只对你的设备地址放行。
- `inventory.txt` 中的 SNMP community 以明文保存；请使用只读 community。

<a id="15-sizing"></a>

## 15. 容量规划

在 2 核机器上以每秒 5,000 条流实测：明细每天约占 12 GB 磁盘，当前小时另占约 1.5 GB；程序约占六分之一个核。长时间范围的概览来自汇总数据，耗时不到 0.2 秒。基于明细的查询每小时约扫描 2200 万行：查单台主机 1 小时的数据不到 1 秒，1 小时内全部会话的 Top 66 约 9 秒；耗时随时间范围增大而增加，随核数增多而减少。

因此，每秒 5,000 条流保存 30 天约需 360 GB 磁盘；请按你的流速率（显示在 **接入** 中）和 `-retention-days` 等比例估算。

内存：`-memory`（默认为物理内存的 10%，至少 256 MB）限制数据库缓存，程序其余部分另有同样大小的软上限。每秒 5,000 条流时，程序自身的数据（解码、去重、批次）约占 90 MB；总计约 0.6–0.8 GB，因此 2 GB 内存的机器即可满足。实测条件：连续采集 10 分钟（8 GB 机器上峰值 0.58 GB），以及在 2 GB 机器的限制下以该速率的 11 倍导入 1 小时的流（峰值 0.74 GB）。

`-memory` 是预算，不是硬上限：Go 的限制是软上限，数据库也可能短时超出自己的份额。需要硬上限时请使用操作系统的机制：systemd unit 中的 `MemoryMax=`（第 2 节），或容器的内存限制。请预留约 `-memory` 份额的 2.5 倍，且不少于 1 GB；按默认份额，物理内存不超过 8 GB 的机器用 `MemoryMax=2G` 即可。这样超限时重启的是 traffic66，而不是整台机器内存耗尽。

<a id="16-troubleshooting"></a>

## 16. 故障排查

| 现象 | 原因与解决方法 |
|---|---|
| **接入** 中看不到设备 | 报文没有到达：见 [确认流数据已到达](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | 设备尚未发送采样器选项；大多数设备几分钟内会重发。如果一直不发，请配置导出（Cisco 上为 `option sampler-table`）；如果确实是 1:1，可在名称中标记为 `unsampled` |
| 统计值低于接口计数器 | 查看 **接口对账**：途中丢包、有接口未采样，或流仍在设备缓存中（活动超时超过 60 秒） |
| 统计值高于接口计数器 | 同一流量在两个接口或两台设备上被采样 |
| 没有国家或网络信息 | 缺少 IP 到 ASN 映射表：见 [国家](#8-countries-networks-and-threat-lists) |
| 页面上出现 "数据库达到内存上限，无法完成这次查询" | 选择更短的时间范围，或用更大的 `-memory` 启动；详细信息见日志 |
| 忘记密码 | 在 traffic66 主机上执行 `traffic66 passwd`（如果 traffic66 运行时使用了 `-data`，也加上它） |
| `Conflicting lock is held` | 另一个 traffic66 正在使用该数据目录 |
| `receive buffer is only … KB` | Linux 限制了 UDP 缓冲区：设置 `net.core.rmem_max=16777216`（见 [Linux](#linux)） |
| `cannot create the data directory` | 当前用户对程序目录没有写权限：请指定 `-data` |
| macOS："cannot be opened" 或 "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows："Windows 已保护你的电脑" | **更多信息** → **仍要运行**；程序目前尚未签名 |
| Windows 抓包：找不到 Npcap | 安装 [Npcap](https://npcap.com) |
| `address already in use` | 端口被其他程序占用：用 `-addr` 或 `-listen` 换用其他端口 |

<a id="17-build-from-source"></a>

## 17. 从源码构建

需要 Go 1.24 和 C 编译器（gcc 或 clang；Windows 上用 MinGW-w64）：

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
