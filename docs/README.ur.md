[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | **اردو** | [日本語](README.ja.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66

sFlow، NetFlow اور IPFIX کے لیے flow analytics، ایک ہی پروگرام میں۔ یہ
switches، routers اور firewalls سے flow exports جمع کرتا ہے، انہیں ایک
embedded database میں رکھتا ہے، اور دکھاتا ہے کہ bandwidth کون استعمال کر رہا
ہے، ٹریفک کہاں جا رہی ہے اور کیا اعداد و شمار ڈیوائس کے اپنے interface counters
سے میل کھاتے ہیں — web UI میں بھی اور terminal UI میں بھی۔

- Windows، Linux اور macOS کے لیے ایک ہی executable۔ کوئی database انسٹال
  نہیں کرنا، کوئی runtime نہیں، offline بھی چلتا ہے۔
- کسی بھی UDP port پر sFlow v5، NetFlow v5، NetFlow v9 اور IPFIX؛ چاہیں تو
  کسی network interface یا mirror port سے local capture بھی۔
- اپنے اعداد کو interface counters (sFlow counters یا SNMP) سے ملا کر جانچتا
  ہے، اور فرق ہو تو بتاتا ہے کہ کیوں۔
- flows میں scans، پاس ورڈ کا اندازہ، lateral movement، غیر معمولی uploads،
  floods اور threat list والی ٹریفک ڈھونڈتا ہے، sampling کے باوجود بھی، اور
  انہیں نمٹانے کے لیے مشتبہ سرگرمیوں کی فہرست میں دکھاتا ہے۔
- Top 66 فہرستیں، کون کس سے بات کرتا ہے ring charts کی صورت میں (servers اور
  ان کے clients، services اور ان کے servers)، interface اور نیٹ ورک (AS) کے
  لحاظ سے وقت کے ساتھ ٹریفک، flow کے راستے، دنیا کے نقشے پر ممالک، threat list
  کے matches، flow records، encapsulation (GRE، IPIP، VXLAN، GENEVE، MPLS)۔
- `traffic66 capture.pcap` زیادہ سے زیادہ 3 پیکٹ کیپچر (کل 3 GB) ویب UI میں کھولتا ہے: پورے کیپچر کے فلو، نتائج، ممالک اور فلو ریکارڈز، بغیر کسی سیٹ اپ کے۔
- web UI اور terminal UI میں 13 زبانیں۔
- تمام خصوصیات کے ساتھ 30 دن مفت آزمائیں؛ اس کے بعد بھی کام کرتا رہتا ہے
  ([آزمائش اور لائسنس](#trial-and-licence) دیکھیں)۔

![جائزہ: کھلی مشتبہ سرگرمیاں، پچھلے ہفتے کے مقابلے میں application کے حساب سے bandwidth، سرفہرست clients اور services](images/overview.png)

<sub>تمام اسکرین شاٹس `traffic66 demo` سے لیے گئے ہیں، ایک simulated کمپنی نیٹ ورک جسے آپ خود چلا سکتے ہیں ([ڈیمو چلا کر دیکھیں](#1-try-the-demo) دیکھیں)۔</sub>

<a id="contents"></a>

## فہرست

1. [ڈیمو چلا کر دیکھیں](#1-try-the-demo)
2. [انسٹال کریں](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [یوزرز اور پاس ورڈ](#3-users-and-passwords)
4. [اپنی ڈیوائسز سے flows بھیجیں](#4-send-flows-from-your-devices)
5. [جانچیں کہ flows پہنچ رہے ہیں](#5-check-that-flows-arrive)
6. [اعداد کو interface counters سے ملائیں](#6-make-the-numbers-match-the-interface-counters)
7. [نام، SNMP اور آپ کے اپنے نیٹ ورک](#7-names-snmp-and-your-own-networks)
8. [ممالک، نیٹ ورک اور threat lists](#8-countries-networks-and-threat-lists)
9. [web UI کا استعمال](#9-using-the-web-ui)
10. [Terminal UI](#10-terminal-ui)
11. [Local capture](#11-local-capture)
12. [آپشنز](#12-options)
13. [ڈیٹا، بیک اپ، اپ گریڈ، اَن انسٹال](#13-data-backup-upgrade-uninstall)
14. [سکیورٹی](#14-security)
15. [گنجائش کا اندازہ](#15-sizing)
16. [مسائل کا حل](#16-troubleshooting)
17. [سورس سے بِلڈ کریں](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. ڈیمو چلا کر دیکھیں

اپنے سسٹم کے لیے archive
[releases صفحے](https://github.com/githubflyideas/traffic66/releases) سے ڈاؤن لوڈ کریں:

| سسٹم | Archive |
|---|---|
| Windows 10/11، Server 2016 یا نیا (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: kernel 3.2 یا اس کے بعد والی کوئی بھی distribution، بشمول CentOS 7 اور Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: وہی distributions | `traffic66-linux-arm64.tar.gz` |
| macOS 11 یا نیا، Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 یا نیا، Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (دوسری لائن macOS کو انٹرنیٹ سے ڈاؤن لوڈ کیا گیا ایسا پروگرام چلانے دیتی
ہے جو App Store سے نہیں آیا):

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows (PowerShell):

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

http://127.0.0.1:8066 کھولیں اور `admin` / `try66` سے سائن اِن کریں۔ ڈیمو ایک
چھوٹی کمپنی کا نیٹ ورک بناتا ہے جس میں ایک دن کی ہسٹری اور چار simulated
ڈیوائسز سے live ٹریفک ہوتی ہے، اور اس میں ایک حملہ بھی ہے: **مشتبہ سرگرمی** اس
کا ہر مرحلہ دکھاتا ہے (ایک scan، ایک port scan، پاس ورڈ کا اندازہ، lateral
movement، ایک control server پر upload) اور public website پر ایک flood بھی۔
فہرست کے کسی اندراج پر **تفصیل** پر کلک کریں، یا **جائزہ** سے شروع کریں،
**سرفہرست کلائنٹ** میں کسی host پر کلک کریں، **تفصیل دیکھیں** چنیں، اور وہاں
سے کلک کرتے ہوئے آگے بڑھیں۔ Ctrl+C سے بند کریں۔ ڈیمو کا ڈیٹا پروگرام کے ساتھ
`traffic66-demo` میں رہتا ہے؛ ڈیمو نئے سرے سے شروع کرنے کے لیے وہ فولڈر ڈیلیٹ کر دیں۔

ڈیمو وہی ports استعمال کرتا ہے جو اصل installation کرتی ہے (8066، اور UDP
6343، 2055، 4739)۔ اصل installation کے ساتھ ساتھ چلانے کے لیے اسے دوسرے ports
دیں: `traffic66 demo -password try66 -addr :8067 -listen ""`۔

Windows پر آپ سیدھے `traffic66.exe` پر double-click بھی کر سکتے ہیں۔ اس سے
traffic66 اصل طور پر (ڈیمو نہیں) شروع ہوتا ہے اور آپ کے browser میں web UI
کھل جاتا ہے؛ پہلی بار شروع ہونے کا پاس ورڈ کالی window میں دکھایا جاتا ہے، اور
window بند کرنے سے traffic66 رک جاتا ہے۔ اگر Windows یہ دکھائے:
"Windows protected your PC" (Windows نے آپ کے PC کو محفوظ کیا)، تو
**More info** (مزید معلومات) → **Run anyway** (پھر بھی چلائیں) پر کلک کریں۔

<a id="2-install"></a>

## 2. انسٹال کریں

traffic66 ایک ہی فائل ہے۔ انسٹال کرنے کا مطلب ہے: اسے کہیں رکھنا، ایک data
directory چننا، پاس ورڈ سیٹ کرنا، firewall کھولنا اور boot پر اسے چلانا۔ مثالوں
میں traffic66 مشین کے لیے `192.0.2.50` اور router کے لیے `192.0.2.1` ہے؛ انہیں
اپنے addresses سے بدل لیں۔

Ports:

| Port | کس لیے |
|---|---|
| UDP 6343 | sFlow (ڈیفالٹ) |
| UDP 2055 | NetFlow (ڈیفالٹ) |
| UDP 4739 | IPFIX (ڈیفالٹ) |
| TCP 8066 | web UI اور API |

ہر UDP port ہر protocol قبول کرتا ہے، اس لیے آسان ہو تو ڈیوائس NetFlow بھی 6343
پر بھیج سکتی ہے۔ ports بدلنے یا شامل کرنے کے لیے `-listen` استعمال کریں۔

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

آخری کمانڈ یوزر `admin` کا پاس ورڈ پوچھتی ہے۔

`/etc/systemd/system/traffic66.service` بنائیں:

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

اسے شروع کریں اور بڑے UDP buffers کی اجازت دیں تاکہ bursts میں packets ضائع نہ ہوں:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall، firewalld کے ساتھ (RHEL، Rocky، Alma، Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

یا ufw کے ساتھ (Ubuntu، Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

`C:\traffic66` میں unpack کریں اور پاس ورڈ سیٹ کریں (PowerShell، بطور
Administrator):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

ڈیٹا پروگرام کے ساتھ `C:\traffic66\traffic66-data` میں جاتا ہے۔

Firewall کھولیں:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

foreground میں آزمانے کے لیے `C:\traffic66\traffic66.exe` چلائیں اور Ctrl+C سے
بند کریں۔ boot ہی سے background میں چلانے کے لیے، کسی کے سائن اِن کیے بغیر، اسے
startup task کے طور پر register کریں:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` ضروری ہے: اس کے بغیر Windows تین دن
بعد task روک دیتا ہے۔ روکنے کے لیے `Stop-ScheduledTask -TaskName
traffic66`، ہٹانے کے لیے `Unregister-ScheduledTask -TaskName traffic66`۔

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

`/Library/LaunchDaemons/traffic66.plist` بنائیں:

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

شروع کریں، اور پھر بند کریں:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

اگر macOS firewall آن ہے تو System Settings → Network → Firewall → Options میں
traffic66 کے لیے incoming connections کی اجازت دیں۔

<a id="3-users-and-passwords"></a>

## 3. یوزرز اور پاس ورڈ

**مختصراً:** یوزرز اور پاس ورڈ data directory کی ایک ہی فائل، `password`، میں
رہتے ہیں۔ اسے کبھی ہاتھ سے edit نہ کریں: `traffic66 passwd` command یوزرز کو
شامل کرتا، بدلتا، ان کی فہرست دکھاتا اور انہیں حذف کرتا ہے۔
`http://<traffic66 machine>:8066` کھولیں اور ان میں سے کسی ایک سے سائن اِن کریں۔

<a id="the-first-sign-in"></a>

### پہلا سائن اِن

پہلی بار شروع ہونے پر traffic66 ایک random پاس ورڈ کے ساتھ یوزر `admin` بناتا ہے
اور اسے ایک بار دکھاتا ہے:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Windows پر double-click سے شروع کیا: کالی window میں۔
- terminal میں شروع کیا: اسی terminal میں۔
- Linux service: `journalctl -u traffic66 | grep "first start"`
- macOS service: `grep "first start" /Library/Logs/traffic66.log`

نظر سے رہ گیا؟ `traffic66 passwd` (نیچے دیکھیں) سے نیا سیٹ کریں۔ اگر آپ نے پہلی
بار شروع کرنے سے پہلے `traffic66 passwd` سے پاس ورڈ سیٹ کر دیا ہے، جیسا کہ اوپر
کے install steps کرتے ہیں، تو کچھ بھی generate نہیں ہوتا۔

<a id="where-the-users-are-stored"></a>

### یوزرز کہاں رکھے جاتے ہیں

data directory کی فائل `password` میں:

| traffic66 کیسے چلتا ہے | فائل |
|---|---|
| unpack کر کے اس کے folder سے شروع کیا گیا (default) | پروگرام کے ساتھ `traffic66-data/password` |
| Linux service (حصہ 2) | `/var/lib/traffic66/password` |
| Windows startup task (حصہ 2) | `C:\traffic66\traffic66-data\password` |
| macOS service (حصہ 2) | `/Library/Application Support/traffic66/password` |
| ڈیمو | پروگرام کے ساتھ `traffic66-demo/password` |

ہر یوزر کی ایک line۔ پاس ورڈ salted hashes کی صورت میں رکھے جاتے ہیں، اس لیے کوئی
انہیں فائل سے واپس نہیں پڑھ سکتا، آپ بھی نہیں؛ پاس ورڈ بھول جائیں تو نیا سیٹ کریں۔
فائل صرف اس کا owner پڑھ سکتا ہے۔

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### یوزرز کا انتظام

یہ commands traffic66 مشین پر چلائیں:

| کام | Command |
|---|---|
| `admin` کا پاس ورڈ بدلنا | `traffic66 passwd` |
| یوزر `alice` شامل کرنا، یا اس کا پاس ورڈ بدلنا | `traffic66 passwd -user alice` |
| یوزر `alice` کو حذف کرنا | `traffic66 passwd -user alice -delete` |
| یوزرز کی فہرست دیکھنا | `traffic66 passwd -list` |
| random پاس ورڈ سیٹ کر کے دکھانا | `traffic66 passwd -generate` (دوسرے یوزرز کے لیے `-user` کے ساتھ) |

- command نیا پاس ورڈ دو بار پوچھتا ہے اور جو آپ ٹائپ کرتے ہیں اسے نہیں دکھاتا۔
  کم از کم 8 حروف استعمال کریں۔
- جب traffic66 `-data` کے ساتھ چلتا ہے تو command میں بھی وہی `-data` شامل کریں۔
  حصہ 2 والی Linux service کے لیے:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Windows پر (PowerShell، بطور Administrator):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- تبدیلیاں فوراً لاگو ہوتی ہیں، restart کی ضرورت نہیں: نیا پاس ورڈ اگلے سائن اِن
  پر کام کرتا ہے، اور حذف شدہ یوزر کھلے browsers سے sign out ہو جاتا ہے۔
- آخری بچا ہوا یوزر حذف نہیں کیا جا سکتا؛ پہلے کوئی دوسرا یوزر شامل کریں۔
- سب یوزرز ایک جیسی چیزیں دیکھتے اور بدل سکتے ہیں؛ کوئی roles نہیں ہیں۔

<a id="passwords-for-scripts-and-containers"></a>

### scripts اور containers کے لیے پاس ورڈ

environment میں `TRAFFIC66_PASSWORD=…`، یا command line پر `-password …`، سے
traffic66 اس run کے لیے صرف ایک یوزر قبول کرتا ہے: وہ جس کا نام `-user` میں ہو
(default `admin`)، اسی پاس ورڈ کے ساتھ۔ اس صورت میں `password` فائل نظرانداز کی
جاتی ہے اور بدلی نہیں جاتی۔ environment variable کو ترجیح دیں: command lines
مشین کے دوسرے یوزرز کو نظر آتی ہیں۔

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

ایک منٹ کے اندر پانچ غلط پاس ورڈز کے بعد اس address کو ایک منٹ کے لیے block کر
دیا جاتا ہے۔

<a id="4-send-flows-from-your-devices"></a>

## 4. اپنی ڈیوائسز سے flows بھیجیں

ہر ڈیوائس کو traffic66 مشین کی طرف point کریں۔ کمانڈز models اور software
versions کے حساب سے مختلف ہوتی ہیں؛ اپنی ڈیوائس کا manual دیکھیں۔ تمام مثالوں میں
`192.0.2.50` traffic66 ہے اور `192.0.2.1` ڈیوائس کا اپنا address۔

عمومی مشورے:

- active flow timeout کو 60 سیکنڈ رکھیں۔ لمبے timeouts سے ٹریفک دیر سے اور بڑے
  ٹکڑوں میں آتی ہے۔
- اگر ڈیوائس NetFlow/IPFIX کو sample کرتی ہے تو اسے sampler options export کرنے
  دیں تاکہ rate معلوم ہو۔ traffic66 records کو 1:1 گننے کے بجائے rate آنے تک
  روکے رکھتا ہے۔
- یا تو تمام interfaces کو sample کریں یا صرف edge interfaces کو، ایک ہی سمت
  میں۔ ایک ہی ٹریفک کو آتے اور جاتے دونوں وقت sample کرنے سے وہ دو بار گنی جاتی
  ہے؛ **انٹرفیس جانچ** اس کی نشاندہی کرتا ہے۔
- sFlow sampling rate: 1 Gb/s links کے لیے تقریباً 1:1000، 10 Gb/s کے لیے
  1:4096، 40/100 Gb/s کے لیے 1:8192۔

Cisco IOS / IOS-XE (Flexible NetFlow):

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

Cisco NX-OS (sFlow):

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS (sFlow):

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX (sFlow):

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

Huawei CloudEngine (sFlow):

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

H3C Comware (sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7 (NetFlow v9 / IPFIX):

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 یا نیا (NetFlow v9):

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

Linux servers اور hosts، softflowd کے ساتھ (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. جانچیں کہ flows پہنچ رہے ہیں

**ذرائع** کھولیں۔ جو بھی ڈیوائس کچھ بھیجتی ہے وہ چند سیکنڈ میں نظر آ جاتی ہے،
اپنے protocol، sampling rate، loss، آخری packet اور status کے ساتھ۔ جب status
سبز نہ ہو تو اس کے ساتھ لکھا متن بتاتا ہے کہ کیا خرابی ہے اور کیا بدلنا ہے۔

![ذرائع: ہر ڈیوائس، اس کا protocol، sampling، loss اور کیا ٹھیک کرنا ہے](images/sources.png)

اگر کوئی ڈیوائس نظر نہ آئے:

1. traffic66 مشین پر packets دیکھیں (Linux، macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`۔
   وہاں کچھ نہ ہو تو packets مشین تک پہنچ ہی نہیں رہے: ڈیوائس کی configuration،
   routing اور راستے کے firewalls چیک کریں۔
2. packets آ رہے ہیں مگر **ذرائع** خالی ہے: local firewall انہیں drop کر رہا ہے
   ([انسٹال کریں](#2-install) دیکھیں)، یا traffic66 دوسرے ports پر سن رہا ہے
   (`-listen`)۔
3. کسی ڈیوائس کو چھیڑے بغیر دوسری مشین سے راستہ ٹیسٹ کرنے کے لیے، وہاں چند سیکنڈ
   کے لیے `traffic66 simulate -to 192.0.2.50` چلائیں۔ یہ simulated ڈیوائسز سے
   sFlow، NetFlow اور IPFIX بھیجتا ہے؛ وہ پھر **ذرائع** اور ڈیٹا میں نظر آتی ہیں،
   اس لیے اس کام کے لیے test installation بہتر ہے۔

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. اعداد کو interface counters سے ملائیں

Flow کے اعداد اندازے ہوتے ہیں: sampled packets ضرب sampling rate۔ traffic66
انہیں ڈیوائس کے اپنے interface counters سے ملاتا ہے اور فرق **انٹرفیس جانچ** پر
دکھاتا ہے؛ جب فرق صرف sampling سے سمجھ میں آنے والے فرق سے زیادہ ہو تو ممکنہ
وجہ بھی بتاتا ہے۔ ہر interface کا ایک chart ہے جس میں ingress (سبز) اور egress
(نیلا) ہوتے ہیں؛ فہرست میں کوئی interface چننے سے اس کے charts دکھتے ہیں۔

![انٹرفیس جانچ: ہر interface کی ٹریفک، اور ڈیوائس کے counter کے ساتھ flow کا اندازہ](images/interfaces.png)

موازنے کے لیے counters حاصل کرنے کے لیے:

- sFlow ڈیوائسز counter interval سیٹ ہونے پر انہیں خود بھیجتی ہیں
  (`sflow counter interval 30` اور اسی طرح کی کمانڈز)۔
- NetFlow اور IPFIX ڈیوائسز کے لیے **ذرائع → نام** میں ایک `snmp` لائن شامل کریں
  ([نام](#7-names-snmp-and-your-own-networks) دیکھیں)۔ پھر traffic66 ہر منٹ
  interface counters پڑھتا ہے۔

اعداد ملانے کے لیے traffic66 پہلے سے یہ کرتا ہے: ڈیوائس نے اصل میں جو sampling
rate لگایا وہی استعمال کرتا ہے، sampling rate معلوم ہونے تک NetFlow/IPFIX
records روکے رکھتا ہے، راستے میں ضائع ہونے والے export packets کی تلافی کرتا ہے،
لمبے flows کو ان منٹوں میں بانٹتا ہے جتنی دیر وہ چلے، اور NetFlow/IPFIX byte
counts میں ہر packet پر 18 bytes کا Ethernet overhead جوڑتا ہے (interface
counters میں یہ شامل ہوتا ہے، IP-layer flow counts میں نہیں؛ `-l2-overhead` سے
بدلیں)۔

باقی رہ جانے والے فرق کی عام وجوہات، جو سب **انٹرفیس جانچ** پر بتائی جاتی ہیں:
کچھ interfaces sample نہیں ہو رہے، ایک ہی ٹریفک دو interfaces پر sample ہو رہی
ہے، export packets traffic66 تک پہنچنے سے پہلے ضائع ہو جاتے ہیں، یا sampling
rate ابھی معلوم نہیں۔

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. نام، SNMP اور آپ کے اپنے نیٹ ورک

کسی host یا ڈیوائس کو نام دینے کا سب سے تیز طریقہ: کسی بھی صفحے پر اس کے address
پر کلک کریں اور **نام دیں…** چنیں۔ نام ٹائپ کر کے Enter دبائیں؛ یہ فوراً محفوظ ہو
جاتا ہے اور ہر جگہ خالی address کی بجائے دکھایا جاتا ہے۔

نیٹ ورکس، interfaces اور SNMP کے لیے **ذرائع → نام** استعمال کریں: قسم چنیں
(host، network، device، interface، SNMP)، address اور نام بھریں، اور **شامل
کریں** پر کلک کریں۔ نیچے کی table ہر نام کو **ترمیم** اور **حذف** کے ساتھ دکھاتی
ہے؛ وہی address دوبارہ شامل کرنے سے پرانی entry بدل جاتی ہے۔ محفوظ کرنے سے پہلے
addresses اور نیٹ ورکس جانچے جاتے ہیں۔

نام data directory میں `inventory.txt` کے طور پر محفوظ ہوتے ہیں، ہر لائن میں ایک
entry۔ **متن کے طور پر ترمیم (ایڈوانسڈ)** وہ فائل دکھاتا ہے، اور آپ اسے براہ راست
بھی edit کر سکتے ہیں (`inventory.txt.example` دیکھیں)۔ ہر لائن اختیاری ہے۔

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers country=JP

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

- `net`: private ranges (10/8، 172.16/12، 192.168/16، 100.64/10) ہمیشہ آپ کی
  مانی جاتی ہیں۔ اپنی public ranges شامل کریں تاکہ ان سے آنے جانے والی ٹریفک بھی
  آپ کی گنی جائے؛ نام **ٹاپ 66** میں segment کے لحاظ سے گروپ بندی پر اور network
  کے لحاظ سے flow کے راستوں میں نظر آتا ہے۔ `country=JP` (دو حرفی ملک کوڈ) بتاتا
  ہے کہ نیٹ ورک کہاں ہے؛ پھر دنیا کا نقشہ وہاں سے ان ممالک تک لکیریں کھینچتا ہے
  جن سے وہ بات کرتا ہے۔
- `snmp <device> <community> [<management address>[:port]]`: device وہ address
  ہے جہاں سے flows آتے ہیں۔ جب ڈیوائس SNMP کا جواب کسی دوسرے address پر دیتی ہو
  تو management address شامل کریں۔ SNMP سے پڑھی گئی interface descriptions بطور
  نام استعمال ہوتی ہیں، جب تک آپ `iface` سے interface کا نام نہ دیں۔ ڈیوائس کی
  SNMP access list میں traffic66 مشین کو اجازت دیں۔
- تبدیلیاں **محفوظ کریں** پر کلک کرتے ہی لاگو ہو جاتی ہیں؛ restart کی ضرورت نہیں۔

<a id="8-countries-networks-and-threat-lists"></a>

## 8. ممالک، نیٹ ورک اور threat lists

ممالک اور نیٹ ورکس (AS) شروع سے کام کرتے ہیں: traffic66 میں DB-IP کے مفت **IP to Country Lite** اور **IP to ASN Lite** ڈیٹابیس اندرونی طور پر شامل ہیں (لائسنس [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)؛ "IP Geolocation by DB-IP"، [db-ip.com](https://db-ip.com))۔ ممالک اور نیٹ ورکس دکھانے والے صفحات ڈیٹا کا ماخذ بتاتے ہیں۔

اندرونی کاپی آپ کے چلائے جا رہے ریلیز کے وقت کی ہے۔ DB-IP ہر ماہ نیا ورژن جاری کرتا ہے؛ **ذرائع ← ممالک اور نیٹ ورکس کا ڈیٹابیس ← DB-IP Lite ابھی اپ ڈیٹ کریں** db-ip.com سے تازہ ترین ڈاؤن لوڈ کرتا ہے (traffic66 چلانے والے سرور کو انٹرنیٹ چاہیے؛ ناکامی پر ویب UI بتاتا ہے)۔

آپ کوئی دوسرا مفت ڈیٹابیس بھی استعمال کر سکتے ہیں۔ اسے ڈاؤن لوڈ کریں، پھر اسی صفحے پر **ڈیٹابیس فائل اپ لوڈ کریں…** سے اپ لوڈ کریں۔ فائل جانچی جاتی ہے، ڈیٹا ڈائریکٹری میں محفوظ ہوتی ہے اور نئے ٹریفک کے لیے فوراً استعمال ہوتی ہے؛ ری اسٹارٹ کی ضرورت نہیں۔ پہلے سے محفوظ ٹریفک وہی ملک رکھتا ہے جس کے ساتھ محفوظ ہوا تھا۔

| ڈیٹابیس | کیا دیتا ہے | لائسنس | کہاں سے ملے گا |
|---|---|---|---|
| DB-IP Lite (اندرونی) | ممالک؛ نیٹ ورکس | CC BY 4.0، اکاؤنٹ کی ضرورت نہیں | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country اور ASN، `.mmdb` | ممالک؛ نیٹ ورکس | GeoLite2 EULA، مفت اکاؤنٹ | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite، `ipinfo_lite.mmdb` | ممالک اور نیٹ ورکس ایک فائل میں | CC BY-SA 4.0، مفت اکاؤنٹ | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN، `ip2asn-combined.tsv.gz` | نیٹ ورکس اور ان کا ملک | PDDL 1.0، اکاؤنٹ کی ضرورت نہیں | [iptoasn.com](https://iptoasn.com) |

آپ کی اپنی فائلیں پہلے استعمال ہوتی ہیں؛ جو ان میں نہیں اس کا جواب اندرونی DB-IP Lite دیتا ہے۔ فائل کے ساتھ **ہٹائیں** باقی پر واپس لے جاتا ہے۔ صفحہ دکھاتا ہے کہ کیا استعمال ہو رہا ہے اور ہر ڈیٹابیس کی تاریخ۔

ویب UI کے بغیر، فائل کو ڈیٹا ڈائریکٹری میں `country.mmdb`، `asn.mmdb`، `both.mmdb` (ممالک اور نیٹ ورکس والی ایک فائل، جیسے IPinfo Lite) یا `asn.tsv.gz` کے نام سے کاپی کریں اور traffic66 ری اسٹارٹ کریں۔

**مقامات اور نیٹ ورک** دوسرے ممالک کے ساتھ ٹریفک کو دنیا کے نقشے پر دکھاتا ہے: ملک جتنا گہرا، ٹریفک اتنی زیادہ۔ کسی ملک پر پوائنٹر لے جائیں تو اس کی ٹریفک نظر آتی ہے؛ کلک کر کے فلٹر کریں یا اس کے flow records کھولیں۔ جب آپ کے نیٹ ورکس کا ملک دیا گیا ہو (`net` لائن میں `country=`، [نام](#7-names-snmp-and-your-own-networks) دیکھیں)، تو اس ملک سے ان ممالک تک لکیریں جاتی ہیں جن سے ٹریفک کا تبادلہ ہوتا ہے، جتنی زیادہ ٹریفک اتنی موٹی لکیر۔ ممالک کی سرحدیں [Natural Earth](https://www.naturalearthdata.com) (پبلک ڈومین) سے ہیں۔

![مقامات اور نیٹ ورک: دنیا کے نقشے پر ملک کے لحاظ سے بیرونی ٹریفک](images/geo.png)

Threat lists سادہ text فائلیں ہیں، ہر لائن میں ایک address یا نیٹ ورک (`#` یا
`;` کے بعد کا متن نظرانداز ہوتا ہے)، جو
`<data directory>/threats/<name>.txt` کے طور پر محفوظ کی جاتی ہیں، مثلاً:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

lists شامل کرنے یا بدلنے کے بعد traffic66 restart کریں۔ matches
**خطرے کی معلومات** پر list کے نام کے حساب سے نظر آتے ہیں۔

![خطرے کی معلومات: ایک internal host جو threat list کے ایک address کو ڈیٹا بھیج رہا ہے](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. web UI کا استعمال

آپ کو شاید ہی کچھ ٹائپ کرنا پڑے۔ ہر صفحے کی ہر value — address، port،
application، ملک، ڈیوائس — پر کلک کیا جا سکتا ہے:

- **صرف یہ دکھائیں** / **اسے خارج کریں** ایک filter شامل کرتا ہے۔ filters اوپر
  والی bar کے نیچے نظر آتے ہیں اور ہٹائے جانے تک ہر صفحے پر لاگو رہتے ہیں۔
- **اس کے فلو ریکارڈ دکھائیں** متعلقہ انفرادی flows کھولتا ہے۔
- **تفصیل دیکھیں** (hosts، ڈیوائسز اور services) اس ایک host یا service کے بارے
  میں ایک صفحہ کھولتا ہے: وقت کے ساتھ application کے حساب سے اس کی ٹریفک، وہ کس
  سے بات کرتا ہے، کون سی services یا clients، ممالک اور اس کے تازہ ترین flows۔
  وہاں کی ہر value پر دوبارہ کلک کیا جا سکتا ہے، تاکہ آپ مزید گہرائی میں جا سکیں؛
  browser کا Back بٹن واپس لے جاتا ہے۔
- **نام دیں…** (hosts اور ڈیوائسز) address کو ایک نام دیتا ہے، جو اس کے بعد ہر
  جگہ دکھایا جاتا ہے۔
- **آن لائن معلوم کریں** address یا AS کو کسی public lookup سائٹ پر کھولتا ہے۔
- **کاپی کریں** value کاپی کرتا ہے۔

صفحات:

| صفحہ | کس سوال کا جواب دیتا ہے |
|---|---|
| جائزہ | ابھی کتنی ٹریفک ہے اور پچھلے ہفتے کے مقابلے میں کتنی، application کے حساب سے؛ کھلی مشتبہ سرگرمیاں؛ سمت اور protocol؛ سرفہرست clients اور services |
| ٹاپ 66 | **سب سے فعال ہوسٹ** پر کھلتا ہے: سرفہرست 30 clients اور servers ساتھ ساتھ، ٹریفک، پیکٹ اور flow records کے ساتھ، پوری ٹریفک کی ایک row کے اوپر۔ **جدول** سرفہرست 66 کی ایک table ہے: ڈیفالٹ طور پر conversations (client، server، service، ملک)۔ ہر column heading سے sort ہوتا ہے؛ عددی columns (ٹریفک، پیکٹ، اوسط پیکٹ، flows) وقت کی حد کے پورے ٹریفک سے سرفہرست 66 دوبارہ چنتے ہیں، اس لیے سب سے چھوٹے اوسط پیکٹ سے scan اور flood پکڑے جاتے ہیں۔ **گروپ بندی بلحاظ** سے applications، نیٹ ورکس، segments، ڈیوائسز، encapsulation اور VLAN پر جائیں |
| ٹریفک کی تفصیل | دو ring charts۔ **سرورز اور کلائنٹس**: اندرونی دائرہ 8 سب سے مصروف servers، بیرونی دائرہ ہر ایک کے clients؛ **کلائنٹس اندر** اسے الٹ دیتا ہے (clients اندر، اور ہر ایک کے استعمال کردہ servers باہر)، کیونکہ اکثر ایک طرف دوسری سے زیادہ وضاحت کرتی ہے۔ **سروسز اور سرورز**: services اندر اور انہیں فراہم کرنے والے servers باہر، یا اس کے الٹ۔ کسی حصے پر پوائنٹر لے جائیں تو اس کی ٹریفک نظر آتی ہے؛ کسی بھی value کی طرح اس پر click کریں |
| ٹریفک کے راستے | کون سا host کس ملک کی طرف کون سی application استعمال کرتا ہے: 8 سب سے مصروف hosts، باقی "دیگر" کے طور پر۔ **کلائنٹ ← سرور** کلائنٹ ← سروس ← سرور دکھاتا ہے؛ **سیگمنٹ کے لحاظ سے** hosts کی جگہ نیٹ ورکس دکھاتا ہے۔ لمبے نام 22 حروف تک مختصر کیے جاتے ہیں؛ پورا نام دیکھنے کے لیے اس پر پوائنٹر لے جائیں |
| مشتبہ سرگرمی | کس چیز پر توجہ چاہیے: scans، پاس ورڈ کا اندازہ، lateral movement، غیر معمولی uploads، floods اور threat list والی ٹریفک ([مزید](#findings)) |
| خطرے کی معلومات | وہ hosts جنہوں نے آپ کی threat lists کے addresses سے بات کی، اور کتنا بھیجا |
| مقامات اور نیٹ ورک | ملک کے لحاظ سے ٹریفک کا دنیا کا نقشہ، آپ کے نیٹ ورکس سے لکیروں کے ساتھ؛ وہ نیٹ ورکس (AS) جہاں سے ٹریفک آئی اور جہاں گئی، وقت کے ساتھ bits/s اور packets/s میں؛ ملک کے حساب سے اور نیٹ ورک کے حساب سے ٹریفک |
| ذرائع | ڈیوائسز، sampling، loss، collectors، SNMP، ممالک اور نیٹ ورکس کا ڈیٹابیس، لوگو، اور **نام** |
| انٹرفیس جانچ | وقت کے ساتھ ہر interface کی ٹریفک، ایک chart میں ingress (سبز) اور egress (نیلا)، bits/s اور packets/s میں، اور interface counters کے ساتھ flow کے اعداد، بدترین پہلے، وجوہات کے ساتھ |
| فلو ریکارڈ | کتنے flow records تھے اور کب (ہر interval کے لیے ایک bar)، اور خود records، نئے پہلے، صفحہ بہ صفحہ، منتخب کیے جا سکنے والے columns کے ساتھ |
| ڈیٹا کی صفائی | 120، 90، 60، 30 یا 7 دن سے پرانا ڈیٹا، یا سارا ڈیٹا حذف کرتا ہے، اور بتاتا ہے کہ ہر انتخاب کتنی جگہ خالی کرتا ہے ([مزید](#13-data-backup-upgrade-uninstall)) |
| آف لائن pcap تجزیہ | pcap، pcapng کیپچر کا لائیو ڈیٹا سے الگ تجزیہ ([مزید](#آف-لائن-pcap-تجزیہ)) |

سائیڈ مینو صفحات کو چار گروپوں میں دکھاتا ہے: ٹریفک (جائزہ، ٹاپ 66، ٹریفک کی
تفصیل، ٹریفک کے راستے)، سیکیورٹی (مشتبہ سرگرمی، خطرے کی معلومات، مقامات اور
نیٹ ورک)، سیٹ اپ اور ڈیٹا (ذرائع، انٹرفیس جانچ، فلو ریکارڈ، ڈیٹا کی صفائی) اور
آف لائن pcap تجزیہ۔ لوگو کے نیچے version اور server کی تاریخ اور وقت ہوتے ہیں۔

صفحات کے اوپر: time range (15 منٹ سے 30 دن، یا کسی بھی آغاز اور اختتام کے لیے
**اپنی مرضی…**، 30 دن سے پہلے کا بھی)، ہر 30 سیکنڈ پر automatic refresh، اور
**لنک کاپی کریں**، جو بالکل موجودہ view (صفحہ، time range اور filters) کا link
کاپی کرتا ہے تاکہ آپ اسے کسی ساتھی کو بھیج سکیں۔ **ٹاپ 66** اور **ٹریفک کی
تفصیل** پر ایک search box اور **آلہ**، **کلائنٹ**، **سرور** اور **سروس** وقت کی
حد کی سب سے مصروف values دکھاتے ہیں: filter کرنے کے لیے ایک چنیں یا ٹائپ کریں؛
پھر filter ہر صفحے پر لاگو رہتا ہے جب تک آپ box خالی نہ کریں۔ زبان browser کے
مطابق ہوتی ہے؛ menu کے نیچے، **لاگ آؤٹ** کے اوپر سے بدلیں۔

وقت کے charts سب سے بڑی 8 values مقررہ رنگوں میں اور باقی کو "دیگر" کے طور پر
دکھاتے ہیں؛ legend ہر value کا کل دیتا ہے اور اس پر کسی بھی دوسری value کی طرح
click کیا جا سکتا ہے۔ clients اور servers کے charts باقی کو drawing سے باہر
رکھتے ہیں، کیونکہ ہزاروں hosts کے ساتھ وہ سرفہرست 8 کو چپٹا کر دیتا؛ legend
پھر بھی اس کا کل دیتا ہے۔

6 گھنٹے سے لمبی ranges پورے گھنٹے سے شروع ہوتی ہیں، تاکہ صفحے کا ہر عدد
بالکل ایک ہی وقت گنے: "24 گھنٹے" میں آخری 24 پورے گھنٹے اور موجودہ گھنٹہ
شامل ہیں۔ ان ranges پر ٹاپ 66 گھنٹہ وار summaries سے آتا ہے؛ وہاں filters
دستیاب نہیں، اور صفحہ یہ بتا دیتا ہے۔ filter کرنے کے لیے چھوٹی range چنیں۔ گفتگو (conversations) ہمیشہ flow کی تفصیل پڑھتی ہے، اس لیے
زیادہ flow rates پر لمبی ranges میں اس میں کچھ وقت لگ سکتا ہے؛ ایک گھنٹہ سب سے تیز ہے۔

سائیڈ menu دکھاتا ہے کہ data کتنی disk استعمال کر رہا ہے اور کتنی خالی ہے؛ خالی
جگہ پر hover کریں تو پتا چلتا ہے کہ موجودہ رفتار پر رکھے گئے دنوں کی تفصیل کو کتنی
جگہ چاہیے (ایک دن کا data جمع ہونے کے بعد اندازہ لگایا جاتا ہے)۔

سائن اِن پیج پر اور menu کے اوپر اپنا لوگو دکھانے کے لیے
**ذرائع → لوگو → لوگو اپ لوڈ کریں…** استعمال کریں: PNG، SVG، JPEG، WebP یا GIF،
1 MB تک، 272 × 92 pixels پر بہترین (دوسرے سائز فٹ ہونے کے لیے scale کیے جاتے
ہیں)۔ **بلٹ اِن لوگو پر واپس جائیں** سے traffic66 کا لوگو واپس آ جاتا ہے۔

<a id="findings"></a>

### مشتبہ سرگرمی

**مشتبہ سرگرمی** وہ سب دکھاتا ہے جو traffic66 نے flows میں پایا، سب سے سنگین
پہلے۔ یہ ہر 5 منٹ پر پچھلے 10 منٹ جانچتا ہے؛ جو چیز ایک گھنٹے تک جاری رہے وہ
ایک ہی اندراج ہے جو بڑھتا رہتا ہے، ہر جانچ پر نیا نہیں۔

| سرگرمی | اس کا مطلب | شدت |
|---|---|---|
| اسکین | ایک address نے ایک ہی port پر بہت سے addresses کو چھوٹے probes بھیجے (TCP یا ping) | آپ کے نیٹ ورک کے اندر سے ہو تو زیادہ، انٹرنیٹ سے ہو تو کم |
| پورٹ اسکین | ایک address نے ایک host کے بہت سے ports پر چھوٹے probes بھیجے | اندر سے زیادہ، انٹرنیٹ سے کم |
| پاس ورڈ کا اندازہ | کسی login service (SSH، RDP، SMB، databases وغیرہ) سے بہت سے مختصر connections | اندر سے زیادہ، انٹرنیٹ سے کم |
| لیٹرل موومنٹ | آپ کے نیٹ ورک کے اندر، file sharing یا remote administration sessions (SMB، RDP، SSH، WinRM، VNC) ایسے hosts سے جنہوں نے پہلے کبھی وہ service پیش نہیں کی | زیادہ |
| غیر معمولی اپ لوڈ | ایک internal host نے جتنا وصول کیا اس سے کہیں زیادہ بھیجا (10 منٹ میں 100 MB، وصول شدہ کا تین گنا) ایسے address کو جس سے پہلے کبھی ڈیٹا کا تبادلہ نہیں ہوا تھا | زیادہ |
| فلڈ | ایک address پر فی سیکنڈ 20,000 یا اس سے زیادہ چھوٹے packets، اس کی معمول کی رفتار کا دس گنا | درمیانہ |
| خطرے کی فہرست | آپ کی کسی threat list کے address کے ساتھ ٹریفک | زیادہ جب آپ کے host نے اس سے connect کیا، کم جب فہرست والا address باہر سے دستک دے رہا تھا |

ہر اندراج بتاتا ہے کہ کس نے کس کے ساتھ کیا کیا، کب اور کتنی دیر تک، اس کے
پیچھے کے اعداد کے ساتھ، اور یہ کہ ڈیٹا کیسے sample کیا گیا تھا۔ **تفصیل** host
کا صفحہ کھولتا ہے، جس میں اس کے بارے میں سرگرمیاں بھی درج ہوتی ہیں۔
**نمٹا دیا** اندراج کو بند کرتا ہے؛ اگر وہی دوبارہ ہو تو نیا اندراج کھلتا ہے۔
**مسئلہ نہیں** اسے ہمیشہ کے لیے بند کر دیتا ہے: وہ پھر کبھی رپورٹ نہیں ہوتا۔
سائیڈ menu میں **مشتبہ سرگرمی** کے ساتھ والا سرخ نمبر پچھلے 24 گھنٹوں کے کھلے
زیادہ اور درمیانہ شدت والے اندراج گنتا ہے۔

لیٹرل موومنٹ اور غیر معمولی uploads کے لیے یہ جاننا ضروری ہے کہ معمول کیا ہے،
اس لیے یہ ایک دن کی ہسٹری جمع ہونے کے بعد ہی رپورٹ ہوتے ہیں۔ پہلی بار شروع
ہونے پر traffic66 پہلے سے موجود ہسٹری سے سیکھتا ہے۔

Sampled data (sFlow، sampled NetFlow) کے ساتھ قواعد وہی گنتے ہیں جو samples
دکھاتے ہیں اور کم تعداد مانگتے ہیں، لیکن پھر ہر ایک کو ایک مختصر probe جیسا
دکھنا چاہیے، تاکہ مصروف عام hosts انہیں trigger نہ کریں۔ جو sampling چھپا دے
وہ پکڑا نہیں جا سکتا: 1:4096 sampling کے پیچھے، چند درجن hosts کا scan اتنے کم
packets بھیجتا ہے کہ نظر نہیں آتا۔ ڈیمو کا حملہ ایک ایسے switch سے گزرتا ہے جو
1:4096 پر sample کرتا ہے، اور پورا پکڑا جاتا ہے؛ ڈیمو کی ایک دن کی عام ٹریفک
سے کوئی سرگرمی نہیں نکلتی، سوائے website پر دستک دینے والے internet scanner
کے۔

![مشتبہ سرگرمی: ایک حملے کا ہر مرحلہ، 1:4096 sFlow sampling کے باوجود پکڑا گیا](images/findings.png)

![ٹاپ 66، سب سے فعال ہوسٹ: پوری ٹریفک کی ایک row کے ساتھ سرفہرست 30 clients اور servers](images/topn.png)

![ٹریفک کی تفصیل: servers اپنے clients کے ساتھ، اور services اپنے servers کے ساتھ، ring charts کی صورت میں](images/traffic.png)

![ایک host کی تفصیل: اس کے بارے میں مشتبہ سرگرمیاں، اس کی ٹریفک، وہ کس سے بات کرتا ہے، services، ممالک اور تازہ ترین flows](images/detail.png)

![ٹریفک کے راستے: کون سا host کس ملک کی طرف کون سی application استعمال کرتا ہے](images/paths.png)

یہی جائزہ چینی زبان میں؛ ہر صفحہ 13 زبانوں میں دستیاب ہے:

![چینی زبان میں جائزہ](images/overview-zh.png)

<a id="10-terminal-ui"></a>

### آف لائن pcap تجزیہ

**آف لائن pcap تجزیہ** Wireshark یا tcpdump کے کیپچر کو لائیو ڈیٹا والے صفحات پر ہی دکھاتا ہے، انہیں ملائے بغیر۔

یہ تمام پیکٹوں کو فلو میں سمیٹتا ہے: کس نے کس سے، کتنی، کب بات کی، اور کیا حملے جیسا لگتا ہے۔ یہ پروٹوکول ڈی کوڈ نہیں کرتا اور پیکٹ کا مواد نہیں دکھاتا؛ ایک پیکٹ یا ایک TCP اسٹریم دیکھنے کے لیے Wireshark استعمال کریں۔

کمانڈ لائن سے، بغیر کسی سیٹ اپ کے:

```
traffic66 office.pcap
traffic66 a.pcap b.pcapng c.pcap
```

traffic66 صرف اسی کمپیوٹر پر (127.0.0.1، کوئی خالی پورٹ) شروع ہوتا ہے، پتا، پاس ورڈ اور ایک بار چلنے والا سائن اِن لنک دکھاتا ہے، اور براؤزر میں کیپچر کھول دیتا ہے۔ زیادہ سے زیادہ 3 فائلیں، کل 3 GB؛ فائلیں اپنی جگہ سے پڑھی جاتی ہیں اور کبھی بدلی نہیں جاتیں۔ کچھ بھی جمع یا بھیجا نہیں جاتا، اور ہوسٹ نام نہیں ڈھونڈے جاتے (`-dns` سے چالو کریں)۔ Ctrl+C روکتا ہے اور امپورٹ کیا گیا ڈیٹا مٹا دیتا ہے۔ 2 کور مشین پر 1 GB کا کیپچر تقریباً 5 سیکنڈ (12 لاکھ پورے سائز کے پیکٹ) سے 30 سیکنڈ (1.4 کروڑ چھوٹے پیکٹ) میں تیار ہوتا ہے۔

```
$ traffic66 office.pcap

traffic66 0.3.1: analysing 1 capture file(s); nothing is collected or sent
  Web UI    http://127.0.0.1:38217  (port 38217, this computer only)
  Sign in   user admin, password gfhfhbuutz2e
  Open      http://127.0.0.1:38217/auto?t=b9388f…  (signs in once)
  Stop      Ctrl+C; the imported data is deleted, your files are kept
```

چلتے ہوئے traffic66 کے ویب UI میں:

1. **کیپچر فائلیں اپ لوڈ کریں…**: ‎`.pcap` یا ‎`.pcapng`، کمپریس شدہ نہیں۔ زیادہ سے زیادہ 3 فائلیں، ہر ایک 50 MB تک۔ فائلیں فلو میں بدل کر الگ ڈیٹابیس (`<data>/sandbox/`) میں جاتی ہیں؛ لائیو ڈیٹا، اس کے اعداد اور نتائج متاثر نہیں ہوتے۔
2. **تجزیہ کریں**: تمام صفحات (جائزہ، Top 66، ٹریفک کی تفصیل، نتائج، فلو راستے، نقشہ، فلو ریکارڈز) کیپچر فائلوں کا پورا وقت دکھاتے ہیں۔ نارنجی پٹی فائلوں کے نام بتاتی ہے؛ **لائیو ڈیٹا پر واپس** سے واپس جائیں۔ ہر فائل ایک ڈیوائس کی طرح دکھتی ہے، اس لیے **آلہ** خانے سے ایک ایک فائل دیکھی جا سکتی ہے۔
3. شناخت کے قواعد کیپچر پر بھی چلتے ہیں: اسکین، پورٹ اسکین اور پاس ورڈ کا اندازہ **مشتبہ سرگرمی** میں آتے ہیں۔ جن قواعد کو ایک دن کی تاریخ چاہیے (لیٹرل موومنٹ، غیر معمولی اپ لوڈ) وہ کیپچر پر لاگو نہیں ہوتے۔
4. **حذف کریں** ایک فائل اور اس کا ڈیٹا ہٹاتا ہے؛ **سب حذف کریں** سب کچھ ہٹاتا ہے۔

ڈیمو میں حملے والی ایک مثال کیپچر شامل ہے۔

![آف لائن تجزیہ: کیپچر فائلیں، ان کے پیکٹ، فلو اور وقت](images/sandbox.png)

## 10. Terminal UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

traffic66 مشین پر `traffic66 tui` خود ہی سائن اِن کر لیتا ہے جب وہ data
directory پڑھ سکے (ڈیفالٹ نہ ہو تو `-data` دیں)۔ اگر traffic66 کسی دوسرے یوزر
کے طور پر چلتا ہے، جیسا کہ service چلتی ہے، تو اس کے بجائے `-user` اور
`-password` استعمال کریں۔ `-lang` زبان چنتا ہے (`en`، `zh`، `hi`،
`es`، `ar`، `fr`، `bn`، `pt`، `ru`، `id`، `ur`، `ja`، `ko`)۔

Keys: 1–8 صفحات، ↑↓ منتخب کریں، Enter منتخب value پر actions، f صرف یہ
دکھائیں، x خارج کریں، / search، t time range، c filters صاف کریں، w یہی view
browser میں کھولیں، q باہر نکلیں۔

![Terminal UI: جائزہ](images/tui-overview.png)

![Terminal UI: ٹاپ 66 conversations](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Local capture

Flow exports وصول کرنے کے علاوہ traffic66 جس مشین پر چلتا ہے، اس کے کسی
network interface کے packets سے خود بھی flows بنا سکتا ہے۔ اسے کیا نظر آتا
ہے، یہ interface پر منحصر ہے:

| Interface | traffic66 کو کیا نظر آتا ہے |
|---|---|
| switch کے mirror (SPAN) port سے جڑا ایک فالتو network port | وہ ساری ٹریفک جو switch mirror کرتا ہے: پورا نیٹ ورک یا uplink |
| مشین کا اپنا Ethernet یا Wi-Fi | صرف اسی مشین کی اپنی ٹریفک |

Wi-Fi adapters دوسری ڈیوائسز کی ٹریفک نہیں دیکھ سکتے۔ پورا Wi-Fi نیٹ ورک دیکھنے
کے لیے router یا access point سے flows export کروائیں (حصہ 4)، یا وہ switch
port mirror کریں جس سے access point جڑا ہے۔

<a id="windows-1"></a>

### Windows

1. [Npcap](https://npcap.com) اس کے default options کے ساتھ انسٹال کریں۔ اگر آپ
   "Restrict Npcap driver's access to Administrators only" پر ٹک لگائیں تو
   traffic66 کو Administrator کے طور پر چلائیں۔
2. Interfaces کی فہرست دیکھیں (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Name والا column، Windows کی network settings میں connection کا نام ہے؛ جو
   interface استعمال میں ہے، اس کا ایک address ہوتا ہے۔
3. Wi-Fi پر capture کریں، نام سے یا نمبر سے:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   جن ناموں میں spaces ہوں، انہیں quotes میں لکھیں: `-capture "Ethernet 2"`۔
   کئی interfaces پر capture کرنے کے لیے `-capture` دہرائیں۔ اگر آپ کو صرف
   capture چاہیے اور کوئی flow collector نہیں، تو `-listen=` شامل کریں۔ حصہ 2
   والے startup task کے لیے یہ option `-Argument` میں شامل کریں:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`۔

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

Capture کے لیے root درکار ہے، یا capabilities `CAP_NET_RAW` اور `CAP_NET_ADMIN`:
اوپر والی `setcap` لائن، یا حصہ 2 کی systemd unit میں `AmbientCapabilities`
والی لائن۔ Wi-Fi interfaces کے نام عموماً `wlan0` یا `wlp…` ہوتے ہیں۔

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

Capture کے لیے root درکار ہے؛ کچھ انسٹال نہیں کرنا۔ MacBooks پر `en0` ہی Wi-Fi ہے۔

<a id="checking-that-it-works"></a>

### جانچیں کہ یہ کام کر رہا ہے

**ذرائع** ہر capture ہونے والا interface، capture کے طریقے اور دیکھے گئے packets
کی تعداد کے ساتھ دکھاتا ہے۔ Flows ڈیوائس `127.0.0.1` (یہ مشین) سے آتے دکھائی
دیتے ہیں، ہر صفحے پر، کسی بھی دوسری ڈیوائس کی طرح۔ دو بار دیکھے گئے packets
(مثلاً دو mirror ports پر) دو بار گنے جاتے ہیں۔

<a id="12-options"></a>

## 12. آپشنز

`traffic66 -h` اور `traffic66 <command> -h` سب کچھ دکھاتے ہیں۔

کمانڈز:

| کمانڈ | |
|---|---|
| `traffic66` | flows جمع کرتا ہے اور web UI چلاتا ہے |
| `traffic66 demo` | یہی، ایک simulated نیٹ ورک کے ساتھ |
| `traffic66 tui` | چلتے ہوئے traffic66 کے لیے terminal UI |
| `traffic66 passwd` | یوزرز شامل کرتا، بدلتا، ان کی فہرست دکھاتا یا انہیں حذف کرتا ہے (دیکھیں [یوزرز اور پاس ورڈ](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | کسی collector کو simulated exports بھیجتا ہے |
| `traffic66 interfaces` | local capture کے لیے interfaces کی فہرست |
| `traffic66 version` | version دکھاتا ہے |

`traffic66` اور `traffic66 demo` کے آپشنز:

| آپشن | ڈیفالٹ | |
|---|---|---|
| `-addr` | `:8066` | web UI کا address؛ صرف اسی مشین کے لیے `127.0.0.1:8066` |
| `-data` | پروگرام کے ساتھ `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors، `name=address` کی شکل میں، comma سے الگ؛ خالی ہو تو بند |
| `-user` | `admin` | پہلی بار شروع ہونے پر بننے والے یوزر کا نام، اور اس یوزر کا جس پر `-password` لاگو ہوتا ہے |
| `-password` | سیٹ نہیں | اس run میں صرف اسی پاس ورڈ کے ساتھ `-user` کو قبول کرتا ہے، `password` فائل کو نظرانداز کر کے (`TRAFFIC66_PASSWORD` بھی) |
| `-retention-days` | `30` | flow detail کتنے دن رکھی جائے؛ summaries 400 دن رکھی جاتی ہیں |
| `-memory` | `0.10` | physical memory کا کتنا حصہ database cache کے لیے، اور اتنا ہی باقی پروگرام کے لیے soft limit (ہر ایک کم از کم 256 MB) |
| `-l2-overhead` | `18` | NetFlow/IPFIX byte counts میں ہر packet پر جوڑے جانے والے bytes |
| `-sampling-wait` | `5m` | records کتنی دیر sampling rate کا انتظار کریں |
| `-capture` | | local interface پر capture (دہرایا جا سکتا ہے) |
| `-inventory` | `<data>/inventory.txt` | ناموں کی فائل |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table (`.mmdb` فائلیں: انہیں اپ لوڈ کریں، یا `<data>/country.mmdb` اور `<data>/asn.mmdb`, `<data>/both.mmdb`) |
| `-threat` | `<data>/threats/*.txt` | اضافی threat list، `name=path` کی شکل میں (دہرایا جا سکتا ہے) |
| `-dns-upstream` | system resolver | host names دکھانے کے لیے DNS server |
| `-dns-rate` | `20` | فی سیکنڈ زیادہ سے زیادہ reverse lookups |
| `-dns-cache` | `2m` | host names کتنی دیر cache میں رہیں |
| `-no-dns` | | کوئی reverse lookup نہیں |
| `-tui` | | ساتھ میں terminal UI بھی کھولیں |

مثال: ایک دوسرا collector port، ایک سال کی detail، اور web UI صرف local مشین پر:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. ڈیٹا، بیک اپ، اپ گریڈ، اَن انسٹال

سب کچھ data directory میں رہتا ہے:

| | |
|---|---|
| `raw/` | flow detail، ہر گھنٹے کی ایک compressed فائل |
| `traffic66.duckdb` | summaries، interface counters اور موجودہ گھنٹہ |
| `password` | login پاس ورڈز (hashed) |
| `inventory.txt` | نام (**ذرائع → نام**) |
| `license.json` | انسٹالیشن نمبر اور لائسنس ([آزمائش اور لائسنس](#trial-and-licence) دیکھیں) |
| `logo.png` (یا `.svg`، `.jpg`، `.webp`، `.gif`) | آپ کا لوگو (**ذرائع → لوگو**)، اگر آپ نے اپ لوڈ کیا ہو |
| `country.mmdb`، `asn.mmdb`، `both.mmdb`، `asn.tsv.gz`، `dbip-country.mmdb`، `dbip-asn.mmdb`، `threats/`، `sandbox/` | آپ کے شامل کردہ ممالک اور نیٹ ورکس کے databases اور threat lists |

**ڈیٹا کتنی دیر رکھا جاتا ہے**: flow detail 30 دن، summaries (overview اور لمبی مدتیں) 400 دن۔ اس سے پرانا
ڈیٹا خود بخود حذف ہوتا ہے، ہر 5 منٹ میں جانچ ہوتی ہے؛ اس کے علاوہ کچھ حذف نہیں ہوتا اور کوئی اور حد نہیں۔ detail
کی مدت `-retention-days` سے بدلیں، جتنے چاہیں دن، مثلاً `-retention-days 365`۔ ڈسک کا استعمال بھی اسی حساب سے
بڑھتا ہے: رکھے گئے دن نہ سمائیں تو سائیڈ مینو میں **خالی** سرخ ہو جاتا ہے۔ ڈسک بھر جائے تو جگہ خالی ہونے تک نئے
flows محفوظ نہیں ہو سکتے۔

سائیڈ مینو میں **ڈیٹا کی صفائی** ضرورت پڑنے سے پہلے ڈیٹا حذف کرتا ہے: 120، 90،
60، 30 یا 7 دن سے پرانا، یا سارا ڈیٹا۔ یہ ہر انتخاب کے لیے دکھاتا ہے کہ کتنے
flow records جائیں گے اور تقریباً کتنی ڈسک خالی ہوگی، اور حذف کرنے سے پہلے پوچھتا
ہے۔ flow records، گھنٹہ وار اور روزانہ summaries، interface counters اور مشتبہ
سرگرمیاں حذف ہوتی ہیں؛ سارا ڈیٹا حذف کرنے سے detection rules کا سیکھا ہوا بھی
reset ہو جاتا ہے۔ اسے واپس نہیں کیا جا سکتا۔

- **بیک اپ**: traffic66 روکیں اور directory کاپی کریں۔ روکے بغیر
  `raw/`، `password` اور `inventory.txt` کاپی کریں؛ اس صورت میں موجودہ گھنٹہ اور
  summaries شامل نہیں ہوں گی۔
- **منتقلی**: traffic66 روکیں، directory منتقل کریں، اور نئی جگہ کی طرف اشارہ
  کرتے `-data` کے ساتھ شروع کریں۔
- **اپ گریڈ**: traffic66 روکیں، پروگرام فائل بدلیں، دوبارہ شروع کریں۔ ڈیٹا
  برقرار رہتا ہے۔ مثلاً Linux پر:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **اَن انسٹال**: service یا startup task روکیں اور ہٹائیں
  ([انسٹال کریں](#2-install) دیکھیں)، پھر پروگرام فولڈر اور data directory
  ڈیلیٹ کر دیں۔

<a id="trial-and-licence"></a>

### آزمائش اور لائسنس

traffic66 کو 30 دن آزمایا جا سکتا ہے۔ پہلی بار شروع ہونے پر یہ data directory
میں 8 ہندسوں کے انسٹالیشن نمبر کے ساتھ `license.json` لکھتا ہے۔ ہر صفحے کے نیچے
دکھایا جاتا ہے کہ آزمائش کے کتنے دن باقی ہیں، پھر یہ کہ آزمائش ختم ہو گئی۔ دونوں
صورتوں میں کچھ بند نہیں ہوتا: تمام خصوصیات کام کرتی رہتی ہیں۔

رجسٹر کرنے کے لیے مصنف کو انسٹالیشن نمبر بھیجیں (یہ ہر صفحے کے نیچے بھی دکھتا
ہے)۔ لائسنس ایک نئی `license.json` کی صورت میں واپس آتا ہے؛ اسے پرانی کی جگہ
data directory میں رکھ دیں۔ traffic66 شروع ہونے پر اور ہر 4 گھنٹے بعد اس کی
جانچ ہوتی ہے، اس لیے restart کی ضرورت نہیں؛ پھر صفحے کے نیچے دکھتا ہے کہ لائسنس
کس کے نام ہے اور کتنے دن باقی ہیں۔

<a id="14-security"></a>

## 14. سکیورٹی

- web UI سادہ HTTP استعمال کرتا ہے: پاس ورڈز اور ڈیٹا بغیر encryption کے نیٹ ورک
  پر جاتے ہیں۔ جن نیٹ ورکس پر آپ کو پورا بھروسا نہیں، وہاں صرف اسی مشین پر سنیں
  (`-addr 127.0.0.1:8066`) اور آگے ایک TLS reverse proxy لگائیں، مثلاً
  [Caddy](https://caddyserver.com) کے ساتھ:
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`۔
  یا VPN یا SSH tunnel کے ذریعے رسائی حاصل کریں:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`، پھر
  http://127.0.0.1:8066 کھولیں۔
- UDP collector ports کی اجازت صرف اپنی ڈیوائسز کے addresses سے دیں۔
- `inventory.txt` میں SNMP communities سادہ متن (clear text) میں محفوظ ہوتی ہیں؛
  read-only community استعمال کریں۔

<a id="15-sizing"></a>

## 15. گنجائش کا اندازہ

2-core مشین پر 5,000 flows فی سیکنڈ پر ناپا گیا: detail روزانہ تقریباً 12 GB
disk لیتی ہے، ساتھ موجودہ گھنٹے کے لیے تقریباً 1.5 GB؛ پروگرام ایک core کا
چھٹا حصہ لیتا ہے۔ لمبی time ranges کے overviews
summaries سے آتے ہیں اور 0.2 s سے کم لیتے ہیں۔ detail پر queries فی گھنٹہ
تقریباً 2 کروڑ 20 لاکھ rows scan کرتی ہیں: 1 گھنٹے میں ایک host کے لیے 1 s سے کم،
تمام conversations کا 1 گھنٹے کا Top 66 تقریباً 9 s؛ وقت range کے ساتھ بڑھتا ہے
اور زیادہ cores کے ساتھ کم ہوتا ہے۔

لہٰذا 5,000 flows/s پر 30 دن کے لیے disk تقریباً 360 GB ہے؛ اسے اپنے flow rate
(**ذرائع** پر نظر آتا ہے) اور `-retention-days` کے تناسب سے بڑھائیں یا گھٹائیں۔

Memory: `-memory` (ڈیفالٹ RAM کا 10%، کم از کم 256 MB) database cache کو
محدود کرتا ہے، اور باقی پروگرام کو اتنے ہی سائز کی soft limit ملتی ہے۔ 5,000
flows فی سیکنڈ پر پروگرام کا اپنا data (decoding، duplicate detection، batches)
تقریباً 90 MB لیتا ہے؛ کل ملا کر 0.6–0.8 GB مان کر چلیں، اس لیے 2 GB RAM والی
مشین کافی ہے۔ مسلسل 10 منٹ collection کے دوران (8 GB مشین پر peak 0.58 GB) اور
2 GB مشین کی limits کے ساتھ اس سے گیارہ گنا rate پر ایک گھنٹے کے flows load
کرتے وقت (peak 0.74 GB) ناپا گیا۔

`-memory` ایک budget ہے، hard cap نہیں: Go کی limit soft ہے اور database
تھوڑی دیر کے لیے اپنے حصے سے زیادہ لے سکتا ہے۔ Hard cap کے لیے operating
system کی limit استعمال کریں: systemd unit میں `MemoryMax=` (حصہ 2) یا
container کی memory limit۔ `-memory` والے حصے کا تقریباً 2.5 گنا اور کم از کم
1 GB رکھیں؛ ڈیفالٹ حصے پر 8 GB تک کی مشینوں کے لیے `MemoryMax=2G` ٹھیک ہے۔
تب مشین کی memory ختم ہونے کے بجائے traffic66 restart ہو جاتا ہے۔

<a id="16-troubleshooting"></a>

## 16. مسائل کا حل

| علامت | وجہ اور حل |
|---|---|
| ڈیوائس **ذرائع** میں نہیں | packets نہیں پہنچ رہے: [جانچیں کہ flows پہنچ رہے ہیں](#5-check-that-flows-arrive) دیکھیں |
| "waiting for the sampling rate" | ڈیوائس نے ابھی تک اپنے sampler options نہیں بھیجے؛ زیادہ تر چند منٹ میں دوبارہ بھیج دیتی ہیں۔ اگر کبھی نہ بھیجے تو انہیں export کروائیں (Cisco پر `option sampler-table`) یا اگر وہ واقعی 1:1 ہے تو نام میں اسے `unsampled` لکھیں |
| اعداد interface counters سے کم | **انٹرفیس جانچ** دیکھیں: راستے میں loss، interfaces sample نہیں ہو رہے، یا flows ابھی ڈیوائس کی cache میں ہیں (active timeout 60 s سے لمبا) |
| اعداد interface counters سے زیادہ | ایک ہی ٹریفک دو interfaces یا دو ڈیوائسز پر sample ہو رہی ہے |
| کوئی ملک یا نیٹ ورک نہیں ("نامعلوم") | کوئی database لوڈ نہیں: **ذرائع** پر ایک اپ لوڈ کریں، [ممالک](#8-countries-networks-and-threat-lists) دیکھیں |
| صفحے پر "ڈیٹا بیس اپنی میموری کی حد تک پہنچ گیا اور جواب نہیں دے سکا" | کم وقت کی حد چنیں، یا بڑے `-memory` کے ساتھ شروع کریں؛ تفصیل log میں ہے |
| پاس ورڈ بھول گئے | traffic66 مشین پر `traffic66 passwd` (اگر traffic66 `-data` کے ساتھ چلتا ہے تو `-data` شامل کریں) |
| `Conflicting lock is held` | کوئی دوسرا traffic66 پہلے سے یہی data directory استعمال کر رہا ہے |
| `receive buffer is only … KB` | Linux UDP buffers کو محدود رکھتا ہے: `net.core.rmem_max=16777216` سیٹ کریں ([Linux](#linux) دیکھیں) |
| `cannot create the data directory` | اس یوزر کے لیے پروگرام فولڈر writable نہیں: `-data` دیں |
| macOS: "cannot be opened" یا "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" (Windows نے آپ کے PC کو محفوظ کیا) | **More info** (مزید معلومات) → **Run anyway** (پھر بھی چلائیں)؛ پروگرام ابھی signed نہیں ہے |
| Windows capture: Npcap نہیں ملا | [Npcap](https://npcap.com) انسٹال کریں |
| `address already in use` | کوئی دوسرا پروگرام وہ port استعمال کر رہا ہے: `-addr` یا `-listen` سے دوسرے ports چنیں |

<a id="17-build-from-source"></a>

## 17. سورس سے بِلڈ کریں

Go 1.24 اور ایک C compiler (gcc یا clang؛ Windows پر MinGW-w64):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```

</div>
