[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | **한국어**

# traffic66 — NetFlow, sFlow, IPFIX 수집기 및 트래픽 분석기

sFlow, NetFlow, IPFIX 플로 분석을 하나의 프로그램으로 처리합니다. 누가 대역폭을 쓰는지, 트래픽이 어디로 가는지, 그 수치가 장비 자체의 인터페이스 카운터와 맞는지를 웹 UI와 터미널 UI로 보여 줍니다.

## 빠른 시작

```
# Linux, macOS: ./traffic66 에 내려받고 수집 시작
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows(PowerShell): 동일
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

터미널에 웹 UI 주소와 장비가 보낼 IP·UDP 포트가 표시됩니다([장비 설정](MANUAL.ko.md#device-configuration)). `admin` / `traffic66` 로 로그인하고 새 비밀번호를 정하세요. 구경만 하려면 가상 장비 데모:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

포트: TCP 8066 웹 UI, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX(모든 UDP 포트가 모든 프로토콜을 받음). 장비가 보낼 주소는 터미널에 표시되고, 첫 로그인 때 새 비밀번호를 정합니다.

macOS 에서 직접 설치할 때는 먼저 `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows "PC 보호" 창: **추가 정보** → **실행**.

## 명령 요약

| 용도 | 명령 |
|---|---|
| sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 수집(기본) | `./traffic66` |
| NetFlow v5/v9(2055)와 IPFIX(4739)만 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| sFlow 만, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| 웹 UI 를 지정한 IP·포트로 | `./traffic66 -addr 10.0.0.5:8080` |
| 이 컴퓨터 NIC 캡처(Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin 비밀번호 설정·재설정(-data 사용 시 추가) | `./traffic66 passwd` |
| 원격 서버에 터미널 UI 접속 | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![개요: 미처리 탐지, 어제 같은 시각과 비교한 애플리케이션별 대역폭, 상위 클라이언트와 서비스](images/overview.png)

## 전체 설명서

웹 UI 메뉴의 **도움말**, GitHub: [MANUAL.ko.md](MANUAL.ko.md)

15개 언어 지원. 평가 용도와 100명 미만 조직은 무료입니다([라이선스](../LICENSE.md)).
