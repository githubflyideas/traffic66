[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | **বাংলা** | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

sFlow, NetFlow ও IPFIX-এর জন্য flow analytics, একটিমাত্র প্রোগ্রামে: কে
bandwidth ব্যবহার করছে, ট্রাফিক কোথায় যাচ্ছে, আর সংখ্যাগুলো ডিভাইসের নিজের
interface counter-এর সাথে মেলে কি না — web UI ও terminal UI-তে।

- Windows, Linux ও macOS-এর জন্য একটিই executable; কোনো database ইনস্টল করতে হয় না, offline-এও চলে।
- যেকোনো UDP port-এ sFlow v5, NetFlow v5/v9 ও IPFIX, অথবা কোনো interface থেকে local capture।
- নিজের সংখ্যাগুলো interface counter (sFlow বা SNMP)-এর সাথে মিলিয়ে দেখে এবং পার্থক্য থাকলে কেন তা জানায়।
- scan, পাসওয়ার্ড অনুমান, lateral movement, অস্বাভাবিক upload, flood ও threat list-এর ট্রাফিক খুঁজে বের করে, sampling-এর মধ্যেও।
- `traffic66 capture.pcap` কোনো সেটআপ ছাড়াই প্যাকেট ক্যাপচার বিশ্লেষণ করে।
- 13টি ভাষা। মূল্যায়নের জন্য এবং 100 জনের কম মানুষের প্রতিষ্ঠানের জন্য বিনামূল্যে ([লাইসেন্স](#licence))।

![সারসংক্ষেপ: খোলা সন্দেহজনক কার্যকলাপ, গতকাল একই সময়ের তুলনায় application অনুযায়ী bandwidth, শীর্ষ client ও service](images/overview.png)

<sub>সব স্ক্রিনশট `traffic66 demo` থেকে নেওয়া, একটি simulated কোম্পানি নেটওয়ার্ক।</sub>

<a id="contents"></a>

## সূচিপত্র

1. [ডেমো চালিয়ে দেখুন](#1-try-the-demo)
2. [ইনস্টল](#2-install)
3. [ইউজার ও পাসওয়ার্ড](#3-users-and-passwords)
4. [আপনার ডিভাইস থেকে flow পাঠান](#4-send-flows-from-your-devices)
5. [flow পৌঁছাচ্ছে কি না দেখুন](#5-check-that-flows-arrive)
6. [Interface ও counter](#6-interfaces-and-counters)
7. [নাম, দেশ ও threat list](#7-names-countries-and-threat-lists)
8. [web UI ব্যবহার](#8-using-the-web-ui)
9. [অফলাইন pcap, terminal UI, local capture](#9-offline-pcap-terminal-ui-local-capture)
10. [অপশন ও ডেটা](#10-options-and-data)
11. [নিরাপত্তা, সক্ষমতার হিসাব, সমস্যা সমাধান](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. ডেমো চালিয়ে দেখুন

আপনার সিস্টেমের archive
[releases পেজ](https://github.com/githubflyideas/traffic66/releases) থেকে
ডাউনলোড করুন (Windows x64, kernel 3.2+ সহ Linux x86-64/ARM64, macOS 11+),
unpack করে চালান:

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

macOS-এ আগে `xattr -dr com.apple.quarantine <folder>` চালান।
http://127.0.0.1:8066 খুলে `admin` / `try66` দিয়ে সাইন ইন করুন: এক দিনের
ইতিহাস আর চারটি simulated ডিভাইস থেকে live ট্রাফিক, সাথে একটি আক্রমণ যা
**সন্দেহজনক কার্যকলাপ**-এ ধাপে ধাপে দেখানো হয়। Ctrl+C দিয়ে বন্ধ করুন; নতুন
করে শুরু করতে `traffic66-demo` মুছে দিন। আসল installation-এর পাশাপাশি চালাতে:
`-addr :8067 -listen ""`।

<a id="2-install"></a>

## 2. ইনস্টল

traffic66 একটিমাত্র ফাইল। Port: UDP 6343 (sFlow), 2055 (NetFlow), 4739
(IPFIX), TCP 8066 (web UI); প্রতিটি UDP port সব protocol গ্রহণ করে।

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

**Windows** (PowerShell, Administrator হিসেবে): `C:\traffic66`-এ unpack করুন,
`C:\traffic66\traffic66.exe passwd` চালান, port খুলুন এবং boot-এর সময় চালু
হওয়ার ব্যবস্থা করুন:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`traffic66.exe`-এ double-click করলেও চলে: এটি web UI খোলে এবং প্রথম
পাসওয়ার্ড তার window-তে দেখায়।

**macOS**: `/usr/local/traffic66`-এ unpack করুন, quarantine flag সরান,
`traffic66 passwd -data "/Library/Application Support/traffic66"` চালান, এবং
একটি LaunchDaemon থেকে চালু করুন যার `ProgramArguments` হলো প্রোগ্রাম,
`-data` ও ওই directory, সাথে `RunAtLoad` ও `KeepAlive`।

<a id="3-users-and-passwords"></a>

## 3. ইউজার ও পাসওয়ার্ড

প্রথমবার চালু হলে traffic66 একটি random পাসওয়ার্ডসহ ইউজার `admin` তৈরি করে
এবং সেটি একবার দেখায় (window-তে, terminal-এ, অথবা
`journalctl -u traffic66 | grep "first start"`)। ইউজাররা data directory-র
`password`-এ salted hash হিসেবে থাকে এবং traffic66 মেশিনে একটি command দিয়ে
পরিচালিত হয় (traffic66 `-data …` দিয়ে চললে সেটিও যোগ করুন):

| কাজ | Command |
|---|---|
| `admin`-এর পাসওয়ার্ড বদলানো | `traffic66 passwd` |
| `alice` যোগ করা বা তার পাসওয়ার্ড বদলানো | `traffic66 passwd -user alice` |
| `alice` মুছে ফেলা | `traffic66 passwd -user alice -delete` |
| ইউজারদের তালিকা দেখা | `traffic66 passwd -list` |

পরিবর্তন সঙ্গে সঙ্গে কার্যকর হয়। সব ইউজারের অধিকার একই। script ও
container-এর জন্য `TRAFFIC66_PASSWORD=…` (বা `-password`) ওই run-এ শুধু ওই
পাসওয়ার্ডসহ `-user`-কে গ্রহণ করে। এক মিনিটের মধ্যে পাঁচবার ভুল পাসওয়ার্ড দিলে
সেই address এক মিনিটের জন্য block হয়।

<a id="4-send-flows-from-your-devices"></a>

## 4. আপনার ডিভাইস থেকে flow পাঠান

`192.0.2.50` হলো traffic66, `192.0.2.1` ডিভাইস। active timeout 60 সেকেন্ড
রাখুন, NetFlow/IPFIX ডিভাইসকে তাদের sampler options export করতে দিন, এবং
**প্রতিটি interface inbound** sample করুন (অথবা শুধু edge interface): তাহলে
প্রতিটি packet একবারই গোনা হয়। sFlow rate: 1 Gb/s-এ প্রায় 1:1000, 10 Gb/s-এ
1:4096, 40/100 Gb/s-এ 1:8192।

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

## 5. flow পৌঁছাচ্ছে কি না দেখুন

**সেটিংস** কয়েক সেকেন্ডের মধ্যে কিছু পাঠানো প্রতিটি ডিভাইস দেখায়: protocol,
sampling rate, loss, যে interface-গুলো এটি sample করে, এবং status সবুজ না হলে
কী ঠিক করতে হবে। sFlow-এর loss দুই ভাগে দেখানো হয়: পথে হারানো
(`netstat -su` buffer error দেখালে `net.core.rmem_max` বাড়ান) এবং ডিভাইস
নিজেই যে sample ফেলে দিয়েছে।

![সেটিংস: প্রতিটি ডিভাইস, তার protocol, sampling, loss আর কী ঠিক করতে হবে](images/sources.png)

কোনো ডিভাইস নেই? `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739` চালান: সেখানে কিছু না থাকলে সমস্যা routing, firewall বা
ডিভাইসের configuration-এ; packet আছে কিন্তু **সেটিংস**-এ কিছু নেই মানে local
firewall বা `-listen`। অন্য মেশিন থেকে `traffic66 simulate -to 192.0.2.50`
simulated ডিভাইস দিয়ে পথটি পরীক্ষা করে।

<a id="6-interfaces-and-counters"></a>

## 6. Interface ও counter

Flow-এর সংখ্যা অনুমান (sample × sampling rate)। **ইন্টারফেস মিলানো** এগুলো
ডিভাইসের interface counter-এর সাথে তুলনা করে (sFlow counter, অথবা নাম-এ একটি
`snmp` লাইনের মাধ্যমে SNMP) এবং পার্থক্য কেন তা জানায়: কিছু interface sample
হচ্ছে না, একই ট্রাফিক দুবার sample হচ্ছে, পথে loss, অথবা sampling rate অজানা।
প্রতিটি interface-এর একটি bits/s ও একটি packets/s chart আছে, ingress সবুজ ও
egress নীল, counter ড্যাশ রেখায়।

প্রতিটি row-এ **✎** একটি নাম ও একটি ছোট ট্যাগ (যেমন *uplink*) দেয়, আর **☆**
এটিকে ডিফল্ট interface (★) করে, যেটিতে পেজগুলো খোলে।

যে ডিভাইস শুধু কিছু interface sample করে, সেটি ওই flow-গুলোর অন্য প্রান্তও
দেখায়। এই **বিপরীত ইন্টারফেস** তালিকার শেষে ছোট ধূসর লেখায় দেখানো হয়: তাদের
সংখ্যায় শুধু স্যাম্পল করা interface দিয়ে যাওয়া ট্রাফিক থাকে। স্যাম্পল করা
interface জানা যায় sFlow-এর data source বা flowDirection field (IPFIX 61)
থেকে; তা না থাকলে, ডিভাইসের 90% ট্রাফিকে থাকা interface।

![ইন্টারফেস মিলানো: প্রতিটি interface-এর ট্রাফিক, আর ডিভাইসের counter-এর পাশে flow-এর অনুমান](images/interfaces.png)

traffic66 আগে থেকেই ডিভাইস যে rate প্রয়োগ করেছে সেটি ব্যবহার করে, অজানা rate-এর
জন্য অপেক্ষা করে, export loss-এর ক্ষতিপূরণ করে, লম্বা flow-কে তাদের মিনিটগুলোতে
ভাগ করে দেয়, এবং NetFlow/IPFIX-এ প্রতি packet-এ 18 byte Ethernet overhead যোগ
করে (`-l2-overhead`)।

<a id="7-names-countries-and-threat-lists"></a>

## 7. নাম, দেশ ও threat list

যেকোনো address-এ ক্লিক করে **নাম দিন…** বেছে নিন, অথবা **সেটিংস → নাম**
ব্যবহার করুন। নামগুলো data directory-র `inventory.txt`-এ থাকে:

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

private range সবসময় আপনার ধরা হয়। **সংরক্ষণ**-এ পরিবর্তন কার্যকর হয়, restart
লাগে না।

দেশ ও নেটওয়ার্ক (AS) শুরু থেকেই কাজ করে DB-IP-এর বিনামূল্যের Lite
ডেটাবেস দিয়ে ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **সেটিংস** সেগুলো আপডেট
করে, অথবা তার বদলে MaxMind GeoLite2, IPinfo Lite বা IPtoASN ফাইল নেয়। মানচিত্রের
সীমানা: [Natural Earth](https://www.naturalearthdata.com)।

![ভূগোল ও নেটওয়ার্ক: বিশ্ব মানচিত্রে দেশ অনুযায়ী বাইরের ট্রাফিক](images/geo.png)

Threat list হলো text ফাইল, প্রতি লাইনে একটি address বা নেটওয়ার্ক,
`<data>/threats/<name>.txt`-এ (যেমন Spamhaus DROP); বদলানোর পর restart করুন।
match-গুলো **হুমকির তথ্য**-তে দেখা যায়।

![হুমকির তথ্য: একটি internal host একটি threat list-এর address-এ ডেটা পাঠাচ্ছে](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. web UI ব্যবহার

প্রতিটি পেজের প্রতিটি value ক্লিক করা যায়: **শুধু এটি দেখান** / **এটি বাদ
দিন** (filter প্রতিটি পেজে প্রযোজ্য), **এর ফ্লো রেকর্ড দেখুন**, **বিস্তারিত
দেখুন** (একটি host বা service সম্পর্কে একটি পেজ), **নাম দিন…**, **অনলাইনে
খুঁজুন**, **কপি করুন**।

| পেজ | কী দেখায় |
|---|---|
| সারসংক্ষেপ | বেছে নেওয়া interface-এর bandwidth; গতকাল বা গত সপ্তাহের তুলনায় application অনুযায়ী ট্রাফিক (মোট, আগত বা বহির্গামী); খোলা সন্দেহজনক কার্যকলাপ; শীর্ষ client ও service |
| শীর্ষ 66 | শীর্ষ 66টি conversation, যেকোনো column অনুযায়ী sort করা যায়, অথবা application, নেটওয়ার্ক, segment, ডিভাইস, encapsulation, VLAN অনুযায়ী গ্রুপ করা; শীর্ষ 30 সবচেয়ে সক্রিয় host |
| ট্রাফিকের বিস্তারিত | Ring chart: server ও তাদের client (বা উল্টোটা), এবং service |
| ট্রাফিকের পথ | host → application → দেশ, অথবা client → service → server, অথবা নেটওয়ার্ক অনুযায়ী |
| ইন্টারফেস মিলানো | সময়ের সাথে প্রতিটি interface তার counter-এর বিপরীতে; নাম, ট্যাগ, ডিফল্ট |
| ফ্লো রেকর্ড | আলাদা আলাদা flow, প্রতি 5 সেকেন্ডে live বা যেকোনো সময়সীমার জন্য |
| সন্দেহজনক কার্যকলাপ, হুমকির তথ্য | কীসে নজর দিতে হবে ([নিচে](#findings)); তালিকাভুক্ত address-এর সাথে ট্রাফিক |
| ভূগোল ও নেটওয়ার্ক | দেশ অনুযায়ী বিশ্ব মানচিত্র, সময়ের সাথে নেটওয়ার্ক (AS) |
| সেটিংস | ডিভাইস, sampling, loss, SNMP, ডেটাবেস, লোগো, নাম |
| অফলাইন pcap বিশ্লেষণ, ডেটা পরিষ্কার | ক্যাপচার ফাইল ([নিচে](#9-offline-pcap-terminal-ui-local-capture)); পুরোনো ডেটা মোছা |

পেজগুলোর ওপরে: **ইন্টারফেস** (সব, অথবা একটি স্যাম্পল করা interface; তখন
ট্রাফিক পেজগুলো শুধু সেটির মধ্য দিয়ে যাওয়া ট্রাফিক দেখায়), time range (15
মিনিট থেকে 30 দিন, বা নিজের মতো), প্রতি 30 s-এ refresh এবং ঠিক বর্তমান view-এর
জন্য **লিংক কপি করুন**। ভাষা ও পাঁচটি রঙের থিম menu-র নিচে থাকে। Chart
সেখানেই শেষ হয় যেখানে ডেটা সম্পূর্ণ: NetFlow/IPFIX-এ ডিভাইসগুলো যতটা দেরিতে
export করে ততটা (সর্বোচ্চ 2 মিনিট)। 6 ঘণ্টার চেয়ে লম্বা range পূর্ণ ঘণ্টা থেকে
শুরু হয়; 7 বা 30 দিনের জন্য একটি interface flow detail পড়ে, তাই বেশি সময় নেয়
এবং ততদূর পেছনে যায় যতদিন detail রাখা হয়।

![শীর্ষ 66: শীর্ষ 66টি conversation, যেকোনো column অনুযায়ী সাজানো](images/topn.png)

![ট্রাফিকের বিস্তারিত: server ও তাদের client, service ও তাদের server, ring chart হিসেবে](images/traffic.png)

![একটি host-এর বিস্তারিত: তার সম্পর্কে পাওয়া সন্দেহজনক কার্যকলাপ, তার ট্রাফিক, সে কার সাথে কথা বলে, service, দেশ ও সর্বশেষ flow](images/detail.png)

![ট্রাফিকের পথ: কোন host কোন দেশের দিকে কোন application ব্যবহার করে](images/paths.png)

![চীনা ভাষায় সারসংক্ষেপ](images/overview-zh.png)

<a id="findings"></a>

### সন্দেহজনক কার্যকলাপ

প্রতি 5 মিনিটে শেষ 10 মিনিট পরীক্ষা করা হয়; যা এক ঘণ্টা ধরে চলে তা একটিই
এন্ট্রি যা বাড়তে থাকে।

| কার্যকলাপ | এর মানে |
|---|---|
| স্ক্যান, পোর্ট স্ক্যান | এক port-এ অনেক host-এ, অথবা একটি host-এর অনেক port-এ ছোট probe |
| পাসওয়ার্ড অনুমান | একটি login service-এ অনেক ছোট connection |
| ল্যাটেরাল মুভমেন্ট | এমন internal host-এ file sharing বা remote administration, যারা আগে কখনো তা দেয়নি |
| অস্বাভাবিক আপলোড | একটি নতুন address-এ 10 মিনিটে 100 MB, ফিরে আসা ডেটার তিন গুণ |
| ফ্লাড | একটি address-এ প্রতি সেকেন্ডে 20,000+ ছোট packet, তার স্বাভাবিক হারের দশ গুণ |
| হুমকি তালিকা | তালিকাভুক্ত address-এর সাথে ট্রাফিক |

আপনার নেটওয়ার্কের ভেতর থেকে হলে এগুলো উচ্চ, ইন্টারনেট থেকে হলে নিম্ন।
**সমাধান হয়েছে** এন্ট্রিটি বন্ধ করে, **সমস্যা নয়** এটিকে চিরতরে বন্ধ করে।
ল্যাটেরাল মুভমেন্ট ও আপলোডের জন্য এক দিনের ইতিহাস লাগে। 1:4096 sampling-এর
মধ্য দিয়েও ডেমোর আক্রমণ পুরোটাই ধরা পড়ে; খুব ছোট scan sampling-এর পেছনে
লুকিয়ে থাকতে পারে।

![সন্দেহজনক কার্যকলাপ: একটি আক্রমণের প্রতিটি ধাপ, 1:4096 sFlow sampling-এর মধ্য দিয়েও ধরা পড়েছে](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. অফলাইন pcap, terminal UI, local capture

**অফলাইন pcap বিশ্লেষণ** প্যাকেট ক্যাপচার (pcap, pcapng) একই পেজে দেখায়, লাইভ
ডেটা থেকে আলাদা রেখে: `traffic66 a.pcap b.pcapng` 127.0.0.1-এ চালু হয় এবং
browser খোলে (সর্বোচ্চ 3টি ফাইল, 3 GB; Ctrl+C ইমপোর্ট করা ডেটা মুছে দেয়),
অথবা ওই পেজে 50 MB পর্যন্ত 3টি ফাইল আপলোড করুন। এটি flow নিয়ে কাজ করে,
প্যাকেটের বিষয়বস্তু নিয়ে নয়।

![অফলাইন বিশ্লেষণ: ক্যাপচার ফাইল, তাদের প্যাকেট, ফ্লো ও সময়](images/sandbox.png)

**Terminal UI**: traffic66 মেশিনে `traffic66 tui`, অথবা
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`।
Key: 1–8 পেজ, Enter action, f শুধু এটি দেখান, x বাদ দিন, t time range, w
browser-এ খুলুন, q বেরিয়ে যান; `-lang` ভাষা বেছে নেয়।

![Terminal UI: সারসংক্ষেপ](images/tui-overview.png)

![Terminal UI: শীর্ষ 66 conversation](images/tui-topn.png)

**Local capture** একটি local interface থেকে flow তৈরি করে, সবচেয়ে ভালো হয়
switch-এর mirror port-এ যুক্ত একটি port: `traffic66 interfaces` সেগুলোর তালিকা
দেয়, `-capture eth1` (অথবা Windows-এর নাম বা নম্বর) capture করে। Linux-এ root
বা `setcap cap_net_raw,cap_net_admin+ep` লাগে, macOS-এ root, Windows-এ
[Npcap](https://npcap.com)। capture করা flow ডিভাইস `127.0.0.1` থেকে আসে।

<a id="10-options-and-data"></a>

## 10. অপশন ও ডেটা

`traffic66 -h` সবকিছুর তালিকা দেয়। সবচেয়ে বেশি ব্যবহৃত:

| অপশন | ডিফল্ট | |
|---|---|---|
| `-data` | প্রোগ্রামের পাশে `traffic66-data` | data directory |
| `-addr` | `:8066` | web UI; শুধু এই মেশিনের জন্য `127.0.0.1:8066` |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collector; খালি রাখলে বন্ধ |
| `-retention-days` | `30` | কত দিনের flow detail; summary 400 দিন রাখা হয় |
| `-memory` | `0.10` | database cache-এর জন্য RAM-এর অংশ |
| `-sampling-wait` | `5m` | record কতক্ষণ sampling rate-এর জন্য অপেক্ষা করবে |
| `-capture` | | local interface (একাধিকবার দেওয়া যায়) |
| `-no-dns` | | কোনো reverse lookup নয় |

data directory-তে থাকে `raw/` (detail, প্রতি ঘণ্টায় একটি ফাইল),
`traffic66.duckdb` (summary ও counter), `password`, `inventory.txt`,
`license.json`, আপনার লোগো ও ডেটাবেস। ব্যাকআপ: traffic66 বন্ধ করে directory কপি
করুন; আপগ্রেড: প্রোগ্রাম ফাইল বদলে দিন। **ডেটা পরিষ্কার** 7–120 দিনের চেয়ে
পুরোনো ডেটা, বা সব ডেটা মুছে দেয়।

<a id="licence"></a>

### লাইসেন্স

[PolyForm Noncommercial License 1.0.0](../LICENSE.md) এবং
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md)-এর অধীনে source
available (ইংরেজি পাঠ বাধ্যতামূলক): মূল্যায়নের জন্য এবং 100 জনের কম মানুষের
প্রতিষ্ঠানের জন্য বিনামূল্যে; বড় প্রতিষ্ঠান production ব্যবহারের 30 দিন পরে
নিবন্ধন করে; বিক্রি করা, অন্যদের জন্য host করা বা প্রতিদ্বন্দ্বী পণ্যের জন্য
বাণিজ্যিক লাইসেন্স লাগে। কোনো কিছুই কখনো বন্ধ করা হয় না। প্রতিটি পেজের নিচে 8
অঙ্কের ইনস্টলেশন নম্বর দেখায়; সেটি লেখককে পাঠান, আর যে `license.json` ফেরত
পাবেন তা data directory-তে রাখুন। যোগাযোগ:
<https://github.com/githubflyideas/traffic66>।

<a id="11-security-sizing-troubleshooting"></a>

## 11. নিরাপত্তা, সক্ষমতার হিসাব, সমস্যা সমাধান

web UI সাধারণ HTTP: যে নেটওয়ার্কে ভরসা নেই সেখানে `-addr 127.0.0.1:8066`
ব্যবহার করুন, একটি TLS proxy-র পেছনে (`caddy reverse-proxy --from
traffic66.example.com --to 127.0.0.1:8066`) অথবা SSH tunnel দিয়ে। UDP port শুধু
আপনার ডিভাইসগুলো থেকে অনুমতি দিন। SNMP community সাধারণ লেখায় (clear text)
রাখা হয়; read-only community ব্যবহার করুন।

2 core-এ প্রতি সেকেন্ডে 5,000 flow-এ: detail-এর প্রতিদিন প্রায় 12 GB disk (30
দিনে 360 GB), একটি core-এর ছয় ভাগের এক ভাগ, 0.6–0.8 GB memory। লম্বা range-এর
overview 0.2 s-এর কম সময় নেয়; সব conversation-এর 1 ঘণ্টার Top 66 প্রায় 9 s।

| লক্ষণ | সমাধান |
|---|---|
| "waiting for the sampling rate" | sampler options export করান, অথবা device লাইনে `sampling=N` / `unsampled` |
| counter-এর চেয়ে কম | interface sample হচ্ছে না, loss, অথবা active timeout 60 s-এর বেশি |
| counter-এর চেয়ে বেশি | একই ট্রাফিক দুটি interface বা ডিভাইসে sample হচ্ছে |
| পাসওয়ার্ড ভুলে গেছেন | traffic66 মেশিনে `traffic66 passwd` |
| `Conflicting lock is held` | অন্য একটি traffic66 এই data directory ব্যবহার করছে |
| `address already in use` | `-addr` বা `-listen` দিয়ে অন্য port বেছে নিন |
| Windows "protected your PC" | **More info** (আরও তথ্য) → **Run anyway** (তবুও চালান) |

সোর্স থেকে বিল্ড: Go 1.24 ও একটি C compiler, তারপর `scripts/build.sh 0.1.0 traffic66`।
