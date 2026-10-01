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
- Top 66 فہرستیں، flow کے راستے، ممالک اور نیٹ ورک، threat list کے matches،
  flow records، encapsulation (GRE، IPIP، VXLAN، GENEVE، MPLS)۔
- web UI اور terminal UI میں 13 زبانیں۔

<a id="contents"></a>

## فہرست

1. [ڈیمو چلا کر دیکھیں](#1-try-the-demo)
2. [انسٹال کریں](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [سائن اِن اور پاس ورڈ](#3-sign-in-and-passwords)
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
ڈیوائسز سے live ٹریفک ہوتی ہے، اور ڈھونڈنے کے لیے دو incidents بھی ہیں:
**جائزہ** سے شروع کریں، **کس میں اضافہ ہوا** دیکھیں، اور وہاں سے کلک کرتے ہوئے
آگے بڑھیں۔ Ctrl+C سے بند کریں۔ ڈیمو کا ڈیٹا پروگرام کے ساتھ `traffic66-demo`
میں رہتا ہے؛ ڈیمو نئے سرے سے شروع کرنے کے لیے وہ فولڈر ڈیلیٹ کر دیں۔

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

<a id="3-sign-in-and-passwords"></a>

## 3. سائن اِن اور پاس ورڈ

`http://<traffic66 machine>:8066` کھولیں اور سائن اِن کریں۔ یوزر `admin` ہے،
جب تک آپ نے کوئی اور نہ چنا ہو۔

- اگر پہلی بار چلانے سے پہلے آپ نے پاس ورڈ سیٹ نہیں کیا تو traffic66 خود ایک
  بناتا ہے اور اسے ایک بار اپنے log میں لکھتا ہے:
  `first start: sign in as user "admin" with password "…"`۔
  Linux پر اسے `journalctl -u traffic66 | grep "first start"` سے ڈھونڈیں۔
- پاس ورڈ hash کر کے data directory کی فائل `password` میں رکھا جاتا ہے۔
  restart کے بعد بھی وہی رہتا ہے۔
- اسے بدلنے کے لیے، یا بھول جانے پر نیا سیٹ کرنے کے لیے، traffic66 مشین پر:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` ایک random پاس ورڈ بنا کر دکھاتا ہے۔ چلتا ہوا
  traffic66 اگلے سائن اِن پر نیا پاس ورڈ قبول کر لیتا ہے؛ restart کی ضرورت نہیں۔
- مزید یوزرز: `traffic66 passwd -data <data directory> -user alice`۔ سب
  یوزرز ایک جیسا ہی دیکھتے ہیں۔
- scripts اور containers کے لیے، environment میں `TRAFFIC66_PASSWORD=…` یا
  command line پر `-password …` اس run کے لیے محفوظ شدہ پاس ورڈ کی جگہ پاس ورڈ
  سیٹ کرتا ہے۔ environment کو ترجیح دیں: command lines مشین کے دوسرے یوزرز کو
  نظر آتی ہیں۔

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
وجہ بھی بتاتا ہے۔

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

web UI میں **ذرائع → نام** ہر لائن میں ایک entry لیتا ہے۔ یہ data directory میں
`inventory.txt` کے طور پر محفوظ ہوتا ہے، اس لیے آپ وہ فائل براہ راست بھی edit کر
سکتے ہیں (`inventory.txt.example` دیکھیں)۔ ہر لائن اختیاری ہے۔

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

- `net`: private ranges (10/8، 172.16/12، 192.168/16، 100.64/10) ہمیشہ آپ کی
  مانی جاتی ہیں۔ اپنی public ranges شامل کریں تاکہ ان سے آنے جانے والی ٹریفک بھی
  آپ کی گنی جائے؛ نام **ٹاپ N → سیگمنٹس** میں اور flow کے راستوں میں نظر آتا ہے۔
- `snmp <device> <community> [<management address>[:port]]`: device وہ address
  ہے جہاں سے flows آتے ہیں۔ جب ڈیوائس SNMP کا جواب کسی دوسرے address پر دیتی ہو
  تو management address شامل کریں۔ SNMP سے پڑھی گئی interface descriptions بطور
  نام استعمال ہوتی ہیں، جب تک آپ `iface` سے interface کا نام نہ دیں۔ ڈیوائس کی
  SNMP access list میں traffic66 مشین کو اجازت دیں۔
- تبدیلیاں **محفوظ کریں** پر کلک کرتے ہی لاگو ہو جاتی ہیں؛ restart کی ضرورت نہیں۔

<a id="8-countries-networks-and-threat-lists"></a>

## 8. ممالک، نیٹ ورک اور threat lists

ممالک اور نیٹ ورک (AS) کے ناموں کے لیے IP-to-ASN table درکار ہے۔
[iptoasn.com](https://iptoasn.com) سے مفت table ڈاؤن لوڈ کریں:

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

اسی format کی کوئی بھی فائل چلے گی (tab سے الگ: پہلا address، آخری address، AS
number، country code، AS name؛ plain یا gzip)۔ اسے بدلنے کے بعد traffic66
restart کریں؛ تقریباً ہر مہینے نئی فائل ڈاؤن لوڈ کریں۔

Threat lists سادہ text فائلیں ہیں، ہر لائن میں ایک address یا نیٹ ورک (`#` یا
`;` کے بعد کا متن نظرانداز ہوتا ہے)، جو
`<data directory>/threats/<name>.txt` کے طور پر محفوظ کی جاتی ہیں، مثلاً:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

lists شامل کرنے یا بدلنے کے بعد traffic66 restart کریں۔ matches
**خطرے کی معلومات** پر list کے نام کے حساب سے نظر آتے ہیں۔

<a id="9-using-the-web-ui"></a>

## 9. web UI کا استعمال

آپ کو شاید ہی کچھ ٹائپ کرنا پڑے۔ ہر صفحے کی ہر value — address، port،
application، ملک، ڈیوائس — پر کلک کیا جا سکتا ہے:

- **صرف یہ دکھائیں** / **اسے خارج کریں** ایک filter شامل کرتا ہے۔ filters اوپر
  والی bar کے نیچے نظر آتے ہیں اور ہٹائے جانے تک ہر صفحے پر لاگو رہتے ہیں۔
- **اس کے فلو ریکارڈ دکھائیں** متعلقہ انفرادی flows کھولتا ہے۔
- **آن لائن معلوم کریں** address یا AS کو کسی public lookup سائٹ پر کھولتا ہے۔
- **کاپی کریں** value کاپی کرتا ہے۔

صفحات:

| صفحہ | کس سوال کا جواب دیتا ہے |
|---|---|
| جائزہ | ابھی کتنی ٹریفک ہے اور پچھلے ہفتے کے مقابلے میں کتنی، application کے حساب سے؛ کیا بڑھا؛ سرفہرست clients اور services |
| ٹاپ N | clients، servers، conversations، applications، ports، ممالک، نیٹ ورکس، segments، ڈیوائسز، encapsulation یا VLAN کے top 66 |
| ٹریفک کے راستے | کون سا segment کس ملک میں کس application سے بات کرتا ہے |
| مقامات اور نیٹ ورک | ملک کے حساب سے اور نیٹ ورک (AS) کے حساب سے ٹریفک |
| خطرے کی معلومات | وہ hosts جنہوں نے آپ کی threat lists کے addresses سے بات کی، اور کتنا بھیجا |
| فلو ریکارڈ | انفرادی flows، نئے پہلے، منتخب کیے جا سکنے والے columns کے ساتھ |
| انٹرفیس جانچ | interface counters کے ساتھ flow کے اعداد، بدترین پہلے، وجوہات کے ساتھ |
| ذرائع | ڈیوائسز، sampling، loss، collectors، SNMP، اور **نام** |

صفحات کے اوپر: time range (15 منٹ سے 30 دن)، ایک اختیاری search box، ہر 30
سیکنڈ پر automatic refresh، اور **لنک کاپی کریں**، جو بالکل موجودہ view (صفحہ،
time range اور filters) کا link کاپی کرتا ہے تاکہ آپ اسے کسی ساتھی کو بھیج
سکیں۔ زبان browser کے مطابق ہوتی ہے؛ menu کے نیچے سے بدلیں۔

لمبی time ranges پر ٹاپ N گھنٹہ وار summaries سے آتا ہے؛ وہاں filters دستیاب
نہیں، اور صفحہ یہ بتا دیتا ہے۔ filter کرنے کے لیے چھوٹی range چنیں۔

<a id="10-terminal-ui"></a>

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

<a id="11-local-capture"></a>

## 11. Local capture

Flow exports کے علاوہ traffic66 کسی local network interface، مثلاً mirror
(SPAN) port، کے packets سے خود بھی flows بنا سکتا ہے:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: root درکار ہے، یا capabilities `CAP_NET_RAW` اور `CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`، یا
  اوپر والی systemd unit میں `AmbientCapabilities` والی لائن)۔
- macOS: root درکار ہے (BPF devices)؛ کچھ انسٹال نہیں کرنا۔
- Windows: پہلے [Npcap](https://npcap.com) انسٹال کریں۔

Capture ہونے والے interfaces **ذرائع** پر نظر آتے ہیں۔ دو بار دیکھے گئے packets
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
| `traffic66 passwd` | login پاس ورڈ سیٹ کرتا ہے |
| `traffic66 simulate -to HOST` | کسی collector کو simulated exports بھیجتا ہے |
| `traffic66 interfaces` | local capture کے لیے interfaces کی فہرست |
| `traffic66 version` | version دکھاتا ہے |

`traffic66` اور `traffic66 demo` کے آپشنز:

| آپشن | ڈیفالٹ | |
|---|---|---|
| `-addr` | `:8066` | web UI کا address؛ صرف اسی مشین کے لیے `127.0.0.1:8066` |
| `-data` | پروگرام کے ساتھ `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors، `name=address` کی شکل میں، comma سے الگ؛ خالی ہو تو بند |
| `-user` | `admin` | پہلے generated پاس ورڈ اور `-password` کے لیے یوزر |
| `-password` | محفوظ شدہ پاس ورڈ | صرف اس run کے لیے پاس ورڈ (`TRAFFIC66_PASSWORD` بھی) |
| `-retention-days` | `30` | flow detail کتنے دن رکھی جائے؛ summaries 400 دن رکھی جاتی ہیں |
| `-memory` | `0.10` | physical memory کا کتنا حصہ database استعمال کر سکتا ہے |
| `-l2-overhead` | `18` | NetFlow/IPFIX byte counts میں ہر packet پر جوڑے جانے والے bytes |
| `-sampling-wait` | `5m` | records کتنی دیر sampling rate کا انتظار کریں |
| `-capture` | | local interface پر capture (دہرایا جا سکتا ہے) |
| `-inventory` | `<data>/inventory.txt` | ناموں کی فائل |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table |
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
| `asn.tsv.gz`، `threats/` | آپ کی شامل کردہ lookup tables |

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
disk لیتی ہے، ساتھ موجودہ گھنٹے کے لیے تقریباً 1.5 GB؛ پروگرام تقریباً 0.5 GB
memory اور ایک core کا چھٹا حصہ لیتا ہے۔ لمبی time ranges کے overviews
summaries سے آتے ہیں اور 0.2 s سے کم لیتے ہیں۔ detail پر queries فی گھنٹہ
تقریباً 2 کروڑ 20 لاکھ rows scan کرتی ہیں: 1 گھنٹے میں ایک host کے لیے 1 s سے کم،
تمام conversations کا 1 گھنٹے کا Top 66 تقریباً 9 s؛ وقت range کے ساتھ بڑھتا ہے
اور زیادہ cores کے ساتھ کم ہوتا ہے۔

لہٰذا 5,000 flows/s پر 30 دن کے لیے disk تقریباً 360 GB ہے؛ اسے اپنے flow rate
(**ذرائع** پر نظر آتا ہے) اور `-retention-days` کے تناسب سے بڑھائیں یا گھٹائیں۔

<a id="16-troubleshooting"></a>

## 16. مسائل کا حل

| علامت | وجہ اور حل |
|---|---|
| ڈیوائس **ذرائع** میں نہیں | packets نہیں پہنچ رہے: [جانچیں کہ flows پہنچ رہے ہیں](#5-check-that-flows-arrive) دیکھیں |
| "waiting for the sampling rate" | ڈیوائس نے ابھی تک اپنے sampler options نہیں بھیجے؛ زیادہ تر چند منٹ میں دوبارہ بھیج دیتی ہیں۔ اگر کبھی نہ بھیجے تو انہیں export کروائیں (Cisco پر `option sampler-table`) یا اگر وہ واقعی 1:1 ہے تو نام میں اسے `unsampled` لکھیں |
| اعداد interface counters سے کم | **انٹرفیس جانچ** دیکھیں: راستے میں loss، interfaces sample نہیں ہو رہے، یا flows ابھی ڈیوائس کی cache میں ہیں (active timeout 60 s سے لمبا) |
| اعداد interface counters سے زیادہ | ایک ہی ٹریفک دو interfaces یا دو ڈیوائسز پر sample ہو رہی ہے |
| کوئی ملک یا نیٹ ورک نہیں | IP-to-ASN table نہیں ہے: [ممالک](#8-countries-networks-and-threat-lists) دیکھیں |
| پاس ورڈ بھول گئے | traffic66 مشین پر `traffic66 passwd -data <data directory>` |
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
