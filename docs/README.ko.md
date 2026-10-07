[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | **한국어**

# traffic66 — NetFlow, sFlow, IPFIX 수집기 및 트래픽 분석기

sFlow, NetFlow, IPFIX 플로 분석을 하나의 프로그램으로 처리합니다. 누가 대역폭을 쓰는지, 트래픽이 어디로 가는지, 그 수치가 장비 자체의 인터페이스 카운터와 맞는지를 웹 UI와 터미널 UI로 보여 줍니다.

ntopng, ElastiFlow, pmacct와 Grafana 조합, 또는 PRTG와 SolarWinds NTA의 플로 모듈을 대신하는 셀프 호스팅 대안입니다. 네트워크 모니터링(network monitoring), 대역폭 모니터링(bandwidth monitoring), 상위 트래픽 사용자(top talkers), DDoS 및 스캔 탐지, pcap 분석에 쓸 수 있으며, Elasticsearch, Kafka나 별도의 데이터베이스가 필요 없습니다.

- Windows, Linux, macOS 모두 실행 파일 하나입니다. 설치할 데이터베이스가 없고, 오프라인에서도 동작합니다.
- 어떤 UDP 포트로든 sFlow v5, NetFlow v5/v9, IPFIX를 받으며, 인터페이스에서 로컬 캡처도 할 수 있습니다.
- 자체 수치를 인터페이스 카운터(sFlow 또는 SNMP)와 대조하고, 차이가 나는 이유를 알려 줍니다.
- 스캔, 비밀번호 대입, 내부 확산, 비정상 업로드, 플러드, 위협 목록 트래픽을 샘플링을 거쳐서도 찾아냅니다.
- `traffic66 capture.pcap`은 설정할 것 없이 패킷 캡처를 분석합니다.
- 13개 언어 지원. 평가 용도와 100명 미만 조직은 무료입니다([라이선스](#licence)).

![개요: 미처리 탐지, 어제 같은 시각과 비교한 애플리케이션별 대역폭, 상위 클라이언트와 서비스](images/overview.png)

<sub>모든 스크린샷은 가상의 회사 네트워크인 `traffic66 demo`에서 찍은 것입니다.</sub>

<a id="contents"></a>

## 목차

1. [데모 실행해 보기](#1-try-the-demo)
2. [설치](#2-install)
3. [사용자와 비밀번호](#3-users-and-passwords)
4. [장비에서 플로 보내기](#4-send-flows-from-your-devices)
5. [플로 수신 확인](#5-check-that-flows-arrive)
6. [인터페이스와 카운터](#6-interfaces-and-counters)
7. [이름, 국가, 위협 목록](#7-names-countries-and-threat-lists)
8. [웹 UI 사용법](#8-using-the-web-ui)
9. [오프라인 pcap, 터미널 UI, 로컬 캡처](#9-offline-pcap-terminal-ui-local-capture)
10. [옵션과 데이터](#10-options-and-data)
11. [보안, 용량 산정, 문제 해결](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. 데모 실행해 보기

[Releases 페이지](https://github.com/githubflyideas/traffic66/releases)에서 시스템에 맞는 압축 파일(Windows x64, 커널 3.2 이상의 Linux x86-64/ARM64, macOS 11 이상)을 내려받아 풀고 실행합니다:

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

macOS에서는 먼저 `xattr -dr com.apple.quarantine <folder>`를 실행하십시오. http://127.0.0.1:8066을 열고 `admin` / `try66`으로 로그인하면, 하루치 이력과 가상 장비 네 대의 실시간 트래픽이 보입니다. 여기에는 공격이 하나 포함되어 있으며 **탐지**에서 단계별로 보여 줍니다. Ctrl+C로 중지하고, `traffic66-demo`를 삭제하면 처음부터 다시 시작합니다. 실제 설치와 나란히 실행하려면: `-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. 설치

traffic66은 파일 하나입니다. 포트: UDP 6343(sFlow), 2055(NetFlow), 4739(IPFIX), TCP 8066(웹 UI). 모든 UDP 포트가 모든 프로토콜을 받습니다.

**Linux**(systemd):

```
sudo mkdir -p /opt/traffic66 && sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

`/etc/systemd/system/traffic66.service`:

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

**Windows**(관리자 권한 PowerShell): `C:\traffic66`에 압축을 풀고, `C:\traffic66\traffic66.exe passwd`를 실행하고, 포트를 연 다음, 부팅 시 시작하도록 등록합니다:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`traffic66.exe`를 더블클릭해도 됩니다. 웹 UI가 열리고, 첫 비밀번호가 창에 표시됩니다.

**macOS**: `/usr/local/traffic66`에 압축을 풀고, quarantine 플래그를 제거하고, `traffic66 passwd -data "/Library/Application Support/traffic66"`를 실행한 뒤, LaunchDaemon으로 시작합니다. `ProgramArguments`는 프로그램, `-data`, 그 디렉터리로 하고 `RunAtLoad`와 `KeepAlive`를 지정합니다.

<a id="3-users-and-passwords"></a>

## 3. 사용자와 비밀번호

처음 시작할 때 traffic66은 임의의 비밀번호로 사용자 `admin`을 만들고 그 비밀번호를 한 번만 출력합니다(창, 터미널, 또는 `journalctl -u traffic66 | grep "first start"`). 사용자는 솔트를 적용한 해시로 데이터 디렉터리의 `password`에 저장되며, traffic66 서버에서 명령 하나로 관리합니다(traffic66을 `-data …`와 함께 실행한다면 같은 옵션을 추가):

| 작업 | 명령 |
|---|---|
| `admin`의 비밀번호 변경 | `traffic66 passwd` |
| `alice` 추가 또는 비밀번호 변경 | `traffic66 passwd -user alice` |
| `alice` 삭제 | `traffic66 passwd -user alice -delete` |
| 사용자 목록 보기 | `traffic66 passwd -list` |

변경은 즉시 적용됩니다. 모든 사용자의 권한은 같습니다. 스크립트와 컨테이너에서는 `TRAFFIC66_PASSWORD=…`(또는 `-password`)를 지정하면 그 실행에서는 `-user`와 그 비밀번호만 받습니다. 1분 안에 비밀번호를 다섯 번 틀리면 그 주소는 1분 동안 차단됩니다.

<a id="4-send-flows-from-your-devices"></a>

## 4. 장비에서 플로 보내기

`192.0.2.50`은 traffic66, `192.0.2.1`은 장비입니다. 액티브 타임아웃을 60초로 설정하고, NetFlow/IPFIX 장비가 샘플러 옵션을 익스포트하게 하고, **모든 인터페이스의 수신 방향**(또는 에지 인터페이스만)을 샘플링하십시오. 그러면 각 패킷이 한 번만 집계됩니다. sFlow 샘플링 비율: 1 Gb/s는 약 1:1000, 10 Gb/s는 1:4096, 40/100 Gb/s는 1:8192.

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

## 5. 플로 수신 확인

**설정**에는 무엇이든 보내온 장비가 몇 초 안에 모두 나타납니다: 프로토콜, 샘플링 비율, 유실, 샘플링하는 인터페이스, 그리고 상태가 초록이 아닐 때 무엇을 고쳐야 하는지. sFlow 유실은 경로상의 유실(`netstat -su`에 버퍼 오류가 보이면 `net.core.rmem_max`를 올리십시오)과 장비 자체가 버린 샘플로 나뉘어 표시됩니다.

![설정: 장비별 프로토콜, 샘플링, 유실, 고쳐야 할 점](images/sources.png)

장비가 보이지 않습니까? `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`를 실행하십시오. 아무것도 보이지 않으면 라우팅, 방화벽 또는 장비 설정 문제이고, 패킷은 보이는데 **설정**에 아무것도 없으면 로컬 방화벽이나 `-listen` 문제입니다. 다른 머신에서 `traffic66 simulate -to 192.0.2.50`을 실행하면 가상 장비로 경로를 테스트할 수 있습니다.

<a id="6-interfaces-and-counters"></a>

## 6. 인터페이스와 카운터

플로 수치는 추정값입니다(샘플 수 × 샘플링 비율). **인터페이스 대조**는 이를 장비의 인터페이스 카운터(sFlow 카운터, 또는 **이름**의 `snmp` 줄을 통한 SNMP)와 비교하고 차이 나는 이유를 알려 줍니다: 샘플링되지 않은 인터페이스, 같은 트래픽의 중복 샘플링, 경로상 유실, 또는 알 수 없는 샘플링 비율. 각 인터페이스에는 bits/s 차트와 packets/s 차트가 있으며, 수신은 초록, 송신은 파랑, 카운터는 점선입니다.

각 행에서 **✎**는 이름과 짧은 태그(*uplink* 등)를 설정하고, **☆**는 그 인터페이스를 기본 인터페이스(★)로 만듭니다. 페이지는 이 인터페이스로 열립니다.

일부 인터페이스만 샘플링하는 장비는 그 플로의 반대쪽 인터페이스도 보여 줍니다. 이 **상대 인터페이스**는 맨 끝에 작은 회색 글씨로 나열되며, 샘플링된 인터페이스를 지나간 트래픽만 담고 있습니다. 샘플링된 인터페이스는 sFlow 데이터 소스나 flowDirection 필드(IPFIX 61)로 알 수 있고, 그것이 없으면 장비 트래픽의 90%가 지나는 인터페이스로 판단합니다.

![인터페이스 대조: 모든 인터페이스의 트래픽, 그리고 플로 추정값과 장비 카운터를 나란히 표시](images/interfaces.png)

traffic66은 이미 장비가 적용한 비율을 사용하고, 알 수 없는 비율은 기다리고, 익스포트 유실을 보정하고, 긴 플로를 해당 분들에 나눠 담고, NetFlow/IPFIX에는 패킷당 18바이트의 이더넷 오버헤드를 더합니다(`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. 이름, 국가, 위협 목록

아무 주소나 클릭해 **이름 붙이기…**를 선택하거나, **설정 → 이름**을 사용하십시오. 이름은 데이터 디렉터리의 `inventory.txt`에 저장됩니다:

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

사설 주소 대역은 항상 자체 네트워크로 취급합니다. 변경은 **저장**하면 적용되며 재시작이 필요 없습니다.

국가와 네트워크(AS)는 DB-IP의 무료 Lite 데이터베이스([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP Geolocation by DB-IP", [db-ip.com](https://db-ip.com))로 바로 동작합니다. **설정**에서 이를 업데이트하거나, 대신 MaxMind GeoLite2, IPinfo Lite, IPtoASN 파일을 쓸 수 있습니다. 지도 윤곽: [Natural Earth](https://www.naturalearthdata.com).

![지역 및 네트워크: 세계 지도에 표시한 국가별 외부 트래픽](images/geo.png)

위협 목록은 한 줄에 주소나 네트워크 하나를 적은 텍스트 파일로, `<data>/threats/<name>.txt`에 둡니다(예: Spamhaus DROP). 변경한 뒤에는 재시작하십시오. 일치 항목은 **위협 정보**에 나타납니다.

![위협 정보: 위협 목록에 있는 주소로 데이터를 보내는 내부 호스트](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. 웹 UI 사용법

모든 페이지의 모든 값을 클릭할 수 있습니다: **이것만 보기** / **이것 제외**(필터는 모든 페이지에 적용), **플로 레코드 보기**, **상세 보기**(호스트나 서비스 하나에 대한 페이지), **이름 붙이기…**, **온라인에서 조회**, **복사**.

| 페이지 | 보여 주는 것 |
|---|---|
| 개요 | 선택한 인터페이스의 대역폭, 애플리케이션별 트래픽(합계, 인바운드, 아웃바운드)과 어제 또는 지난주와의 비교, 미처리 탐지, 상위 클라이언트와 서비스 |
| Top 66 | 상위 66개 대화(어느 열로든 정렬), 또는 애플리케이션, 네트워크, 세그먼트, 장비, 캡슐화, VLAN별 그룹, 상위 30개 통신 상대 |
| 트래픽 상세 | 링 차트: 서버와 그 클라이언트(또는 그 반대), 서비스 |
| 트래픽 경로 | 호스트 → 애플리케이션 → 국가, 또는 클라이언트 → 서비스 → 서버, 또는 네트워크별 |
| 인터페이스 대조 | 모든 인터페이스의 시간에 따른 트래픽과 카운터 비교, 이름, 태그, 기본 인터페이스 |
| 플로 레코드 | 개별 플로. 5초마다 실시간으로, 또는 임의의 시간 범위 |
| 탐지, 위협 정보 | 살펴봐야 할 것([아래](#findings)), 목록에 있는 주소와의 트래픽 |
| 지역 및 네트워크 | 국가별 세계 지도, 시간에 따른 네트워크(AS) |
| 설정 | 장비, 샘플링, 유실, SNMP, 데이터베이스, 로고, 이름 |
| 오프라인 pcap 분석, 데이터 정리 | 캡처 파일([아래](#9-offline-pcap-terminal-ui-local-capture)), 오래된 데이터 삭제 |

페이지 위쪽에는 **인터페이스**(전체, 또는 샘플링된 인터페이스 하나. 이때 트래픽 페이지는 그 인터페이스를 지나는 트래픽만 보여 줍니다), 시간 범위(15분~30일, 또는 사용자 지정), 30초마다 새로 고침, 그리고 정확히 현재 화면을 가리키는 **링크 복사**가 있습니다. 언어와 다섯 가지 색상 테마는 메뉴 맨 아래에 있습니다. 차트는 데이터가 완전한 지점에서 끝납니다. NetFlow/IPFIX에서는 장비가 익스포트하는 만큼 늦어집니다(최대 2분). 6시간이 넘는 범위는 정시에 시작합니다. 인터페이스 하나를 7일 또는 30일로 보면 플로 상세를 읽으므로 더 느리고, 상세 보관 기간까지만 거슬러 올라갑니다.

![Top 66: 상위 66개 대화, 어느 열로든 정렬](images/topn.png)

![트래픽 상세: 서버와 그 클라이언트, 서비스와 그 서버를 보여 주는 링 차트](images/traffic.png)

![호스트 하나의 상세: 그 호스트에 관한 탐지, 트래픽, 통신 상대, 서비스, 국가, 최근 플로](images/detail.png)

![트래픽 경로: 어느 호스트가 어느 국가로 어느 애플리케이션을 쓰는지](images/paths.png)

![중국어로 본 개요](images/overview-zh.png)

<a id="findings"></a>

### 탐지

5분마다 최근 10분을 검사합니다. 한 시간 동안 이어지는 일은 하나의 탐지 항목으로 계속 커집니다.

| 탐지 | 의미 |
|---|---|
| 스캔, 포트 스캔 | 여러 호스트의 같은 포트, 또는 한 호스트의 여러 포트로 보내는 작은 탐색 패킷 |
| 비밀번호 대입 | 로그인 서비스로의 짧은 연결이 많음 |
| 내부 확산 | 파일 공유나 원격 관리를 제공한 적이 없는 내부 호스트로의 그런 세션 |
| 비정상 업로드 | 새 주소로 10분 동안 100 MB, 돌아온 양의 세 배 |
| 플러드 | 한 주소로 초당 20,000개 이상의 작은 패킷, 평소 속도의 열 배 |
| 위협 목록 | 목록에 있는 주소와의 트래픽 |

내부 네트워크에서 오면 높음, 인터넷에서 오면 낮음입니다. **처리 완료**는 탐지 항목을 닫고, **문제 없음**은 영구히 끕니다. 내부 확산과 비정상 업로드에는 하루치 이력이 필요합니다. 1:4096 샘플링을 거쳐도 데모의 공격은 모두 탐지되지만, 아주 작은 스캔은 샘플링 뒤에 숨을 수 있습니다.

![탐지: 1:4096 sFlow 샘플링을 거쳐 찾아낸 공격의 모든 단계](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. 오프라인 pcap, 터미널 UI, 로컬 캡처

**오프라인 pcap 분석**은 패킷 캡처(pcap, pcapng)를 실시간 데이터와 따로, 같은 페이지로 보여 줍니다. `traffic66 a.pcap b.pcapng`는 127.0.0.1에서 시작해 브라우저를 엽니다(최대 3개 파일, 3 GB. Ctrl+C로 가져온 데이터를 삭제). 또는 그 페이지에서 50 MB까지의 파일을 최대 3개 업로드할 수 있습니다. 다루는 것은 플로이며, 패킷 내용이 아닙니다.

![오프라인 분석: 캡처 파일과 패킷, 플로, 시간](images/sandbox.png)

**터미널 UI**: traffic66 서버에서 `traffic66 tui`, 또는 `traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`. 키: 1–8 페이지, Enter 작업, f 이것만 보기, x 제외, t 시간 범위, w 브라우저에서 열기, q 종료. `-lang`으로 언어를 고릅니다.

![터미널 UI: 개요](images/tui-overview.png)

![터미널 UI: Top 66 대화](images/tui-topn.png)

**로컬 캡처**는 로컬 인터페이스에서 플로를 만듭니다. 스위치의 미러 포트에 연결한 포트가 가장 좋습니다. `traffic66 interfaces`로 목록을 보고, `-capture eth1`(또는 Windows의 이름이나 번호)로 캡처합니다. Linux는 root 또는 `setcap cap_net_raw,cap_net_admin+ep`, macOS는 root, Windows는 [Npcap](https://npcap.com)이 필요합니다. 캡처한 플로는 장비 `127.0.0.1`에서 온 것으로 표시됩니다.

<a id="10-options-and-data"></a>

## 10. 옵션과 데이터

`traffic66 -h`가 전체를 보여 줍니다. 자주 쓰는 옵션:

| 옵션 | 기본값 | |
|---|---|---|
| `-data` | 프로그램 옆의 `traffic66-data` | 데이터 디렉터리 |
| `-addr` | `:8066` | 웹 UI. `127.0.0.1:8066`이면 이 머신에서만 |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP 수집기. 비우면 비활성화 |
| `-retention-days` | `30` | 플로 상세 보관 일수. 요약은 400일 보관 |
| `-memory` | `0.10` | 데이터베이스 캐시에 쓰는 메모리 비율 |
| `-sampling-wait` | `5m` | 레코드가 샘플링 비율을 기다리는 시간 |
| `-capture` | | 로컬 인터페이스(반복 지정 가능) |
| `-no-dns` | | 역방향 조회 안 함 |

데이터 디렉터리에는 `raw/`(상세, 시간당 파일 하나), `traffic66.duckdb`(요약과 카운터), `password`, `inventory.txt`, `license.json`, 로고와 데이터베이스가 있습니다. 백업은 traffic66을 중지하고 디렉터리를 복사하며, 업그레이드는 프로그램 파일을 교체합니다. **데이터 정리**는 7~120일보다 오래된 데이터 또는 모든 데이터를 삭제합니다.

<a id="licence"></a>

### 라이선스

[PolyForm Noncommercial License 1.0.0](../LICENSE.md)과 [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md)(영문이 구속력을 가짐)에 따른 소스 공개: 평가 용도와 100명 미만 조직은 무료이고, 더 큰 조직은 운영 사용 30일 뒤 등록합니다. 판매, 타인을 위한 호스팅, 경쟁 제품에는 상용 라이선스가 필요합니다. 꺼지는 기능은 전혀 없습니다. 모든 페이지 하단에 8자리 설치 번호가 표시되니 작성자에게 보내고, 돌려받은 `license.json`을 데이터 디렉터리에 넣으십시오. 연락처: <https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. 보안, 용량 산정, 문제 해결

웹 UI는 평문 HTTP입니다. 신뢰할 수 없는 네트워크에서는 `-addr 127.0.0.1:8066`을 쓰고 TLS 프록시(`caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`)나 SSH 터널 뒤에 두십시오. UDP 포트는 자체 장비에서만 허용하십시오. SNMP 커뮤니티는 평문으로 저장되므로 읽기 전용 커뮤니티를 쓰십시오.

2코어에서 초당 5,000 플로일 때: 상세 데이터 하루당 디스크 약 12 GB(30일이면 360 GB), 코어 1개의 6분의 1, 메모리 0.6–0.8 GB. 긴 범위의 개요는 0.2초 미만, 전체 대화의 1시간 Top 66은 약 9초 걸립니다.

| 증상 | 해결 |
|---|---|
| "waiting for the sampling rate" | 샘플러 옵션을 익스포트하거나, device 줄에 `sampling=N` / `unsampled` 지정 |
| 카운터보다 낮음 | 샘플링되지 않은 인터페이스, 유실, 또는 60초를 넘는 액티브 타임아웃 |
| 카운터보다 높음 | 같은 트래픽을 두 인터페이스 또는 두 장비에서 샘플링함 |
| 비밀번호를 잊어버림 | traffic66 서버에서 `traffic66 passwd` |
| `Conflicting lock is held` | 다른 traffic66이 이 데이터 디렉터리를 사용 중 |
| `address already in use` | `-addr` 또는 `-listen`으로 다른 포트 지정 |
| Windows: "Windows의 PC 보호" | **추가 정보** → **실행** |

소스에서 빌드: Go 1.24와 C 컴파일러를 준비한 뒤 `scripts/build.sh 0.1.0 traffic66`.
