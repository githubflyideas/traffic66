[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | **日本語** | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow・sFlow・IPFIX コレクター兼トラフィックアナライザー

sFlow・NetFlow・IPFIX のフロー分析を 1 つのプログラムで行います。誰が帯域を使っているか、トラフィックがどこへ流れているか、その数値が機器自身のインターフェースカウンターと一致しているかを、Web UI とターミナル UI で表示します。

## クイックスタート

```
# Linux・macOS：./traffic66 にダウンロードして収集を開始
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows（PowerShell）：同じく
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

ターミナルに Web UI のアドレスと、機器の送信先 IP・UDP ポートが表示されます（[機器の設定](MANUAL.ja.md#device-configuration)）。`admin` / `traffic66` でサインインし新しいパスワードを設定します。まず見るだけなら、模擬機器のデモ：

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

ポート：TCP 8066 Web UI、UDP 6343 sFlow、2055 NetFlow、4739 IPFIX（どの UDP ポートもすべてのプロトコルを受信）。機器の送信先アドレスはターミナルに表示されます。初回サインイン時に新しいパスワードを設定します。

macOS で手動の場合は先に `xattr -dr com.apple.quarantine traffic66-darwin-*`。Windows の「PC が保護されました」：**詳細情報** → **実行**。

## コマンド早見表

| 用途 | コマンド |
|---|---|
| sFlow 6343・NetFlow v5/v9 2055・IPFIX 4739 を収集（既定） | `./traffic66` |
| NetFlow v5/v9（2055）と IPFIX（4739）のみ | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| sFlow のみ、UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Web UI を指定の IP とポートで | `./traffic66 -addr 10.0.0.5:8080` |
| このマシンの NIC をキャプチャ（Linux） | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin のパスワードを設定・リセット（-data 使用時は付ける） | `./traffic66 passwd` |
| リモートサーバーにターミナル UI で接続 | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![概要：未対応の検知、アプリケーション別の帯域と昨日の同じ時刻との比較、上位のクライアントとサービス](images/overview.png)

## 完全なマニュアル

Web UI のメニューの **ヘルプ**、GitHub では： [MANUAL.ja.md](MANUAL.ja.md)

15 言語に対応。評価用途と 100 人未満の組織は無料です（[ライセンス](../LICENSE.md)）。
