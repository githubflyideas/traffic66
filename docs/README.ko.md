[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | **한국어**

# traffic66

sFlow, NetFlow, IPFIX 플로 분석을 하나의 프로그램으로 처리합니다. 스위치, 라우터, 방화벽이
익스포트한 플로를 수집해 내장 데이터베이스에 저장하고, 누가 대역폭을 쓰는지, 트래픽이 어디로
가는지, 그 수치가 장비 자체의 인터페이스 카운터와 맞는지를 웹 UI와 터미널 UI로 보여 줍니다.

- Windows, Linux, macOS 모두 실행 파일 하나입니다. 데이터베이스 설치나 런타임이 필요 없고,
  오프라인에서도 동작합니다.
- 어떤 UDP 포트로든 sFlow v5, NetFlow v5, NetFlow v9, IPFIX를 받습니다. 네트워크
  인터페이스나 미러 포트에서 직접 로컬 캡처도 할 수 있습니다.
- 자체 집계 수치를 인터페이스 카운터(sFlow 카운터 또는 SNMP)와 대조하고, 차이가 나면 그
  이유를 알려 줍니다.
- Top 66 목록, 트래픽 경로, 국가와 네트워크, 위협 목록 일치 항목, 플로 레코드,
  캡슐화(GRE, IPIP, VXLAN, GENEVE, MPLS).
- 웹 UI와 터미널 UI 모두 13개 언어를 지원합니다.

<a id="contents"></a>

## 목차

1. [데모 실행해 보기](#1-try-the-demo)
2. [설치](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [로그인과 비밀번호](#3-sign-in-and-passwords)
4. [장비에서 플로 보내기](#4-send-flows-from-your-devices)
5. [플로 수신 확인](#5-check-that-flows-arrive)
6. [수치를 인터페이스 카운터와 맞추기](#6-make-the-numbers-match-the-interface-counters)
7. [이름, SNMP, 자체 네트워크](#7-names-snmp-and-your-own-networks)
8. [국가, 네트워크, 위협 목록](#8-countries-networks-and-threat-lists)
9. [웹 UI 사용법](#9-using-the-web-ui)
10. [터미널 UI](#10-terminal-ui)
11. [로컬 캡처](#11-local-capture)
12. [옵션](#12-options)
13. [데이터, 백업, 업그레이드, 제거](#13-data-backup-upgrade-uninstall)
14. [보안](#14-security)
15. [용량 산정](#15-sizing)
16. [문제 해결](#16-troubleshooting)
17. [소스에서 빌드](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. 데모 실행해 보기

[릴리스 페이지](https://github.com/githubflyideas/traffic66/releases)에서 사용 중인
시스템용 압축 파일을 내려받습니다.

| 시스템 | 압축 파일 |
|---|---|
| Windows 10/11, Server 2016 이상(x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64 | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64 | `traffic66-linux-arm64.tar.gz` |
| macOS Apple 실리콘 | `traffic66-darwin-arm64.tar.gz` |
| macOS Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS(두 번째 줄은 App Store가 아닌 인터넷에서 내려받은 프로그램을 macOS에서 실행할 수 있게 해 줍니다):

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows(PowerShell):

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

http://127.0.0.1:8066 을 열고 `admin` / `try66`으로 로그인합니다. 데모는 작은 회사
네트워크를 만들어 하루치 이력과 시뮬레이션 장비 4대의 실시간 트래픽을 보여 주며, 찾아볼
만한 사고 두 건이 들어 있습니다. **개요**에서 시작해 **평소보다 늘어난 곳**을 보고, 거기서부터
클릭하며 따라가 보십시오. Ctrl+C로 중지합니다. 데모 데이터는 프로그램 옆의 `traffic66-demo`에
저장되며, 이 폴더를 삭제하면 데모를 처음부터 다시 시작할 수 있습니다.

데모는 실제 설치와 같은 포트(8066, UDP 6343, 2055, 4739)를 사용합니다. 실제 인스턴스와
함께 실행하려면 다른 포트를 지정합니다:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. 설치

traffic66은 파일 하나입니다. 설치란 이 파일을 적당한 곳에 두고, 데이터 디렉터리를 정하고,
비밀번호를 설정하고, 방화벽을 열고, 부팅 시 자동으로 시작되게 하는 것입니다. 예시에서는
traffic66 서버를 `192.0.2.50`, 라우터를 `192.0.2.1`로 표기합니다. 실제 주소로 바꿔
사용하십시오.

포트:

| 포트 | 용도 |
|---|---|
| UDP 6343 | sFlow(기본값) |
| UDP 2055 | NetFlow(기본값) |
| UDP 4739 | IPFIX(기본값) |
| TCP 8066 | 웹 UI 및 API |

모든 UDP 포트가 모든 프로토콜을 받으므로, 편하다면 장비에서 NetFlow를 6343으로 보내도
됩니다. 포트 변경이나 추가는 `-listen`으로 합니다.

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

마지막 명령은 사용자 `admin`의 비밀번호를 묻습니다.

`/etc/systemd/system/traffic66.service`를 만듭니다:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target
Wants=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
# only needed for local capture (-capture):
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

서비스를 시작하고, 버스트 시 유실되지 않도록 UDP 버퍼 한도를 늘립니다:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

방화벽, firewalld 사용 시(RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

ufw 사용 시(Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

`C:\traffic66`에 압축을 풀고 비밀번호를 설정합니다(관리자 권한 PowerShell):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

데이터는 프로그램 옆의 `C:\traffic66\traffic66-data`에 저장됩니다.

방화벽을 엽니다:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

포그라운드에서 시험해 보려면 `C:\traffic66\traffic66.exe`를 실행하고 Ctrl+C로 중지합니다.
아무도 로그인하지 않은 상태에서도 부팅 시부터 백그라운드로 실행하려면 시작 작업으로
등록합니다:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)`는 반드시 필요합니다. 이것이 없으면 Windows가
3일 후 작업을 중지합니다. 중지는 `Stop-ScheduledTask -TaskName
traffic66`, 제거는 `Unregister-ScheduledTask -TaskName traffic66`으로 합니다.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

`/Library/LaunchDaemons/traffic66.plist`를 만듭니다:

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

시작과 중지:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

macOS 방화벽이 켜져 있다면 시스템 설정 → 네트워크 → 방화벽 → 옵션에서 traffic66의 수신
연결을 허용하십시오.

<a id="3-sign-in-and-passwords"></a>

## 3. 로그인과 비밀번호

`http://<traffic66 machine>:8066`을 열고 로그인합니다. 따로 정하지 않았다면 사용자는
`admin`입니다.

- 첫 시작 전에 비밀번호를 설정하지 않았다면 traffic66이 비밀번호를 만들어 로그에 한 번만
  출력합니다:
  `first start: sign in as user "admin" with password "…"`.
  Linux에서는 `journalctl -u traffic66 | grep "first start"`로 찾을 수 있습니다.
- 비밀번호는 해시되어 데이터 디렉터리의 `password` 파일에 저장됩니다. 재시작해도
  바뀌지 않습니다.
- 비밀번호를 바꾸거나, 잊어버려서 새로 설정하려면 traffic66 서버에서 다음을 실행합니다:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate`는 임의의 비밀번호를 만들어 출력합니다. 실행 중인 traffic66은
  다음 로그인부터 새 비밀번호를 받아들이며, 재시작할 필요가 없습니다.
- 사용자 추가: `traffic66 passwd -data <data directory> -user alice`. 모든 사용자에게
  같은 화면이 보입니다.
- 스크립트나 컨테이너에서는 환경 변수 `TRAFFIC66_PASSWORD=…` 또는 명령줄의
  `-password …`로, 저장된 비밀번호 대신 해당 실행에만 쓸 비밀번호를 지정할 수 있습니다.
  환경 변수를 권장합니다. 명령줄은 같은 서버의 다른 사용자에게 보이기 때문입니다.

1분 안에 비밀번호를 다섯 번 틀린 주소는 1분간 차단됩니다.

<a id="4-send-flows-from-your-devices"></a>

## 4. 장비에서 플로 보내기

각 장비가 traffic66 서버로 플로를 보내도록 설정합니다. 명령은 모델과 소프트웨어 버전에 따라
다르므로 장비 매뉴얼을 확인하십시오. 모든 예시에서 `192.0.2.50`은 traffic66,
`192.0.2.1`은 장비 자신의 주소입니다.

일반적인 권장 사항:

- 액티브 플로 타임아웃은 60초로 설정합니다. 타임아웃이 길수록 트래픽이 늦게, 큰 덩어리로
  도착합니다.
- 장비가 NetFlow/IPFIX를 샘플링한다면 샘플러 옵션도 익스포트하도록 해서 샘플링 레이트를 알 수
  있게 합니다. traffic66은 레이트가 도착할 때까지 레코드를 보류하며, 1:1로 계산하지 않습니다.
- 모든 인터페이스 또는 경계 인터페이스만, 한 방향으로 샘플링합니다. 같은 트래픽을 들어올 때와
  나갈 때 모두 샘플링하면 두 번 집계됩니다. **인터페이스 대조**가 이를 지적해 줍니다.
- sFlow 샘플링 레이트: 1 Gb/s 링크는 약 1:1000, 10 Gb/s는 1:4096, 40/100 Gb/s는 1:8192.

Cisco IOS / IOS-XE(Flexible NetFlow):

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

Cisco NX-OS(sFlow):

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS(sFlow):

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX(sFlow):

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

Huawei CloudEngine(sFlow):

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

H3C Comware(sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7(NetFlow v9 / IPFIX):

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 이상(NetFlow v9):

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

Linux 서버와 호스트는 softflowd 사용(NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. 플로 수신 확인

**수집 상태**를 엽니다. 무엇이든 보내는 장비는 몇 초 안에 나타나며, 프로토콜, 샘플링 레이트,
유실, 마지막 패킷, 상태가 표시됩니다. 상태가 녹색이 아니면 옆의 설명에 무엇이 잘못됐고 무엇을
바꿔야 하는지 나와 있습니다.

장비가 나타나지 않으면:

1. traffic66 서버에서 패킷을 확인합니다(Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   아무것도 보이지 않으면 패킷이 서버까지 오지 않는 것입니다. 장비 설정, 라우팅, 경로상의
   방화벽을 확인하십시오.
2. 패킷은 들어오는데 **수집 상태**가 비어 있다면: 로컬 방화벽이 패킷을 버리고 있거나
   ([설치](#2-install) 참조), traffic66이 다른 포트에서 수신하고 있는 것입니다(`-listen`).
3. 장비를 건드리지 않고 다른 서버에서 경로를 테스트하려면 그 서버에서
   `traffic66 simulate -to 192.0.2.50`을 몇 초간 실행합니다. 시뮬레이션 장비에서 sFlow,
   NetFlow, IPFIX를 보내며, 이 장비들이 **수집 상태**와 데이터에 나타나므로 테스트용 설치에서
   하는 것이 좋습니다.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. 수치를 인터페이스 카운터와 맞추기

플로 수치는 추정값입니다(샘플링된 패킷 수 × 샘플링 레이트). traffic66은 이를 장비 자체의
인터페이스 카운터와 비교해 **인터페이스 대조**에 차이를 보여 주며, 샘플링만으로 설명되지 않을
만큼 차이가 크면 가능한 원인도 함께 표시합니다.

비교할 카운터를 얻으려면:

- sFlow 장비는 카운터 전송 주기를 설정하면(`sflow counter interval 30` 등) 알아서
  보냅니다.
- NetFlow와 IPFIX 장비는 **수집 상태 → 이름**에 `snmp` 줄을 추가합니다
  ([이름](#7-names-snmp-and-your-own-networks) 참조). 그러면 traffic66이 1분마다
  인터페이스 카운터를 읽습니다.

수치를 맞추기 위해 traffic66이 이미 하고 있는 일: 장비가 실제로 적용한 샘플링 레이트를
사용하고, 샘플링 레이트가 확인될 때까지 NetFlow/IPFIX 레코드를 보류하고, 중간에 유실된
익스포트 패킷을 보정하고, 긴 플로를 지속된 각 분에 나눠 배분하며, NetFlow/IPFIX 바이트 수에
패킷당 18바이트의 Ethernet 오버헤드를 더합니다(인터페이스 카운터에는 포함되고 IP 계층 플로
집계에는 포함되지 않기 때문이며, `-l2-overhead`로 변경 가능).

그래도 남는 차이의 흔한 원인은 다음과 같으며, 모두 **인터페이스 대조**에 보고됩니다: 일부
인터페이스가 샘플링되지 않음, 같은 트래픽을 두 인터페이스에서 샘플링함, 익스포트 패킷이
traffic66에 도착하기 전에 유실됨, 샘플링 레이트를 아직 모름.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. 이름, SNMP, 자체 네트워크

웹 UI의 **수집 상태 → 이름**에는 한 줄에 하나씩 항목을 적습니다. 내용은 데이터 디렉터리에
`inventory.txt`로 저장되므로 이 파일을 직접 편집해도 됩니다(`inventory.txt.example` 참조).
모든 줄은 선택 사항입니다.

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

- `net`: 사설 대역(10/8, 172.16/12, 192.168/16, 100.64/10)은 항상 자체 네트워크로
  취급됩니다. 공인 대역도 추가하면 그 대역을 오가는 트래픽도 자체 트래픽으로 집계됩니다.
  이름은 **Top-N → 세그먼트**와 트래픽 경로에 표시됩니다.
- `snmp <device> <community> [<management address>[:port]]`: device는 플로를 보내는
  주소입니다. 장비가 다른 주소로 SNMP에 응답한다면 관리 주소를 추가합니다. SNMP로 읽은
  인터페이스 설명은 `iface`로 이름을 붙이지 않은 경우 인터페이스 이름으로 쓰입니다. 장비의
  SNMP 접근 목록에서 traffic66 서버를 허용하십시오.
- 변경 사항은 **저장**을 클릭하면 적용되며, 재시작은 필요 없습니다.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. 국가, 네트워크, 위협 목록

국가와 네트워크(AS) 이름을 표시하려면 IP-ASN 매핑 테이블이 필요합니다.
[iptoasn.com](https://iptoasn.com)에서 무료로 내려받을 수 있습니다:

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

같은 형식이면 어떤 파일이든 됩니다(탭 구분: 시작 주소, 끝 주소, AS 번호, 국가 코드, AS 이름.
일반 텍스트 또는 gzip). 파일을 교체한 뒤에는 traffic66을 재시작하고, 한 달에 한 번 정도 새로
내려받으십시오.

위협 목록은 한 줄에 주소나 네트워크 하나씩 적은 일반 텍스트 파일이며(`#` 또는 `;` 뒤는
무시됨), `<data directory>/threats/<name>.txt`로 저장합니다. 예:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

목록을 추가하거나 변경한 뒤에는 traffic66을 재시작합니다. 일치 항목은 **위협 정보**에 목록
이름별로 나타납니다.

<a id="9-using-the-web-ui"></a>

## 9. 웹 UI 사용법

직접 입력할 일은 거의 없습니다. 모든 페이지의 모든 값(주소, 포트, 애플리케이션, 국가, 장비)을
클릭할 수 있습니다:

- **이것만 보기** / **이것 제외**는 필터를 추가합니다. 필터는 상단 바 아래에 표시되며, 제거할
  때까지 모든 페이지에 적용됩니다.
- **플로 레코드 보기**는 해당하는 개별 플로를 엽니다.
- **온라인에서 조회**는 주소나 AS를 공개 조회 사이트에서 엽니다.
- **복사**는 값을 복사합니다.

페이지:

| 페이지 | 알 수 있는 것 |
|---|---|
| 개요 | 현재 트래픽 양과 지난주 대비 변화(애플리케이션별), 늘어난 항목, 상위 클라이언트와 서비스 |
| Top-N | 클라이언트, 서버, 대화, 애플리케이션, 포트, 국가, 네트워크, 세그먼트, 장비, 캡슐화 또는 VLAN의 상위 66개 |
| 트래픽 경로 | 어느 세그먼트가 어느 국가의 어느 애플리케이션과 통신하는지 |
| 지역 및 네트워크 | 국가별, 네트워크(AS)별 트래픽 |
| 위협 정보 | 위협 목록에 있는 주소와 통신한 호스트와 그 전송량 |
| 플로 레코드 | 개별 플로, 최신순, 표시할 열 선택 가능 |
| 인터페이스 대조 | 플로 수치와 인터페이스 카운터 비교, 차이가 큰 순, 원인 포함 |
| 수집 상태 | 장비, 샘플링, 유실, 수집기, SNMP, **이름** |

페이지 위쪽에는 시간 범위(15분~30일), 선택적 검색 상자, 30초마다 자동 새로 고침, 그리고
**링크 복사**가 있습니다. **링크 복사**는 현재 보고 있는 화면(페이지, 시간 범위, 필터) 그대로의
링크를 복사하므로 동료에게 보낼 때 편리합니다. 언어는 브라우저 설정을 따르며, 메뉴 맨
아래에서 바꿀 수 있습니다.

긴 시간 범위의 Top-N은 시간 단위 요약에서 가져오므로 필터를 쓸 수 없으며, 페이지에도 그렇게
표시됩니다. 필터를 쓰려면 더 짧은 범위를 선택하십시오.

<a id="10-terminal-ui"></a>

## 10. 터미널 UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

traffic66 서버에서는 데이터 디렉터리를 읽을 수 있으면 `traffic66 tui`가 알아서 로그인합니다
(기본 위치가 아니면 `-data` 지정). 서비스로 실행할 때처럼 traffic66이 다른 사용자로 실행 중이면
대신 `-user`와 `-password`를 사용합니다. `-lang`으로 언어를 고릅니다(`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

키: 1–8 페이지, ↑↓ 선택, Enter 선택한 값에 대한 동작, f 이것만 보기, x 제외, / 검색,
t 시간 범위, c 필터 해제, w 같은 화면을 브라우저에서 열기, q 종료.

<a id="11-local-capture"></a>

## 11. 로컬 캡처

traffic66은 플로 익스포트 외에도, 미러(SPAN) 포트 같은 로컬 네트워크 인터페이스의 패킷으로
직접 플로를 만들 수 있습니다:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: root 권한 또는 `CAP_NET_RAW`, `CAP_NET_ADMIN` 케이퍼빌리티가 필요합니다
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`, 또는 위
  systemd 유닛의 `AmbientCapabilities` 줄).
- macOS: root 권한이 필요합니다(BPF 장치). 따로 설치할 것은 없습니다.
- Windows: 먼저 [Npcap](https://npcap.com)을 설치하십시오.

캡처 중인 인터페이스는 **수집 상태**에 표시됩니다. 같은 패킷이 두 번 보이면(예: 미러 포트 두
곳에서) 두 번 집계됩니다.

<a id="12-options"></a>

## 12. 옵션

`traffic66 -h`와 `traffic66 <command> -h`로 전체 목록을 볼 수 있습니다.

명령:

| 명령 | |
|---|---|
| `traffic66` | 플로를 수집하고 웹 UI 제공 |
| `traffic66 demo` | 위와 같으나 시뮬레이션 네트워크 사용 |
| `traffic66 tui` | 실행 중인 traffic66용 터미널 UI |
| `traffic66 passwd` | 로그인 비밀번호 설정 |
| `traffic66 simulate -to HOST` | 수집기로 시뮬레이션 익스포트 전송 |
| `traffic66 interfaces` | 로컬 캡처용 인터페이스 목록 |
| `traffic66 version` | 버전 출력 |

`traffic66`과 `traffic66 demo`의 옵션:

| 옵션 | 기본값 | |
|---|---|---|
| `-addr` | `:8066` | 웹 UI 주소. `127.0.0.1:8066`이면 이 서버에서만 접근 |
| `-data` | 프로그램 옆의 `traffic66-data` | 데이터 디렉터리 |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP 수집기를 `name=address` 형식으로 쉼표 구분. 비우면 비활성화 |
| `-user` | `admin` | 처음 생성되는 비밀번호와 `-password`에 해당하는 사용자 |
| `-password` | 저장된 비밀번호 | 이번 실행에만 쓰는 비밀번호(`TRAFFIC66_PASSWORD`도 가능) |
| `-retention-days` | `30` | 플로 상세 보관 일수. 요약은 400일 보관 |
| `-memory` | `0.10` | 데이터베이스가 사용할 수 있는 물리 메모리 비율 |
| `-l2-overhead` | `18` | NetFlow/IPFIX 바이트 수에 패킷당 더하는 바이트 |
| `-sampling-wait` | `5m` | 레코드가 샘플링 레이트를 기다리는 시간 |
| `-capture` | | 로컬 인터페이스에서 캡처(반복 지정 가능) |
| `-inventory` | `<data>/inventory.txt` | 이름 파일 |
| `-asn` | `<data>/asn.tsv.gz` | IP-ASN 매핑 테이블 |
| `-threat` | `<data>/threats/*.txt` | 추가 위협 목록, `name=path` 형식(반복 지정 가능) |
| `-dns-upstream` | 시스템 리졸버 | 호스트 이름 표시에 쓰는 DNS 서버 |
| `-dns-rate` | `20` | 초당 최대 역방향 조회 횟수 |
| `-dns-cache` | `2m` | 호스트 이름 캐시 시간 |
| `-no-dns` | | 역방향 조회 안 함 |
| `-tui` | | 터미널 UI도 함께 열기 |

예: 수집 포트 하나 추가, 상세 1년 보관, 웹 UI는 로컬 서버에서만:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. 데이터, 백업, 업그레이드, 제거

모든 데이터는 데이터 디렉터리에 있습니다:

| | |
|---|---|
| `raw/` | 플로 상세, 시간당 압축 파일 하나 |
| `traffic66.duckdb` | 요약, 인터페이스 카운터, 현재 시간대 데이터 |
| `password` | 로그인 비밀번호(해시) |
| `inventory.txt` | 이름(**수집 상태 → 이름**) |
| `asn.tsv.gz`, `threats/` | 추가한 조회 테이블 |

- **백업**: traffic66을 중지하고 디렉터리를 복사합니다. 중지하지 않고 하려면 `raw/`,
  `password`, `inventory.txt`를 복사합니다. 이 경우 현재 시간대 데이터와 요약은 빠집니다.
- **이전**: traffic66을 중지하고 디렉터리를 옮긴 뒤, `-data`로 새 위치를 지정해 시작합니다.
- **업그레이드**: traffic66을 중지하고 프로그램 파일을 교체한 뒤 다시 시작합니다. 데이터는
  그대로 유지됩니다. Linux 예:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **제거**: 서비스나 시작 작업을 중지하고 삭제한 다음([설치](#2-install) 참조), 프로그램
  폴더와 데이터 디렉터리를 삭제합니다.

<a id="14-security"></a>

## 14. 보안

- 웹 UI는 평문 HTTP를 사용합니다. 비밀번호와 데이터가 암호화되지 않은 채 네트워크를 지납니다.
  완전히 신뢰할 수 없는 네트워크에서는 이 서버에서만 수신하도록 하고(`-addr 127.0.0.1:8066`)
  앞단에 TLS 리버스 프록시를 두십시오. 예를 들어 [Caddy](https://caddyserver.com)를 쓴다면:
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  또는 VPN이나 SSH 터널로 접속합니다:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50` 후
  http://127.0.0.1:8066 을 엽니다.
- UDP 수집 포트는 자체 장비 주소에서만 허용하십시오.
- `inventory.txt`의 SNMP 커뮤니티는 평문으로 저장됩니다. 읽기 전용 커뮤니티를 사용하십시오.

<a id="15-sizing"></a>

## 15. 용량 산정

2코어 서버에서 초당 5,000 플로로 측정한 결과: 상세 데이터는 하루 약 12 GB의 디스크와 현재
시간대용 약 1.5 GB를 사용하고, 프로그램은 약 0.5 GB의 메모리와 코어 하나의 6분의 1 정도를
사용합니다. 긴 시간 범위의 개요는 요약에서 가져오며 0.2초 미만이 걸립니다. 상세 데이터 쿼리는
시간당 약 2,200만 행을 스캔합니다. 호스트 하나의 1시간 조회는 1초 미만, 전체 대화의 1시간
Top 66은 약 9초이며, 소요 시간은 범위에 비례해 늘고 코어가 많을수록 줄어듭니다.

따라서 초당 5,000 플로로 30일을 보관하면 디스크가 약 360 GB 필요합니다. 실제 플로 속도
(**수집 상태**에 표시)와 `-retention-days`에 맞춰 환산하십시오.

<a id="16-troubleshooting"></a>

## 16. 문제 해결

| 증상 | 원인과 해결 |
|---|---|
| **수집 상태**에 장비가 없음 | 패킷이 도착하지 않음: [플로 수신 확인](#5-check-that-flows-arrive) 참조 |
| "waiting for the sampling rate" | 장비가 아직 샘플러 옵션을 보내지 않았습니다. 대부분 몇 분 안에 다시 보냅니다. 끝내 보내지 않으면 익스포트하도록 설정하거나(Cisco는 `option sampler-table`), 정말 1:1이라면 이름에서 `unsampled`로 표시합니다 |
| 수치가 인터페이스 카운터보다 낮음 | **인터페이스 대조** 확인: 경로상 유실, 샘플링되지 않은 인터페이스, 또는 플로가 아직 장비 캐시에 있음(액티브 타임아웃이 60초보다 김) |
| 수치가 인터페이스 카운터보다 높음 | 같은 트래픽을 두 인터페이스 또는 두 장비에서 샘플링함 |
| 국가나 네트워크가 표시되지 않음 | IP-ASN 매핑 테이블이 없음: [국가](#8-countries-networks-and-threat-lists) 참조 |
| 비밀번호를 잊어버림 | traffic66 서버에서 `traffic66 passwd -data <data directory>` |
| `Conflicting lock is held` | 다른 traffic66이 이미 이 데이터 디렉터리를 사용 중 |
| `receive buffer is only … KB` | Linux가 UDP 버퍼를 제한함: `net.core.rmem_max=16777216` 설정([Linux](#linux) 참조) |
| `cannot create the data directory` | 이 사용자에게 프로그램 폴더 쓰기 권한이 없음: `-data` 지정 |
| macOS: "cannot be opened" 또는 "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows 캡처: Npcap을 찾을 수 없음 | [Npcap](https://npcap.com) 설치 |
| `address already in use` | 다른 프로그램이 포트를 사용 중: `-addr` 또는 `-listen`으로 다른 포트 지정 |

<a id="17-build-from-source"></a>

## 17. 소스에서 빌드

Go 1.24와 C 컴파일러(gcc 또는 clang. Windows에서는 MinGW-w64)가 필요합니다:

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
