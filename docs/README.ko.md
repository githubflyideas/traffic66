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

다운로드 확인: `sha256sum -c SHA256SUMS --ignore-missing`. 업그레이드, 되돌리기, 리버스 프록시는 [설명서](MANUAL.ko.md#12-install-upgrade-reverse-proxy-monitoring)를 보세요.

그다음:

- 웹 UI: `http://<서버 IP>:8066`. 모든 네트워크 인터페이스에서 대기합니다.
- 로그인: 사용자 `admin`, 비밀번호 `traffic66`. 첫 로그인 때 새 비밀번호를 정해야 합니다.
- 장비가 서버 IP 로 보내게 하세요: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739(모든 인터페이스, 모든 포트가 모든 프로토콜을 받음).
- 시작 직후 데이터가 없는 것은 정상이니 잠시 기다리세요: 장비는 첫 패킷 후 몇 초 안에 **설정**에 나타나고, 차트는 약 1분 늦게 채워집니다. NetFlow v9/IPFIX는 먼저 장비의 템플릿이 필요하며, 샘플링 레이트를 아직 알리지 않은 장비의 레코드는 최대 5분(`-sampling-wait`) 보관된 뒤 1:1로 집계됩니다.
- 요구 사항: 프로그램 약 73 MB, 따로 설치할 것 없음. 메모리는 유휴 시 약 0.1 GB, 초당 5,000 플로에서 약 0.6–0.8 GB이므로 1 GB 이상을 권장합니다. 웹 UI를 인터넷에 직접 노출하지 마세요. 내부망에서 쓰거나 HTTPS 리버스 프록시 뒤에 두세요.

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
  가상 장비로 체험. 첫 시작 때 하루치 모의 기록을 먼저 만들고(몇 초, 느린 디스크에서는 더 오래), 끝나면 웹 UI를 엽니다.

## 스크린샷

<img src="images/overview.png" alt="개요" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="지역 및 네트워크" width="49%"> <img src="images/interfaces.png" alt="인터페이스 대조" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## 전체 설명서

웹 UI 메뉴의 **도움말**, GitHub: [MANUAL.ko.md](MANUAL.ko.md)

15개 언어 지원. 평가 용도와 100명 미만 조직은 무료입니다([라이선스](../LICENSE.md)).

- **무료**: 평가·테스트·데모는 어떤 조직이든 기간 제한 없이 무료. 직원과 외주 인력을 합쳐 100명 미만인 조직과 비상업적 용도는 운영 사용도 무료. 그 밖의 조직은 운영 사용 30일 후 등록이 필요합니다. 자세한 내용: [LICENSE.md](../LICENSE.md), [추가 사용 허가](../ADDITIONAL-USE-GRANT.md).
- **미등록**: 보고서와 차트에 '평가판' 워터마크, 웹 계정은 최대 3개.
- **등록**: 워터마크가 사라집니다. 문의 route66traffic66@gmail.com.
