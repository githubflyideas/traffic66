[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | **اردو** | [日本語](README.ja.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66 — NetFlow، sFlow اور IPFIX کلیکٹر اور ٹریفک اینالائزر

sFlow، NetFlow اور IPFIX کے لیے flow analytics، ایک ہی پروگرام میں: bandwidth
کون استعمال کر رہا ہے، ٹریفک کہاں جا رہی ہے، اور کیا اعداد ڈیوائسز کے اپنے
interface counters سے میل کھاتے ہیں — web UI اور terminal UI میں۔

ntopng، ElastiFlow، Grafana کے ساتھ pmacct، یا PRTG اور SolarWinds NTA کے
flow modules کا self-hosted متبادل — نیٹ ورک مانیٹرنگ (network monitoring)،
bandwidth مانیٹرنگ (bandwidth monitoring)، سب سے زیادہ ٹریفک والے hosts
(top talkers)، DDoS اور scan کی نشاندہی، اور pcap تجزیے کے لیے، بغیر
Elasticsearch، Kafka یا الگ database کے۔

- Windows، Linux اور macOS کے لیے ایک ہی executable؛ کوئی database انسٹال نہیں کرنا، offline بھی چلتا ہے۔
- کسی بھی UDP port پر sFlow v5، NetFlow v5/v9 اور IPFIX، یا کسی interface سے local capture۔
- اپنے اعداد کو interface counters (sFlow یا SNMP) سے ملا کر جانچتا ہے اور بتاتا ہے کہ فرق کیوں ہے۔
- scans، پاس ورڈ کا اندازہ، lateral movement، غیر معمولی uploads، floods اور threat list والی ٹریفک ڈھونڈتا ہے، sampling کے باوجود بھی۔
- `traffic66 capture.pcap` بغیر کسی سیٹ اپ کے پیکٹ کیپچر کا تجزیہ کرتا ہے۔
- 13 زبانیں۔ جانچ کے لیے اور 100 سے کم افراد والی تنظیموں کے لیے مفت ([لائسنس](#licence))۔

![جائزہ: کھلی مشتبہ سرگرمیاں، کل اسی وقت کے مقابلے میں application کے حساب سے bandwidth، سرفہرست clients اور services](images/overview.png)

<sub>تمام اسکرین شاٹس `traffic66 demo` سے لیے گئے ہیں، ایک simulated کمپنی نیٹ ورک۔</sub>

<a id="contents"></a>

## فہرست

1. [ڈیمو چلا کر دیکھیں](#1-try-the-demo)
2. [انسٹال کریں](#2-install)
3. [یوزرز اور پاس ورڈ](#3-users-and-passwords)
4. [اپنی ڈیوائسز سے flows بھیجیں](#4-send-flows-from-your-devices)
5. [جانچیں کہ flows پہنچ رہے ہیں](#5-check-that-flows-arrive)
6. [Interfaces اور counters](#6-interfaces-and-counters)
7. [نام، ممالک اور threat lists](#7-names-countries-and-threat-lists)
8. [web UI کا استعمال](#8-using-the-web-ui)
9. [آف لائن pcap، terminal UI، local capture](#9-offline-pcap-terminal-ui-local-capture)
10. [آپشنز اور ڈیٹا](#10-options-and-data)
11. [سکیورٹی، گنجائش کا اندازہ، مسائل کا حل](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. ڈیمو چلا کر دیکھیں

اپنے سسٹم کا archive
[releases صفحے](https://github.com/githubflyideas/traffic66/releases) سے
ڈاؤن لوڈ کریں (Windows x64، kernel 3.2+ والا Linux x86-64/ARM64، macOS 11+)،
اسے unpack کریں اور چلائیں:

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

macOS پر پہلے `xattr -dr com.apple.quarantine <folder>` چلائیں۔
http://127.0.0.1:8066 کھولیں اور `admin` / `try66` سے سائن اِن کریں: ایک دن کی
تاریخ اور چار simulated ڈیوائسز سے live ٹریفک، جس میں ایک حملہ بھی ہے جو
**مشتبہ سرگرمی** پر قدم بہ قدم دکھایا جاتا ہے۔ Ctrl+C اسے روکتا ہے؛ نئے سرے سے
شروع کرنے کے لیے `traffic66-demo` حذف کر دیں۔ اصل installation کے ساتھ ساتھ
چلانے کے لیے: `-addr :8067 -listen ""`۔

<a id="2-install"></a>

## 2. انسٹال کریں

traffic66 ایک ہی فائل ہے۔ Ports: UDP 6343 (sFlow)، 2055 (NetFlow)، 4739
(IPFIX)، TCP 8066 (web UI)؛ ہر UDP port ہر protocol قبول کرتا ہے۔

**Linux** (systemd):

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

**Windows** (PowerShell، Administrator کے طور پر): `C:\traffic66` میں unpack
کریں، `C:\traffic66\traffic66.exe passwd` چلائیں، ports کھولیں، اور اسے boot
پر شروع ہونے کے لیے سیٹ کریں:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`traffic66.exe` پر double-click بھی کام کرتا ہے: یہ web UI کھولتا ہے اور پہلا
پاس ورڈ اپنی window میں دکھاتا ہے۔

**macOS**: `/usr/local/traffic66` میں unpack کریں، quarantine flag ہٹائیں،
`traffic66 passwd -data "/Library/Application Support/traffic66"` چلائیں، اور
اسے ایک LaunchDaemon سے شروع کریں جس کے `ProgramArguments` پروگرام، `-data` اور
وہ directory ہوں، ساتھ `RunAtLoad` اور `KeepAlive`۔

<a id="3-users-and-passwords"></a>

## 3. یوزرز اور پاس ورڈ

پہلی بار شروع ہونے پر traffic66 ایک random پاس ورڈ کے ساتھ یوزر `admin` بناتا
ہے اور اسے ایک بار دکھاتا ہے (window میں، terminal میں، یا
`journalctl -u traffic66 | grep "first start"`)۔ یوزرز data directory میں
`password` کے اندر salted hashes کی صورت میں رکھے جاتے ہیں اور traffic66 مشین
پر ایک command سے سنبھالے جاتے ہیں (اگر traffic66 `-data …` کے ساتھ چلتا ہے تو
وہ بھی شامل کریں):

| کام | Command |
|---|---|
| `admin` کا پاس ورڈ بدلنا | `traffic66 passwd` |
| `alice` کو شامل کرنا یا اس کا پاس ورڈ بدلنا | `traffic66 passwd -user alice` |
| `alice` کو حذف کرنا | `traffic66 passwd -user alice -delete` |
| یوزرز کی فہرست | `traffic66 passwd -list` |

تبدیلیاں فوراً لاگو ہوتی ہیں۔ تمام یوزرز کے اختیارات یکساں ہیں۔ scripts اور
containers کے لیے `TRAFFIC66_PASSWORD=…` (یا `-password`) اس run میں صرف اسی
پاس ورڈ کے ساتھ `-user` کو قبول کرتا ہے۔ ایک منٹ میں پانچ غلط پاس ورڈ اس address
کو ایک منٹ کے لیے block کر دیتے ہیں۔

<a id="4-send-flows-from-your-devices"></a>

## 4. اپنی ڈیوائسز سے flows بھیجیں

`192.0.2.50` traffic66 ہے، `192.0.2.1` ڈیوائس۔ active timeout کو 60 سیکنڈ پر
رکھیں، NetFlow/IPFIX ڈیوائسز کو اپنے sampler options export کرنے دیں، اور
**ہر interface کو inbound** sample کریں (یا صرف edge interfaces): اس طرح ہر
packet ایک ہی بار گنا جاتا ہے۔ sFlow rates: 1 Gb/s کے لیے تقریباً 1:1000،
10 Gb/s کے لیے 1:4096، 40/100 Gb/s کے لیے 1:8192۔

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

## 5. جانچیں کہ flows پہنچ رہے ہیں

**ترتیبات** ہر اس ڈیوائس کو چند سیکنڈ میں دکھاتی ہیں جو کچھ بھی بھیجتی ہے:
protocol، sampling rate، loss، وہ interfaces جنہیں وہ sample کرتی ہے، اور status
سبز نہ ہو تو کیا ٹھیک کرنا ہے۔ sFlow کا loss دو حصوں میں بٹا ہوتا ہے: راستے میں
ضائع (اگر `netstat -su` buffer errors دکھائے تو `net.core.rmem_max` بڑھائیں)
اور وہ samples جو ڈیوائس نے خود گرا دیے۔

![ترتیبات: ہر ڈیوائس، اس کا protocol، sampling، loss اور کیا ٹھیک کرنا ہے](images/sources.png)

کوئی ڈیوائس غائب ہے؟ `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739` چلائیں: وہاں کچھ نہ ہو تو مسئلہ routing، کسی firewall یا
ڈیوائس کی configuration میں ہے؛ packets وہاں ہوں مگر **ترتیبات** میں کچھ نہ ہو
تو local firewall یا `-listen`۔ کسی دوسری مشین سے
`traffic66 simulate -to 192.0.2.50` simulated ڈیوائسز کے ساتھ راستہ جانچتا ہے۔

<a id="6-interfaces-and-counters"></a>

## 6. Interfaces اور counters

Flow کے اعداد اندازے ہیں (samples × sampling rate)۔ **انٹرفیس جانچ** انہیں
ڈیوائس کے interface counters سے ملاتی ہے (sFlow counters، یا نام میں `snmp`
لائن کے ذریعے SNMP) اور بتاتی ہے کہ فرق کیوں ہے: interfaces sample نہیں ہو رہے،
ایک ہی ٹریفک دو بار sample ہوئی، راستے میں loss، یا sampling rate نامعلوم۔ ہر
interface کا ایک bits/s اور ایک packets/s chart ہے، ingress سبز اور egress
نیلا، counters ڈیشڈ لکیروں میں۔

ہر row پر **✎** ایک نام اور ایک مختصر ٹیگ (جیسے *uplink*) دیتا ہے اور **☆** اسے
ڈیفالٹ interface (★) بناتا ہے، جس پر صفحات کھلتے ہیں۔

جو ڈیوائس صرف کچھ interfaces sample کرتی ہے وہ ان flows کے دوسرے سرے بھی دکھاتی
ہے۔ یہ **مخالف سرے کے انٹرفیس** آخر میں چھوٹے سرمئی حروف میں دکھائے جاتے ہیں: ان
میں صرف وہ ٹریفک ہوتی ہے جو سیمپل والے interface سے گزری۔ سیمپل والا interface
sFlow کے data source یا flowDirection field (IPFIX 61) سے معلوم ہوتا ہے؛ اس کے
بغیر، وہ interface جو ڈیوائس کی 90% ٹریفک میں ہو۔

![انٹرفیس جانچ: ہر interface کی ٹریفک، اور ڈیوائس کے counter کے ساتھ flow کا اندازہ](images/interfaces.png)

traffic66 پہلے ہی وہ rate استعمال کرتا ہے جو ڈیوائس نے لاگو کیا، نامعلوم rates کا
انتظار کرتا ہے، export loss کی تلافی کرتا ہے، لمبے flows کو ان کے منٹوں پر بانٹتا
ہے اور NetFlow/IPFIX میں ہر packet پر 18 bytes کا Ethernet overhead جوڑتا ہے
(`-l2-overhead`)۔

<a id="7-names-countries-and-threat-lists"></a>

## 7. نام، ممالک اور threat lists

کسی بھی address پر کلک کر کے **نام دیں…** چنیں، یا **ترتیبات → نام** استعمال
کریں۔ نام data directory میں `inventory.txt` میں رکھے جاتے ہیں:

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

private ranges ہمیشہ آپ کی ہوتی ہیں۔ تبدیلیاں **محفوظ کریں** پر لاگو ہوتی ہیں،
restart کی ضرورت نہیں۔

ممالک اور نیٹ ورکس (AS) شروع سے DB-IP کے مفت Lite ڈیٹابیس کے ساتھ کام کرتے ہیں
([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)، "IP
Geolocation by DB-IP"، [db-ip.com](https://db-ip.com))؛ **ترتیبات** انہیں اپ
ڈیٹ کرتی ہیں، یا ان کی جگہ MaxMind GeoLite2، IPinfo Lite یا IPtoASN فائلیں لیتی
ہیں۔ نقشے کی سرحدیں: [Natural Earth](https://www.naturalearthdata.com)۔

![مقامات اور نیٹ ورک: دنیا کے نقشے پر ملک کے لحاظ سے بیرونی ٹریفک](images/geo.png)

Threat lists متن کی فائلیں ہیں، ہر لائن میں ایک address یا نیٹ ورک،
`<data>/threats/<name>.txt` میں (مثلاً Spamhaus DROP)؛ انہیں بدلنے کے بعد restart
کریں۔ matches **خطرے کی معلومات** پر نظر آتے ہیں۔

![خطرے کی معلومات: ایک internal host جو threat list کے ایک address کو ڈیٹا بھیج رہا ہے](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. web UI کا استعمال

ہر صفحے کی ہر value پر کلک کیا جا سکتا ہے: **صرف یہ دکھائیں** / **اسے خارج
کریں** (filters ہر صفحے پر لاگو ہوتے ہیں)، **اس کے فلو ریکارڈ دکھائیں**،
**تفصیل دیکھیں** (ایک host یا service کے بارے میں صفحہ)، **نام دیں…**، **آن
لائن معلوم کریں**، **کاپی کریں**۔

| صفحہ | کیا دکھاتا ہے |
|---|---|
| جائزہ | منتخب interface کی bandwidth؛ کل یا پچھلے ہفتے کے مقابلے میں application کے حساب سے ٹریفک (کل، اندر آنے والی یا باہر جانے والی)؛ کھلی مشتبہ سرگرمیاں؛ سرفہرست clients اور services |
| ٹاپ 66 | سرفہرست 66 conversations، کسی بھی column سے sort ہونے والی، یا application، نیٹ ورک، segment، ڈیوائس، encapsulation، VLAN کے لحاظ سے گروپ؛ سب سے فعال 30 hosts |
| ٹریفک کی تفصیل | Ring charts: servers اور ان کے clients (یا الٹا)، اور services |
| ٹریفک کے راستے | host ← application ← ملک، یا client ← service ← server، یا نیٹ ورک کے لحاظ سے |
| انٹرفیس جانچ | وقت کے ساتھ ہر interface اپنے counters کے مقابلے میں؛ نام، ٹیگ، ڈیفالٹ |
| فلو ریکارڈ | انفرادی flows، ہر 5 سیکنڈ میں لائیو یا کسی بھی وقت کی حد کے لیے |
| مشتبہ سرگرمی، خطرے کی معلومات | کس چیز پر توجہ چاہیے ([نیچے](#findings))؛ فہرست والے addresses کے ساتھ ٹریفک |
| مقامات اور نیٹ ورک | ملک کے لحاظ سے دنیا کا نقشہ، وقت کے ساتھ نیٹ ورکس (AS) |
| ترتیبات | ڈیوائسز، sampling، loss، SNMP، ڈیٹابیس، لوگو، نام |
| آف لائن pcap تجزیہ، ڈیٹا کی صفائی | کیپچر فائلیں ([نیچے](#9-offline-pcap-terminal-ui-local-capture))؛ پرانا ڈیٹا حذف کرنا |

صفحات کے اوپر: **انٹرفیس** (سب، یا ایک سیمپل والا interface؛ تب ٹریفک کے صفحات
صرف اس سے گزرنے والی ٹریفک دکھاتے ہیں)، وقت کی حد (15 منٹ سے 30 دن، یا اپنی
مرضی)، ہر 30 s پر refresh اور بالکل موجودہ view کے لیے **لنک کاپی کریں**۔ زبان
اور رنگوں کی پانچ تھیمیں menu کے نیچے ہیں۔ Charts وہاں ختم ہوتے ہیں جہاں ڈیٹا
مکمل ہو: NetFlow/IPFIX کے ساتھ اتنی دیر پہلے جتنی دیر سے ڈیوائسز export کرتی
ہیں (زیادہ سے زیادہ 2 منٹ)۔ 6 گھنٹے سے لمبی حدیں پورے گھنٹے سے شروع ہوتی ہیں؛
7 یا 30 دن کے لیے ایک interface flow detail پڑھتا ہے، اس لیے سست ہوتا ہے اور
صرف اتنا پیچھے جاتا ہے جتنی دیر detail رکھی جاتی ہے۔

![ٹاپ 66: سرفہرست 66 conversations، کسی بھی column کے لحاظ سے ترتیب دی گئیں](images/topn.png)

![ٹریفک کی تفصیل: servers اپنے clients کے ساتھ، اور services اپنے servers کے ساتھ، ring charts کی صورت میں](images/traffic.png)

![ایک host کی تفصیل: اس کے بارے میں مشتبہ سرگرمیاں، اس کی ٹریفک، وہ کس سے بات کرتا ہے، services، ممالک اور تازہ ترین flows](images/detail.png)

![ٹریفک کے راستے: کون سا host کس ملک کی طرف کون سی application استعمال کرتا ہے](images/paths.png)

![چینی زبان میں جائزہ](images/overview-zh.png)

<a id="findings"></a>

### مشتبہ سرگرمی

ہر 5 منٹ پر پچھلے 10 منٹ جانچے جاتے ہیں؛ جو چیز ایک گھنٹہ چلے وہ ایک ہی اندراج
ہے جو بڑھتا رہتا ہے۔

| سرگرمی | اس کا مطلب |
|---|---|
| اسکین، پورٹ اسکین | ایک port پر بہت سے hosts کو، یا ایک host کے بہت سے ports کو چھوٹے probes |
| پاس ورڈ کا اندازہ | کسی login service سے بہت سے مختصر connections |
| لیٹرل موومنٹ | ایسے internal hosts سے file sharing یا remote administration جنہوں نے پہلے کبھی یہ پیش نہیں کیا |
| غیر معمولی اپ لوڈ | ایک نئے address کو 10 منٹ میں 100 MB، واپس آنے والے کا تین گنا |
| فلڈ | ایک address پر فی سیکنڈ 20,000+ چھوٹے packets، اس کی معمول کی رفتار کا دس گنا |
| خطرے کی فہرست | فہرست والے address کے ساتھ ٹریفک |

آپ کے نیٹ ورک کے اندر سے ہوں تو یہ زیادہ شدت کی ہوتی ہیں، انٹرنیٹ سے کم۔
**نمٹا دیا** اندراج بند کرتا ہے، **مسئلہ نہیں** اسے ہمیشہ کے لیے خاموش کر دیتا
ہے۔ لیٹرل موومنٹ اور اپ لوڈ کے لیے ایک دن کی تاریخ چاہیے۔ 1:4096 sampling کے
باوجود ڈیمو کا حملہ پورا پکڑا جاتا ہے؛ بہت چھوٹے scans sampling کے پیچھے چھپ
سکتے ہیں۔

![مشتبہ سرگرمی: ایک حملے کا ہر مرحلہ، 1:4096 sFlow sampling کے باوجود پکڑا گیا](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. آف لائن pcap، terminal UI، local capture

**آف لائن pcap تجزیہ** پیکٹ کیپچر (pcap، pcapng) کو انہی صفحات پر دکھاتا ہے، لائیو
ڈیٹا سے الگ: `traffic66 a.pcap b.pcapng` 127.0.0.1 پر شروع ہوتا ہے اور browser
کھولتا ہے (زیادہ سے زیادہ 3 فائلیں، 3 GB؛ Ctrl+C امپورٹ شدہ ڈیٹا حذف کر دیتا
ہے)، یا اسی صفحے پر 50 MB تک کی 3 فائلیں اپ لوڈ کریں۔ یہ flows پر کام کرتا ہے،
پیکٹ کے مواد پر نہیں۔

![آف لائن تجزیہ: کیپچر فائلیں، ان کے پیکٹ، فلو اور وقت](images/sandbox.png)

**Terminal UI**: traffic66 مشین پر `traffic66 tui`، یا
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`۔
Keys: 1–8 صفحات، Enter actions، f صرف یہ دکھائیں، x خارج کریں، t وقت کی حد، w
browser میں کھولیں، q باہر نکلیں؛ `-lang` زبان چنتا ہے۔

![Terminal UI: جائزہ](images/tui-overview.png)

![Terminal UI: ٹاپ 66 conversations](images/tui-topn.png)

**Local capture** ایک local interface سے flows بناتا ہے، بہترین ہے کہ وہ کسی
switch کے mirror port سے جڑا port ہو: `traffic66 interfaces` ان کی فہرست دیتا
ہے، `-capture eth1` (یا Windows کا نام یا نمبر) capture کرتا ہے۔ Linux کو root یا
`setcap cap_net_raw,cap_net_admin+ep` چاہیے، macOS کو root، Windows کو
[Npcap](https://npcap.com)۔ capture شدہ flows ڈیوائس `127.0.0.1` سے آتے ہیں۔

<a id="10-options-and-data"></a>

## 10. آپشنز اور ڈیٹا

`traffic66 -h` سب کچھ دکھاتا ہے۔ سب سے زیادہ استعمال ہونے والے:

| آپشن | ڈیفالٹ | |
|---|---|---|
| `-data` | پروگرام کے ساتھ `traffic66-data` | data directory |
| `-addr` | `:8066` | web UI؛ صرف اس مشین کے لیے `127.0.0.1:8066` |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors؛ خالی ہو تو بند |
| `-retention-days` | `30` | flow detail کے دن؛ summaries 400 دن رکھی جاتی ہیں |
| `-memory` | `0.10` | database cache کے لیے RAM کا حصہ |
| `-sampling-wait` | `5m` | records کتنی دیر sampling rate کا انتظار کریں |
| `-capture` | | local interface (بار بار دیا جا سکتا ہے) |
| `-no-dns` | | کوئی reverse lookup نہیں |

data directory میں `raw/` (detail، ہر گھنٹے کی ایک فائل)، `traffic66.duckdb`
(summaries اور counters)، `password`، `inventory.txt`، `license.json`، آپ کا
لوگو اور ڈیٹابیس ہوتے ہیں۔ بیک اپ کے لیے traffic66 روک کر اسے کاپی کریں؛ اپ گریڈ
کے لیے پروگرام فائل بدل دیں۔ **ڈیٹا کی صفائی** 7–120 دن سے پرانا ڈیٹا، یا سارا
ڈیٹا حذف کرتی ہے۔

<a id="licence"></a>

### لائسنس

[PolyForm Noncommercial License 1.0.0](../LICENSE.md) اور
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) کے تحت source
available (انگریزی متن لازم ہے): جانچ کے لیے اور 100 سے کم افراد والی تنظیموں کے
لیے مفت؛ بڑی تنظیمیں 30 دن کے production استعمال کے بعد رجسٹر کرتی ہیں؛ بیچنے،
دوسروں کے لیے host کرنے یا مقابل مصنوعات کے لیے تجارتی لائسنس چاہیے۔ کچھ بھی کبھی
بند نہیں کیا جاتا۔ ہر صفحے کے نیچے 8 ہندسوں کا installation نمبر دکھتا ہے؛ اسے
مصنف کو بھیجیں، اور جو `license.json` واپس ملے اسے data directory میں رکھیں۔
رابطہ: <https://github.com/githubflyideas/traffic66>۔

<a id="11-security-sizing-troubleshooting"></a>

## 11. سکیورٹی، گنجائش کا اندازہ، مسائل کا حل

web UI سادہ HTTP ہے: غیر بھروسہ مند نیٹ ورکس پر `-addr 127.0.0.1:8066` استعمال
کریں، کسی TLS proxy کے پیچھے (`caddy reverse-proxy --from
traffic66.example.com --to 127.0.0.1:8066`) یا SSH tunnel کے ذریعے۔ UDP ports
کی اجازت صرف اپنی ڈیوائسز سے دیں۔ SNMP communities سادہ متن (clear text) میں
رکھی جاتی ہیں؛ read-only communities استعمال کریں۔

2 cores پر 5,000 flows/s پر: detail کے ہر دن کے لیے تقریباً 12 GB disk (30 دن
کے لیے 360 GB)، ایک core کا چھٹا حصہ، 0.6–0.8 GB memory۔ لمبی حدوں کے overviews
0.2 s سے کم لیتے ہیں؛ تمام conversations کا 1 گھنٹے کا Top 66 تقریباً 9 s۔

| علامت | حل |
|---|---|
| "waiting for the sampling rate" | sampler options export کریں، یا device لائن میں `sampling=N` / `unsampled` |
| counters سے کم | interfaces sample نہیں ہو رہے، loss، یا active timeout 60 s سے زیادہ |
| counters سے زیادہ | ایک ہی ٹریفک دو interfaces یا ڈیوائسز پر sample ہو رہی ہے |
| پاس ورڈ بھول گئے | traffic66 مشین پر `traffic66 passwd` |
| `Conflicting lock is held` | کوئی دوسرا traffic66 یہ data directory استعمال کر رہا ہے |
| `address already in use` | `-addr` یا `-listen` سے دوسرے ports چنیں |
| Windows "protected your PC" | **More info** (مزید معلومات) → **Run anyway** (پھر بھی چلائیں) |

سورس سے بِلڈ: Go 1.24 اور ایک C compiler، پھر `scripts/build.sh 0.1.0 traffic66`۔

</div>
