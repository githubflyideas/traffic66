[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | **日本語** | [한국어](README.ko.md)

# traffic66

sFlow・NetFlow・IPFIX のフロー分析を 1 つのプログラムで行います。スイッチ、ルーター、ファイアウォールからエクスポートされたフローを受信して組み込みデータベースに保存し、誰が帯域を使っているか、トラフィックがどこへ流れているか、その数値が機器自身のインターフェースカウンターと一致しているかを、Web UI とターミナル UI で表示します。

- Windows・Linux・macOS それぞれ実行ファイル 1 つだけです。データベースのインストールもランタイムも不要で、オフラインで動作します。
- sFlow v5、NetFlow v5、NetFlow v9、IPFIX を任意の UDP ポートで受信できます。ネットワークインターフェースやミラーポートでのローカルキャプチャにも対応しています。
- 自身の集計値をインターフェースカウンター（sFlow カウンターまたは SNMP）と照合し、ずれがあればその理由を示します。
- フローからスキャン、パスワード総当たり、横展開、不審なアップロード、フラッド、脅威リストとの通信を見つけ出し（サンプリングされたフローでも可能）、対応すべき検知として一覧にします。
- Top 66 ランキング、クライアント・サーバー・サービス・インターフェース・ネットワーク（AS）別のトラフィックの推移、フローの流れ、国、脅威リストとの一致、フローレコード、カプセル化（GRE、IPIP、VXLAN、GENEVE、MPLS）。
- Web UI とターミナル UI は 13 言語に対応しています。

![概要：未対応の検知、アプリケーション別の帯域と先週との比較、上位のクライアントとサービス](images/overview.png)

<sub>スクリーンショットはすべて `traffic66 demo` で撮影したものです。これは自分で実行できる模擬的な社内ネットワークです（[デモを試す](#1-try-the-demo) を参照）。</sub>

<a id="contents"></a>

## 目次

1. [デモを試す](#1-try-the-demo)
2. [インストール](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [ユーザーとパスワード](#3-users-and-passwords)
4. [機器からフローを送る](#4-send-flows-from-your-devices)
5. [フローの受信を確認する](#5-check-that-flows-arrive)
6. [数値をインターフェースカウンターと一致させる](#6-make-the-numbers-match-the-interface-counters)
7. [名前、SNMP、自社ネットワーク](#7-names-snmp-and-your-own-networks)
8. [国、ネットワーク、脅威リスト](#8-countries-networks-and-threat-lists)
9. [Web UI の使い方](#9-using-the-web-ui)
10. [ターミナル UI](#10-terminal-ui)
11. [ローカルキャプチャ](#11-local-capture)
12. [オプション](#12-options)
13. [データ、バックアップ、アップグレード、アンインストール](#13-data-backup-upgrade-uninstall)
14. [セキュリティ](#14-security)
15. [サイジング](#15-sizing)
16. [トラブルシューティング](#16-troubleshooting)
17. [ソースからのビルド](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. デモを試す

[リリースページ](https://github.com/githubflyideas/traffic66/releases) から、お使いのシステム用のアーカイブをダウンロードします。

| システム | アーカイブ |
|---|---|
| Windows 10/11、Server 2016 以降（x64） | `traffic66-windows-amd64.zip` |
| Linux x86-64：カーネル 3.2 以降の任意のディストリビューション（CentOS 7、Alpine を含む） | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64：同じディストリビューション | `traffic66-linux-arm64.tar.gz` |
| macOS 11 以降、Apple シリコン | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 以降、Intel | `traffic66-darwin-amd64.tar.gz` |

Linux：

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS（2 行目は、App Store 以外からダウンロードしたプログラムを macOS で実行できるようにするためのものです）：

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

http://127.0.0.1:8066 を開き、`admin` / `try66` でサインインします。デモでは小さな社内ネットワークが構築され、1 日分の履歴と、シミュレートした 4 台の機器からのライブトラフィックが流れます。中には攻撃が 1 件含まれています。**検知** にはその各段階（スキャン、ポートスキャン、パスワード総当たり、横展開、制御サーバーへのアップロード）と、公開 Web サイトへのフラッドが表示されます。検知の **詳細** をクリックするか、**概要** から始めて **上位クライアント** のホストをクリックし、**詳細を表示** を選んで、そこからさらにクリックでたどってください。Ctrl+C で停止します。デモデータはプログラムと同じ場所の `traffic66-demo` に保存されます。このフォルダーを削除するとデモを最初からやり直せます。

デモは本番インストールと同じポート（8066 と UDP 6343、2055、4739）を使います。本番環境と並べて動かす場合は、別のポートを指定してください：`traffic66 demo -password try66 -addr :8067 -listen ""`。

Windows では `traffic66.exe` をダブルクリックするだけでも起動できます。この場合はデモではなく本番として traffic66 が起動し、ブラウザーで Web UI が開きます。初回起動時のパスワードは黒いウィンドウに表示され、ウィンドウを閉じると traffic66 は停止します。"Windows によって PC が保護されました" と表示された場合は、**詳細情報** → **実行** をクリックしてください。

<a id="2-install"></a>

## 2. インストール

traffic66 はファイル 1 つです。インストールとは、ファイルを配置し、データディレクトリを決め、パスワードを設定し、ファイアウォールを開け、起動時に自動実行されるようにすることです。例では traffic66 のマシンを `192.0.2.50`、ルーターを `192.0.2.1` としています。実際のアドレスに置き換えてください。

ポート：

| ポート | 用途 |
|---|---|
| UDP 6343 | sFlow（デフォルト） |
| UDP 2055 | NetFlow（デフォルト） |
| UDP 4739 | IPFIX（デフォルト） |
| TCP 8066 | Web UI と API |

どの UDP ポートでもすべてのプロトコルを受け付けるので、都合がよければ NetFlow を 6343 に送っても構いません。ポートの変更や追加は `-listen` で行います。

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

最後のコマンドで、ユーザー `admin` のパスワードを入力します。

`/etc/systemd/system/traffic66.service` を作成します：

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

起動し、バーストで取りこぼさないよう UDP バッファの上限を引き上げます：

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

ファイアウォール（firewalld の場合。RHEL、Rocky、Alma、Fedora）：

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

ufw の場合（Ubuntu、Debian）：

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

`C:\traffic66` に展開してパスワードを設定します（管理者として PowerShell を実行）：

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

データはプログラムと同じ場所の `C:\traffic66\traffic66-data` に保存されます。

ファイアウォールを開けます：

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

フォアグラウンドで試すには `C:\traffic66\traffic66.exe` を実行し、Ctrl+C で停止します。誰もサインインしていなくても起動時からバックグラウンドで動かすには、スタートアップタスクとして登録します：

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` は必須です。これがないと Windows は 3 日後にタスクを停止します。停止は `Stop-ScheduledTask -TaskName
traffic66`、削除は `Unregister-ScheduledTask -TaskName traffic66` で行います。

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

`/Library/LaunchDaemons/traffic66.plist` を作成します：

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

起動と停止：

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

macOS のファイアウォールが有効な場合は、「システム設定 → ネットワーク → ファイアウォール →オプション」で traffic66 への着信接続を許可してください。

<a id="3-users-and-passwords"></a>

## 3. ユーザーとパスワード

**要点**：ユーザーとパスワードは、データディレクトリ内の 1 つのファイル `password` に保存されます。このファイルを手で編集することはありません。ユーザーの追加・変更・一覧表示・削除は `traffic66 passwd` コマンドで行います。`http://<traffic66 machine>:8066` を開き、いずれかのユーザーでサインインします。

<a id="the-first-sign-in"></a>

### 初回のサインイン

traffic66 は初回起動時に、ランダムなパスワードでユーザー `admin` を作成し、そのパスワードを 1 回だけ表示します：

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Windows でダブルクリックして起動した場合：黒いウィンドウに表示されます。
- ターミナルから起動した場合：そのターミナルに表示されます。
- Linux サービス：`journalctl -u traffic66 | grep "first start"`
- macOS サービス：`grep "first start" /Library/Logs/traffic66.log`

見逃した場合は、`traffic66 passwd`（後述）で新しいパスワードを設定してください。上のインストール手順のように、初回起動前に `traffic66 passwd` でパスワードを設定しておけば、パスワードは生成されません。

<a id="where-the-users-are-stored"></a>

### ユーザーの保存場所

データディレクトリ内の `password` ファイルです：

| traffic66 の動かし方 | ファイル |
|---|---|
| 展開したフォルダーから起動（デフォルト） | プログラムと同じ場所の `traffic66-data/password` |
| Linux サービス（セクション 2） | `/var/lib/traffic66/password` |
| Windows スタートアップタスク（セクション 2） | `C:\traffic66\traffic66-data\password` |
| macOS サービス（セクション 2） | `/Library/Application Support/traffic66/password` |
| デモ | プログラムと同じ場所の `traffic66-demo/password` |

1 行に 1 ユーザーです。パスワードはソルト付きハッシュで保存されるため、誰もファイルから読み戻すことはできません。あなた自身も同じです。パスワードを忘れた場合は、新しく設定してください。このファイルは所有者だけが読み取れます。

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### ユーザーの管理

traffic66 のマシンで次を実行します：

| 目的 | コマンド |
|---|---|
| `admin` のパスワードを変更 | `traffic66 passwd` |
| ユーザー `alice` を追加、またはそのパスワードを変更 | `traffic66 passwd -user alice` |
| ユーザー `alice` を削除 | `traffic66 passwd -user alice -delete` |
| ユーザーを一覧表示 | `traffic66 passwd -list` |
| ランダムなパスワードを設定して表示 | `traffic66 passwd -generate`（他のユーザーには `-user` を付ける） |

- コマンドは新しいパスワードを 2 回入力するよう求め、入力内容は表示しません。8 文字以上にしてください。
- traffic66 を `-data` 付きで動かしている場合は、コマンドにも同じ `-data` を付けます。セクション 2 の Linux サービスの場合：

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Windows の場合（管理者として PowerShell を実行）：

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- 変更は再起動なしですぐに反映されます。新しいパスワードは次回のサインインから使え、削除したユーザーは開いているブラウザーからサインアウトされます。
- 最後に残ったユーザーは削除できません。先に別のユーザーを追加してください。
- どのユーザーも同じものを閲覧・変更できます。ロールはありません。

<a id="passwords-for-scripts-and-containers"></a>

### スクリプトやコンテナ用のパスワード

環境変数 `TRAFFIC66_PASSWORD=…` またはコマンドラインの `-password …` を指定すると、traffic66 はその実行中、`-user` で指定したユーザー（デフォルトは `admin`）1 人だけを、そのパスワードで受け付けます。このとき `password` ファイルは無視され、変更もされません。環境変数を推奨します。コマンドラインはマシンの他のユーザーからも見えるためです。

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

1 分以内に 5 回パスワードを間違えたアドレスは、1 分間ブロックされます。

<a id="4-send-flows-from-your-devices"></a>

## 4. 機器からフローを送る

各機器のエクスポート先を traffic66 のマシンに向けます。コマンドは機種やソフトウェアのバージョンによって異なるので、機器のマニュアルを確認してください。すべての例で、`192.0.2.50` が traffic66、`192.0.2.1` が機器自身のアドレスです。

全般的な注意点：

- アクティブフロータイムアウトは 60 秒に設定してください。長くすると、トラフィックが遅れて大きな塊で届くようになります。
- 機器が NetFlow/IPFIX でサンプリングしている場合は、サンプラーオプションもエクスポートさせてサンプリングレートが分かるようにしてください。traffic66 はレートが届くまでレコードを保留し、1:1 として数えることはしません。
- サンプリングは全インターフェースか、エッジのインターフェースだけかのどちらかで、片方向で行ってください。同じトラフィックを入りと出の両方でサンプリングすると二重にカウントされます。**インターフェース照合** がこれを指摘します。
- sFlow のサンプリングレートの目安：1 Gb/s リンクで約 1:1000、10 Gb/s で 1:4096、40/100 Gb/s で 1:8192。

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

Huawei CloudEngine（sFlow）：

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

FortiGate FortiOS 7.4.2 以降（NetFlow v9）：

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

Linux サーバー・ホストでは softflowd を使います（NetFlow v9）：

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. フローの受信を確認する

**受信状況** を開きます。何かを送ってきた機器は数秒以内に表示され、プロトコル、サンプリングレート、ロス、最終受信パケット、ステータスが示されます。ステータスが緑でない場合は、横のテキストに何が問題で何を変えればよいかが書かれています。

![受信状況：機器ごとのプロトコル、サンプリング、ロス、直すべき点](images/sources.png)

機器が表示されない場合：

1. traffic66 のマシンでパケットを確認します（Linux、macOS）：`sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`。何も見えなければ、パケットがマシンに届いていません。機器の設定、ルーティング、経路上のファイアウォールを確認してください。
2. パケットは届いているのに **受信状況** が空のまま：ローカルのファイアウォールが破棄している（[インストール](#2-install) を参照）か、traffic66 が別のポートで待ち受けています（`-listen`）。
3. 機器に触れずに別のマシンから経路をテストするには、そのマシンで`traffic66 simulate -to 192.0.2.50` を数秒間実行します。シミュレートした機器からsFlow、NetFlow、IPFIX を送信し、それらは **受信状況** とデータに現れるので、テスト用のインストールで行うことをおすすめします。

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. 数値をインターフェースカウンターと一致させる

フローの数値は推定値です（サンプリングされたパケット数 × サンプリングレート）。traffic66 はこれを機器自身のインターフェースカウンターと比較して **インターフェース照合** に差分を表示し、サンプリングだけでは説明できないほど大きい場合は考えられる原因も示します。

![インターフェース照合：全インターフェースのトラフィックと、フローの推定値と機器のカウンターの並列表示](images/interfaces.png)

比較用のカウンターを取得するには：

- sFlow 機器は、カウンターの送信間隔を設定すれば（`sflow counter interval 30` など）自動的に送ってきます。
- NetFlow・IPFIX 機器の場合は、**受信状況 → 名前** に `snmp` 行を追加します（[名前](#7-names-snmp-and-your-own-networks) を参照）。traffic66 が毎分インターフェースカウンターを読み取ります。

数値を一致させるために traffic66 がすでに行っていること：機器が実際に適用したサンプリングレートを使う、サンプリングレートが判明するまで NetFlow/IPFIX レコードを保留する、途中で失われたエクスポートパケットを補正する、長いフローを継続した各分に按分する、そしてNetFlow/IPFIX のバイト数にパケットあたり 18 バイトの Ethernet オーバーヘッドを加算する（インターフェースカウンターには含まれ、IP レイヤーのフローの集計には含まれないため。`-l2-overhead` で変更可能）。

それでも残る差分のよくある原因は次のとおりで、いずれも **インターフェース照合** に報告されます：一部のインターフェースがサンプリングされていない、同じトラフィックを 2 つのインターフェースでサンプリングしている、エクスポートパケットが traffic66 に届く前に失われている、サンプリングレートがまだ分かっていない。

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. 名前、SNMP、自社ネットワーク

ホストや機器に名前を付ける一番手早い方法は、どのページでもそのアドレスをクリックして **名前を付ける…** を選ぶことです。名前を入力して Enter を押すとすぐに保存され、以後どこでもアドレスの代わりにその名前が表示されます。

ネットワーク、インターフェース、SNMP については、Web UI の **受信状況 → 名前** に 1 行に 1 エントリを書きます。内容はデータディレクトリに`inventory.txt` として保存されるので、このファイルを直接編集することもできます（`inventory.txt.example` を参照）。どの行も省略可能です。

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

- `net`：プライベートアドレス範囲（10/8、172.16/12、192.168/16、100.64/10）は常に自社のものとして扱われます。グローバルアドレス範囲も追加すると、それらとの間のトラフィックも自社扱いになります。名前は、セグメントでグループ化した **上位 66** と、ネットワーク別のフローの流れに表示されます。
- `snmp <device> <community> [<management address>[:port]]`：device はフローの送信元アドレスです。機器が別のアドレスで SNMP に応答する場合は管理アドレスを追加します。SNMP で読み取ったインターフェースの説明は、`iface` で名前を付けていない限りインターフェース名として使われます。機器の SNMP アクセスリストで traffic66 のマシンを許可してください。
- 変更は **保存** をクリックすると反映されます。再起動は不要です。

<a id="8-countries-networks-and-threat-lists"></a>

## 8. 国、ネットワーク、脅威リスト

国とネットワーク（AS）は最初から表示されます。traffic66 には DB-IP の無料データベース **IP to Country Lite** と **IP to ASN Lite** が内蔵されています（ライセンス [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)、"IP Geolocation by DB-IP"、[db-ip.com](https://db-ip.com)）。国とネットワークを表示するページにはデータの出典を表示します。

内蔵データは、使っているリリース時点のものです。DB-IP は毎月新版を公開します。**受信状況 → 国・ネットワークデータベース → DB-IP Lite を今すぐ更新** で db-ip.com から最新版をダウンロードできます（traffic66 を動かしているサーバーからインターネットに出られる必要があります。失敗した場合は画面に表示されます）。

ほかの無料データベースも使えます。ダウンロードして、同じページの **データベースファイルをアップロード…** でアップロードします。ファイルは検査されてデータディレクトリに保存され、すぐに新しいトラフィックに使われます。再起動は不要です。保存済みのトラフィックは保存時の国のままです。

| データベース | 内容 | ライセンス | 入手先 |
|---|---|---|---|
| DB-IP Lite（内蔵） | 国、ネットワーク | CC BY 4.0、登録不要 | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country と ASN、`.mmdb` | 国、ネットワーク | GeoLite2 EULA、無料アカウントが必要 | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite、`ipinfo_lite.mmdb` | 国とネットワークを 1 ファイルで | CC BY-SA 4.0、無料アカウントが必要 | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN、`ip2asn-combined.tsv.gz` | ネットワークとその国 | PDDL 1.0、登録不要 | [iptoasn.com](https://iptoasn.com) |

アップロードしたファイルが優先され、そこにない分は内蔵の DB-IP Lite が答えます。ファイルの横の **削除** で残りの構成に戻ります。ページには使用中のデータベースとそれぞれの日付が表示されます。

Web UI を使わない場合は、ファイルを `country.mmdb`、`asn.mmdb`、`both.mmdb`（IPinfo Lite のように国とネットワークが 1 ファイルのもの）または `asn.tsv.gz` という名前でデータディレクトリに置き、traffic66 を再起動します。

**地域とネットワーク** には他の国とのトラフィックが世界地図で表示されます。色が濃いほどトラフィックが多い国です。国にポインタを合わせるとトラフィック量が、クリックすると絞り込みやフローレコードの表示ができます。国境は [Natural Earth](https://www.naturalearthdata.com)（パブリックドメイン）のものです。

![地域とネットワーク：国別の外部トラフィックを示す世界地図](images/geo.png)

脅威リストは 1 行に 1 つのアドレスまたはネットワークを書いたプレーンテキストファイルで（`#` または `;` 以降は無視されます）、`<data directory>/threats/<name>.txt` として保存します。例：

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

リストを追加・変更したら traffic66 を再起動してください。一致したものは **脅威インテル** にリスト名ごとに表示されます。

![脅威インテル：脅威リストに載っているアドレスへデータを送っている内部ホスト](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Web UI の使い方

キーボードで入力する場面はほとんどありません。どのページのどの値も（アドレス、ポート、アプリケーション、国、機器）クリックできます：

- **これだけ表示** / **これを除外** でフィルターを追加します。フィルターはトップバーの下に表示され、外すまですべてのページに適用されます。
- **フローレコードを見る** で、該当する個々のフローを開きます。
- **詳細を表示**（ホスト、機器、サービス）で、そのホストまたはサービス 1 つについてのページを開きます。アプリケーション別のトラフィックの推移、通信相手、サービスまたはクライアント、国、最新のフローが表示されます。そこでもすべての値をクリックできるので、さらに掘り下げていけます。ブラウザーの戻るボタンで戻れます。
- **名前を付ける…**（ホストと機器）で、アドレスに名前を付けます。以後どこでもその名前が表示されます。
- **外部サイトで調べる** で、アドレスや AS を公開の検索サイトで開きます。
- **コピー** で値をコピーします。

ページ：

| ページ | 分かること |
|---|---|
| 概要 | 現在のトラフィック量と先週との比較（アプリケーション別）、未対応の検知、方向とプロトコル、上位のクライアントとサービス |
| 上位 66 | 最初は **主な通信相手** を表示：サービス別のトラフィックの推移と、上位 30 のクライアントとサーバーを並べてトラフィック、パケット数、フローレコード数とともに示し、その下に全トラフィックの行があります。**表** は上位 66 件を 1 つの表で表示。既定は会話（クライアント、サーバー、サービス、国）。どの列見出しでも並べ替えられます。数値の列（トラフィック、パケット数、平均パケット長、フロー数）は期間内の全トラフィックから上位 66 件を取り直すので、平均パケット長の小さい順でスキャンやフラッドが見つかります。**グループ化** でアプリケーション、ネットワーク、セグメント、機器、カプセル化、VLAN に切り替えられます |
| トラフィック詳細 | クライアント、サーバー、サービスの推移（bits/s と packets/s）：それぞれの上位 8 件と、その総数。初期表示は **サーバー**、タブで **クライアント**、**両端**（並べて表示）、**サービス** に切り替え |
| 検知 | 対応が必要なもの：スキャン、パスワード総当たり、横展開、不審なアップロード、フラッド、脅威リストとの通信（[詳しく](#findings)） |
| フローの流れ | どのホストがどの国に向けてどのアプリケーションを使っているか。通信量の多い 8 ホストを表示し、残りは「その他」にまとめます。**クライアント → サーバー** でクライアント → サービス → サーバーを表示し、**ネットワーク別** でホストの代わりにネットワークを表示します |
| 地域とネットワーク | 国別トラフィックの世界地図、トラフィックの送信元・送信先のネットワーク（AS）の推移（bits/s と packets/s）、国別・ネットワーク別のトラフィック |
| 脅威インテル | 脅威リスト上のアドレスと通信したホストと、その送信量 |
| フローレコード | フローレコードの数とその時期（間隔ごとの棒グラフ）と、レコードそのもの（新しい順、ページ送り、表示列は選択可能） |
| インターフェース照合 | 全インターフェースのトラフィックの推移（受信と送信、bits/s と packets/s）と、フローの数値とインターフェースカウンターの対比（差の大きい順、理由付き） |
| 受信状況 | 機器、サンプリング、ロス、コレクター、SNMP、国・ネットワークデータベース、ロゴ、**名前** |

ページの上部には、時間範囲（15 分〜30 日）、検索ボックス（任意）、30 秒ごとの自動更新、そして **リンクをコピー** があります。**リンクをコピー** は、現在の表示（ページ、時間範囲、フィルター）そのものへのリンクをコピーするので、同僚に送るのに便利です。その下の **機器**、**クライアント**、**サーバー**、**サービス** には、時間範囲内で通信量の多い値が並びます。値を選ぶか入力すると全ページが絞り込まれ、ボックスを空にするとフィルターが外れます。言語はブラウザーの設定に従い、メニューの一番下で変更できます。

推移のグラフは、上位 8 つの値を固定の色で、残りを「その他」として表示します。凡例には各値の合計が示され、他の値と同じようにクリックできます。クライアントとサーバーのグラフでは「その他」を描画しません。ホストが数千台あると上位 8 つが平らになってしまうためです。凡例にはその合計が引き続き示されます。

6 時間より長い時間範囲は正時から始まるため、ページ上のすべての数値がまったく同じ時間を集計します。「24 時間」は、直近の区切りのよい 24 時間に現在の 1 時間を加えた範囲です。これらの範囲の上位 66 は 1 時間ごとの集計から作られるため、フィルターは使えません（ページにもその旨が表示されます）。フィルターを使うには短い範囲を選んでください。会話は常にフローの明細を読むため、フローの多い環境で長い範囲を選ぶと時間がかかることがあります。1 時間が最速です。

サイドメニューには、データが使っているディスク容量と空き容量が表示されます。空き容量にマウスを重ねると、現在のペースで明細を保持日数分残すのに必要な容量が分かります（1 日分のデータがたまると推定されます）。

サインイン画面とメニュー上部に独自のロゴを表示するには、**受信状況 → ロゴ → ロゴをアップロード…** を使います。PNG、SVG、JPEG、WebP、GIF に対応し、1 MB まで、推奨サイズは 272 × 92 ピクセルです（他のサイズは収まるように縮小・拡大されます）。**内蔵ロゴに戻す** で traffic66 のロゴに戻ります。

<a id="findings"></a>

### 検知

**検知** には、traffic66 がフローから見つけたものが深刻な順に表示されます。5 分ごとに直近 10 分間をチェックします。1 時間続く事象は、チェックのたびに新しい検知になるのではなく、1 件の検知として育っていきます。

| 検知 | 意味 | 重大度 |
|---|---|---|
| スキャン | 1 つのアドレスが、多数のアドレスの同じポートに小さなプローブを送った（TCP または ping） | 社内ネットワークからなら高、インターネットからなら低 |
| ポートスキャン | 1 つのアドレスが、1 台のホストの多数のポートに小さなプローブを送った | 社内からなら高、インターネットからなら低 |
| パスワード総当たり | ログインサービス（SSH、RDP、SMB、データベースなど）への短い接続が多数 | 社内からなら高、インターネットからなら低 |
| 横展開 | 社内ネットワーク内で、それまでそのサービスを提供したことのないホストへのファイル共有またはリモート管理のセッション（SMB、RDP、SSH、WinRM、VNC） | 高 |
| 不審なアップロード | 内部ホストが、それまでデータをやり取りしたことのないアドレスへ、受信よりはるかに多く送信した（10 分間で 100 MB、受信量の 3 倍） | 高 |
| フラッド | 1 つのアドレスへ毎秒 20,000 個以上の小さなパケット、普段の 10 倍のレート | 中 |
| 脅威リスト | 脅威リストのいずれかに載っているアドレスとの通信 | 自分のホストから接続した場合は高、リスト上のアドレスが外から叩いてきた場合は低 |

各検知には、誰が誰に何をしたか、いつ、どれだけの時間続いたかが、根拠となる数値とデータのサンプリング方法とともに示されます。**詳細** はそのホストのページを開きます。そこにもそのホストに関する検知が表示されます。**対応済み** は検知を閉じます。同じことが再び起きれば新しい検知が開きます。**問題なし** は検知を永久に閉じ、二度と報告されません。サイドメニューの **検知** の横にある赤い数字は、直近 24 時間の未対応の重大度「高」と「中」の検知の件数です。

横展開と不審なアップロードは何が普段どおりかを知る必要があるため、1 日分の履歴がたまってから報告されます。初回起動時には、traffic66 はすでにある履歴から学習します。

サンプリングされたデータ（sFlow、サンプリングされた NetFlow）では、ルールはサンプルに見えるものを数え、必要な件数を減らします。その代わり、各サンプルが 1 回の短いプローブに見えなければならないので、正常で忙しいホストが引っかかることはありません。サンプリングで隠れたものは見つけられません。1:4096 のサンプリングでは、数十台のホストへのスキャンは送るパケットが少なすぎて見えません。デモの攻撃は 1:4096 でサンプリングするスイッチを通りますが、すべて検知されます。デモの正常なトラフィックは 1 日分でも、Web サイトを叩くインターネット上のスキャナー以外の検知を生みません。

![検知：攻撃の各段階を 1:4096 の sFlow サンプリング越しに検知](images/findings.png)

![上位 66、主な通信相手：サービス別のトラフィックと、上位 30 のクライアントとサーバー（全トラフィックの行付き）](images/topn.png)

![トラフィック詳細：クライアント、サーバー、サービスの推移（bits/s と packets/s）](images/traffic.png)

![1 台のホストの詳細：そのホストに関する検知、トラフィック、通信相手、サービス、国、最新のフロー](images/detail.png)

![フローの流れ：どのホストがどの国に向けてどのアプリケーションを使っているか](images/paths.png)

同じ概要画面の中国語版です。すべてのページが 13 言語で表示できます：

![中国語の概要画面](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. ターミナル UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

traffic66 のマシン上では、データディレクトリを読み取れれば `traffic66 tui` は自動でサインインします（デフォルト以外の場所なら `-data` を指定）。サービスとして動かす場合のようにtraffic66 が別のユーザーで動いているときは、代わりに `-user` と `-password` を使います。`-lang` で言語を選びます（`en`、`zh`、`hi`、`es`、`ar`、`fr`、`bn`、`pt`、`ru`、`id`、`ur`、`ja`、`ko`）。

キー操作：1–8 ページ切り替え、↑↓ 選択、Enter 選択した値に対する操作、f これだけ表示、x 除外、/ 検索、t 時間範囲、c フィルター解除、w 同じ表示をブラウザーで開く、q 終了。

![ターミナル UI：概要](images/tui-overview.png)

![ターミナル UI：上位 66 の会話](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. ローカルキャプチャ

traffic66 はフローのエクスポートを受信するだけでなく、自身が動いているマシンのネットワークインターフェース上のパケットから自分でフローを生成することもできます。何が見えるかはインターフェースによって変わります：

| インターフェース | traffic66 から見えるもの |
|---|---|
| スイッチのミラー（SPAN）ポートにつないだ空きのネットワークポート | スイッチがミラーするすべてのトラフィック：ネットワーク全体やアップリンク |
| そのマシン自身の Ethernet または Wi-Fi | そのマシン自身のトラフィックのみ |

Wi-Fi アダプターでは他の機器のトラフィックは見えません。Wi-Fi ネットワーク全体を見るには、ルーターやアクセスポイントからフローをエクスポートする（セクション 4）か、アクセスポイントがつながっているスイッチポートをミラーしてください。

<a id="windows-1"></a>

### Windows

1. [Npcap](https://npcap.com) をデフォルトのオプションでインストールします。"Restrict Npcap driver's access to Administrators only" にチェックを入れた場合は、traffic66 を管理者として実行してください。
2. インターフェースを一覧表示します（PowerShell）：

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Name 列は Windows のネットワーク設定にある接続名です。使用中のインターフェースにはアドレスが付いています。
3. Wi-Fi で、名前または番号を指定してキャプチャします：

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   スペースを含む名前は引用符で囲みます：`-capture "Ethernet 2"`。複数のインターフェースでキャプチャするには `-capture` を繰り返します。キャプチャだけを行い、フローコレクターが不要なら `-listen=` を付けます。セクション 2 のスタートアップタスクでは、このオプションを `-Argument` に追加します：`-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`。

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

キャプチャには root 権限、またはケーパビリティ `CAP_NET_RAW` と `CAP_NET_ADMIN` が必要です。上記の `setcap` の行、またはセクション 2 の systemd ユニットにある `AmbientCapabilities` 行を使います。Wi-Fi インターフェースの名前は通常 `wlan0` か `wlp…` です。

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

キャプチャには root 権限が必要です。追加のインストールは不要です。MacBook では `en0` が Wi-Fi です。

<a id="checking-that-it-works"></a>

### 動作の確認

**受信状況** に、キャプチャ中の各インターフェースが、キャプチャ方式と受信したパケット数とともに表示されます。フローはすべてのページで、他の機器のフローと同じように機器 `127.0.0.1`（このマシン）からのものとして表示されます。同じパケットが 2 回見えた場合（例えば 2 つのミラーポートで）は 2 回カウントされます。

<a id="12-options"></a>

## 12. オプション

`traffic66 -h` と `traffic66 <command> -h` ですべて確認できます。

コマンド：

| コマンド | |
|---|---|
| `traffic66` | フローを収集し Web UI を提供 |
| `traffic66 demo` | 同上、シミュレートしたネットワークで動作 |
| `traffic66 tui` | 稼働中の traffic66 用のターミナル UI |
| `traffic66 passwd` | ユーザーの追加・変更・一覧表示・削除（[ユーザーとパスワード](#3-users-and-passwords) を参照） |
| `traffic66 simulate -to HOST` | シミュレートしたエクスポートをコレクターに送信 |
| `traffic66 interfaces` | ローカルキャプチャ用のインターフェースを一覧表示 |
| `traffic66 version` | バージョンを表示 |

`traffic66` と `traffic66 demo` のオプション：

| オプション | デフォルト | |
|---|---|---|
| `-addr` | `:8066` | Web UI のアドレス。`127.0.0.1:8066` でこのマシンからのみ |
| `-data` | プログラムと同じ場所の `traffic66-data` | データディレクトリ |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP コレクターを `name=address` のカンマ区切りで指定。空にすると無効 |
| `-user` | `admin` | 初回起動時に作成されるユーザーの名前。`-password` が適用されるユーザーでもある |
| `-password` | 未設定 | この実行では `password` ファイルを無視し、`-user` とこのパスワードだけを受け付ける（`TRAFFIC66_PASSWORD` でも可） |
| `-retention-days` | `30` | フロー詳細の保存日数。集計は 400 日保存 |
| `-memory` | `0.10` | データベースのキャッシュに使う物理メモリの割合。プログラムのその他の部分にも同じ量をソフト上限として設定（それぞれ最低 256 MB） |
| `-l2-overhead` | `18` | NetFlow/IPFIX のバイト数にパケットあたり加算するバイト数 |
| `-sampling-wait` | `5m` | レコードがサンプリングレートを待つ時間 |
| `-capture` | | ローカルインターフェースでキャプチャ（複数指定可） |
| `-inventory` | `<data>/inventory.txt` | 名前ファイル |
| `-asn` | `<data>/asn.tsv.gz` | IP-ASN 対応表（`.mmdb` ファイルはアップロードするか、`<data>/country.mmdb` と `<data>/asn.mmdb`, `<data>/both.mmdb` に置く） |
| `-threat` | `<data>/threats/*.txt` | 追加の脅威リストを `name=path` で指定（複数指定可） |
| `-dns-upstream` | システムのリゾルバー | ホスト名表示に使う DNS サーバー |
| `-dns-rate` | `20` | 1 秒あたりの逆引き回数の上限 |
| `-dns-cache` | `2m` | ホスト名のキャッシュ時間 |
| `-no-dns` | | 逆引きを行わない |
| `-tui` | | ターミナル UI も開く |

例：コレクターのポートを 1 つ追加し、詳細を 1 年保存し、Web UI をローカルマシンだけに公開する：

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. データ、バックアップ、アップグレード、アンインストール

すべてのデータはデータディレクトリにあります：

| | |
|---|---|
| `raw/` | フロー詳細。1 時間ごとに 1 つの圧縮ファイル |
| `traffic66.duckdb` | 集計、インターフェースカウンター、現在の 1 時間分 |
| `password` | ログインパスワード（ハッシュ化） |
| `inventory.txt` | 名前（**受信状況 → 名前**） |
| `logo.png`（または `.svg`、`.jpg`、`.webp`、`.gif`） | アップロードしたロゴ（**受信状況 → ロゴ**）。アップロードした場合のみ |
| `country.mmdb`, `asn.mmdb`, `both.mmdb`, `asn.tsv.gz`, `dbip-country.mmdb`, `dbip-asn.mmdb`, `threats/` | 追加した国・ネットワークデータベースと脅威リスト |

**データの保存期間**：フロー明細は 30 日、集計（概要と長い期間）は 400 日です。それより古いデータは自動で削除され、5 分ごとに確認します。それ以外は何も削除せず、ほかの制限もありません。明細の保存日数は `-retention-days` で変更でき、何日でも指定できます（例：`-retention-days 365`）。ディスク使用量もそれに応じて増えます。保存日数分が入りきらないときは、サイドメニューの **空き** が赤くなります。ディスクが満杯になると、空きができるまで新しいフローは保存できません。

- **バックアップ**：traffic66 を停止してディレクトリをコピーします。停止せずに行う場合は`raw/`、`password`、`inventory.txt` をコピーします。この場合、現在の 1 時間分と集計は含まれません。
- **移動**：traffic66 を停止し、ディレクトリを移動して、新しい場所を `-data` で指定して起動します。
- **アップグレード**：traffic66 を停止し、プログラムファイルを置き換えて、再び起動します。データはそのまま残ります。Linux の例：

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **アンインストール**：サービスまたはスタートアップタスクを停止・削除し（[インストール](#2-install) を参照）、プログラムのフォルダーとデータディレクトリを削除します。

<a id="14-security"></a>

## 14. セキュリティ

- Web UI は平文の HTTP を使います。パスワードとデータは暗号化されずにネットワークを流れます。完全には信頼できないネットワークでは、このマシンだけで待ち受け（`-addr 127.0.0.1:8066`）、前段に TLS リバースプロキシを置いてください。例えば [Caddy](https://caddyserver.com) なら：`caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`。あるいは VPN や SSH トンネル経由でアクセスします：`ssh -L 8066:127.0.0.1:8066 user@192.0.2.50` の後、http://127.0.0.1:8066 を開きます。
- UDP のコレクターポートは、自社機器のアドレスからのみ許可してください。
- `inventory.txt` 内の SNMP コミュニティは平文で保存されます。読み取り専用のコミュニティを使ってください。

<a id="15-sizing"></a>

## 15. サイジング

2 コアのマシンで毎秒 5,000 フローを処理した実測値：詳細データはディスクを 1 日あたり約 12 GB、加えて現在の 1 時間分に約 1.5 GB 使用し、プログラムは CPU 1 コアの 6 分の 1程度を使います。長い時間範囲の概要は集計から作られ、0.2 秒未満で表示されます。詳細データへのクエリは 1 時間あたり約 2,200 万行をスキャンします。1 ホストの 1 時間分なら 1 秒未満、全通信ペアの 1 時間分の Top 66 なら約 9 秒です。所要時間は範囲に比例して増え、コア数が多いほど短くなります。

したがって毎秒 5,000 フローで 30 日分のディスクは約 360 GB です。実際のフローレート（**受信状況** に表示）と `-retention-days` に応じて見積もってください。

メモリ：`-memory`（デフォルトは物理メモリの 10%、最低 256 MB）はデータベースのキャッシュを制限し、プログラムのその他の部分には同じ大きさのソフト上限がかかります。毎秒 5,000 フローでは、プログラム自身のデータ（デコード、重複検出、バッチ）は約 90 MB です。全体では 0.6–0.8 GB を見込んでください。RAM 2 GB のマシンで足ります。10 分間の連続収集（8 GB のマシンでピーク 0.58 GB）と、2 GB のマシンの制限下でその 11 倍のレートで 1 時間分のフローを取り込んだとき（ピーク 0.74 GB）に実測しました。

`-memory` は予算であり、厳密な上限ではありません。Go の制限はソフト上限で、データベースも一時的に自分の割り当てを超えることがあります。厳密な上限が必要なら OS の仕組みを使ってください：systemd ユニットの `MemoryMax=`（セクション 2）またはコンテナのメモリ制限です。`-memory` の割り当ての約 2.5 倍、かつ 1 GB 以上を見込んでください。デフォルトの割り当てなら、物理メモリ 8 GB までのマシンには `MemoryMax=2G` が適しています。こうすると、マシンのメモリが枯渇する代わりに traffic66 が再起動します。

<a id="16-troubleshooting"></a>

## 16. トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| **受信状況** に機器が表示されない | パケットが届いていません：[フローの受信を確認する](#5-check-that-flows-arrive) を参照 |
| "waiting for the sampling rate" | 機器がまだサンプラーオプションを送っていません。多くの機器は数分以内に再送します。いつまでも送られない場合はエクスポートを設定する（Cisco では `option sampler-table`）か、本当に 1:1 なら名前で `unsampled` を付けます |
| 数値がインターフェースカウンターより小さい | **インターフェース照合** を確認：途中でのロス、サンプリングされていないインターフェース、またはフローがまだ機器のキャッシュ内にある（アクティブタイムアウトが 60 秒より長い） |
| 数値がインターフェースカウンターより大きい | 同じトラフィックを 2 つのインターフェースまたは 2 台の機器でサンプリングしています |
| 国やネットワークが表示されない（"不明"） | データベースが読み込まれていません。**受信状況** でアップロードしてください：[国](#8-countries-networks-and-threat-lists) を参照 |
| ページに "データベースがメモリ上限に達したため、応答できませんでした" と表示される | 期間を短くするか、より大きな `-memory` で起動してください。詳細はログにあります |
| パスワードを忘れた | traffic66 のマシンで `traffic66 passwd`（traffic66 を `-data` 付きで動かしている場合は `-data` も付ける） |
| `Conflicting lock is held` | 別の traffic66 がすでにこのデータディレクトリを使っています |
| `receive buffer is only … KB` | Linux が UDP バッファを制限しています：`net.core.rmem_max=16777216` を設定（[Linux](#linux) を参照） |
| `cannot create the data directory` | このユーザーはプログラムのフォルダーに書き込めません：`-data` を指定してください |
| macOS："cannot be opened" または "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows："Windows によって PC が保護されました" | **詳細情報** → **実行**。プログラムはまだ署名されていません |
| Windows でのキャプチャ：Npcap が見つからない | [Npcap](https://npcap.com) をインストール |
| `address already in use` | 別のプログラムがポートを使用中：`-addr` または `-listen` で別のポートを指定 |

<a id="17-build-from-source"></a>

## 17. ソースからのビルド

Go 1.24 と C コンパイラー（gcc または clang。Windows では MinGW-w64）が必要です：

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
