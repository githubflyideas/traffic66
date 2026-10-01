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

![개요: 지난주와 비교한 애플리케이션별 대역폭, 상위 클라이언트와 서비스](images/overview.png)

<sub>모든 스크린샷은 직접 실행해 볼 수 있는 가상의 회사 네트워크인 `traffic66 demo`에서 찍은 것입니다([데모 실행해 보기](#1-try-the-demo) 참조).</sub>

<a id="contents"></a>

## 목차

1. [데모 실행해 보기](#1-try-the-demo)
2. [설치](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [사용자와 비밀번호](#3-users-and-passwords)
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
| Linux x86-64: 커널 3.2 이상의 모든 배포판(CentOS 7, Alpine 포함) | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: 동일한 배포판 | `traffic66-linux-arm64.tar.gz` |
| macOS 11 이상, Apple 실리콘 | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 이상, Intel | `traffic66-darwin-amd64.tar.gz` |

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
만한 사고 두 건이 들어 있습니다. **개요**에서 시작해 **상위 클라이언트**의 호스트를 클릭하고
**상세 보기**를 고른 뒤, 거기서부터 계속 클릭해 보십시오. Ctrl+C로 중지합니다. 데모 데이터는 프로그램 옆의 `traffic66-demo`에
저장되며, 이 폴더를 삭제하면 데모를 처음부터 다시 시작할 수 있습니다.

데모는 실제 설치와 같은 포트(8066, UDP 6343, 2055, 4739)를 사용합니다. 실제 인스턴스와
함께 실행하려면 다른 포트를 지정합니다:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

Windows에서는 `traffic66.exe`를 더블클릭해도 됩니다. 이렇게 하면 데모가 아닌
실제 traffic66이 시작되고 브라우저에서 웹 UI가 열립니다. 첫 시작 비밀번호는 검은
창에 표시되며, 창을 닫으면 traffic66이 중지됩니다. Windows가 "Windows의 PC 보호"
메시지를 표시하면 **추가 정보** → **실행**을 클릭합니다.

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
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
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

<a id="3-users-and-passwords"></a>

## 3. 사용자와 비밀번호

**요약:** 사용자와 비밀번호는 데이터 디렉터리의 파일 하나, `password`에 저장됩니다.
이 파일을 직접 편집할 일은 없습니다. 사용자 추가, 변경, 목록 확인, 삭제는
`traffic66 passwd` 명령으로 합니다. `http://<traffic66 machine>:8066`을 열고
그중 한 사용자로 로그인합니다.

<a id="the-first-sign-in"></a>

### 첫 로그인

traffic66은 처음 시작할 때 임의의 비밀번호로 사용자 `admin`을 만들고, 그 비밀번호를
한 번만 보여 줍니다:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Windows에서 더블클릭으로 시작한 경우: 검은 창에 표시됩니다.
- 터미널에서 시작한 경우: 그 터미널에 표시됩니다.
- Linux 서비스: `journalctl -u traffic66 | grep "first start"`
- macOS 서비스: `grep "first start" /Library/Logs/traffic66.log`

놓쳤다면 `traffic66 passwd`(아래 참조)로 새 비밀번호를 설정합니다. 위의 설치 절차처럼
첫 시작 전에 `traffic66 passwd`로 비밀번호를 설정해 두었다면 비밀번호가 생성되지
않습니다.

<a id="where-the-users-are-stored"></a>

### 사용자가 저장되는 곳

데이터 디렉터리의 `password` 파일입니다:

| traffic66 실행 방식 | 파일 |
|---|---|
| 압축을 풀고 그 폴더에서 시작(기본) | 프로그램 옆의 `traffic66-data/password` |
| Linux 서비스(섹션 2) | `/var/lib/traffic66/password` |
| Windows 시작 작업(섹션 2) | `C:\traffic66\traffic66-data\password` |
| macOS 서비스(섹션 2) | `/Library/Application Support/traffic66/password` |
| 데모 | 프로그램 옆의 `traffic66-demo/password` |

사용자마다 한 줄입니다. 비밀번호는 솔트를 넣은 해시로 저장되므로 누구도 파일에서
비밀번호를 다시 읽어 낼 수 없으며, 본인도 마찬가지입니다. 비밀번호를 잊어버렸다면
새로 설정합니다. 이 파일은 소유자만 읽을 수 있습니다.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### 사용자 관리

traffic66 서버에서 다음을 실행합니다:

| 작업 | 명령 |
|---|---|
| `admin`의 비밀번호 변경 | `traffic66 passwd` |
| 사용자 `alice` 추가 또는 비밀번호 변경 | `traffic66 passwd -user alice` |
| 사용자 `alice` 삭제 | `traffic66 passwd -user alice -delete` |
| 사용자 목록 보기 | `traffic66 passwd -list` |
| 임의의 비밀번호를 설정하고 출력 | `traffic66 passwd -generate`(다른 사용자는 `-user` 추가) |

- 명령은 새 비밀번호를 두 번 묻고, 입력하는 내용은 화면에 표시하지 않습니다.
  8자 이상을 사용합니다.
- traffic66을 `-data`와 함께 실행한다면 명령에도 같은 `-data`를 붙입니다.
  섹션 2의 Linux 서비스라면:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Windows에서는(관리자 권한 PowerShell):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- 변경 사항은 재시작 없이 바로 적용됩니다. 새 비밀번호는 다음 로그인부터 쓸 수 있고,
  삭제된 사용자는 열려 있는 브라우저에서 로그아웃됩니다.
- 마지막 남은 사용자는 삭제할 수 없습니다. 먼저 다른 사용자를 추가합니다.
- 모든 사용자가 같은 것을 보고 바꿀 수 있습니다. 역할 구분은 없습니다.

<a id="passwords-for-scripts-and-containers"></a>

### 스크립트와 컨테이너용 비밀번호

환경 변수 `TRAFFIC66_PASSWORD=…` 또는 명령줄의 `-password …`를 쓰면 traffic66은
그 실행 동안 사용자 한 명만 받아들입니다. `-user`로 지정한 사용자(기본값 `admin`)와
그 비밀번호입니다. 이때 `password` 파일은 무시되며 변경되지도 않습니다. 환경 변수를
권장합니다. 명령줄은 같은 서버의 다른 사용자에게 보이기 때문입니다.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

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

![수집 상태: 장비별 프로토콜, 샘플링, 유실, 고쳐야 할 점](images/sources.png)

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

![인터페이스 대조: 인터페이스마다 플로 추정값과 장비 카운터를 나란히 표시](images/interfaces.png)

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

호스트나 장비에 이름을 붙이는 가장 빠른 방법: 아무 페이지에서나 그 주소를 클릭하고
**이름 붙이기…** 메뉴를 고릅니다. 이름을 입력하고 Enter를 누르면 바로 저장되고, 이후 모든
곳에서 주소 대신 그 이름이 표시됩니다.

네트워크, 인터페이스, SNMP는 웹 UI의 **수집 상태 → 이름**에 한 줄에 하나씩 항목을 적습니다.
내용은 데이터 디렉터리에
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

국가와 네트워크(AS) 이름을 표시하려면 주소를 국가와 네트워크로 매핑하는 데이터베이스가
필요합니다. 웹 UI에서 업로드하십시오: **수집 상태 → 국가 및 네트워크 데이터베이스 →
데이터베이스 파일 업로드…**. 파일은 검사를 거쳐 데이터 디렉터리에 저장되고 새 트래픽에 바로
사용되며, 재시작할 필요는 없습니다. 이미 저장된 트래픽은 저장될 때의 국가를 유지합니다.

사용할 수 있는 파일:

| 파일 | 제공 정보 | 구하는 곳 |
|---|---|---|
| DB-IP Lite country 또는 ASN, `.mmdb` | 국가, 또는 AS 번호와 이름 | 무료, 계정 불필요: [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country 또는 ASN, `.mmdb` | 국가, 또는 AS 번호와 이름 | MaxMind 계정이 있으면 무료: [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IP-ASN 매핑 테이블, `.tsv` 또는 `.tsv.gz` | AS 번호, AS 이름, 국가 | 무료: [iptoasn.com](https://iptoasn.com) (`ip2asn-combined.tsv.gz`) |

국가 데이터베이스와 ASN 데이터베이스를 함께 업로드하면 둘 다 얻을 수 있습니다. 여러 개가
로드되어 있으면 `.mmdb` 파일에 담긴 정보는 그 파일이 우선합니다. 새 버전은 매달 나옵니다.
같은 방법으로 새 파일을 업로드하면 이전 파일을 대체합니다.

웹 UI 없이 하려면 파일을 데이터 디렉터리에 `country.mmdb`, `asn.mmdb` 또는 `asn.tsv.gz`로
복사하고 traffic66을 재시작하십시오.

위협 목록은 한 줄에 주소나 네트워크 하나씩 적은 일반 텍스트 파일이며(`#` 또는 `;` 뒤는
무시됨), `<data directory>/threats/<name>.txt`로 저장합니다. 예:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

목록을 추가하거나 변경한 뒤에는 traffic66을 재시작합니다. 일치 항목은 **위협 정보**에 목록
이름별로 나타납니다.

![위협 정보: 위협 목록에 있는 주소로 데이터를 보내는 내부 호스트](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. 웹 UI 사용법

직접 입력할 일은 거의 없습니다. 모든 페이지의 모든 값(주소, 포트, 애플리케이션, 국가, 장비)을
클릭할 수 있습니다:

- **이것만 보기** / **이것 제외**는 필터를 추가합니다. 필터는 상단 바 아래에 표시되며, 제거할
  때까지 모든 페이지에 적용됩니다.
- **플로 레코드 보기**는 해당하는 개별 플로를 엽니다.
- **상세 보기**(호스트, 장비, 서비스)는 그 호스트나 서비스 하나에 대한 페이지를 엽니다.
  애플리케이션별 트래픽 추이, 통신 상대, 서비스 또는 클라이언트, 국가, 최근 플로를 보여
  줍니다. 그곳의 모든 값도 다시 클릭할 수 있어 계속 파고들 수 있으며, 브라우저의 뒤로
  버튼으로 돌아옵니다.
- **이름 붙이기…**(호스트와 장비)는 주소에 이름을 붙이며, 이후 모든 곳에 그 이름이
  표시됩니다.
- **온라인에서 조회**는 주소나 AS를 공개 조회 사이트에서 엽니다.
- **복사**는 값을 복사합니다.

페이지:

| 페이지 | 알 수 있는 것 |
|---|---|
| 개요 | 현재 트래픽 양과 지난주 대비 변화(애플리케이션별), 상위 클라이언트와 서비스 |
| Top-N | 상위 66개를 담은 하나의 표: 기본은 대화(클라이언트, 서버, 서비스, 국가). 모든 열 제목으로 정렬할 수 있고, 숫자 열(트래픽, 패킷, 평균 패킷 크기, 플로)은 기간 내 전체 트래픽에서 상위 66개를 다시 뽑으므로 평균 패킷 크기가 작은 순으로 스캔과 플러드를 찾을 수 있습니다. **그룹 기준**에서 애플리케이션, 네트워크, 세그먼트, 장비, 캡슐화, VLAN으로 바꿀 수 있습니다 |
| 트래픽 경로 | 어느 세그먼트가 어느 국가의 어느 애플리케이션과 통신하는지 |
| 지역 및 네트워크 | 국가별, 네트워크(AS)별 트래픽 |
| 위협 정보 | 위협 목록에 있는 주소와 통신한 호스트와 그 전송량 |
| 플로 레코드 | 개별 플로, 최신순, 표시할 열 선택 가능 |
| 인터페이스 대조 | 플로 수치와 인터페이스 카운터 비교, 차이가 큰 순, 원인 포함 |
| 수집 상태 | 장비, 샘플링, 유실, 수집기, SNMP, 국가 및 네트워크 데이터베이스, **이름** |

페이지 위쪽에는 시간 범위(15분~30일), 선택적 검색 상자, 30초마다 자동 새로 고침, 그리고
**링크 복사**가 있습니다. **링크 복사**는 현재 보고 있는 화면(페이지, 시간 범위, 필터) 그대로의
링크를 복사하므로 동료에게 보낼 때 편리합니다. 언어는 브라우저 설정을 따르며, 메뉴 맨
아래에서 바꿀 수 있습니다.

6시간보다 긴 범위는 정시에 시작하므로 페이지의 모든 수치가 정확히 같은 시간을 집계합니다.
"24시간"은 지난 24개의 온전한 시간에 현재 시간을 더한 범위입니다. 이런 범위의 Top-N은
시간 단위 요약에서 가져오므로 필터를 쓸 수 없으며, 페이지에도 그렇게 표시됩니다. 필터를
쓰려면 더 짧은 범위를 선택하십시오. 대화는 항상 플로 상세를 읽으므로 플로가 많은 환경에서
긴 범위를 고르면 시간이 걸릴 수 있습니다. 1시간이 가장 빠릅니다.

사이드 메뉴에는 데이터가 쓰는 디스크 용량과 남은 용량이 표시됩니다. 남은 용량에 마우스를
올리면 현재 속도로 상세 데이터를 보관 일수만큼 유지하는 데 필요한 용량을 볼 수 있습니다
(하루치 데이터가 쌓이면 추정됨).

![Top-N: 최근 1시간의 상위 66개 대화](images/topn.png)

![호스트 하나의 상세: 트래픽, 통신 상대, 서비스, 국가, 최근 플로](images/detail.png)

![트래픽 경로: 어느 세그먼트가 어느 국가로 어느 애플리케이션을 쓰는지](images/paths.png)

같은 개요 화면의 중국어 버전입니다. 모든 페이지를 13개 언어로 볼 수 있습니다:

![중국어로 본 개요](images/overview-zh.png)

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

![터미널 UI: 개요](images/tui-overview.png)

![터미널 UI: Top-N 대화](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. 로컬 캡처

traffic66은 플로 익스포트를 받는 것 외에도, 자신이 실행 중인 머신의 네트워크 인터페이스에서
패킷을 받아 직접 플로를 만들 수 있습니다. 무엇이 보이는지는 인터페이스에 따라 다릅니다:

| 인터페이스 | traffic66에 보이는 것 |
|---|---|
| 스위치의 미러(SPAN) 포트에 연결한 남는 네트워크 포트 | 스위치가 미러링하는 모든 트래픽: 네트워크 전체 또는 업링크 |
| 머신 자체의 이더넷 또는 Wi-Fi | 이 머신 자신의 트래픽만 |

Wi-Fi 어댑터로는 다른 장비의 트래픽을 볼 수 없습니다. Wi-Fi 네트워크 전체를 보려면 라우터나
액세스 포인트가 플로를 익스포트하게 하거나(섹션 4), 액세스 포인트가 연결된 스위치 포트를
미러링하십시오.

<a id="windows-1"></a>

### Windows

1. [Npcap](https://npcap.com)을 기본 옵션으로 설치합니다.
   "Restrict Npcap driver's access to Administrators only"를 선택했다면 traffic66을
   관리자 권한으로 실행하십시오.
2. 인터페이스 목록을 봅니다(PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Name 열은 Windows 네트워크 설정의 연결 이름이며, 사용 중인 인터페이스에는 주소가
   있습니다.
3. Wi-Fi에서 이름이나 번호로 캡처합니다:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   공백이 들어간 이름은 따옴표로 감쌉니다: `-capture "Ethernet 2"`. 여러 인터페이스에서
   캡처하려면 `-capture`를 반복합니다. 플로 수집기 없이 캡처만 하려면 `-listen=`을
   추가합니다. 섹션 2의 시작 작업이라면 `-Argument`에 옵션을 추가합니다:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

캡처에는 root 권한 또는 `CAP_NET_RAW`, `CAP_NET_ADMIN` 케이퍼빌리티가 필요합니다. 위의
`setcap` 줄이나 섹션 2 systemd 유닛의 `AmbientCapabilities` 줄을 쓰면 됩니다. Wi-Fi
인터페이스 이름은 보통 `wlan0` 또는 `wlp…`입니다.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

캡처에는 root 권한이 필요하며, 따로 설치할 것은 없습니다. MacBook에서는 `en0`이 Wi-Fi입니다.

<a id="checking-that-it-works"></a>

### 동작 확인

**수집 상태**에 캡처 중인 인터페이스마다 캡처 방식과 지금까지 본 패킷 수가 표시됩니다. 플로는
모든 페이지에서 다른 장비의 플로와 마찬가지로 장비 `127.0.0.1`(이 머신)에서 온 것으로
나타납니다. 같은 패킷이 두 번 보이면(예: 미러 포트 두 곳에서) 두 번 집계됩니다.

<a id="12-options"></a>

## 12. 옵션

`traffic66 -h`와 `traffic66 <command> -h`로 전체 목록을 볼 수 있습니다.

명령:

| 명령 | |
|---|---|
| `traffic66` | 플로를 수집하고 웹 UI 제공 |
| `traffic66 demo` | 위와 같으나 시뮬레이션 네트워크 사용 |
| `traffic66 tui` | 실행 중인 traffic66용 터미널 UI |
| `traffic66 passwd` | 사용자 추가, 변경, 목록 확인, 삭제([사용자와 비밀번호](#3-users-and-passwords) 참조) |
| `traffic66 simulate -to HOST` | 수집기로 시뮬레이션 익스포트 전송 |
| `traffic66 interfaces` | 로컬 캡처용 인터페이스 목록 |
| `traffic66 version` | 버전 출력 |

`traffic66`과 `traffic66 demo`의 옵션:

| 옵션 | 기본값 | |
|---|---|---|
| `-addr` | `:8066` | 웹 UI 주소. `127.0.0.1:8066`이면 이 서버에서만 접근 |
| `-data` | 프로그램 옆의 `traffic66-data` | 데이터 디렉터리 |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP 수집기를 `name=address` 형식으로 쉼표 구분. 비우면 비활성화 |
| `-user` | `admin` | 첫 시작 때 만들어지는 사용자의 이름이자 `-password`가 적용되는 사용자 |
| `-password` | 설정 안 함 | 이번 실행에서는 `password` 파일을 무시하고 `-user`와 이 비밀번호만 허용(`TRAFFIC66_PASSWORD`도 가능) |
| `-retention-days` | `30` | 플로 상세 보관 일수. 요약은 400일 보관 |
| `-memory` | `0.10` | 데이터베이스 캐시에 쓸 물리 메모리 비율. 프로그램의 나머지 부분에도 같은 크기의 소프트 한도 적용(각각 최소 256 MB) |
| `-l2-overhead` | `18` | NetFlow/IPFIX 바이트 수에 패킷당 더하는 바이트 |
| `-sampling-wait` | `5m` | 레코드가 샘플링 레이트를 기다리는 시간 |
| `-capture` | | 로컬 인터페이스에서 캡처(반복 지정 가능) |
| `-inventory` | `<data>/inventory.txt` | 이름 파일 |
| `-asn` | `<data>/asn.tsv.gz` | IP-ASN 매핑 테이블(`.mmdb` 파일은 업로드하거나 `<data>/country.mmdb`와 `<data>/asn.mmdb`에 둠) |
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
| `country.mmdb`, `asn.mmdb`, `asn.tsv.gz`, `threats/` | 추가한 국가 및 네트워크 데이터베이스와 위협 목록 |

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
시간대용 약 1.5 GB를 사용하고, 프로그램은 코어 하나의 6분의 1 정도를
사용합니다. 긴 시간 범위의 개요는 요약에서 가져오며 0.2초 미만이 걸립니다. 상세 데이터 쿼리는
시간당 약 2,200만 행을 스캔합니다. 호스트 하나의 1시간 조회는 1초 미만, 전체 대화의 1시간
Top 66은 약 9초이며, 소요 시간은 범위에 비례해 늘고 코어가 많을수록 줄어듭니다.

따라서 초당 5,000 플로로 30일을 보관하면 디스크가 약 360 GB 필요합니다. 실제 플로 속도
(**수집 상태**에 표시)와 `-retention-days`에 맞춰 환산하십시오.

메모리: `-memory`(기본값 RAM의 10%, 최소 256 MB)는 데이터베이스 캐시를 제한하며, 프로그램의
나머지 부분에도 같은 크기의 소프트 한도가 적용됩니다. 초당 5,000 플로에서 프로그램 자체 데이터
(디코딩, 중복 검출, 배치)는 약 90 MB이며, 전체로는 0.6–0.8 GB를 예상하면 되므로 RAM 2 GB
서버면 충분합니다. 10분간 연속 수집(8 GB 서버에서 최대 0.58 GB)과, 2 GB 서버의 한도로 그
11배 속도에서 1시간 분량의 플로를 적재할 때(최대 0.74 GB) 측정했습니다.

`-memory`는 예산이지 하드 한도가 아닙니다. Go 한도는 소프트 한도이고 데이터베이스도
잠시 자기 몫을 넘을 수 있습니다. 하드 한도가 필요하면 운영체제의 기능을 쓰세요: systemd 유닛의
`MemoryMax=`(섹션 2) 또는 컨테이너의 메모리 한도. `-memory` 몫의 약 2.5배, 최소 1 GB를
잡으세요. 기본 몫이라면 RAM 8 GB 이하 서버에는 `MemoryMax=2G`가 적당합니다. 그러면 서버
메모리가 바닥나는 대신 traffic66이 재시작됩니다.

<a id="16-troubleshooting"></a>

## 16. 문제 해결

| 증상 | 원인과 해결 |
|---|---|
| **수집 상태**에 장비가 없음 | 패킷이 도착하지 않음: [플로 수신 확인](#5-check-that-flows-arrive) 참조 |
| "waiting for the sampling rate" | 장비가 아직 샘플러 옵션을 보내지 않았습니다. 대부분 몇 분 안에 다시 보냅니다. 끝내 보내지 않으면 익스포트하도록 설정하거나(Cisco는 `option sampler-table`), 정말 1:1이라면 이름에서 `unsampled`로 표시합니다 |
| 수치가 인터페이스 카운터보다 낮음 | **인터페이스 대조** 확인: 경로상 유실, 샘플링되지 않은 인터페이스, 또는 플로가 아직 장비 캐시에 있음(액티브 타임아웃이 60초보다 김) |
| 수치가 인터페이스 카운터보다 높음 | 같은 트래픽을 두 인터페이스 또는 두 장비에서 샘플링함 |
| 국가나 네트워크가 표시되지 않음("알 수 없음") | 로드된 데이터베이스가 없음: **수집 상태**에서 업로드. [국가](#8-countries-networks-and-threat-lists) 참조 |
| 페이지에 "데이터베이스가 메모리 한도에 도달해 응답하지 못했습니다" 표시 | 기간을 줄이거나 더 큰 `-memory`로 시작. 자세한 내용은 로그 참조 |
| 비밀번호를 잊어버림 | traffic66 서버에서 `traffic66 passwd`(traffic66을 `-data`와 함께 실행한다면 `-data` 추가) |
| `Conflicting lock is held` | 다른 traffic66이 이미 이 데이터 디렉터리를 사용 중 |
| `receive buffer is only … KB` | Linux가 UDP 버퍼를 제한함: `net.core.rmem_max=16777216` 설정([Linux](#linux) 참조) |
| `cannot create the data directory` | 이 사용자에게 프로그램 폴더 쓰기 권한이 없음: `-data` 지정 |
| macOS: "cannot be opened" 또는 "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows의 PC 보호" | **추가 정보** → **실행**. 프로그램이 아직 서명되지 않음 |
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
