[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | **العربية** | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

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
- 13 لغة. مجاني للتقييم وللمؤسسات التي يقل عدد أفرادها عن 100 شخص ([الترخيص](#licence)).

![نظرة عامة: الاكتشافات المفتوحة، واستهلاك عرض النطاق حسب التطبيق مقارنةً بالوقت نفسه أمس، وأبرز العملاء والخدمات](images/overview.png)

<sub>جميع لقطات الشاشة مأخوذة من `traffic66 demo`، وهي شبكة شركة محاكاة.</sub>

<a id="contents"></a>

## المحتويات

1. [جرّب العرض التوضيحي](#1-try-the-demo)
2. [التثبيت](#2-install)
3. [المستخدمون وكلمات المرور](#3-users-and-passwords)
4. [إرسال التدفقات من أجهزتك](#4-send-flows-from-your-devices)
5. [التحقق من وصول التدفقات](#5-check-that-flows-arrive)
6. [الواجهات والعدّادات](#6-interfaces-and-counters)
7. [الأسماء والدول وقوائم التهديدات](#7-names-countries-and-threat-lists)
8. [استخدام واجهة الويب](#8-using-the-web-ui)
9. [تحليل pcap دون اتصال، الواجهة الطرفية، الالتقاط المحلي](#9-offline-pcap-terminal-ui-local-capture)
10. [الخيارات والبيانات](#10-options-and-data)
11. [الأمان، تقدير الموارد، استكشاف الأخطاء](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. جرّب العرض التوضيحي

نزّل الأرشيف المناسب لنظامك من
[صفحة الإصدارات](https://github.com/githubflyideas/traffic66/releases)
(Windows x64، وLinux x86-64/ARM64 بنواة 3.2+، وmacOS 11+)، وفكّ ضغطه وشغّل:

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

على macOS شغّل أولًا `xattr -dr com.apple.quarantine <folder>`. افتح
http://127.0.0.1:8066 وسجّل الدخول بـ `admin` / `try66`: سجلّ يوم كامل وحركة
حية من أربعة أجهزة محاكاة، ومنها هجوم يُعرض خطوةً خطوة في **الاكتشافات**. يوقفه
Ctrl+C؛ احذف `traffic66-demo` لتبدأ من جديد. لتشغيله بجوار تثبيت حقيقي:
`-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. التثبيت

traffic66 ملف واحد. المنافذ: UDP 6343 (sFlow)، و2055 (NetFlow)، و4739
(IPFIX)، وTCP 8066 (واجهة الويب)؛ وكل منفذ UDP يقبل كل البروتوكولات.

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

**Windows** (PowerShell بصلاحيات المسؤول): فكّ الضغط إلى `C:\traffic66`،
وشغّل `C:\traffic66\traffic66.exe passwd`، وافتح المنافذ، واجعله يبدأ مع
الإقلاع:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

النقر المزدوج على `traffic66.exe` يعمل أيضًا: يفتح واجهة الويب ويعرض كلمة
المرور الأولى في نافذته.

**macOS**: فكّ الضغط إلى `/usr/local/traffic66`، وأزل علامة الحجر،
وشغّل `traffic66 passwd -data "/Library/Application Support/traffic66"`، ثم
شغّله من LaunchDaemon تكون `ProgramArguments` فيه البرنامج و`-data` وذلك
الدليل، مع `RunAtLoad` و`KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. المستخدمون وكلمات المرور

عند التشغيل الأول ينشئ traffic66 المستخدم `admin` بكلمة مرور عشوائية ويطبعها
مرة واحدة (في النافذة، أو الطرفية، أو
`journalctl -u traffic66 | grep "first start"`). يُحفظ المستخدمون بصيغة
تجزئات مملّحة في `password` داخل دليل البيانات، ويُدارون بأمر واحد على جهاز
traffic66 (أضف `-data …` إن كان traffic66 يعمل به):

| المطلوب | الأمر |
|---|---|
| تغيير كلمة مرور `admin` | `traffic66 passwd` |
| إضافة `alice` أو تغيير كلمة مرورها | `traffic66 passwd -user alice` |
| حذف `alice` | `traffic66 passwd -user alice -delete` |
| عرض المستخدمين | `traffic66 passwd -list` |

تُطبَّق التغييرات فورًا. لجميع المستخدمين الصلاحيات نفسها. للسكربتات
والحاويات، يقبل `TRAFFIC66_PASSWORD=…` (أو `-password`) المستخدم `-user` فقط
بكلمة المرور تلك طوال ذلك التشغيل. خمس كلمات مرور خاطئة خلال دقيقة تحظر
العنوان لمدة دقيقة.

<a id="4-send-flows-from-your-devices"></a>

## 4. إرسال التدفقات من أجهزتك

`192.0.2.50` هو traffic66، و`192.0.2.1` هو الجهاز. اضبط مهلة التدفق النشط على
60 ثانية، ودع أجهزة NetFlow/IPFIX تصدّر خيارات أخذ العينات، وخذ العينات من
**كل واجهة في اتجاه الدخول** (أو من واجهات الحافة فقط): بذلك تُحسب كل حزمة مرة
واحدة. نسب sFlow: نحو 1:1000 لـ 1 Gb/s، و1:4096 لـ 10 Gb/s، و1:8192 لـ
40/100 Gb/s.

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
ملفات بحجم 50 MB في تلك الصفحة. يعمل على التدفقات، لا على محتوى الحزم.

![التحليل دون اتصال: ملفات الالتقاط مع حزمها وتدفقاتها ووقتها](images/sandbox.png)

**الواجهة الطرفية**: `traffic66 tui` على جهاز traffic66، أو
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
المفاتيح: 1–8 للصفحات، Enter للإجراءات، f لعرض هذا فقط، x للاستبعاد، t للنطاق
الزمني، w للفتح في المتصفح، q للخروج؛ ويختار `-lang` اللغة.

![الواجهة الطرفية: نظرة عامة](images/tui-overview.png)

![الواجهة الطرفية: محادثات أعلى 66](images/tui-topn.png)

**الالتقاط المحلي** يبني التدفقات من واجهة محلية، ويُفضَّل منفذ موصول بمنفذ
المرآة في مبدّل: يسردها `traffic66 interfaces`، ويلتقط `-capture eth1` (أو اسم
أو رقم في Windows). يحتاج Linux إلى root أو
`setcap cap_net_raw,cap_net_admin+ep`، وmacOS إلى root، وWindows إلى
[Npcap](https://npcap.com). تأتي التدفقات الملتقطة من الجهاز `127.0.0.1`.

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
<https://github.com/githubflyideas/traffic66>.

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

البناء من المصدر: Go 1.24 ومترجم C، ثم `scripts/build.sh 0.1.0 traffic66`.

</div>
