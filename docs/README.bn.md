[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | **বাংলা** | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

sFlow, NetFlow ও IPFIX-এর জন্য flow analytics, একটিমাত্র প্রোগ্রামে। এটি
switch, router ও firewall থেকে flow export সংগ্রহ করে, একটি embedded
database-এ রাখে, এবং দেখায় কে bandwidth ব্যবহার করছে, ট্রাফিক কোথায় যাচ্ছে
আর সংখ্যাগুলো ডিভাইসের নিজের interface counter-এর সাথে মেলে কি না — web UI
এবং terminal UI দুটোতেই।

- Windows, Linux ও macOS-এর জন্য একটিই executable। কোনো database ইনস্টল
  করতে হয় না, কোনো runtime লাগে না, offline-এও চলে।
- যেকোনো UDP port-এ sFlow v5, NetFlow v5, NetFlow v9 ও IPFIX; চাইলে কোনো
  network interface বা mirror port থেকে local capture-ও।
- নিজের সংখ্যাগুলো interface counter (sFlow counter বা SNMP)-এর সাথে মিলিয়ে
  দেখে, আর পার্থক্য থাকলে কেন তা জানায়।
- Top 66 তালিকা, flow-এর পথ, দেশ ও নেটওয়ার্ক, threat list-এর match, flow
  record, encapsulation (GRE, IPIP, VXLAN, GENEVE, MPLS)।
- web UI ও terminal UI-তে 13টি ভাষা।

<a id="contents"></a>

## সূচিপত্র

1. [ডেমো চালিয়ে দেখুন](#1-try-the-demo)
2. [ইনস্টল](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [সাইন ইন ও পাসওয়ার্ড](#3-sign-in-and-passwords)
4. [আপনার ডিভাইস থেকে flow পাঠান](#4-send-flows-from-your-devices)
5. [flow পৌঁছাচ্ছে কি না দেখুন](#5-check-that-flows-arrive)
6. [সংখ্যাগুলো interface counter-এর সাথে মেলান](#6-make-the-numbers-match-the-interface-counters)
7. [নাম, SNMP ও আপনার নিজের নেটওয়ার্ক](#7-names-snmp-and-your-own-networks)
8. [দেশ, নেটওয়ার্ক ও threat list](#8-countries-networks-and-threat-lists)
9. [web UI ব্যবহার](#9-using-the-web-ui)
10. [Terminal UI](#10-terminal-ui)
11. [Local capture](#11-local-capture)
12. [অপশন](#12-options)
13. [ডেটা, ব্যাকআপ, আপগ্রেড, আনইনস্টল](#13-data-backup-upgrade-uninstall)
14. [নিরাপত্তা](#14-security)
15. [সক্ষমতার হিসাব](#15-sizing)
16. [সমস্যা সমাধান](#16-troubleshooting)
17. [সোর্স থেকে বিল্ড](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. ডেমো চালিয়ে দেখুন

আপনার সিস্টেমের archive
[releases পেজ](https://github.com/githubflyideas/traffic66/releases) থেকে ডাউনলোড করুন:

| সিস্টেম | Archive |
|---|---|
| Windows 10/11, Server 2016 বা নতুন (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: RHEL/Rocky/Alma 8+, Ubuntu 18.04+, Debian 10+ | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: একই distribution | `traffic66-linux-arm64.tar.gz` |
| macOS 11 বা নতুন, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 বা নতুন, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (দ্বিতীয় লাইনটি macOS-কে ইন্টারনেট থেকে ডাউনলোড করা এমন প্রোগ্রাম চালাতে
দেয় যা App Store থেকে আসেনি):

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

http://127.0.0.1:8066 খুলুন এবং `admin` / `try66` দিয়ে সাইন ইন করুন। ডেমো
একটি ছোট কোম্পানির নেটওয়ার্ক তৈরি করে, যাতে এক দিনের ইতিহাস আর চারটি
simulated ডিভাইস থেকে live ট্রাফিক থাকে, সাথে খুঁজে বের করার মতো দুটি
incident: **সারসংক্ষেপ** থেকে শুরু করুন, **কার ট্রাফিক বেড়েছে** দেখুন, তারপর
ক্লিক করে করে এগোন। Ctrl+C দিয়ে বন্ধ করুন। ডেমোর ডেটা প্রোগ্রামের পাশে
`traffic66-demo`-তে থাকে; ডেমো নতুন করে শুরু করতে ওই ফোল্ডারটি মুছে দিন।

ডেমো আসল installation-এর মতো একই port ব্যবহার করে (8066, এবং UDP 6343,
2055, 4739)। আসলটির পাশাপাশি চালাতে অন্য port দিন:
`traffic66 demo -password try66 -addr :8067 -listen ""`।

Windows-এ আপনি সরাসরি `traffic66.exe`-এ double-click-ও করতে পারেন। এতে
traffic66 আসলভাবে (ডেমো নয়) চালু হয় এবং আপনার browser-এ web UI খোলে; প্রথম
চালুর পাসওয়ার্ড কালো window-তে দেখানো হয়, আর window বন্ধ করলে traffic66
থেমে যায়। Windows যদি "Windows protected your PC" (Windows আপনার PC সুরক্ষিত
করেছে) দেখায়, তাহলে **More info** (আরও তথ্য) → **Run anyway** (তবুও চালান)
ক্লিক করুন।

<a id="2-install"></a>

## 2. ইনস্টল

traffic66 একটিমাত্র ফাইল। ইনস্টল মানে: এটিকে কোথাও রাখা, একটি data
directory বেছে নেওয়া, পাসওয়ার্ড সেট করা, firewall খোলা এবং boot-এর সময় চালু
হওয়ার ব্যবস্থা করা। উদাহরণে traffic66 মেশিনের জন্য `192.0.2.50` আর router-এর
জন্য `192.0.2.1` ব্যবহার করা হয়েছে; এগুলো আপনার address দিয়ে বদলে নিন।

Port:

| Port | কাজ |
|---|---|
| UDP 6343 | sFlow (ডিফল্ট) |
| UDP 2055 | NetFlow (ডিফল্ট) |
| UDP 4739 | IPFIX (ডিফল্ট) |
| TCP 8066 | web UI ও API |

প্রতিটি UDP port সব protocol গ্রহণ করে, তাই সুবিধা হলে কোনো ডিভাইস NetFlow-ও
6343-এ পাঠাতে পারে। port বদলাতে বা যোগ করতে `-listen` ব্যবহার করুন।

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

শেষ কমান্ডটি ইউজার `admin`-এর পাসওয়ার্ড চায়।

`/etc/systemd/system/traffic66.service` তৈরি করুন:

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

চালু করুন এবং বড় UDP buffer-এর অনুমতি দিন যাতে burst-এর সময় packet না হারায়:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, firewalld দিয়ে (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

অথবা ufw দিয়ে (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

`C:\traffic66`-এ unpack করুন এবং পাসওয়ার্ড সেট করুন (PowerShell,
Administrator হিসেবে):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

ডেটা প্রোগ্রামের পাশে `C:\traffic66\traffic66-data`-তে যায়।

Firewall খুলুন:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

foreground-এ চালিয়ে দেখতে `C:\traffic66\traffic66.exe` চালান এবং Ctrl+C দিয়ে
বন্ধ করুন। boot থেকেই background-এ চালাতে, কেউ সাইন ইন না করলেও, এটিকে
startup task হিসেবে register করুন:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` জরুরি: এটি ছাড়া Windows তিন দিন পর
task থামিয়ে দেয়। থামাতে `Stop-ScheduledTask -TaskName
traffic66`, সরাতে `Unregister-ScheduledTask -TaskName traffic66`।

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

`/Library/LaunchDaemons/traffic66.plist` তৈরি করুন:

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

চালু করুন, আবার বন্ধ করুন:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

macOS firewall চালু থাকলে System Settings → Network → Firewall → Options-এ
traffic66-এর জন্য incoming connection-এর অনুমতি দিন।

<a id="3-sign-in-and-passwords"></a>

## 3. সাইন ইন ও পাসওয়ার্ড

`http://<traffic66 machine>:8066` খুলে সাইন ইন করুন। ইউজার `admin`, যদি না
আপনি অন্য কিছু বেছে নিয়ে থাকেন।

- প্রথমবার চালুর আগে পাসওয়ার্ড সেট না করলে traffic66 নিজেই একটি তৈরি করে
  এবং log-এ একবার প্রিন্ট করে:
  `first start: sign in as user "admin" with password "…"`।
  Linux-এ এটি খুঁজুন `journalctl -u traffic66 | grep "first start"` দিয়ে।
- পাসওয়ার্ড hash করে data directory-র `password` ফাইলে রাখা হয়। restart-এর
  পরেও একই থাকে।
- বদলাতে, বা ভুলে গেলে নতুন সেট করতে, traffic66 মেশিনে:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` একটি random পাসওয়ার্ড তৈরি করে প্রিন্ট করে।
  চলমান traffic66 পরের সাইন ইনেই নতুন পাসওয়ার্ড গ্রহণ করে; restart লাগে না।
- আরও ইউজার: `traffic66 passwd -data <data directory> -user alice`। সব
  ইউজার একই জিনিস দেখে।
- script ও container-এর জন্য, environment-এ `TRAFFIC66_PASSWORD=…` বা command
  line-এ `-password …` ওই run-এর জন্য সংরক্ষিত পাসওয়ার্ডের বদলে পাসওয়ার্ড সেট
  করে। environment-ই ভালো: command line মেশিনের অন্য ইউজাররা দেখতে পায়।

এক মিনিটের মধ্যে পাঁচবার ভুল পাসওয়ার্ড দিলে সেই address এক মিনিটের জন্য block
হয়।

<a id="4-send-flows-from-your-devices"></a>

## 4. আপনার ডিভাইস থেকে flow পাঠান

প্রতিটি ডিভাইসকে traffic66 মেশিনের দিকে point করুন। কমান্ড model ও software
version অনুযায়ী আলাদা হয়; ডিভাইসের manual দেখে নিন। সব উদাহরণে `192.0.2.50`
হলো traffic66 আর `192.0.2.1` ডিভাইসের নিজের address।

সাধারণ পরামর্শ:

- active flow timeout 60 সেকেন্ড রাখুন। বেশি timeout হলে ট্রাফিক দেরিতে, বড়
  বড় চাংকে আসে।
- ডিভাইস NetFlow/IPFIX sample করলে, তাকে sampler options export করতে দিন
  যাতে rate জানা যায়। traffic66 record-গুলো 1:1 গোনার বদলে rate না আসা পর্যন্ত
  ধরে রাখে।
- হয় সব interface sample করুন, নয়তো শুধু edge interface, এক দিকেই। একই
  ট্রাফিক ঢোকার আর বেরোনোর সময় দুবার sample করলে সেটি দুবার গোনা হয়;
  **ইন্টারফেস মিলানো** এটি ধরিয়ে দেয়।
- sFlow sampling rate: 1 Gb/s link-এ প্রায় 1:1000, 10 Gb/s-এ 1:4096,
  40/100 Gb/s-এ 1:8192।

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

FortiGate FortiOS 7.4.2 বা নতুন (NetFlow v9):

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

Linux server ও host, softflowd দিয়ে (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. flow পৌঁছাচ্ছে কি না দেখুন

**উৎস** খুলুন। যে ডিভাইস কিছু পাঠায়, সেটি কয়েক সেকেন্ডের মধ্যে দেখা যায়,
তার protocol, sampling rate, loss, শেষ packet আর status সহ। status সবুজ না
হলে পাশের লেখাটি বলে দেয় কী সমস্যা আর কী বদলাতে হবে।

কোনো ডিভাইস না দেখা গেলে:

1. traffic66 মেশিনে packet দেখুন (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`।
   সেখানে কিছু না থাকলে packet মেশিন পর্যন্ত পৌঁছাচ্ছেই না: ডিভাইসের
   configuration, routing আর পথের firewall পরীক্ষা করুন।
2. packet আসছে কিন্তু **উৎস** খালি: local firewall সেগুলো drop করছে
   ([ইনস্টল](#2-install) দেখুন), অথবা traffic66 অন্য port-এ শুনছে
   (`-listen`)।
3. কোনো ডিভাইস না ছুঁয়ে অন্য মেশিন থেকে পথটি পরীক্ষা করতে, সেখানে কয়েক
   সেকেন্ডের জন্য `traffic66 simulate -to 192.0.2.50` চালান। এটি simulated
   ডিভাইস থেকে sFlow, NetFlow ও IPFIX পাঠায়; সেগুলো তখন **উৎস**-এ ও ডেটায়
   দেখা যায়, তাই এর জন্য test installation ব্যবহার করাই ভালো।

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. সংখ্যাগুলো interface counter-এর সাথে মেলান

Flow-এর সংখ্যা আসলে অনুমান: sampled packet গুণ sampling rate। traffic66
এগুলো ডিভাইসের নিজের interface counter-এর সাথে তুলনা করে এবং পার্থক্য
**ইন্টারফেস মিলানো**-তে দেখায়; পার্থক্য শুধু sampling দিয়ে ব্যাখ্যা করা যায়
তার চেয়ে বেশি হলে সম্ভাব্য কারণও দেখায়।

তুলনার জন্য counter পেতে:

- sFlow ডিভাইস counter interval সেট থাকলে নিজেরাই সেগুলো পাঠায়
  (`sflow counter interval 30` বা এ জাতীয়)।
- NetFlow ও IPFIX ডিভাইসের জন্য **উৎস → নাম**-এ একটি `snmp` লাইন যোগ করুন
  ([নাম](#7-names-snmp-and-your-own-networks) দেখুন)। তখন traffic66 প্রতি
  মিনিটে interface counter পড়ে।

সংখ্যা মেলাতে traffic66 আগে থেকেই যা করে: ডিভাইস আসলে যে sampling rate
প্রয়োগ করেছে সেটিই ব্যবহার করে, sampling rate জানা না যাওয়া পর্যন্ত
NetFlow/IPFIX record ধরে রাখে, পথে হারানো export packet-এর ক্ষতিপূরণ করে,
লম্বা flow-কে যত মিনিট সেটি চলেছে সেই মিনিটগুলোতে ভাগ করে দেয়, এবং
NetFlow/IPFIX byte count-এ প্রতি packet-এ 18 byte Ethernet overhead যোগ করে
(interface counter-এ এটি থাকে, IP-layer flow count-এ থাকে না; `-l2-overhead`
দিয়ে বদলান)।

বাকি পার্থক্যের সাধারণ কারণ, সবই **ইন্টারফেস মিলানো**-তে জানানো হয়: কিছু
interface sample হচ্ছে না, একই ট্রাফিক দুটি interface-এ sample হচ্ছে, export
packet traffic66-এ পৌঁছানোর আগেই হারাচ্ছে, অথবা sampling rate এখনও জানা নেই।

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. নাম, SNMP ও আপনার নিজের নেটওয়ার্ক

web UI-র **উৎস → নাম** প্রতি লাইনে একটি entry নেয়। এটি data directory-তে
`inventory.txt` হিসেবে সেভ হয়, তাই আপনি ফাইলটি সরাসরিও edit করতে পারেন
(`inventory.txt.example` দেখুন)। প্রতিটি লাইন ঐচ্ছিক।

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

- `net`: private range (10/8, 172.16/12, 192.168/16, 100.64/10) সবসময় আপনার
  ধরা হয়। আপনার public range যোগ করুন যাতে সেগুলোতে আসা-যাওয়া ট্রাফিকও আপনার
  হিসেবে গোনা হয়; নামটি **Top-N → সেগমেন্ট**-এ এবং flow-এর পথে দেখা যায়।
- `snmp <device> <community> [<management address>[:port]]`: device হলো সেই
  address যেখান থেকে flow আসে। ডিভাইস অন্য address-এ SNMP-র উত্তর দিলে
  management address যোগ করুন। SNMP দিয়ে পড়া interface description নাম হিসেবে
  ব্যবহার হয়, যদি না আপনি `iface` দিয়ে interface-এর নাম দেন। ডিভাইসের SNMP
  access list-এ traffic66 মেশিনকে অনুমতি দিন।
- **সংরক্ষণ**-এ ক্লিক করলেই পরিবর্তন কার্যকর হয়; restart লাগে না।

<a id="8-countries-networks-and-threat-lists"></a>

## 8. দেশ, নেটওয়ার্ক ও threat list

দেশ ও নেটওয়ার্ক (AS)-এর নামের জন্য একটি IP-to-ASN table লাগে।
[iptoasn.com](https://iptoasn.com) থেকে বিনামূল্যের table ডাউনলোড করুন:

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

একই format-এর যেকোনো ফাইল চলবে (tab দিয়ে আলাদা: প্রথম address, শেষ address,
AS number, country code, AS name; plain বা gzip)। বদলানোর পর traffic66
restart করুন; মোটামুটি প্রতি মাসে নতুন একটি ডাউনলোড করুন।

Threat list হলো সাধারণ text ফাইল, প্রতি লাইনে একটি address বা নেটওয়ার্ক
(`#` বা `;`-এর পরের লেখা উপেক্ষা করা হয়), সেভ করা হয়
`<data directory>/threats/<name>.txt` হিসেবে, যেমন:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

list যোগ বা পরিবর্তনের পর traffic66 restart করুন। match-গুলো
**হুমকির তথ্য**-তে list-এর নাম অনুযায়ী দেখা যায়।

<a id="9-using-the-web-ui"></a>

## 9. web UI ব্যবহার

খুব কমই কিছু টাইপ করতে হবে। প্রতিটি পেজের প্রতিটি value — address, port,
application, দেশ, ডিভাইস — ক্লিক করা যায়:

- **শুধু এটি দেখান** / **এটি বাদ দিন** একটি filter যোগ করে। Filter top bar-এর
  নিচে দেখা যায় এবং সরিয়ে না দেওয়া পর্যন্ত প্রতিটি পেজে প্রযোজ্য থাকে।
- **এর ফ্লো রেকর্ড দেখুন** মিলে যাওয়া আলাদা আলাদা flow খোলে।
- **অনলাইনে খুঁজুন** address বা AS কোনো public lookup সাইটে খোলে।
- **কপি করুন** value-টি কপি করে।

পেজ:

| পেজ | কোন প্রশ্নের উত্তর দেয় |
|---|---|
| সারসংক্ষেপ | এখন কত ট্রাফিক আর গত সপ্তাহের তুলনায় কত, application অনুযায়ী; কী বেড়েছে; শীর্ষ client ও service |
| Top-N | client, server, conversation, application, port, দেশ, নেটওয়ার্ক, segment, ডিভাইস, encapsulation বা VLAN-এর শীর্ষ 66 |
| ট্রাফিকের পথ | কোন segment কোন দেশের কোন application-এর সাথে কথা বলে |
| ভূগোল ও নেটওয়ার্ক | দেশ ও নেটওয়ার্ক (AS) অনুযায়ী ট্রাফিক |
| হুমকির তথ্য | যেসব host আপনার threat list-এর address-এর সাথে কথা বলেছে, এবং কতটা পাঠিয়েছে |
| ফ্লো রেকর্ড | আলাদা আলাদা flow, নতুনগুলো আগে, বেছে নেওয়া যায় এমন column সহ |
| ইন্টারফেস মিলানো | interface counter-এর পাশে flow-এর সংখ্যা, সবচেয়ে খারাপগুলো আগে, কারণসহ |
| উৎস | ডিভাইস, sampling, loss, collector, SNMP, এবং **নাম** |

পেজগুলোর ওপরে: time range (15 মিনিট থেকে 30 দিন), একটি ঐচ্ছিক search box,
প্রতি 30 সেকেন্ডে automatic refresh, এবং **লিংক কপি করুন**, যা ঠিক বর্তমান
view-এর (পেজ, time range ও filter) link কপি করে, যাতে সহকর্মীকে পাঠানো যায়।
ভাষা browser অনুযায়ী ঠিক হয়; menu-র নিচ থেকে বদলান।

লম্বা time range-এ Top-N আসে ঘণ্টাভিত্তিক summary থেকে; সেখানে filter পাওয়া
যায় না, আর পেজেই তা বলা থাকে। filter করতে ছোট range বেছে নিন।

<a id="10-terminal-ui"></a>

## 10. Terminal UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

traffic66 মেশিনে `traffic66 tui` data directory পড়তে পারলে নিজেই সাইন ইন করে
নেয় (ডিফল্ট না হলে `-data` দিন)। traffic66 অন্য ইউজার হিসেবে চললে, যেমন
service হিসেবে চলে, তার বদলে `-user` ও `-password` ব্যবহার করুন। `-lang`
ভাষা বেছে নেয় (`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`)।

Key: 1–8 পেজ, ↑↓ বাছাই, Enter বাছাই করা value-র action, f শুধু এটি দেখান,
x বাদ দিন, / search, t time range, c filter মুছুন, w একই view browser-এ খুলুন,
q বেরিয়ে যান।

<a id="11-local-capture"></a>

## 11. Local capture

Flow export ছাড়াও traffic66 কোনো local network interface-এর packet থেকে
নিজেই flow তৈরি করতে পারে, যেমন একটি mirror (SPAN) port:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: root লাগে, অথবা `CAP_NET_RAW` ও `CAP_NET_ADMIN` capability
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`, অথবা
  ওপরের systemd unit-এর `AmbientCapabilities` লাইন)।
- macOS: root লাগে (BPF device); কিছু ইনস্টল করতে হয় না।
- Windows: আগে [Npcap](https://npcap.com) ইনস্টল করুন।

Capture করা interface **উৎস**-এ দেখা যায়। দুবার দেখা packet (যেমন দুটি mirror
port-এ) দুবার গোনা হয়।

<a id="12-options"></a>

## 12. অপশন

`traffic66 -h` ও `traffic66 <command> -h` সবকিছুর তালিকা দেয়।

কমান্ড:

| কমান্ড | |
|---|---|
| `traffic66` | flow সংগ্রহ করে ও web UI চালায় |
| `traffic66 demo` | একই, একটি simulated নেটওয়ার্ক সহ |
| `traffic66 tui` | চলমান traffic66-এর জন্য terminal UI |
| `traffic66 passwd` | login পাসওয়ার্ড সেট করে |
| `traffic66 simulate -to HOST` | কোনো collector-এ simulated export পাঠায় |
| `traffic66 interfaces` | local capture-এর জন্য interface-এর তালিকা |
| `traffic66 version` | version প্রিন্ট করে |

`traffic66` ও `traffic66 demo`-এর অপশন:

| অপশন | ডিফল্ট | |
|---|---|---|
| `-addr` | `:8066` | web UI-র address; শুধু এই মেশিনের জন্য `127.0.0.1:8066` |
| `-data` | প্রোগ্রামের পাশে `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collector, `name=address` আকারে, comma দিয়ে আলাদা; খালি রাখলে বন্ধ |
| `-user` | `admin` | প্রথম generated পাসওয়ার্ড ও `-password`-এর ইউজার |
| `-password` | সংরক্ষিত পাসওয়ার্ড | শুধু এই run-এর পাসওয়ার্ড (`TRAFFIC66_PASSWORD`-ও) |
| `-retention-days` | `30` | কত দিনের flow detail রাখা হবে; summary 400 দিন রাখা হয় |
| `-memory` | `0.10` | physical memory-র কত অংশ database ব্যবহার করতে পারবে |
| `-l2-overhead` | `18` | NetFlow/IPFIX byte count-এ প্রতি packet-এ যোগ করা byte |
| `-sampling-wait` | `5m` | record কতক্ষণ sampling rate-এর জন্য অপেক্ষা করবে |
| `-capture` | | local interface-এ capture (একাধিকবার দেওয়া যায়) |
| `-inventory` | `<data>/inventory.txt` | নামের ফাইল |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table |
| `-threat` | `<data>/threats/*.txt` | অতিরিক্ত threat list, `name=path` আকারে (একাধিকবার দেওয়া যায়) |
| `-dns-upstream` | system resolver | host name দেখানোর জন্য DNS server |
| `-dns-rate` | `20` | প্রতি সেকেন্ডে সর্বোচ্চ reverse lookup |
| `-dns-cache` | `2m` | host name কতক্ষণ cache-এ থাকবে |
| `-no-dns` | | কোনো reverse lookup নয় |
| `-tui` | | সাথে terminal UI-ও খোলে |

উদাহরণ: দ্বিতীয় একটি collector port, এক বছরের detail, আর web UI শুধু local
মেশিনে:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. ডেটা, ব্যাকআপ, আপগ্রেড, আনইনস্টল

সবকিছু data directory-তে থাকে:

| | |
|---|---|
| `raw/` | flow detail, প্রতি ঘণ্টায় একটি compressed ফাইল |
| `traffic66.duckdb` | summary, interface counter ও চলতি ঘণ্টা |
| `password` | login পাসওয়ার্ড (hashed) |
| `inventory.txt` | নাম (**উৎস → নাম**) |
| `asn.tsv.gz`, `threats/` | আপনার যোগ করা lookup table |

- **ব্যাকআপ**: traffic66 বন্ধ করে directory কপি করুন। বন্ধ না করে কপি করলে
  `raw/`, `password` ও `inventory.txt` কপি করুন; তখন চলতি ঘণ্টা আর summary বাদ
  পড়ে।
- **সরানো**: traffic66 বন্ধ করুন, directory সরান, নতুন জায়গার দিকে নির্দেশ করা
  `-data` দিয়ে চালু করুন।
- **আপগ্রেড**: traffic66 বন্ধ করুন, প্রোগ্রাম ফাইল বদলান, আবার চালু করুন।
  ডেটা থেকে যায়। যেমন Linux-এ:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **আনইনস্টল**: service বা startup task বন্ধ করে সরিয়ে দিন
  ([ইনস্টল](#2-install) দেখুন), তারপর প্রোগ্রাম ফোল্ডার ও data directory মুছে
  দিন।

<a id="14-security"></a>

## 14. নিরাপত্তা

- web UI সাধারণ HTTP ব্যবহার করে: পাসওয়ার্ড ও ডেটা encryption ছাড়াই নেটওয়ার্ক
  দিয়ে যায়। যে নেটওয়ার্কে পুরো ভরসা নেই, সেখানে শুধু এই মেশিনে শুনুন
  (`-addr 127.0.0.1:8066`) এবং সামনে একটি TLS reverse proxy বসান, যেমন
  [Caddy](https://caddyserver.com) দিয়ে:
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`।
  অথবা VPN বা SSH tunnel দিয়ে ঢুকুন:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, তারপর
  http://127.0.0.1:8066 খুলুন।
- UDP collector port শুধু আপনার ডিভাইসগুলোর address থেকে অনুমতি দিন।
- `inventory.txt`-এ SNMP community সাধারণ লেখায় (clear text) থাকে; read-only
  community ব্যবহার করুন।

<a id="15-sizing"></a>

## 15. সক্ষমতার হিসাব

2-core মেশিনে প্রতি সেকেন্ডে 5,000 flow-এ মাপা হয়েছে: detail প্রতিদিন প্রায়
12 GB disk নেয়, সাথে চলতি ঘণ্টার জন্য প্রায় 1.5 GB; প্রোগ্রাম নেয় প্রায় 0.5 GB
memory আর একটি core-এর ছয় ভাগের এক ভাগ। লম্বা time range-এর overview আসে
summary থেকে এবং 0.2 s-এর কম সময় নেয়। detail-এর ওপর query প্রতি ঘণ্টায় প্রায়
2 কোটি 20 লাখ row scan করে: 1 ঘণ্টায় একটি host 1 s-এর কম, সব conversation-এর
1 ঘণ্টার Top 66 প্রায় 9 s; সময় range-এর সাথে বাড়ে আর বেশি core-এ কমে।

তাই 5,000 flows/s-এ 30 দিনের জন্য disk প্রায় 360 GB; আপনার flow rate
(**উৎস**-এ দেখা যায়) ও `-retention-days` অনুযায়ী আনুপাতিক হিসাব করুন।

<a id="16-troubleshooting"></a>

## 16. সমস্যা সমাধান

| লক্ষণ | কারণ ও সমাধান |
|---|---|
| ডিভাইস **উৎস**-এ নেই | packet পৌঁছাচ্ছে না: [flow পৌঁছাচ্ছে কি না দেখুন](#5-check-that-flows-arrive) দেখুন |
| "waiting for the sampling rate" | ডিভাইস এখনও sampler options পাঠায়নি; বেশিরভাগই কয়েক মিনিটের মধ্যে আবার পাঠায়। কখনো না পাঠালে সেগুলো export করান (Cisco-তে `option sampler-table`) অথবা সত্যিই 1:1 হলে নাম-এ সেটিকে `unsampled` চিহ্নিত করুন |
| সংখ্যা interface counter-এর চেয়ে কম | **ইন্টারফেস মিলানো** দেখুন: পথে loss, interface sample হচ্ছে না, অথবা flow এখনও ডিভাইসের cache-এ (active timeout 60 s-এর বেশি) |
| সংখ্যা interface counter-এর চেয়ে বেশি | একই ট্রাফিক দুটি interface বা দুটি ডিভাইসে sample হচ্ছে |
| কোনো দেশ বা নেটওয়ার্ক নেই | IP-to-ASN table নেই: [দেশ](#8-countries-networks-and-threat-lists) দেখুন |
| পাসওয়ার্ড ভুলে গেছেন | traffic66 মেশিনে `traffic66 passwd -data <data directory>` |
| `Conflicting lock is held` | অন্য একটি traffic66 ইতিমধ্যে এই data directory ব্যবহার করছে |
| `receive buffer is only … KB` | Linux UDP buffer সীমিত রাখে: `net.core.rmem_max=16777216` সেট করুন ([Linux](#linux) দেখুন) |
| `cannot create the data directory` | এই ইউজারের জন্য প্রোগ্রাম ফোল্ডার writable নয়: `-data` দিন |
| macOS: "cannot be opened" বা "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" (Windows আপনার PC সুরক্ষিত করেছে) | **More info** (আরও তথ্য) → **Run anyway** (তবুও চালান); প্রোগ্রামটি এখনও signed নয় |
| Linux: `GLIBC_2.xx not found` | distribution-টি RHEL 8 / Ubuntu 18.04 / Debian 10-এর চেয়ে পুরোনো |
| Windows capture: Npcap পাওয়া যায়নি | [Npcap](https://npcap.com) ইনস্টল করুন |
| `address already in use` | অন্য কোনো প্রোগ্রাম port-টি ব্যবহার করছে: `-addr` বা `-listen` দিয়ে অন্য port বেছে নিন |

<a id="17-build-from-source"></a>

## 17. সোর্স থেকে বিল্ড

Go 1.24 এবং একটি C compiler (gcc বা clang; Windows-এ MinGW-w64):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
