[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | **العربية** | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66

تحليل تدفقات sFlow وNetFlow وIPFIX في برنامج واحد. يستقبل traffic66
التدفقات المُصدَّرة من المبدّلات والموجّهات والجدران النارية، ويخزّنها في
قاعدة بيانات مدمجة، ويعرض من يستهلك عرض النطاق، وإلى أين تذهب الحركة، وهل
تتطابق الأرقام مع عدّادات الواجهات في الأجهزة نفسها — في واجهة ويب وفي
واجهة طرفية.

- ملف تنفيذي واحد لأنظمة Windows وLinux وmacOS. لا قاعدة بيانات تحتاج إلى
  تثبيت، ولا بيئة تشغيل، ويعمل دون اتصال بالإنترنت.
- sFlow v5 وNetFlow v5 وNetFlow v9 وIPFIX على أي منفذ UDP؛ مع التقاط محلي
  اختياري من واجهة شبكة أو منفذ مرآة.
- يقارن أرقامه بعدّادات الواجهات (عدّادات sFlow أو SNMP) ويشرح سبب
  الاختلاف حين يقع.
- قوائم أعلى 66، ومسارات الحركة، والدول والشبكات، والتطابقات مع قوائم
  التهديدات، وسجلات التدفق، والتغليف (GRE وIPIP وVXLAN وGENEVE وMPLS).
- 13 لغة في واجهة الويب والواجهة الطرفية.

<a id="contents"></a>

## المحتويات

1. [جرّب العرض التوضيحي](#1-try-the-demo)
2. [التثبيت](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [تسجيل الدخول وكلمات المرور](#3-sign-in-and-passwords)
4. [إرسال التدفقات من أجهزتك](#4-send-flows-from-your-devices)
5. [التحقق من وصول التدفقات](#5-check-that-flows-arrive)
6. [مطابقة الأرقام مع عدّادات الواجهات](#6-make-the-numbers-match-the-interface-counters)
7. [الأسماء وSNMP وشبكاتك الخاصة](#7-names-snmp-and-your-own-networks)
8. [الدول والشبكات وقوائم التهديدات](#8-countries-networks-and-threat-lists)
9. [استخدام واجهة الويب](#9-using-the-web-ui)
10. [الواجهة الطرفية](#10-terminal-ui)
11. [الالتقاط المحلي](#11-local-capture)
12. [الخيارات](#12-options)
13. [البيانات والنسخ الاحتياطي والترقية وإزالة التثبيت](#13-data-backup-upgrade-uninstall)
14. [الأمان](#14-security)
15. [تقدير الموارد](#15-sizing)
16. [استكشاف الأخطاء وإصلاحها](#16-troubleshooting)
17. [البناء من المصدر](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. جرّب العرض التوضيحي

نزّل الأرشيف المناسب لنظامك من
[صفحة الإصدارات](https://github.com/githubflyideas/traffic66/releases):

| النظام | الأرشيف |
|---|---|
| Windows 10/11 وServer 2016 أو أحدث (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: أي توزيعة بنواة 3.2 أو أحدث، بما فيها CentOS 7 وAlpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: التوزيعات نفسها | `traffic66-linux-arm64.tar.gz` |
| macOS 11 أو أحدث بمعالج Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 أو أحدث بمعالج Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (السطر الثاني يسمح لنظام macOS بتشغيل برنامج نُزّل من الإنترنت وليس
من App Store):

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

افتح http://127.0.0.1:8066 وسجّل الدخول باسم `admin` وكلمة المرور `try66`.
ينشئ العرض التوضيحي شبكة شركة صغيرة بسجلّ يوم كامل وحركة حية من أربعة أجهزة
محاكاة، وفيها حادثتان عليك اكتشافهما: ابدأ من **نظرة عامة**، وانظر إلى
**من زادت حركته**، ثم تابع بالنقر. أوقفه بـ Ctrl+C. تُحفظ بيانات العرض في
`traffic66-demo` بجوار البرنامج؛ احذف هذا المجلد لتبدأ العرض من جديد.

يستخدم العرض التوضيحي المنافذ نفسها التي يستخدمها التثبيت الفعلي (8066،
وUDP 6343 و2055 و4739). لتشغيله بجانب تثبيت فعلي، امنحه منافذ أخرى:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

على Windows يمكنك أيضًا ببساطة النقر المزدوج على `traffic66.exe`. يؤدي ذلك
إلى تشغيل traffic66 فعليًا (وليس العرض التوضيحي) وفتح واجهة الويب في متصفحك؛
تظهر كلمة مرور التشغيل الأول في النافذة السوداء، وإغلاق النافذة يوقف traffic66.
إذا عرض Windows الرسالة "Windows protected your PC" (حمى Windows جهازك)،
فانقر **More info** (مزيد من المعلومات) → **Run anyway** (التشغيل على أي حال).

<a id="2-install"></a>

## 2. التثبيت

traffic66 ملف واحد. التثبيت يعني وضعه في مكان ما، واختيار دليل للبيانات،
وتعيين كلمة مرور، وفتح الجدار الناري، وتشغيله عند الإقلاع. تستخدم الأمثلة
`192.0.2.50` لجهاز traffic66 و`192.0.2.1` لموجّه؛ استبدلهما بعناوينك.

المنافذ:

| المنفذ | الاستخدام |
|---|---|
| UDP 6343 | sFlow (افتراضي) |
| UDP 2055 | NetFlow (افتراضي) |
| UDP 4739 | IPFIX (افتراضي) |
| TCP 8066 | واجهة الويب وواجهة API |

كل منفذ UDP يقبل كل البروتوكولات، فيمكن للجهاز إرسال NetFlow إلى 6343 إن
كان ذلك أسهل. غيّر المنافذ أو أضف غيرها باستخدام `-listen`.

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

يطلب الأمر الأخير كلمة مرور المستخدم `admin`.

أنشئ `/etc/systemd/system/traffic66.service`:

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

شغّله، واسمح بمخازن UDP مؤقتة أكبر حتى لا تضيع الحزم عند الذروات:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

الجدار الناري باستخدام firewalld (RHEL وRocky وAlma وFedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

أو باستخدام ufw (Ubuntu وDebian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

فك الضغط إلى `C:\traffic66` وعيّن كلمة المرور (PowerShell بصلاحيات
المسؤول):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

تُحفظ البيانات في `C:\traffic66\traffic66-data` بجوار البرنامج.

افتح الجدار الناري:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

لتجربته في الواجهة الأمامية، شغّل `C:\traffic66\traffic66.exe` وأوقفه
بـ Ctrl+C. ولتشغيله في الخلفية منذ الإقلاع دون أن يكون أحد مسجّلًا الدخول،
سجّله مهمةً عند بدء التشغيل:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

الخيار `-ExecutionTimeLimit ([TimeSpan]::Zero)` ضروري: من دونه يوقف Windows
المهمة بعد ثلاثة أيام. أوقفها بـ `Stop-ScheduledTask -TaskName
traffic66`، واحذفها بـ `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

أنشئ `/Library/LaunchDaemons/traffic66.plist`:

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

التشغيل ثم الإيقاف:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

إذا كان الجدار الناري في macOS مفعّلًا، فاسمح بالاتصالات الواردة لـ
traffic66 من إعدادات النظام ← الشبكة ← جدار الحماية ← الخيارات.

<a id="3-sign-in-and-passwords"></a>

## 3. تسجيل الدخول وكلمات المرور

افتح `http://<traffic66 machine>:8066` وسجّل الدخول. اسم المستخدم هو
`admin` ما لم تختر غيره.

- إذا لم تعيّن كلمة مرور قبل التشغيل الأول، يُنشئ traffic66 واحدة ويطبعها
  مرة واحدة في سجلّه:
  `first start: sign in as user "admin" with password "…"`.
  في Linux تجدها بالأمر `journalctl -u traffic66 | grep "first start"`.
- تُحفظ كلمة المرور مُجزّأة (hashed) في الملف `password` داخل دليل
  البيانات، وتبقى كما هي بعد إعادة التشغيل.
- لتغييرها، أو لتعيين كلمة جديدة بعد نسيانها، نفّذ على جهاز traffic66:

  ```
  traffic66 passwd -data <data directory>
  ```

  الأمر `traffic66 passwd -generate` يُنشئ كلمة عشوائية ويطبعها. يقبل
  traffic66 العامل كلمة المرور الجديدة عند تسجيل الدخول التالي؛ لا حاجة
  إلى إعادة التشغيل.
- مستخدمون إضافيون: `traffic66 passwd -data <data directory> -user alice`.
  يرى جميع المستخدمين الشيء نفسه.
- في السكربتات والحاويات، يعيّن `TRAFFIC66_PASSWORD=…` في متغيرات البيئة أو
  `-password …` في سطر الأوامر كلمةَ المرور لهذا التشغيل بدلًا من المخزّنة.
  فضّل متغيرات البيئة: فأسطر الأوامر مرئية لبقية مستخدمي الجهاز.

بعد خمس محاولات خاطئة خلال دقيقة يُحظر العنوان لمدة دقيقة.

<a id="4-send-flows-from-your-devices"></a>

## 4. إرسال التدفقات من أجهزتك

وجّه كل جهاز إلى جهاز traffic66. تختلف الأوامر بين الطرازات وإصدارات
البرمجيات؛ راجع دليل جهازك. في جميع الأمثلة، `192.0.2.50` هو traffic66
و`192.0.2.1` هو عنوان الجهاز نفسه.

نصائح عامة:

- اضبط مهلة التدفق النشط (active flow timeout) على 60 ثانية. المهل الأطول
  تجعل الحركة تصل متأخرة على دفعات كبيرة.
- إذا كان الجهاز يأخذ عينات NetFlow/IPFIX، فاجعله يُصدّر خيارات أخذ
  العينات (sampler options) ليُعرف المعدّل. يحتجز traffic66 السجلات حتى
  يصل المعدّل بدلًا من عدّها بنسبة 1:1.
- خذ العينات إما على كل الواجهات أو على واجهات الأطراف فقط، وفي اتجاه
  واحد. أخذ عينات الحركة نفسها عند الدخول وعند الخروج يعدّها مرتين؛ وصفحة
  **مطابقة الواجهات** تنبّه إلى ذلك.
- معدّل أخذ العينات في sFlow: نحو 1:1000 لروابط 1 Gb/s، و1:4096 لـ
  10 Gb/s، و1:8192 لـ 40/100 Gb/s.

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

FortiGate FortiOS 7.4.2 أو أحدث (NetFlow v9):

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

خوادم ومضيفات Linux، باستخدام softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. التحقق من وصول التدفقات

افتح **المصادر**. يظهر كل جهاز يرسل أي شيء خلال ثوانٍ، مع البروتوكول ومعدّل
أخذ العينات والفقد وآخر حزمة وحالته. إذا لم تكن الحالة خضراء، فالنص
المجاور يوضّح الخلل وما يجب تغييره.

إذا لم يظهر جهاز:

1. راقب الحزم على جهاز traffic66 (Linux وmacOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   إن لم يظهر شيء، فالحزم لا تصل إلى الجهاز: افحص إعدادات الجهاز المُرسِل
   والتوجيه والجدران النارية على المسار.
2. الحزم تصل لكن **المصادر** تبقى فارغة: الجدار الناري المحلي يُسقطها
   (انظر [التثبيت](#2-install))، أو أن traffic66 يستمع على منافذ أخرى
   (`-listen`).
3. لاختبار المسار من جهاز آخر دون المساس بأي جهاز شبكي، شغّل هناك
   `traffic66 simulate -to 192.0.2.50` لبضع ثوانٍ. يرسل هذا الأمر sFlow
   وNetFlow وIPFIX من أجهزة محاكاة، فتظهر بعدها في **المصادر** وفي
   البيانات؛ لذا يُفضَّل إجراؤه على تثبيت تجريبي.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. مطابقة الأرقام مع عدّادات الواجهات

أرقام التدفقات تقديرية: عدد الحزم المأخوذة عينةً مضروبًا في معدّل أخذ
العينات. يقارنها traffic66 بعدّادات الواجهات في الجهاز نفسه ويعرض الفرق في
**مطابقة الواجهات**، مع السبب المرجّح حين يكون الفرق أكبر مما يفسّره أخذ
العينات وحده.

للحصول على عدّادات للمقارنة:

- أجهزة sFlow ترسلها من تلقاء نفسها عند ضبط فاصل العدّادات
  (`sflow counter interval 30` وما يماثله).
- لأجهزة NetFlow وIPFIX، أضف سطر `snmp` في **المصادر ← الأسماء**
  (انظر [الأسماء](#7-names-snmp-and-your-own-networks)). عندها يقرأ
  traffic66 عدّادات الواجهات كل دقيقة.

ما يفعله traffic66 أصلًا لتتطابق الأرقام: يستخدم معدّل أخذ العينات الذي
طبّقه الجهاز فعلًا، ويحتجز سجلات NetFlow/IPFIX حتى يُعرف المعدّل، ويعوّض
حزم التصدير المفقودة في الطريق، ويوزّع التدفقات الطويلة على الدقائق التي
استغرقتها، ويضيف 18 بايت لكل حزمة كحمل Ethernet إضافي إلى عدد بايتات
NetFlow/IPFIX (عدّادات الواجهات تتضمنه، وعدّ التدفقات على مستوى IP لا
يتضمنه؛ يُغيَّر بـ `-l2-overhead`).

الأسباب الشائعة لأي فرق متبقٍّ، وكلها تظهر في **مطابقة الواجهات**: بعض
الواجهات لا تؤخذ منها عينات، أو تؤخذ عينات الحركة نفسها على واجهتين، أو
تضيع حزم التصدير قبل وصولها إلى traffic66، أو أن معدّل أخذ العينات لم
يُعرف بعد.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. الأسماء وSNMP وشبكاتك الخاصة

تقبل **المصادر ← الأسماء** في واجهة الويب إدخالًا واحدًا في كل سطر. يُحفظ
المحتوى باسم `inventory.txt` في دليل البيانات، لذا يمكنك تعديل هذا الملف
مباشرة أيضًا (انظر `inventory.txt.example`). كل سطر اختياري.

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

- `net`: النطاقات الخاصة (10/8 و172.16/12 و192.168/16 و100.64/10) تُعدّ
  دائمًا نطاقاتك. أضف نطاقاتك العامة كي تُحتسب الحركة منها وإليها لك أيضًا؛
  ويظهر الاسم في **Top-N ← المقاطع** وفي مسارات الحركة.
- `snmp <device> <community> [<management address>[:port]]`: الجهاز هو
  العنوان الذي تأتي منه التدفقات. أضف عنوان الإدارة إذا كان الجهاز يجيب
  على SNMP من عنوان آخر. تُستخدم أوصاف الواجهات المقروءة عبر SNMP أسماءً
  لها ما لم تُسمِّ الواجهة بـ `iface`. اسمح لجهاز traffic66 في قائمة وصول
  SNMP على الجهاز.
- تُطبَّق التغييرات عند النقر على **حفظ**؛ لا حاجة إلى إعادة التشغيل.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. الدول والشبكات وقوائم التهديدات

تحتاج أسماء الدول والشبكات (AS) إلى جدول IP-to-ASN. نزّل الجدول المجاني من
[iptoasn.com](https://iptoasn.com):

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

يصلح أي ملف بالتنسيق نفسه (مفصول بعلامات الجدولة: العنوان الأول، العنوان
الأخير، رقم AS، رمز الدولة، اسم AS؛ نص عادي أو gzip). أعد تشغيل traffic66
بعد استبداله؛ ونزّل نسخة جديدة كل شهر تقريبًا.

قوائم التهديدات ملفات نصية عادية فيها عنوان أو شبكة في كل سطر (يُتجاهل
النص بعد `#` أو `;`)، تُحفظ في `<data directory>/threats/<name>.txt`،
مثلًا:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

أعد تشغيل traffic66 بعد إضافة القوائم أو تعديلها. تظهر التطابقات في
**معلومات التهديدات** مصنّفة حسب اسم القائمة.

<a id="9-using-the-web-ui"></a>

## 9. استخدام واجهة الويب

نادرًا ما ستحتاج إلى الكتابة. كل قيمة في كل صفحة — عنوان، منفذ، تطبيق،
دولة، جهاز — قابلة للنقر:

- **اعرض هذا فقط** / **استبعد هذا** يضيف مرشّحًا. تظهر المرشّحات أسفل
  الشريط العلوي وتنطبق على كل الصفحات حتى تزيلها.
- **اعرض سجلات تدفقه** يفتح التدفقات الفردية المطابقة.
- **ابحث عنه على الإنترنت** يفتح العنوان أو AS في موقع استعلام عام.
- **نسخ** ينسخ القيمة.

الصفحات:

| الصفحة | ما الذي تجيب عنه |
|---|---|
| نظرة عامة | حجم الحركة الآن ومقارنةً بالأسبوع الماضي حسب التطبيق؛ ما الذي زاد؛ أبرز العملاء والخدمات |
| Top-N | أعلى 66 من العملاء أو الخوادم أو المحادثات أو التطبيقات أو المنافذ أو الدول أو الشبكات أو المقاطع أو الأجهزة أو أنواع التغليف أو شبكات VLAN |
| مسارات الحركة | أي مقطع يتواصل مع أي تطبيق في أي دولة |
| الجغرافيا والشبكات | الحركة حسب الدولة وحسب الشبكة (AS) |
| معلومات التهديدات | المضيفات التي تواصلت مع عناوين في قوائم تهديداتك، وكم أرسلت |
| سجلات التدفق | التدفقات الفردية، الأحدث أولًا، مع أعمدة قابلة للاختيار |
| مطابقة الواجهات | أرقام التدفقات بجوار عدّادات الواجهات، الأسوأ أولًا، مع الأسباب |
| المصادر | الأجهزة، وأخذ العينات، والفقد، والمستقبِلات، وSNMP، و**الأسماء** |

فوق الصفحات: النطاق الزمني (من 15 دقيقة إلى 30 يومًا)، ومربع بحث اختياري،
وتحديث تلقائي كل 30 ثانية، و**نسخ الرابط** الذي ينسخ رابطًا إلى العرض
الحالي بالضبط (الصفحة والنطاق الزمني والمرشّحات) لإرساله إلى زميل. تتبع
اللغة إعدادات المتصفح؛ ويمكن تغييرها من أسفل القائمة.

تُبنى Top-N للنطاقات الزمنية الطويلة من ملخّصات ساعية؛ والمرشّحات غير متاحة
فيها، وتنبّه الصفحة إلى ذلك. اختر نطاقًا أقصر لتتمكن من التصفية.

<a id="10-terminal-ui"></a>

## 10. الواجهة الطرفية

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

على جهاز traffic66، يسجّل `traffic66 tui` الدخول تلقائيًا إذا استطاع قراءة
دليل البيانات (مرّر `-data` إن لم يكن الدليل الافتراضي). إذا كان traffic66
يعمل باسم مستخدم آخر، كما هو الحال مع الخدمة، فاستخدم `-user` و
`-password` بدلًا من ذلك. يختار `-lang` اللغة (`en`، `zh`، `hi`،
`es`، `ar`، `fr`، `bn`، `pt`، `ru`، `id`، `ur`، `ja`، `ko`).

المفاتيح: 1–8 للصفحات، ↑↓ للتحديد، Enter لإجراءات القيمة المحددة، f لعرض
هذا فقط، x للاستبعاد، / للبحث، t للنطاق الزمني، c لمسح المرشّحات، w لفتح
العرض نفسه في المتصفح، q للخروج.

<a id="11-local-capture"></a>

## 11. الالتقاط المحلي

إلى جانب استقبال التدفقات المُصدَّرة، يستطيع traffic66 تكوين التدفقات بنفسه
من الحزم على واجهة شبكة محلية، كمنفذ مرآة (SPAN) مثلًا:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: يتطلب صلاحيات root، أو القدرات `CAP_NET_RAW` و`CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`،
  أو سطر `AmbientCapabilities` في وحدة systemd أعلاه).
- macOS: يتطلب root (أجهزة BPF)؛ لا شيء يحتاج إلى تثبيت.
- Windows: ثبّت [Npcap](https://npcap.com) أولًا.

تُدرج الواجهات الملتقَطة في **المصادر**. الحزم التي تُرى مرتين (على منفذَي
مرآة مثلًا) تُعدّ مرتين.

<a id="12-options"></a>

## 12. الخيارات

يعرض `traffic66 -h` و`traffic66 <command> -h` كل الخيارات.

الأوامر:

| الأمر | |
|---|---|
| `traffic66` | جمع التدفقات وتقديم واجهة الويب |
| `traffic66 demo` | الشيء نفسه مع شبكة محاكاة |
| `traffic66 tui` | واجهة طرفية لنسخة traffic66 عاملة |
| `traffic66 passwd` | تعيين كلمة مرور لتسجيل الدخول |
| `traffic66 simulate -to HOST` | إرسال تدفقات محاكاة إلى مستقبِل |
| `traffic66 interfaces` | سرد الواجهات المتاحة للالتقاط المحلي |
| `traffic66 version` | طباعة الإصدار |

خيارات `traffic66` و`traffic66 demo`:

| الخيار | القيمة الافتراضية | |
|---|---|---|
| `-addr` | `:8066` | عنوان واجهة الويب؛ `127.0.0.1:8066` لهذا الجهاز فقط |
| `-data` | `traffic66-data` بجوار البرنامج | دليل البيانات |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | مستقبِلات UDP بصيغة `name=address` مفصولة بفواصل؛ القيمة الفارغة تعطّلها |
| `-user` | `admin` | المستخدم لأول كلمة مرور مولَّدة ولـ `-password` |
| `-password` | كلمة المرور المخزّنة | كلمة مرور لهذا التشغيل فقط (أو `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | عدد أيام الاحتفاظ بتفاصيل التدفقات؛ تُحفظ الملخّصات 400 يوم |
| `-memory` | `0.10` | حصة الذاكرة الفعلية التي يمكن لقاعدة البيانات استخدامها |
| `-l2-overhead` | `18` | بايتات تُضاف لكل حزمة إلى عدد بايتات NetFlow/IPFIX |
| `-sampling-wait` | `5m` | مدة انتظار السجلات لمعدّل أخذ العينات |
| `-capture` | | الالتقاط على واجهة محلية (يمكن تكراره) |
| `-inventory` | `<data>/inventory.txt` | ملف الأسماء |
| `-asn` | `<data>/asn.tsv.gz` | جدول IP-to-ASN |
| `-threat` | `<data>/threats/*.txt` | قائمة تهديدات إضافية بصيغة `name=path` (يمكن تكراره) |
| `-dns-upstream` | محلّل النظام | خادم DNS لعرض أسماء المضيفات |
| `-dns-rate` | `20` | الحد الأقصى لاستعلامات DNS العكسية في الثانية |
| `-dns-cache` | `2m` | مدة الاحتفاظ بأسماء المضيفات في الذاكرة المؤقتة |
| `-no-dns` | | بلا استعلامات DNS عكسية |
| `-tui` | | فتح الواجهة الطرفية أيضًا |

مثال: منفذ مستقبِل ثانٍ، وتفاصيل لمدة سنة، وواجهة الويب على الجهاز المحلي
فقط:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. البيانات والنسخ الاحتياطي والترقية وإزالة التثبيت

يحتوي دليل البيانات على كل شيء:

| | |
|---|---|
| `raw/` | تفاصيل التدفقات، ملف مضغوط واحد لكل ساعة |
| `traffic66.duckdb` | الملخّصات وعدّادات الواجهات والساعة الجارية |
| `password` | كلمات مرور تسجيل الدخول (مُجزّأة) |
| `inventory.txt` | الأسماء (**المصادر ← الأسماء**) |
| `asn.tsv.gz`، `threats/` | جداول الاستعلام التي أضفتها |

- **النسخ الاحتياطي**: أوقف traffic66 وانسخ الدليل. من دون إيقافه، انسخ
  `raw/` و`password` و`inventory.txt`؛ وعندها لن تتضمن النسخة الساعة
  الجارية ولا الملخّصات.
- **النقل**: أوقف traffic66، وانقل الدليل، ثم شغّله مع `-data` مشيرًا إلى
  المكان الجديد.
- **الترقية**: أوقف traffic66، واستبدل ملف البرنامج، ثم شغّله مجددًا. تبقى
  البيانات كما هي. مثلًا في Linux:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **إزالة التثبيت**: أوقف الخدمة أو مهمة بدء التشغيل واحذفها (انظر
  [التثبيت](#2-install))، ثم احذف مجلد البرنامج ودليل البيانات.

<a id="14-security"></a>

## 14. الأمان

- تستخدم واجهة الويب HTTP عاديًا: تعبر كلمات المرور والبيانات الشبكة دون
  تشفير. على الشبكات التي لا تثق بها تمامًا، استمع على هذا الجهاز فقط
  (`-addr 127.0.0.1:8066`) وضع أمامه وكيلًا عكسيًا بـ TLS، مثلًا باستخدام
  [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  أو اتصل به عبر VPN أو نفق SSH:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`، ثم افتح
  http://127.0.0.1:8066.
- اسمح بمنافذ UDP الخاصة بالمستقبِلات من عناوين أجهزتك فقط.
- تُخزَّن مجتمعات SNMP في `inventory.txt` نصًا صريحًا؛ استخدم مجتمعًا للقراءة
  فقط.

<a id="15-sizing"></a>

## 15. تقدير الموارد

القياس عند 5,000 تدفق في الثانية على جهاز بنواتين: تستهلك التفاصيل نحو
12 GB من القرص يوميًا، إضافة إلى نحو 1.5 GB للساعة الجارية، ويستهلك البرنامج
نحو 0.5 GB من الذاكرة وسدس نواة واحدة. تُبنى النظرات العامة للنطاقات
الزمنية الطويلة من الملخّصات وتستغرق أقل من 0.2 ثانية. تمسح الاستعلامات على
التفاصيل نحو 22 مليون صف لكل ساعة: مضيف واحد على مدى ساعة يستغرق أقل من
ثانية، وأعلى 66 لكل المحادثات على مدى ساعة نحو 9 ثوانٍ؛ ويزداد الوقت مع
طول النطاق ويقلّ بزيادة الأنوية.

لذلك يحتاج 30 يومًا عند 5,000 تدفق/ثانية إلى نحو 360 GB من القرص؛ احسبها
وفق معدّل التدفقات لديك (يظهر في **المصادر**) و`-retention-days`.

<a id="16-troubleshooting"></a>

## 16. استكشاف الأخطاء وإصلاحها

| العَرَض | السبب والحل |
|---|---|
| الجهاز غير ظاهر في **المصادر** | الحزم لا تصل: انظر [التحقق من وصول التدفقات](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | لم يرسل الجهاز خيارات أخذ العينات بعد؛ ومعظم الأجهزة تعيد إرسالها خلال دقائق. إن لم يفعل أبدًا، فصدّرها (`option sampler-table` على Cisco) أو علّم الجهاز بـ `unsampled` في الأسماء إن كان يصدّر فعلًا بنسبة 1:1 |
| الأرقام أقل من عدّادات الواجهات | انظر **مطابقة الواجهات**: فقد في الطريق، أو واجهات لا تؤخذ منها عينات، أو تدفقات ما زالت في ذاكرة الجهاز المؤقتة (مهلة التدفق النشط أطول من 60 ثانية) |
| الأرقام أعلى من عدّادات الواجهات | تؤخذ عينات الحركة نفسها على واجهتين أو جهازين |
| لا توجد دول أو شبكات | لا يوجد جدول IP-to-ASN: انظر [الدول](#8-countries-networks-and-threat-lists) |
| نسيت كلمة المرور | `traffic66 passwd -data <data directory>` على جهاز traffic66 |
| `Conflicting lock is held` | نسخة أخرى من traffic66 تستخدم دليل البيانات هذا |
| `receive buffer is only … KB` | يحدّ Linux من مخازن UDP المؤقتة: اضبط `net.core.rmem_max=16777216` (انظر [Linux](#linux)) |
| `cannot create the data directory` | مجلد البرنامج غير قابل للكتابة لهذا المستخدم: مرّر `-data` |
| macOS: "cannot be opened" أو "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" (حمى Windows جهازك) | **More info** (مزيد من المعلومات) → **Run anyway** (التشغيل على أي حال)؛ البرنامج غير موقّع بعد |
| الالتقاط في Windows: لم يُعثر على Npcap | ثبّت [Npcap](https://npcap.com) |
| `address already in use` | برنامج آخر يستخدم المنفذ: اختر منافذ أخرى بـ `-addr` أو `-listen` |

<a id="17-build-from-source"></a>

## 17. البناء من المصدر

Go 1.24 ومترجم C (gcc أو clang؛ وMinGW-w64 على Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```

</div>
