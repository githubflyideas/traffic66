[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | **日本語** | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow・sFlow・IPFIX コレクター兼トラフィックアナライザー

sFlow・NetFlow・IPFIX のフロー分析を 1 つのプログラムで行います。誰が帯域を使っているか、トラフィックがどこへ流れているか、その数値が機器自身のインターフェースカウンターと一致しているかを、Web UI とターミナル UI で表示します。

ntopng、ElastiFlow、pmacct と Grafana の組み合わせ、あるいは PRTG や SolarWinds NTA のフローモジュールに代わる、セルフホスト型の選択肢です。ネットワーク監視（network monitoring）、帯域監視（bandwidth monitoring）、トップトーカー（top talkers）、DDoS・スキャン検知、pcap 分析に使え、Elasticsearch、Kafka、別途のデータベースは不要です。

- Windows・Linux・macOS それぞれ実行ファイル 1 つ。データベースのインストールは不要で、オフラインで動作します。
- sFlow v5、NetFlow v5/v9、IPFIX を任意の UDP ポートで受信。インターフェースでのローカルキャプチャにも対応。
- 自身の数値をインターフェースカウンター（sFlow または SNMP）と照合し、ずれの理由を示します。
- スキャン、パスワード総当たり、横展開、不審なアップロード、フラッド、脅威リストとの通信を、サンプリング越しでも見つけます。
- `traffic66 capture.pcap` で、設定なしにパケットキャプチャを分析できます。
- 15 言語に対応。評価用途と 100 人未満の組織は無料です（[ライセンス](#licence)）。

![概要：未対応の検知、アプリケーション別の帯域と昨日の同じ時刻との比較、上位のクライアントとサービス](images/overview.png)

<sub>スクリーンショットはすべて、模擬的な社内ネットワークである `traffic66 demo` で撮影したものです。</sub>

<a id="contents"></a>

## 目次

1. [デモを試す](#1-try-the-demo)
2. [インストール](#2-install)
3. [ユーザーとパスワード](#3-users-and-passwords)
4. [機器からフローを送る](#4-send-flows-from-your-devices)
5. [フローの受信を確認する](#5-check-that-flows-arrive)
6. [インターフェースとカウンター](#6-interfaces-and-counters)
7. [名前、国、脅威リスト](#7-names-countries-and-threat-lists)
8. [Web UI の使い方](#8-using-the-web-ui)
9. [オフライン pcap、ターミナル UI、ローカルキャプチャ](#9-offline-pcap-terminal-ui-local-capture)
10. [オプションとデータ](#10-options-and-data)
11. [セキュリティ、サイジング、トラブルシューティング](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. デモを試す

[Releases ページ](https://github.com/githubflyideas/traffic66/releases) から自分のシステム用のアーカイブ（Windows x64、カーネル 3.2 以上の Linux x86-64/ARM64、macOS 11 以上）をダウンロードし、展開して実行します：

```
./traffic66 demo          # Linux, macOS
.\traffic66.exe demo      # Windows
```

macOS では先に `xattr -dr com.apple.quarantine <folder>` を実行してください。http://127.0.0.1:8066 を開いて `admin` / `traffic66` でサインインすると、1 日分の履歴と、シミュレートした 4 台の機器からのライブトラフィックが表示されます。その中には攻撃が含まれ、**検知** で段階ごとに示されます。Ctrl+C で停止し、`traffic66-demo` を削除すると最初からやり直せます。本番インストールと並べて動かすには：`-addr :8067 -listen ""`。

<a id="2-install"></a>

## 2. インストール

traffic66 はファイル 1 つです。ポート：UDP 6343（sFlow）、2055（NetFlow）、4739（IPFIX）、TCP 8066（Web UI）。どの UDP ポートもすべてのプロトコルを受け付けます。

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

**Windows**（管理者として PowerShell を実行）：`C:\traffic66` に展開し、`C:\traffic66\traffic66.exe passwd` を実行し、ポートを開けて、起動時に開始するよう登録します：

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`traffic66.exe` をダブルクリックしても動きます。Web UI が開きます。

**macOS**：`/usr/local/traffic66` に展開し、quarantine フラグを外し、`traffic66 passwd -data "/Library/Application Support/traffic66"` を実行して、LaunchDaemon から起動します。`ProgramArguments` にはプログラム、`-data`、そのディレクトリを指定し、`RunAtLoad` と `KeepAlive` を付けます。

<a id="3-users-and-passwords"></a>

## 3. ユーザーとパスワード

新規インストールでは `admin` / `traffic66` でサインインします。初回のサインインでは、ほかの画面を表示する前に新しいパスワードの設定を求められます（デモは `traffic66` のままです）。以降は、メニュー下部の **アカウント** で自分のパスワードを変更できます。管理者（`admin`）はそこでユーザーの追加・削除とパスワードのリセットも行えます。LDAP / Active Directory によるサインインは開発中です。

ユーザーはソルト付きハッシュとしてデータディレクトリの `password` に保存されます。同じ操作は traffic66 のマシン上で 1 つのコマンドでも行えます（traffic66 を `-data …` 付きで動かしている場合は同じものを付けます）：

| 操作 | コマンド |
|---|---|
| `admin` のパスワードを変更 | `traffic66 passwd` |
| `alice` を追加、またはパスワードを変更 | `traffic66 passwd -user alice` |
| `alice` を削除 | `traffic66 passwd -user alice -delete` |
| ユーザーの一覧 | `traffic66 passwd -list` |

変更はすぐに反映されます。ユーザー管理を除き、すべてのユーザーの権限は同じです。スクリプトやコンテナでは、`TRAFFIC66_PASSWORD=…`（または `-password`）を指定すると、その実行では `-user` とそのパスワードだけを受け付けます。1 分間に 5 回パスワードを間違えると、そのアドレスは 1 分間ブロックされます。

<a id="4-send-flows-from-your-devices"></a>

## 4. 機器からフローを送る

`192.0.2.50` が traffic66、`192.0.2.1` が機器です。アクティブタイムアウトは 60 秒に設定し、NetFlow/IPFIX の機器にはサンプラーオプションをエクスポートさせ、**すべてのインターフェースで受信方向**（またはエッジのインターフェースだけ）をサンプリングしてください。こうすると各パケットが 1 回だけ数えられます。sFlow のレート：1 Gb/s で約 1:1000、10 Gb/s で 1:4096、40/100 Gb/s で 1:8192。

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

## 5. フローの受信を確認する

**設定** には、何かを送ってきた機器が数秒以内にすべて表示されます：プロトコル、サンプリングレート、ロス、サンプリングしているインターフェース、そしてステータスが緑でないときに何を直せばよいか。sFlow のロスは、途中でのロス（`netstat -su` にバッファーエラーが出ていれば `net.core.rmem_max` を引き上げる）と、機器自身が破棄したサンプルに分けて示されます。

![設定：機器ごとのプロトコル、サンプリング、ロス、直すべき点](images/sources.png)

機器が表示されない場合は `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739` を実行します。何も見えなければ、ルーティング、ファイアウォール、または機器の設定の問題です。パケットは見えるのに **設定** に何も出なければ、ローカルのファイアウォールか `-listen` の問題です。別のマシンから `traffic66 simulate -to 192.0.2.50` を実行すると、シミュレートした機器で経路をテストできます。

<a id="6-interfaces-and-counters"></a>

## 6. インターフェースとカウンター

フローの数値は推定値です（サンプル数 × サンプリングレート）。**インターフェース照合** はこれを機器のインターフェースカウンター（sFlow カウンター、または **名前** の `snmp` 行による SNMP）と比較し、ずれの理由を示します：サンプリングされていないインターフェース、同じトラフィックの二重サンプリング、途中でのロス、または不明なサンプリングレート。各インターフェースには bits/s と packets/s のグラフがあり、受信は緑、送信は青、カウンターは破線で表示されます。

各行の **✎** は名前と短いタグ（*uplink* など）を設定し、**☆** はそれをデフォルトのインターフェース（★）にします。各ページはこのインターフェースで開きます。

一部のインターフェースだけでサンプリングしている機器では、それらのフローの反対側のインターフェースも表示されます。これらの **対向インターフェース** は最後に小さい灰色の文字で並びます。中身はサンプリング対象のインターフェースを通ったトラフィックだけです。サンプリング対象のインターフェースは sFlow のデータソースか flowDirection フィールド（IPFIX 61）から判断し、それがない場合は機器のトラフィックの 90% が通るインターフェースとみなします。

![インターフェース照合：全インターフェースのトラフィックと、フローの推定値と機器のカウンターの並列表示](images/interfaces.png)

traffic66 はすでに、機器が適用したレートを使い、不明なレートは待ち、エクスポートのロスを補正し、長いフローを各分に按分し、NetFlow/IPFIX にはパケットあたり 18 バイトの Ethernet オーバーヘッドを加算しています（`-l2-overhead`）。

<a id="7-names-countries-and-threat-lists"></a>

## 7. 名前、国、脅威リスト

任意のアドレスをクリックして **名前を付ける…** を選ぶか、**設定 → 名前** を使います。名前はデータディレクトリの `inventory.txt` に保存されます：

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

プライベートアドレス範囲は常に自社のものとして扱われます。変更は **保存** で反映され、再起動は不要です。

国とネットワーク（AS）は、DB-IP の無料 Lite データベース（[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)、"IP Geolocation by DB-IP"、[db-ip.com](https://db-ip.com)）により最初から使えます。**設定** で更新したり、代わりに MaxMind GeoLite2、IPinfo Lite、IPtoASN のファイルを使ったりできます。地図の輪郭：[Natural Earth](https://www.naturalearthdata.com)。

![地域とネットワーク：国別の外部トラフィックを示す世界地図](images/geo.png)

脅威リストは、1 行に 1 つのアドレスまたはネットワークを書いたテキストファイルで、`<data>/threats/<name>.txt` に置きます（例：Spamhaus DROP）。変更後は再起動してください。一致したものは **脅威インテル** に表示されます。

![脅威インテル：脅威リストに載っているアドレスへデータを送っている内部ホスト](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Web UI の使い方

どのページのどの値もクリックできます：**これだけ表示** / **これを除外**（フィルターは全ページに適用）、**フローレコードを見る**、**詳細を表示**（1 つのホストまたはサービスのページ）、**名前を付ける…**、**外部サイトで調べる**、**コピー**。

| ページ | 表示内容 |
|---|---|
| 概要 | 選択したインターフェースの帯域幅、アプリケーション別のトラフィック（合計、内向き、外向き）と昨日・先週との比較、未対応の検知、上位のクライアントとサービス |
| 上位 66 | 上位 66 件の会話（どの列でも並べ替え可能）、またはアプリケーション、ネットワーク、セグメント、機器、カプセル化、VLAN でのグループ化、上位 30 の通信相手 |
| トラフィック詳細 | リングチャート：サーバーとそのクライアント（またはその逆）、サービス |
| フローの流れ | ホスト → アプリケーション → 国、またはクライアント → サービス → サーバー、またはネットワーク別 |
| インターフェース照合 | 全インターフェースの推移とカウンターの比較、名前、タグ、デフォルト |
| フローレコード | 個々のフロー。5 秒ごとのリアルタイム表示、または任意の時間範囲 |
| 検知、脅威インテル | 対応が必要なもの（[後述](#findings)）、リスト上のアドレスとの通信 |
| 地域とネットワーク | 国別の世界地図、ネットワーク（AS）の推移 |
| 設定 | 機器、サンプリング、ロス、SNMP、データベース、ロゴ、名前 |
| オフライン pcap 分析、データ削除 | キャプチャファイル（[後述](#9-offline-pcap-terminal-ui-local-capture)）、古いデータの削除 |

ページの上部には、**インターフェース**（すべて、またはサンプリング対象のインターフェース 1 つ。選ぶとトラフィック系のページにはそこを通るトラフィックだけが表示されます）、時間範囲（15 分〜30 日、または任意指定）、30 秒ごとの更新、そして現在の表示そのものへの **リンクをコピー** があります。言語と 5 つのカラーテーマはメニューの一番下にあります。グラフはデータが揃っている時点で終わります。NetFlow/IPFIX では機器のエクスポートが遅れる分だけ手前になります（最大 2 分）。6 時間より長い範囲は正時から始まります。1 つのインターフェースで 7 日または 30 日を表示するとフロー明細を読むため、時間がかかり、遡れるのは明細の保存期間までです。

![上位 66：上位 66 件の会話、どの列でも並べ替え可能](images/topn.png)

![トラフィック詳細：サーバーとそのクライアント、サービスとそのサーバーのリングチャート](images/traffic.png)

![1 つのホストの詳細：そのホストに関する検知、トラフィック、通信相手、サービス、国、最新のフロー](images/detail.png)

![フローの流れ：どのホストがどの国に向けてどのアプリケーションを使っているか](images/paths.png)

![中国語の概要画面](images/overview-zh.png)

<a id="findings"></a>

### 検知

5 分ごとに直近 10 分間をチェックします。1 時間続く事象は、1 件の検知として育っていきます。

| 検知 | 意味 |
|---|---|
| スキャン、ポートスキャン | 多数のホストの同じポート、または 1 台のホストの多数のポートへの小さなプローブ |
| パスワード総当たり | ログインサービスへの短い接続が多数 |
| 横展開 | それまでファイル共有やリモート管理を提供したことのない内部ホストへの、それらのセッション |
| 不審なアップロード | 新しいアドレスへ 10 分間で 100 MB、戻ってきた量の 3 倍 |
| フラッド | 1 つのアドレスへ毎秒 20,000 個以上の小さなパケット、普段の 10 倍のレート |
| 脅威リスト | リスト上のアドレスとの通信 |

社内ネットワークからなら重大度は高、インターネットからなら低です。**対応済み** は検知を閉じ、**問題なし** は永久に表示しないようにします。横展開と不審なアップロードには 1 日分の履歴が必要です。1:4096 のサンプリング越しでもデモの攻撃はすべて検知されますが、ごく小さなスキャンはサンプリングに隠れることがあります。

![検知：攻撃の各段階を 1:4096 の sFlow サンプリング越しに検知](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. オフライン pcap、ターミナル UI、ローカルキャプチャ

**オフライン pcap 分析** は、パケットキャプチャ（pcap、pcapng）をライブデータとは分けて、同じページで表示します。`traffic66 a.pcap b.pcapng` は 127.0.0.1 で起動してブラウザーを開きます（最大 3 ファイル、3 GB。Ctrl+C で取り込んだデータを削除）。そのページで 50 MB までのファイルを最大 3 つアップロードすることもできます。一度に分析するのは 1 ファイルで、それぞれ専用のデータベースに入ります。ファイルの行の **分析** でそのファイルが全ページに表示され、上部のバーで別のファイルに切り替えられます。扱うのはフローで、パケットの中身ではありません。

![オフライン分析：キャプチャファイルとパケット数、フロー数、時間](images/sandbox.png)

**ターミナル UI**：traffic66 のマシンで `traffic66 tui`、または `traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`。キー操作：1–8 ページ、Enter 操作、f これだけ表示、x 除外、t 時間範囲、w ブラウザーで開く、q 終了。`-lang` で言語を選びます。

![ターミナル UI：概要](images/tui-overview.png)

![ターミナル UI：上位 66 の会話](images/tui-topn.png)

**ローカルキャプチャ** はローカルのインターフェースからフローを作ります。スイッチのミラーポートにつないだポートが最適です。`traffic66 interfaces` で一覧を表示し、`-capture eth1` でキャプチャします。Windows では：

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux では root か `setcap cap_net_raw,cap_net_admin+ep`、macOS では root、Windows では [Npcap](https://npcap.com) が必要です。キャプチャしたフローは機器 `127.0.0.1` からのものとして表示されます。ローカルキャプチャには機器のインターフェースもカウンターもないため、**インターフェース照合** で比較する対象はありません。

<a id="10-options-and-data"></a>

## 10. オプションとデータ

`traffic66 -h` ですべてを表示します。よく使うもの：

| オプション | デフォルト | |
|---|---|---|
| `-data` | プログラムと同じ場所の `traffic66-data` | データディレクトリ |
| `-addr` | `:8066` | Web UI。`127.0.0.1:8066` でこのマシンからのみ |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP コレクター。空にすると無効 |
| `-retention-days` | `30` | フロー明細の保存日数。集計は 400 日保存 |
| `-memory` | `0.10` | データベースのキャッシュに使うメモリの割合 |
| `-sampling-wait` | `5m` | レコードがサンプリングレートを待つ時間 |
| `-capture` | | ローカルインターフェース（複数指定可） |
| `-no-dns` | | 逆引きを行わない |

データディレクトリには `raw/`（明細、1 時間ごとに 1 ファイル）、`traffic66.duckdb`（集計とカウンター）、`password`、`inventory.txt`、`license.json`、ロゴ、データベースが入っています。バックアップは traffic66 を停止してディレクトリをコピーし、アップグレードはプログラムファイルを置き換えます。**データ削除** は 7〜120 日より古いデータ、またはすべてのデータを削除します。

<a id="licence"></a>

### ライセンス

[PolyForm Noncommercial License 1.0.0](../LICENSE.md) と [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md)（英語版が正本）のもとでのソースアベイラブルです。評価用途と 100 人未満の組織は無料。それより大きな組織は本番利用 30 日後に登録が必要です。販売、他者向けのホスティング、競合製品には商用ライセンスが必要です。機能が止まることは一切ありません。各ページの下部に 8 桁のインストール番号が表示されるので、それを作者に送り、返ってきた `license.json` をデータディレクトリに置いてください。連絡先：<https://github.com/githubflyideas/traffic66>。

<a id="11-security-sizing-troubleshooting"></a>

## 11. セキュリティ、サイジング、トラブルシューティング

Web UI は平文の HTTP です。信頼できないネットワークでは `-addr 127.0.0.1:8066` とし、TLS プロキシ（`caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`）か SSH トンネルの背後に置いてください。UDP ポートは自社の機器からのみ許可してください。SNMP コミュニティは平文で保存されるので、読み取り専用のものを使ってください。

2 コアで毎秒 5,000 フローの場合：明細のディスクは 1 日あたり約 12 GB（30 日で 360 GB）、CPU は 1 コアの 6 分の 1、メモリは 0.6–0.8 GB。長い範囲の概要は 0.2 秒未満、全会話の 1 時間分の上位 66 は約 9 秒です。

| 症状 | 対処 |
|---|---|
| "waiting for the sampling rate" | サンプラーオプションをエクスポートするか、device 行に `sampling=N` / `unsampled` を付ける |
| カウンターより小さい | サンプリングされていないインターフェース、ロス、またはアクティブタイムアウトが 60 秒超 |
| カウンターより大きい | 同じトラフィックを 2 つのインターフェースまたは 2 台の機器でサンプリングしている |
| パスワードを忘れた | traffic66 のマシンで `traffic66 passwd` |
| `Conflicting lock is held` | 別の traffic66 がこのデータディレクトリを使っている |
| `address already in use` | `-addr` または `-listen` で別のポートを指定 |
| Windows："Windows によって PC が保護されました" | **詳細情報** → **実行** |

ソースからのビルド：Go 1.24 と C コンパイラーを用意し、`scripts/build.sh 0.1.0 traffic66` を実行します。
