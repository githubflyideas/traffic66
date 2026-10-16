[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | **日本語** | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow・sFlow・IPFIX コレクター兼トラフィックアナライザー

sFlow・NetFlow・IPFIX のフロー分析を 1 つのプログラムで行います。誰が帯域を使っているか、トラフィックがどこへ流れているか、その数値が機器自身のインターフェースカウンターと一致しているかを、Web UI とターミナル UI で表示します。

## クイックスタート

1. お使いのシステム用のアーカイブをダウンロード： [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. 展開してプログラムを実行：

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows：`traffic66-windows-amd64.zip` を展開し `traffic66.exe` を実行（ダブルクリックで可）。

macOS：先にそのフォルダで `xattr -dr com.apple.quarantine .` を実行。

その後：

- Web UI：`http://<サーバーIP>:8066`。すべてのネットワークインターフェースで待ち受けます。
- サインイン：ユーザー `admin`、パスワード `traffic66`。初回サインイン時に新しいパスワードを設定します。
- 機器の送信先をサーバー IP に：sFlow UDP 6343、NetFlow v5/v9 UDP 2055、IPFIX UDP 4739（全インターフェース。どのポートも全プロトコルを受信）。

## よく使うコマンド

- `./traffic66`  
  上記の既定値で起動。

- `./traffic66 -addr :9000`  
  Web UI を 8066 ではなく 9000 番ポートで。

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  コレクタのポートを変更（ここでは NetFlow を 9995 に）。

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  全インターフェースではなく 1 つの IP アドレスだけで待ち受け。

- `./traffic66 -data /var/lib/traffic66`  
  データを別のディレクトリに保存（既定はプログラム横の `traffic66-data`）。

- `./traffic66 passwd`  
  admin のパスワードを設定・リセット（`-data` を使う場合は `-data DIR` を付ける）。

- `traffic66.exe interfaces`  
  Windows：ネットワークカードの一覧（名前と番号）。

- `traffic66.exe -capture Wi-Fi`  
  Windows：この PC のそのカードの通信を解析（名前か番号）。Npcap（npcap.com）が必要。

- `./traffic66 interfaces`  
  Linux・macOS：ネットワークカードの一覧。

- `sudo ./traffic66 -capture eth0`  
  Linux：このマシンの eth0 の通信を解析（macOS は `-capture en0`）。

- `./traffic66 tui`  
  traffic66 のマシンでターミナル UI。

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  リモートの traffic66 にターミナル UI で接続。

- `./traffic66 demo`  
  模擬機器で試す。

## スクリーンショット

<img src="images/overview.png" alt="概要" width="49%"> <img src="images/topn.png" alt="上位 66" width="49%">

<img src="images/geo.png" alt="地域とネットワーク" width="49%"> <img src="images/interfaces.png" alt="インターフェース照合" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## 完全なマニュアル

Web UI のメニューの **ヘルプ**、GitHub では： [MANUAL.ja.md](MANUAL.ja.md)

15 言語に対応。評価用途と 100 人未満の組織は無料です（[ライセンス](../LICENSE.md)）。
