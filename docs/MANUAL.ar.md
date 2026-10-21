[English](MANUAL.md) | [中文](MANUAL.zh.md) | [हिन्दी](MANUAL.hi.md) | [Español](MANUAL.es.md) | **العربية** | [Français](MANUAL.fr.md) | [বাংলা](MANUAL.bn.md) | [Português](MANUAL.pt.md) | [Русский](MANUAL.ru.md) | [Bahasa Indonesia](MANUAL.id.md) | [اردو](MANUAL.ur.md) | [Deutsch](MANUAL.de.md) | [日本語](MANUAL.ja.md) | [Tiếng Việt](MANUAL.vi.md) | [한국어](MANUAL.ko.md)

<div dir="rtl">

# traffic66 — مُجمِّع ومحلّل حركة NetFlow وsFlow وIPFIX

تحليل تدفقات sFlow وNetFlow وIPFIX في برنامج واحد: من يستهلك عرض النطاق،
وإلى أين تذهب الحركة، وهل تتطابق الأرقام مع عدّادات الواجهات في الأجهزة
نفسها، في واجهة ويب وواجهة طرفية.

بديل ذاتي الاستضافة لـ ntopng وElastiFlow وpmacct مع Grafana، أو لوحدات
تحليل التدفقات في PRTG وSolarWinds NTA، لمراقبة الشبكة (network monitoring)
ومراقبة عرض النطاق (bandwidth monitoring) وأكثر المضيفين استهلاكًا
(top talkers) واكتشاف DDoS والمسح وتحليل ملفات pcap، دون Elasticsearch أو
Kafka أو قاعدة بيانات منفصلة.

