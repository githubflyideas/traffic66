[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | **한국어**

# traffic66 — NetFlow, sFlow, IPFIX 수집기 및 트래픽 분석기

sFlow, NetFlow, IPFIX 플로 분석을 하나의 프로그램으로 처리합니다. 누가 대역폭을 쓰는지, 트래픽이 어디로 가는지, 그 수치가 장비 자체의 인터페이스 카운터와 맞는지를 웹 UI와 터미널 UI로 보여 줍니다.

## 빠른 시작

1. 시스템에 맞는 압축 파일을 내려받으세요: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. 압축을 풀고 프로그램을 실행합니다:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: `traffic66-windows-amd64.zip` 을 풀고 `traffic66.exe` 실행(더블클릭 가능).

macOS: 먼저 그 폴더에서 `xattr -dr com.apple.quarantine .` 실행.

그다음:

- 웹 UI: `http://<서버 IP>:8066`. 모든 네트워크 인터페이스에서 대기합니다.
- 로그인: 사용자 `admin`, 비밀번호 `traffic66`. 첫 로그인 때 새 비밀번호를 정해야 합니다.
- 장비가 서버 IP 로 보내게 하세요: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739(모든 인터페이스, 모든 포트가 모든 프로토콜을 받음).

## 자주 쓰는 명령

- `./traffic66`  
  위 기본값으로 시작.

- `./traffic66 -addr :9000`  
  웹 UI 를 8066 대신 9000 포트로.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  수집 포트 변경(여기서는 NetFlow 를 9995 로).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  모든 인터페이스 대신 IP 주소 하나에서만 대기.

- `./traffic66 -data /var/lib/traffic66`  
  데이터를 다른 디렉터리에 저장(기본: 프로그램 옆 `traffic66-data`).

- `./traffic66 passwd`  
  admin 비밀번호 설정·재설정(`-data` 를 쓰면 `-data DIR` 추가).

- `traffic66.exe interfaces`  
  Windows: 네트워크 카드 목록(이름과 번호).

- `traffic66.exe -capture Wi-Fi`  
  Windows: 이 PC 의 해당 카드 트래픽 분석(이름 또는 번호). Npcap(npcap.com) 필요.

- `./traffic66 interfaces`  
  Linux, macOS: 네트워크 카드 목록.

- `sudo ./traffic66 -capture eth0`  
  Linux: 이 컴퓨터 eth0 의 트래픽 분석(macOS: `-capture en0`).

- `./traffic66 tui`  
  traffic66 컴퓨터에서 터미널 UI.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  원격 traffic66 에 터미널 UI 로 접속.

- `./traffic66 demo`  
  가상 장비로 체험.

## 스크린샷

<img src="images/overview.png" alt="개요" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="지역 및 네트워크" width="49%"> <img src="images/interfaces.png" alt="인터페이스 대조" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## 전체 설명서

웹 UI 메뉴의 **도움말**, GitHub: [MANUAL.ko.md](MANUAL.ko.md)

15개 언어 지원. 평가 용도와 100명 미만 조직은 무료입니다([라이선스](../LICENSE.md)).