- ملف تنفيذي واحد لأنظمة Windows وLinux وmacOS؛ لا قاعدة بيانات تحتاج إلى تثبيت، ويعمل دون اتصال بالإنترنت.
- sFlow v5 وNetFlow v5/v9 وIPFIX على أي منفذ UDP، أو التقاط محلي من واجهة.
- يقارن أرقامه بعدّادات الواجهات (sFlow أو SNMP) ويشرح سبب الاختلاف.
- يكتشف عمليات المسح وتخمين كلمات المرور والتحرك الجانبي وعمليات الرفع غير المعتادة والإغراق وحركة قوائم التهديدات، حتى عبر أخذ العينات.
- يحلّل `traffic66 capture.pcap` ملفات التقاط الحزم دون أي إعداد.
- 15 لغة. مجاني للتقييم وللمؤسسات التي يقل عدد أفرادها عن 100 شخص ([الترخيص](#licence)).

![نظرة عامة: الاكتشافات المفتوحة، واستهلاك عرض النطاق حسب التطبيق مقارنةً بالوقت نفسه أمس، وأبرز العملاء والخدمات](images/overview.png)

<sub>جميع لقطات الشاشة مأخوذة من `traffic66 demo`، وهي شبكة شركة محاكاة.</sub>

<a id="contents"></a>

## المحتويات

1. [بدء سريع](#1-quick-start)
2. [أمثلة](#2-examples)
3. [المستخدمون وكلمات المرور](#3-users-and-passwords)
4. [إرسال التدفقات من أجهزتك](#4-send-flows-from-your-devices)
5. [التحقق من وصول التدفقات](#5-check-that-flows-arrive)
6. [الواجهات والعدّادات](#6-interfaces-and-counters)
7. [الأسماء والدول وقوائم التهديدات](#7-names-countries-and-threat-lists)
8. [استخدام واجهة الويب](#8-using-the-web-ui)
9. [تحليل pcap دون اتصال، الواجهة الطرفية، الالتقاط المحلي](#9-offline-pcap-terminal-ui-local-capture)
10. [الخيارات والبيانات](#10-options-and-data)
11. [الأمان، تقدير الموارد، استكشاف الأخطاء](#11-security-sizing-troubleshooting)

<a id="1-quick-start"></a>

<a id="1-try-the-demo"></a>

## 1. بدء سريع

1. نزّل أرشيف نظامك من: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. فك الضغط وشغّل البرنامج:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

ويندوز: فك ضغط `traffic66-windows-amd64.zip` وشغّل `traffic66.exe` (يكفي النقر المزدوج).

macOS: شغّل أولًا `xattr -dr com.apple.quarantine .` داخل المجلد.

ثم:

- واجهة الويب: `http://<IP الخادم>:8066`، وتستمع على كل واجهات الشبكة.
- تسجيل الدخول: المستخدم `admin` وكلمة المرور `traffic66`. يُطلب تعيين كلمة مرور جديدة عند أول دخول.
- وجّه أجهزتك إلى IP الخادم: sFlow على UDP 6343 وNetFlow v5/v9 على UDP 2055 وIPFIX على UDP 4739 (كل الواجهات؛ كل منفذ يقبل كل البروتوكولات).

<a id="2-examples"></a>

<a id="2-install"></a>

## 2. أمثلة

<a id="command-cheat-sheet"></a>

### ملخص الأوامر

| المهمة | الأمر |
|---|---|
| جمع sFlow على 6343 وNetFlow v5/v9 على 2055 وIPFIX على 4739 (افتراضي) | `./traffic66` |
| sFlow فقط، UDP 6343 | `./traffic66 -listen sflow=:6343` |
| NetFlow v5/v9 على 2055 وIPFIX على 4739 فقط | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| منافذ UDP أخرى | `./traffic66 -listen netflow=:9995,sflow=:6344` |
| المجمّعات على عنوان IP واحد | `./traffic66 -listen sflow=10.0.0.5:6343,netflow=10.0.0.5:2055,ipfix=10.0.0.5:4739` |
| واجهة الويب على IP ومنفذ محددين | `./traffic66 -addr 10.0.0.5:8080` |
| واجهة الويب على هذا الجهاز فقط | `./traffic66 -addr 127.0.0.1:8066` |
| واجهة الويب عبر HTTPS | `./traffic66 -tls` |
| مجلد البيانات | `./traffic66 -data /var/lib/traffic66` |
| الاحتفاظ بتفاصيل 7 أيام | `./traffic66 -retention-days 7` |
| الذاكرة: 20% من RAM لقاعدة البيانات | `./traffic66 -memory 0.20` |
| سرد بطاقات الشبكة للالتقاط المحلي | `./traffic66 interfaces` |
| التقاط بطاقة هذا الجهاز (لينكس) | `sudo ./traffic66 -capture eth0 -listen ""` |
| التقاط بطاقة هذا الجهاز (macOS) | `sudo ./traffic66 -capture en0 -listen ""` |
| التقاط بطاقة هذا الجهاز (ويندوز، يتطلب Npcap) | `.\traffic66.exe -capture Wi-Fi -listen ""` |
| تعيين كلمة مرور admin أو إعادة تعيينها (أضف ‎-data DIR إن استُخدم) | `./traffic66 passwd` |
| كلمة مرور عشوائية لـ admin تُطبع | `./traffic66 passwd -generate` |
| إضافة مستخدم أو تغيير كلمة مروره | `./traffic66 passwd -user alice` |
| سرد المستخدمين | `./traffic66 passwd -list` |
| كلمة مرور لهذا التشغيل فقط (سكربتات، حاويات) | `TRAFFIC66_PASSWORD='S3cret-pass' ./traffic66` |
| الواجهة الطرفية على جهاز traffic66 | `./traffic66 tui` |
| الواجهة الطرفية إلى خادم بعيد | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |
| لغة الواجهة الطرفية | `./traffic66 tui -lang ja` |
| تحليل ملف التقاط | `./traffic66 capture.pcap` |
| إرسال تدفقات اختبار إلى مجمّع | `./traffic66 simulate -to 192.0.2.50` |
| عرض توضيحي بأجهزة محاكاة | `./traffic66 demo` |
| كل الخيارات | `./traffic66 serve -h` |

لغة واجهة الويب: أسفل القائمة (لا يوجد خيار). البقية: `traffic66 serve -h`.

<a id="device-configuration"></a>

### إعداد الأجهزة

`192.0.2.50` هو traffic66 و`192.0.2.1` الجهاز؛ المهل ومعدلات أخذ العينات: [القسم 4](#4-send-flows-from-your-devices).

```
! Cisco IOS-XE, NetFlow v9 (IPFIX: add "export-protocol ipfix" and use "transport udp 4739")
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
# Huawei CloudEngine, sFlow
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
# Huawei AR/NE, NetStream (NetFlow v9)
ip netstream export version 9
ip netstream export host 192.0.2.50 2055
ip netstream timeout active 1
interface GigabitEthernet0/0/1
 ip netstream inbound
 ip netstream sampler fix-packets 1000 inbound
```

```
# H3C Comware 7, sFlow
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
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
```

```
# MikroTik RouterOS 7, NetFlow v9 (IPFIX: version=ipfix port=4739)
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

### فتح الجدار الناري

```
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload   # RHEL, Fedora
sudo ufw allow proto udp to any port 6343,2055,4739 && sudo ufw allow 8066/tcp                          # Ubuntu, Debian
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow   # Windows
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

### واجهة الويب على هذا الجهاز فقط، عبر SSH

```
./traffic66 -addr 127.0.0.1:8066                 # on the server
ssh -L 8066:127.0.0.1:8066 user@192.0.2.50        # on your PC, then open http://127.0.0.1:8066
```

### كخدمة: لينكس (systemd)

```
sudo mkdir -p /opt/traffic66 && curl -fsSL https://github.com/githubflyideas/traffic66/releases/latest/download/traffic66-linux-amd64.tar.gz | sudo tar xz -C /opt/traffic66 --strip-components=1
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
journalctl -u traffic66 -f
```

### كخدمة: ويندوز

```
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

### كخدمة: macOS

فك الضغط في `/usr/local/traffic66`، وأزل علامة الحجر، وشغّل `traffic66 passwd -data "/Library/Application Support/traffic66"`، ثم ابدأه من LaunchDaemon تكون `ProgramArguments` فيه البرنامج و`-data` وذلك المجلد، مع `RunAtLoad` و`KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. المستخدمون وكلمات المرور

يُسجَّل الدخول في التثبيت الجديد بـ `admin` / `traffic66`، ويطلب أول تسجيل
دخول كلمة مرور جديدة قبل عرض أي شيء آخر (يحتفظ العرض التجريبي بـ `traffic66`).
بعد ذلك تُغيّر كلمة مرورك من **الحساب** أسفل القائمة؛ ومن هناك أيضًا يضيف
المسؤول (`admin`) المستخدمين ويحذفهم ويعيد تعيين كلمات مرورهم. تسجيل الدخول عبر
LDAP / Active Directory قيد التطوير.

يُحفظ المستخدمون بصيغة تجزئات مملّحة في `password` داخل دليل البيانات. ويمكن
فعل الشيء نفسه بأمر واحد على جهاز traffic66 (أضف `-data …` إن كان traffic66
يعمل به):

| المطلوب | الأمر |
|---|---|
| تغيير كلمة مرور `admin` | `traffic66 passwd` |
| إضافة `alice` أو تغيير كلمة مرورها | `traffic66 passwd -user alice` |
| حذف `alice` | `traffic66 passwd -user alice -delete` |
| عرض المستخدمين | `traffic66 passwd -list` |

تُطبَّق التغييرات فورًا. باستثناء إدارة المستخدمين، لجميع المستخدمين
الصلاحيات نفسها. للسكربتات والحاويات، يقبل `TRAFFIC66_PASSWORD=…` (أو `-password`) المستخدم `-user` فقط
بكلمة المرور تلك طوال ذلك التشغيل. خمس كلمات مرور خاطئة خلال دقيقة تحظر
العنوان لمدة دقيقة.

<a id="4-send-flows-from-your-devices"></a>

## 4. إرسال التدفقات من أجهزتك

`192.0.2.50` هو traffic66، و`192.0.2.1` هو الجهاز. اضبط مهلة التدفق النشط على
60 ثانية، ودع أجهزة NetFlow/IPFIX تصدّر خيارات أخذ العينات، وخذ العينات من
**كل واجهة في اتجاه الدخول** (أو من واجهات الحافة فقط): بذلك تُحسب كل حزمة مرة
واحدة. نسب sFlow: نحو 1:1000 لـ 1 Gb/s، و1:4096 لـ 10 Gb/s، و1:8192 لـ
40/100 Gb/s.

إعدادات Cisco وHuawei وH3C وJuniper وArista وMikroTik وsoftflowd: [أمثلة ← إعداد الأجهزة](#device-configuration).

<a id="5-check-that-flows-arrive"></a>

## 5. التحقق من وصول التدفقات

تسرد **الإعدادات** خلال ثوانٍ كل جهاز يرسل أي شيء: البروتوكول، ومعدّل أخذ
العينات، والفقد، والواجهات التي يأخذ منها العينات، وما يجب إصلاحه حين لا تكون
الحالة خضراء. يُقسَم فقد sFlow إلى فقد في الطريق (ارفع `net.core.rmem_max` إن
أظهر `netstat -su` أخطاء في المخزن المؤقت) وعينات أسقطها الجهاز نفسه.

![الإعدادات: كل جهاز مع البروتوكول وأخذ العينات والفقد وما يجب إصلاحه](images/sources.png)

جهاز مفقود؟ شغّل `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739`: إن لم يظهر شيء فالمشكلة في التوجيه أو جدار ناري أو إعداد
الجهاز؛ وإن ظهرت حزم ولا شيء في **الإعدادات** فالسبب الجدار الناري المحلي أو
`-listen`. يختبر `traffic66 simulate -to 192.0.2.50` من جهاز آخر المسارَ
بأجهزة محاكاة.

<a id="6-interfaces-and-counters"></a>

## 6. الواجهات والعدّادات

أرقام التدفقات تقديرات (العينات × معدّل أخذ العينات). تقارنها **مطابقة الواجهات**
بعدّادات واجهات الجهاز (عدّادات sFlow، أو SNMP عبر سطر `snmp` في الأسماء)
وتشرح سبب الاختلاف: واجهات لا تؤخذ منها عينات، أو الحركة نفسها مأخوذة بالعينات
مرتين، أو فقد في الطريق، أو معدّل أخذ عينات غير معروف. لكل واجهة مخطط bits/s
ومخطط packets/s، الدخول بالأخضر والخروج بالأزرق، والعدّادات بخط متقطع.

في كل صف، يضبط **✎** اسمًا ووسمًا قصيرًا (مثل *uplink*)، ويجعلها **☆** الواجهة
الافتراضية (★) التي تُفتح عليها الصفحات.

الجهاز الذي يأخذ العينات من بعض الواجهات فقط يُظهر أيضًا الأطراف الأخرى لتلك
التدفقات. تُدرَج **واجهات الطرف الآخر** هذه في الآخر بخط أصغر رمادي: لا تحمل إلا
الحركة المارّة عبر الواجهة المأخوذ منها العينات. تُعرف الواجهة المأخوذ منها
العينات من مصدر بيانات sFlow أو من الحقل flowDirection (IPFIX 61)؛ وبدونه،
الواجهة الموجودة في 90% من حركة الجهاز.

![مطابقة الواجهات: حركة كل واجهة، وتقدير التدفقات بجوار عدّاد الجهاز](images/interfaces.png)

يستخدم traffic66 أصلًا المعدّل الذي طبّقه الجهاز، وينتظر المعدّلات غير
المعروفة، ويعوّض فقد التصدير، ويوزّع التدفقات الطويلة على دقائقها، ويضيف 18 بايت
لكل حزمة من حمل Ethernet الإضافي إلى NetFlow/IPFIX (`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. الأسماء والدول وقوائم التهديدات

انقر على أي عنوان واختر **تسمية…**، أو استخدم **الإعدادات ← الأسماء**.
تُحفظ الأسماء في `inventory.txt` داخل دليل البيانات:

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

النطاقات الخاصة تُعدّ دائمًا لك. تُطبَّق التغييرات عند **حفظ**، دون إعادة تشغيل.

تعمل الدول والشبكات (AS) فورًا بقاعدتَي DB-IP Lite المجانيتين
([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)، "IP
Geolocation by DB-IP"، [db-ip.com](https://db-ip.com))؛ وتحدّثهما **الإعدادات**،
أو تقبل بدلًا منهما ملفات MaxMind GeoLite2 أو IPinfo Lite أو IPtoASN. حدود
الخريطة: [Natural Earth](https://www.naturalearthdata.com).

![الجغرافيا والشبكات: الحركة الخارجية حسب الدولة على خريطة العالم](images/geo.png)

قوائم التهديدات ملفات نصية فيها عنوان أو شبكة في كل سطر، في
`<data>/threats/<name>.txt` (مثل Spamhaus DROP)؛ أعد التشغيل بعد تغييرها.
تظهر التطابقات في **معلومات التهديدات**.

![معلومات التهديدات: مضيف داخلي يرسل بيانات إلى عنوان مدرج في قائمة تهديدات](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. استخدام واجهة الويب

كل قيمة في كل صفحة قابلة للنقر: **اعرض هذا فقط** / **استبعد هذا** (تنطبق
المرشّحات على كل الصفحات)، **اعرض سجلات تدفقه**، **عرض التفاصيل** (صفحة عن مضيف
أو خدمة واحدة)، **تسمية…**، **ابحث عنه على الإنترنت**، **نسخ**.

| الصفحة | ما تعرضه |
|---|---|
| نظرة عامة | عرض النطاق للواجهة المختارة؛ الحركة حسب التطبيق (الإجمالي أو الوارد أو الصادر) مقارنةً بأمس أو بالأسبوع الماضي؛ الاكتشافات المفتوحة؛ أبرز العملاء والخدمات |
| أعلى 66 | أعلى 66 محادثة، قابلة للفرز حسب أي عمود، أو مجمّعة حسب التطبيق أو الشبكة أو المقطع أو الجهاز أو التغليف أو VLAN؛ أكثر 30 طرفًا نشاطًا |
| تفاصيل الحركة | مخططات حلقية: الخوادم وعملاؤها (أو العكس)، والخدمات |
| مسارات الحركة | مضيف ← تطبيق ← دولة، أو عميل ← خدمة ← خادم، أو حسب الشبكة |
| مطابقة الواجهات | كل واجهة عبر الزمن مقابل عدّاداتها؛ الأسماء والوسوم والواجهة الافتراضية |
| سجلات التدفق | التدفقات الفردية، مباشرةً كل 5 ثوانٍ أو لأي نطاق زمني |
| الاكتشافات، معلومات التهديدات | ما يحتاج إلى انتباه ([أدناه](#findings))؛ الحركة مع العناوين المدرجة |
| الجغرافيا والشبكات | خريطة العالم حسب الدولة، والشبكات (AS) عبر الزمن |
| الإعدادات | الأجهزة، وأخذ العينات، والفقد، وSNMP، وقواعد البيانات، والشعار، والأسماء |
| تحليل pcap دون اتصال، تنظيف البيانات | ملفات الالتقاط ([أدناه](#9-offline-pcap-terminal-ui-local-capture))؛ حذف البيانات القديمة |

فوق الصفحات: **الواجهة** (الكل، أو واجهة واحدة مأخوذ منها العينات؛ عندها لا
تعرض صفحات الحركة إلا الحركة المارّة عبرها)، والنطاق الزمني (من 15 دقيقة إلى 30
يومًا، أو مخصص)، وتحديث كل 30 ثانية، و**نسخ الرابط** للعرض الحالي بالضبط. اللغة
وسمات الألوان الخمس في أسفل القائمة. تنتهي المخططات حيث تكتمل البيانات: مع
NetFlow/IPFIX بقدر تأخر الأجهزة في التصدير (دقيقتان كحد أقصى). النطاقات التي
تزيد على 6 ساعات تبدأ عند ساعة كاملة؛ واجهة واحدة على مدى 7 أو 30 يومًا تقرأ
تفاصيل التدفقات، لذا فهي أبطأ ولا تمتد إلا بقدر ما تُحفظ التفاصيل.

![أعلى 66: أعلى 66 محادثة، مرتّبة حسب أي عمود](images/topn.png)

![تفاصيل الحركة: الخوادم مع عملائها، والخدمات مع خوادمها، في مخططات حلقية](images/traffic.png)

![تفاصيل مضيف واحد: الاكتشافات المتعلقة به، وحركته، ومن يتواصل معه، والخدمات، والدول، وأحدث التدفقات](images/detail.png)

![مسارات الحركة: أي مضيف يستخدم أي تطبيق نحو أي دولة](images/paths.png)

![النظرة العامة باللغة الصينية](images/overview-zh.png)

<a id="findings"></a>

### الاكتشافات

يُفحص آخر 10 دقائق كل 5 دقائق؛ والشيء الذي يستمر ساعة هو اكتشاف واحد يكبر.

| الاكتشاف | معناه |
|---|---|
| مسح، مسح المنافذ | مجسّات صغيرة إلى مضيفات كثيرة على منفذ واحد، أو إلى منافذ كثيرة لمضيف واحد |
| تخمين كلمات المرور | اتصالات قصيرة كثيرة بخدمة تسجيل دخول |
| تحرك جانبي | مشاركة ملفات أو إدارة عن بُعد إلى مضيفات داخلية لم تقدّم ذلك من قبل |
| رفع غير معتاد | 100 MB في 10 دقائق إلى عنوان جديد، ثلاثة أضعاف ما عاد |
| إغراق | أكثر من 20,000 حزمة صغيرة في الثانية إلى عنوان واحد، عشرة أضعاف معدله المعتاد |
| قائمة التهديدات | حركة مع عنوان مدرج |

من داخل شبكتك تكون خطورتها عالية، ومن الإنترنت منخفضة. يغلق **تمت المعالجة**
الاكتشاف، ويُسكته **ليست مشكلة** نهائيًا. يحتاج التحرك الجانبي والرفع إلى سجلّ
يوم كامل. عبر أخذ عينات بنسبة 1:4096 يُكتشف هجوم العرض التوضيحي كاملًا؛ أما
عمليات المسح الصغيرة جدًا فقد تختبئ خلف أخذ العينات.

![الاكتشافات: كل خطوة من هجوم، اكتُشفت عبر أخذ عينات sFlow بنسبة 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. تحليل pcap دون اتصال، الواجهة الطرفية، الالتقاط المحلي

يعرض **تحليل pcap دون اتصال** ملفات التقاط الحزم (pcap وpcapng) بالصفحات نفسها،
بمعزل عن البيانات الحية: يبدأ `traffic66 a.pcap b.pcapng` على 127.0.0.1 ويفتح
المتصفح (حتى 3 ملفات، 3 GB؛ يحذف Ctrl+C البيانات المستوردة)، أو ارفع حتى 3
ملفات بحجم 50 MB في تلك الصفحة. يُحلَّل ملف واحد في كل مرة، ولكل ملف قاعدة
بيانات خاصة به: يعرض **تحليل** في صف الملف بياناته في كل الصفحات، ويبدّل
الشريط في الأعلى إلى ملف آخر. يعمل على التدفقات، لا على محتوى الحزم.

![التحليل دون اتصال: ملفات الالتقاط مع حزمها وتدفقاتها ووقتها](images/sandbox.png)

**الواجهة الطرفية**: `traffic66 tui` على جهاز traffic66، أو
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
ثلاث صفحات: **1 الأعلى** (العملاء والخوادم والتطبيقات والمنافذ والشبكات (AS) والدول/المناطق حسب المعدل؛ ← → للتبديل، وEnter للإجراءات)، و**2 الواجهات** (المعدل والاستخدام والمطابقة لكل جهاز وواجهة، مع أخذ العينات والفقد لكل جهاز؛ Enter يعرض الاتجاه وأعلى المضيفين/التطبيقات)، و**3 بحث** (سجلات التدفق لعوامل تصفية مثل `ip:10.0.0.5 net:10.0.0.0/8 asn:13335 port:443 cc:US dev:192.0.2.1`). المفاتيح: `/` تصفية، Backspace/`c` إزالة/مسح، `t` النطاق الزمني (5 دقائق إلى 24 ساعة)، `u` بت/ث أو بايت أو حزم/ث، `p` إيقاف مؤقت، `+`/`-` فترة التحديث، `w` فتح العرض نفسه في المتصفح، `?` مساعدة، `q` خروج؛ `-lang` يختار اللغة و`NO_COLOR` يلغي الألوان.

![الواجهة الطرفية: الأعلى](images/tui-top.png)

![الواجهة الطرفية: الواجهات](images/tui-interfaces.png)

**الالتقاط المحلي** يبني التدفقات من واجهة محلية، ويُفضَّل منفذ موصول بمنفذ
المرآة في مبدّل: يسردها `traffic66 interfaces`، ويلتقط `-capture eth1`. على
Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

يحتاج Linux إلى root أو `setcap cap_net_raw,cap_net_admin+ep`، وmacOS إلى
root، وWindows إلى [Npcap](https://npcap.com). تأتي التدفقات الملتقطة من
الجهاز `127.0.0.1`. لا واجهات ولا عدّادات جهاز للالتقاط المحلي، لذا ليس لدى
**مطابقة الواجهات** ما تقارنه فيه.

<a id="10-options-and-data"></a>

## 10. الخيارات والبيانات

يسرد `traffic66 -h` كل شيء. الأكثر استخدامًا:

| الخيار | الافتراضي | |
|---|---|---|
| `-data` | `traffic66-data` بجوار البرنامج | دليل البيانات |
| `-addr` | `:8066` | واجهة الويب؛ `127.0.0.1:8066` لهذا الجهاز فقط |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | مستقبِلات UDP؛ القيمة الفارغة تعطّلها |
| `-retention-days` | `30` | أيام تفاصيل التدفقات؛ تُحفظ الملخصات 400 يوم |
| `-memory` | `0.10` | حصة الذاكرة لذاكرة قاعدة البيانات المؤقتة |
| `-sampling-wait` | `5m` | مدة انتظار السجلات لمعدّل أخذ العينات |
| `-capture` | | واجهة محلية (قابل للتكرار) |
| `-no-dns` | | بلا استعلامات عكسية |

يحوي دليل البيانات `raw/` (التفاصيل، ملف لكل ساعة)، و`traffic66.duckdb`
(الملخصات والعدّادات)، و`password`، و`inventory.txt`، و`license.json`، وشعارك
وقواعد البيانات. للنسخ الاحتياطي أوقف traffic66 وانسخه؛ وللترقية استبدل ملف
البرنامج. يحذف **تنظيف البيانات** البيانات الأقدم من 7–120 يومًا، أو كلها.

<a id="licence"></a>

### الترخيص

متاح المصدر بموجب [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
و[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (النص الإنجليزي
هو الملزم): مجاني للتقييم وللمؤسسات التي يقل عدد أفرادها عن 100 شخص؛ وتسجّل
المؤسسات الأكبر بعد 30 يومًا من الاستخدام الإنتاجي؛ ويحتاج البيع أو الاستضافة
للآخرين أو المنتجات المنافسة إلى ترخيص تجاري. لا يُعطَّل أي شيء أبدًا. يعرض أسفل
كل صفحة رقم التثبيت المكوّن من 8 أرقام؛ أرسله إلى المؤلف، وضع ملف
`license.json` الذي تتلقاه في دليل البيانات. للتواصل:
<route66traffic66@gmail.com>.

تعرض التثبيتات غير المسجلة شريط رعاية أعلى الصفحة. يحمّله متصفحك وحده من GitHub ([ads-sponsor](https://github.com/githubflyideas/ads-sponsor))؛ ولا يتصل خادم traffic66 بـ GitHub لأجله أبدًا ولا يحتفظ بنسخة منه. والمتصفح الذي لا يصل إلى GitHub يعرض بدلًا منه نصًا قصيرًا مدمجًا. لا تُرسل أي بيانات عنك أو عن التثبيت. ويزول بالتسجيل.

<a id="11-security-sizing-troubleshooting"></a>

## 11. الأمان، تقدير الموارد، استكشاف الأخطاء

واجهة الويب HTTP عادي: على الشبكات غير الموثوقة استخدم `-addr 127.0.0.1:8066`
خلف وكيل TLS (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) أو نفق SSH. اسمح بمنافذ UDP من أجهزتك فقط. تُخزَّن مجتمعات
SNMP نصًا صريحًا؛ استخدم مجتمعات للقراءة فقط.

عند 5,000 تدفق/ث على نواتين: نحو 12 GB من القرص لكل يوم من التفاصيل (360 GB
لـ 30 يومًا)، وسدس نواة، و0.6–0.8 GB من الذاكرة. تستغرق النظرات العامة للنطاقات
الطويلة أقل من 0.2 ثانية؛ وأعلى 66 لكل المحادثات لمدة ساعة نحو 9 ثوانٍ.

| العَرَض | الحل |
|---|---|
| "waiting for the sampling rate" | صدّر خيارات أخذ العينات، أو `sampling=N` / `unsampled` في سطر الجهاز |
| أقل من العدّادات | واجهات لا تؤخذ منها عينات، أو فقد، أو مهلة تدفق نشط أطول من 60 ثانية |
| أعلى من العدّادات | الحركة نفسها مأخوذة بالعينات على واجهتين أو جهازين |
| نسيت كلمة المرور | `traffic66 passwd` على جهاز traffic66 |
| `Conflicting lock is held` | نسخة أخرى من traffic66 تستخدم دليل البيانات هذا |
| `address already in use` | اختر منافذ أخرى بـ `-addr` أو `-listen` |
| Windows "protected your PC" | **More info** (مزيد من المعلومات) → **Run anyway** (التشغيل على أي حال) |

البناء من المصدر: Go 1.26 ومترجم C، ثم `CGO_ENABLED=1 go build ./cmd/traffic66`.

</div>
