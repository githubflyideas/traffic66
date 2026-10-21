[English](MANUAL.md) | [中文](MANUAL.zh.md) | **हिन्दी** | [Español](MANUAL.es.md) | [العربية](MANUAL.ar.md) | [Français](MANUAL.fr.md) | [বাংলা](MANUAL.bn.md) | [Português](MANUAL.pt.md) | [Русский](MANUAL.ru.md) | [Bahasa Indonesia](MANUAL.id.md) | [اردو](MANUAL.ur.md) | [Deutsch](MANUAL.de.md) | [日本語](MANUAL.ja.md) | [Tiếng Việt](MANUAL.vi.md) | [한국어](MANUAL.ko.md)

# traffic66 — NetFlow, sFlow और IPFIX कलेक्टर तथा ट्रैफ़िक एनालाइज़र

sFlow, NetFlow और IPFIX के लिए flow analytics, एक ही प्रोग्राम में: bandwidth
कौन इस्तेमाल कर रहा है, ट्रैफ़िक कहाँ जा रहा है, और क्या आँकड़े डिवाइसों के
अपने interface counters से मेल खाते हैं — web UI में भी और terminal UI में भी।

ntopng, ElastiFlow, Grafana के साथ pmacct, या PRTG और SolarWinds NTA के flow
modules का self-hosted विकल्प — नेटवर्क मॉनिटरिंग (network monitoring),
bandwidth मॉनिटरिंग (bandwidth monitoring), सबसे ज़्यादा ट्रैफ़िक वाले hosts
(top talkers), DDoS और scan की पहचान, और pcap विश्लेषण के लिए, बिना
Elasticsearch, Kafka या अलग database के।

- Windows, Linux और macOS के लिए एक ही executable; कोई database इंस्टॉल नहीं करना, offline भी चलता है।
- किसी भी UDP port पर sFlow v5, NetFlow v5/v9 और IPFIX, या किसी interface से local capture।
- अपने आँकड़ों को interface counters (sFlow या SNMP) से मिलाकर जाँचता है और बताता है कि फ़र्क़ क्यों है।
- scans, पासवर्ड का अनुमान, lateral movement, असामान्य uploads, floods और threat list वाला ट्रैफ़िक ढूँढता है, sampling के बावजूद भी।
- `traffic66 capture.pcap` बिना किसी सेटअप के पैकेट कैप्चर का विश्लेषण करता है।
- 15 भाषाएँ। evaluation के लिए और 100 से कम लोगों वाले संगठनों के लिए मुफ़्त ([लाइसेंस](#licence))।

![सारांश: खुली संदिग्ध गतिविधियाँ, कल इसी समय की तुलना में application के हिसाब से bandwidth, top clients और services](images/overview.png)

<sub>सभी स्क्रीनशॉट `traffic66 demo` से लिए गए हैं, एक simulated कंपनी नेटवर्क।</sub>

<a id="contents"></a>

## विषय-सूची

1. [क्विक स्टार्ट](#1-quick-start)
2. [उदाहरण](#2-examples)
3. [यूज़र और पासवर्ड](#3-users-and-passwords)
4. [अपने डिवाइसों से flows भेजें](#4-send-flows-from-your-devices)
5. [जाँचें कि flows पहुँच रहे हैं](#5-check-that-flows-arrive)
6. [Interfaces और counters](#6-interfaces-and-counters)
7. [नाम, देश और threat lists](#7-names-countries-and-threat-lists)
8. [web UI का इस्तेमाल](#8-using-the-web-ui)
9. [ऑफ़लाइन pcap, terminal UI, local capture](#9-offline-pcap-terminal-ui-local-capture)
10. [विकल्प और डेटा](#10-options-and-data)
11. [सुरक्षा, क्षमता का अनुमान, समस्या निवारण](#11-security-sizing-troubleshooting)

<a id="1-quick-start"></a>

<a id="1-try-the-demo"></a>

## 1. क्विक स्टार्ट

1. अपने सिस्टम का आर्काइव डाउनलोड करें: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. खोलें और प्रोग्राम चलाएँ:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: `traffic66-windows-amd64.zip` खोलें और `traffic66.exe` चलाएँ (डबल-क्लिक भी चलेगा)।

macOS: पहले उस फ़ोल्डर में `xattr -dr com.apple.quarantine .` चलाएँ।

फिर:

- Web UI: `http://<सर्वर IP>:8066`। यह सभी नेटवर्क इंटरफ़ेस पर सुनता है।
- साइन इन: यूज़र `admin`, पासवर्ड `traffic66`। पहली बार साइन इन पर नया पासवर्ड सेट करना होता है।
- डिवाइसों को सर्वर IP पर भेजने को कहें: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (सभी इंटरफ़ेस; हर पोर्ट हर प्रोटोकॉल लेता है)।

<a id="2-examples"></a>

<a id="2-install"></a>

## 2. उदाहरण

<a id="command-cheat-sheet"></a>

### कमांड चीट शीट

| काम | कमांड |
|---|---|
| sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 लें (डिफ़ॉल्ट) | `./traffic66` |
| सिर्फ़ sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| सिर्फ़ NetFlow v5/v9 (2055) और IPFIX (4739) | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| दूसरे UDP पोर्ट | `./traffic66 -listen netflow=:9995,sflow=:6344` |
| collectors एक ही IP पर | `./traffic66 -listen sflow=10.0.0.5:6343,netflow=10.0.0.5:2055,ipfix=10.0.0.5:4739` |
| Web UI किसी IP और पोर्ट पर | `./traffic66 -addr 10.0.0.5:8080` |
| Web UI सिर्फ़ इसी मशीन पर | `./traffic66 -addr 127.0.0.1:8066` |
| Web UI HTTPS पर | `./traffic66 -tls` |
| डेटा डायरेक्टरी | `./traffic66 -data /var/lib/traffic66` |
| flow विवरण 7 दिन रखें | `./traffic66 -retention-days 7` |
| मेमोरी: डेटाबेस के लिए 20% RAM | `./traffic66 -memory 0.20` |
| लोकल कैप्चर के लिए नेटवर्क कार्ड सूची | `./traffic66 interfaces` |
| इसी मशीन का NIC कैप्चर करें (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| इसी मशीन का NIC कैप्चर करें (macOS) | `sudo ./traffic66 -capture en0 -listen ""` |
| इसी मशीन का NIC कैप्चर करें (Windows, Npcap चाहिए) | `.\traffic66.exe -capture Wi-Fi -listen ""` |
| admin पासवर्ड सेट/रीसेट करें (-data DIR हो तो जोड़ें) | `./traffic66 passwd` |
| admin का रैंडम पासवर्ड सेट करके दिखाएँ | `./traffic66 passwd -generate` |
| यूज़र जोड़ें या पासवर्ड बदलें | `./traffic66 passwd -user alice` |
| यूज़र सूची | `./traffic66 passwd -list` |
| सिर्फ़ इस रन के लिए पासवर्ड (स्क्रिप्ट, कंटेनर) | `TRAFFIC66_PASSWORD='S3cret-pass' ./traffic66` |
| traffic66 मशीन पर Terminal UI | `./traffic66 tui` |
| रिमोट सर्वर से Terminal UI | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |
| Terminal UI की भाषा | `./traffic66 tui -lang ja` |
| कैप्चर फ़ाइल का विश्लेषण | `./traffic66 capture.pcap` |
| collector को परीक्षण flows भेजें | `./traffic66 simulate -to 192.0.2.50` |
| सिम्युलेटेड डिवाइसों वाला डेमो | `./traffic66 demo` |
| सभी विकल्प | `./traffic66 serve -h` |

Web UI की भाषा: मेनू के नीचे (कोई फ़्लैग नहीं)। बाकी: `traffic66 serve -h`।

<a id="device-configuration"></a>

### डिवाइस कॉन्फ़िगरेशन

`192.0.2.50` traffic66 है, `192.0.2.1` डिवाइस; timeout और sampling rate: [सेक्शन 4](#4-send-flows-from-your-devices)।

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

### फ़ायरवॉल खोलें

```
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload   # RHEL, Fedora
sudo ufw allow proto udp to any port 6343,2055,4739 && sudo ufw allow 8066/tcp                          # Ubuntu, Debian
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow   # Windows
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

### Web UI सिर्फ़ इसी मशीन पर, SSH से

```
./traffic66 -addr 127.0.0.1:8066                 # on the server
ssh -L 8066:127.0.0.1:8066 user@192.0.2.50        # on your PC, then open http://127.0.0.1:8066
```

### सर्विस के रूप में: Linux (systemd)

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

### सर्विस के रूप में: Windows

```
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

### सर्विस के रूप में: macOS

`/usr/local/traffic66` में खोलें, quarantine फ़्लैग हटाएँ, `traffic66 passwd -data "/Library/Application Support/traffic66"` चलाएँ, और ऐसे LaunchDaemon से शुरू करें जिसके `ProgramArguments` प्रोग्राम, `-data` और वह डायरेक्टरी हों, `RunAtLoad` और `KeepAlive` के साथ।

<a id="3-users-and-passwords"></a>

## 3. यूज़र और पासवर्ड

नए installation में `admin` / `traffic66` से साइन इन होता है, और पहला साइन इन
कुछ भी और दिखाने से पहले नया पासवर्ड माँगता है (demo `traffic66` ही रखता है)।
इसके बाद menu के नीचे **खाता** में आप अपना पासवर्ड बदलते हैं; administrator
(`admin`) वहीं यूज़र्स जोड़ और हटा भी सकता है और उनके पासवर्ड reset कर सकता है।
LDAP / Active Directory से साइन इन विकास में है।

यूज़र data directory में `password` में salted hashes के रूप में रखे जाते हैं।
यही काम traffic66 मशीन पर एक command से भी किया जा सकता है (जब traffic66
`-data …` के साथ चलता है तो उसे जोड़ें):

| काम | Command |
|---|---|
| `admin` का पासवर्ड बदलें | `traffic66 passwd` |
| `alice` जोड़ें या उसका पासवर्ड बदलें | `traffic66 passwd -user alice` |
| `alice` को हटाएँ | `traffic66 passwd -user alice -delete` |
| यूज़र्स की सूची देखें | `traffic66 passwd -list` |

बदलाव तुरंत लागू होते हैं। यूज़र्स के प्रबंधन को छोड़कर सभी यूज़र्स के अधिकार
एक जैसे हैं। scripts और containers के लिए `TRAFFIC66_PASSWORD=…` (या `-password`) उस run के लिए केवल
`-user` को उसी पासवर्ड के साथ स्वीकार करता है। एक मिनट में पाँच ग़लत पासवर्ड
होने पर उस address को एक मिनट के लिए block कर दिया जाता है।

<a id="4-send-flows-from-your-devices"></a>

## 4. अपने डिवाइसों से flows भेजें

`192.0.2.50` traffic66 है, `192.0.2.1` डिवाइस। active timeout 60 सेकंड रखें,
NetFlow/IPFIX डिवाइसों को उनके sampler options export करने दें, और **हर
interface को inbound** sample करें (या केवल edge interfaces को): तब हर packet
एक ही बार गिना जाता है। sFlow rates: 1 Gb/s के लिए लगभग 1:1000, 10 Gb/s के
लिए 1:4096, 40/100 Gb/s के लिए 1:8192।

Cisco, Huawei, H3C, Juniper, Arista, MikroTik और softflowd का कॉन्फ़िगरेशन: [उदाहरण → डिवाइस कॉन्फ़िगरेशन](#device-configuration)।

<a id="5-check-that-flows-arrive"></a>

## 5. जाँचें कि flows पहुँच रहे हैं

**सेटिंग्स** कुछ ही सेकंड में हर उस डिवाइस को दिखाता है जो कुछ भी भेजता है:
protocol, sampling rate, loss, वह कौन-से interfaces sample करता है, और status
हरा न हो तो क्या ठीक करना है। sFlow loss को दो हिस्सों में बाँटा जाता है:
रास्ते में loss (अगर `netstat -su` में buffer errors दिखें तो
`net.core.rmem_max` बढ़ाएँ) और वे samples जो ख़ुद डिवाइस ने छोड़ दिए।

![सेटिंग्स: हर डिवाइस, उसका protocol, sampling, loss और क्या ठीक करना है](images/sources.png)

कोई डिवाइस नहीं दिख रहा? `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739` चलाएँ: वहाँ कुछ नहीं है तो समस्या routing, किसी firewall या
डिवाइस के configuration में है; packets वहाँ हैं पर **सेटिंग्स** में कुछ नहीं, तो
local firewall या `-listen`। किसी दूसरी मशीन से `traffic66 simulate -to 192.0.2.50`
simulated डिवाइसों से रास्ता टेस्ट करता है।

<a id="6-interfaces-and-counters"></a>

## 6. Interfaces और counters

Flow के आँकड़े अनुमान होते हैं (samples × sampling rate)। **इंटरफ़ेस मिलान**
उन्हें डिवाइस के interface counters (sFlow counters, या नाम में एक `snmp` लाइन
के ज़रिए SNMP) से मिलाता है और बताता है कि फ़र्क़ क्यों है: interfaces sample
नहीं हो रहे, एक ही ट्रैफ़िक दो बार sample हुआ, रास्ते में loss, या अज्ञात
sampling rate। हर interface का एक bits/s chart और एक packets/s chart होता है,
ingress हरा और egress नीला, counters dashed।

हर row पर **✎** एक नाम और एक छोटा टैग (जैसे *uplink*) सेट करता है, और **☆**
उसे डिफ़ॉल्ट interface (★) बनाता है, जिस पर पेज खुलते हैं।

जो डिवाइस केवल कुछ interfaces sample करता है, वह उन flows के दूसरे सिरे भी
दिखाता है। ये **सामने वाले इंटरफ़ेस** सबसे आख़िर में छोटे धूसर अक्षरों में
दिखते हैं: उनमें केवल सैंपल किए गए interface से होकर गया ट्रैफ़िक होता है।
सैंपल किया गया interface sFlow के data source या flowDirection field (IPFIX
61) से पता चलता है; इसके बिना, वह interface जो डिवाइस के 90% ट्रैफ़िक में है।

![इंटरफ़ेस मिलान: हर interface का ट्रैफ़िक, और डिवाइस के counter के बगल में flow का अनुमान](images/interfaces.png)

traffic66 पहले से डिवाइस द्वारा लगाया गया rate इस्तेमाल करता है, अज्ञात rates
का इंतज़ार करता है, export loss की भरपाई करता है, लंबे flows को उनके मिनटों में
बाँटता है, और NetFlow/IPFIX में हर packet पर 18 bytes का Ethernet overhead
जोड़ता है (`-l2-overhead`)।

<a id="7-names-countries-and-threat-lists"></a>

## 7. नाम, देश और threat lists

किसी भी address पर क्लिक करें और **नाम दें…** चुनें, या **इन्वेंटरी**
इस्तेमाल करें। नाम data directory में `inventory.txt` में रखे जाते हैं:

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

Private ranges हमेशा आपके माने जाते हैं। बदलाव **सहेजें** पर लागू होते हैं,
restart की ज़रूरत नहीं।

देश और नेटवर्क (AS) DB-IP के मुफ़्त Lite databases के साथ शुरू से ही काम करते
हैं ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **सेटिंग्स** उन्हें
अपडेट करता है, या उनकी जगह MaxMind GeoLite2, IPinfo Lite या IPtoASN फ़ाइलें
लेता है। मानचित्र की सीमाएँ: [Natural Earth](https://www.naturalearthdata.com)।

![भूगोल और नेटवर्क: विश्व मानचित्र पर देश के अनुसार बाहरी ट्रैफ़िक](images/geo.png)

Threat lists text फ़ाइलें होती हैं, हर लाइन में एक address या नेटवर्क,
`<data>/threats/<name>.txt` में (उदाहरण के लिए Spamhaus DROP); बदलने के बाद
restart करें। Matches **ख़तरे की जानकारी** पर दिखते हैं।

![ख़तरे की जानकारी: एक internal host जो threat list के किसी address को डेटा भेज रहा है](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. web UI का इस्तेमाल

हर पेज की हर value पर क्लिक किया जा सकता है: **केवल यही दिखाएँ** / **इसे
हटाएँ** (filters हर पेज पर लागू होते हैं), **इसके फ़्लो रिकॉर्ड देखें**,
**विवरण देखें** (किसी एक host या service के बारे में पेज), **नाम दें…**,
**ऑनलाइन खोजें**, **कॉपी करें**।

| पेज | क्या दिखाता है |
|---|---|
| सारांश | चुने हुए interface की bandwidth; application के हिसाब से ट्रैफ़िक (कुल, अंदर आने वाला या बाहर जाने वाला) कल या पिछले हफ़्ते की तुलना में; खुली संदिग्ध गतिविधियाँ; top clients और services |
| शीर्ष 66 | top 66 conversations, किसी भी column से sort होने वाली, या application, नेटवर्क, segment, डिवाइस, encapsulation, VLAN के अनुसार समूह; top 30 सबसे सक्रिय hosts |
| ट्रैफ़िक विवरण | Ring charts: servers और उनके clients (या उल्टा), और services |
| ट्रैफ़िक के रास्ते | Host → application → देश, या client → service → server, या नेटवर्क के अनुसार |
| इंटरफ़ेस मिलान | समय के साथ हर interface, उसके counters की तुलना में; नाम, टैग, डिफ़ॉल्ट |
| फ़्लो रिकॉर्ड | अलग-अलग flows, हर 5 सेकंड पर live या किसी भी समय-सीमा के लिए |
| संदिग्ध गतिविधि, ख़तरे की जानकारी | किस पर ध्यान देना है ([नीचे](#findings)); सूचीबद्ध addresses के साथ ट्रैफ़िक |
| भूगोल और नेटवर्क | देश के अनुसार विश्व मानचित्र, समय के साथ नेटवर्क (AS) |
| सेटिंग्स | डिवाइस, sampling, loss, SNMP, databases, लोगो, नाम |
| ऑफ़लाइन pcap विश्लेषण, डेटा सफ़ाई | कैप्चर फ़ाइलें ([नीचे](#9-offline-pcap-terminal-ui-local-capture)); पुराना डेटा हटाना |

पेजों के ऊपर: **इंटरफ़ेस** (सभी, या एक सैंपल interface; तब ट्रैफ़िक पेज सिर्फ़
उससे होकर गुज़रने वाला ट्रैफ़िक दिखाते हैं), समय-सीमा (15 मिनट से 30 दिन, या
कस्टम), हर 30 s पर refresh, और ठीक इसी view के लिए **लिंक कॉपी करें**। भाषा और
पाँच रंग थीम menu के नीचे हैं। Charts वहीं तक जाते हैं जहाँ तक डेटा पूरा है:
NetFlow/IPFIX के साथ उतनी देर तक जितनी देर से डिवाइस export करते हैं (ज़्यादा
से ज़्यादा 2 मिनट)। 6 घंटे से लंबी ranges पूरे घंटे से शुरू होती हैं; 7 या 30
दिन के लिए एक interface flow detail पढ़ता है, इसलिए वह धीमा है और उतना ही पीछे
जाता है जितने समय तक detail रखा जाता है।

![शीर्ष 66: top 66 conversations, किसी भी column से sort होने वाली](images/topn.png)

![ट्रैफ़िक विवरण: ring charts के रूप में servers अपने clients के साथ, और services अपने servers के साथ](images/traffic.png)

![एक host का विवरण: उसके बारे में मिली संदिग्ध गतिविधियाँ, उसका ट्रैफ़िक, वह किससे बात करता है, services, देश और नवीनतम flows](images/detail.png)

![ट्रैफ़िक के रास्ते: कौन-सा host किस देश की ओर किस application का इस्तेमाल करता है](images/paths.png)

![चीनी में सारांश](images/overview-zh.png)

<a id="findings"></a>

### संदिग्ध गतिविधि

हर 5 मिनट पर पिछले 10 मिनट की जाँच होती है; जो चीज़ एक घंटे तक चलती है वह एक
ही प्रविष्टि है जो बढ़ती जाती है।

| गतिविधि | मतलब |
|---|---|
| स्कैन, पोर्ट स्कैन | एक port पर कई hosts को, या एक host के कई ports को छोटे probes |
| पासवर्ड का अनुमान | किसी login service से कई छोटे connections |
| लेटरल मूवमेंट | ऐसे internal hosts से file sharing या remote administration जिन्होंने पहले कभी वह नहीं दिया |
| असामान्य अपलोड | 10 मिनट में किसी नए address को 100 MB, जो वापस आया उसका तीन गुना |
| फ़्लड | एक address पर 20,000+ छोटे packets/s, उसकी सामान्य दर का दस गुना |
| ख़तरा सूची | किसी सूचीबद्ध address के साथ ट्रैफ़िक |

आपके नेटवर्क के अंदर से हों तो उच्च, इंटरनेट से हों तो निम्न। **निपटा दिया**
प्रविष्टि को बंद करता है, **कोई समस्या नहीं** उसे हमेशा के लिए चुप कर देता है।
लेटरल मूवमेंट और uploads के लिए एक दिन का इतिहास चाहिए। 1:4096 sampling के
बावजूद डेमो का हमला पूरा पकड़ा जाता है; बहुत छोटे scans sampling के पीछे छिप
सकते हैं।

![संदिग्ध गतिविधि: एक हमले का हर चरण, 1:4096 sFlow sampling के बावजूद पकड़ा गया](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. ऑफ़लाइन pcap, terminal UI, local capture

**ऑफ़लाइन pcap विश्लेषण** पैकेट कैप्चर (pcap, pcapng) को उन्हीं पेजों पर
दिखाता है, लाइव डेटा से अलग: `traffic66 a.pcap b.pcapng` 127.0.0.1 पर शुरू
होकर browser खोलता है (अधिकतम 3 फ़ाइलें, 3 GB; Ctrl+C आयात किया डेटा मिटा देता
है), या उस पेज पर 50 MB तक की अधिकतम 3 फ़ाइलें अपलोड करें। एक समय में एक
फ़ाइल का विश्लेषण होता है, हर फ़ाइल अपने अलग database में: किसी फ़ाइल की
पंक्ति पर **विश्लेषण करें** उसे हर पेज पर दिखाता है, और ऊपर की पट्टी दूसरी
फ़ाइल पर बदल देती है। यह flows पर काम करता है, पैकेट की सामग्री पर नहीं।

![ऑफ़लाइन विश्लेषण: कैप्चर फ़ाइलें, उनके पैकेट, फ़्लो और समय](images/sandbox.png)

**Terminal UI**: traffic66 मशीन पर `traffic66 tui`, या
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`।
तीन पेज: **1 टॉप** (क्लाइंट, सर्वर, ऐप्लिकेशन, पोर्ट, नेटवर्क (AS) और देश/क्षेत्र दर के क्रम में; ← → बदलें, Enter क्रियाएँ), **2 इंटरफ़ेस** (हर डिवाइस और इंटरफ़ेस की दर, उपयोग और मिलान, साथ में हर डिवाइस की सैंपलिंग और हानि; Enter ट्रेंड और टॉप होस्ट/ऐप खोलता है) और **3 खोज** (`ip:10.0.0.5 net:10.0.0.0/8 asn:13335 port:443 cc:US dev:192.0.2.1` जैसे फ़िल्टर से फ़्लो रिकॉर्ड)। कुंजियाँ: `/` फ़िल्टर, Backspace/`c` हटाएँ/साफ़, `t` अवधि (5 मिनट से 24 घंटे), `u` बिट/से, बाइट या पैकेट/से, `p` रोकें, `+`/`-` रीफ़्रेश अंतराल, `w` वही दृश्य ब्राउज़र में, `?` मदद, `q` बाहर; `-lang` भाषा चुनता है और `NO_COLOR` रंग बंद करता है।

![Terminal UI: टॉप](images/tui-top.png)

![Terminal UI: इंटरफ़ेस](images/tui-interfaces.png)

**Local capture** किसी local interface से flows बनाता है, सबसे अच्छा किसी
switch के mirror port से जुड़ा port: `traffic66 interfaces` उनकी सूची देता है,
`-capture eth1` capture करता है। Windows पर:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux को root या `setcap cap_net_raw,cap_net_admin+ep` चाहिए, macOS को root,
Windows को [Npcap](https://npcap.com)। Capture किए गए flows डिवाइस `127.0.0.1`
से आते हैं। Local capture में डिवाइस के interfaces या counters नहीं होते,
इसलिए **इंटरफ़ेस मिलान** के पास इसके लिए तुलना करने को कुछ नहीं है।

<a id="10-options-and-data"></a>

## 10. विकल्प और डेटा

`traffic66 -h` सब कुछ सूचीबद्ध करता है। सबसे ज़्यादा इस्तेमाल होने वाले:

| विकल्प | डिफ़ॉल्ट | |
|---|---|---|
| `-data` | प्रोग्राम के बगल में `traffic66-data` | data directory |
| `-addr` | `:8066` | web UI; केवल इसी मशीन के लिए `127.0.0.1:8066` |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors; ख़ाली हो तो बंद |
| `-retention-days` | `30` | flow detail के दिन; summaries 400 दिन रखी जाती हैं |
| `-memory` | `0.10` | database cache के लिए RAM का हिस्सा |
| `-sampling-wait` | `5m` | records sampling rate का कितनी देर इंतज़ार करें |
| `-capture` | | local interface (दोहराया जा सकता है) |
| `-no-dns` | | कोई reverse lookup नहीं |

data directory में `raw/` (detail, हर घंटे की एक फ़ाइल), `traffic66.duckdb`
(summaries और counters), `password`, `inventory.txt`, `license.json`, आपका
लोगो और databases रहते हैं। बैकअप के लिए traffic66 रोकें और इसे कॉपी करें;
अपग्रेड के लिए प्रोग्राम फ़ाइल बदलें। **डेटा सफ़ाई** 7–120 दिन से पुराना डेटा,
या पूरा डेटा, हटाता है।

<a id="licence"></a>

### लाइसेंस

[PolyForm Noncommercial License 1.0.0](../LICENSE.md) और
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) के तहत source
available (अंग्रेज़ी पाठ ही बाध्यकारी है): evaluation के लिए और 100 से कम लोगों
वाले संगठनों के लिए मुफ़्त; बड़े संगठन production use के 30 दिन बाद रजिस्टर
करते हैं; बेचने, दूसरों के लिए host करने या प्रतिस्पर्धी products के लिए
commercial licence चाहिए। कभी कुछ बंद नहीं होता। हर पेज के नीचे 8 अंकों का
installation number दिखता है; उसे लेखक को भेजें, और जो `license.json` वापस
मिले उसे data directory में रखें। संपर्क:
<route66traffic66@gmail.com>।

अपंजीकृत इंस्टॉलेशन में पेज के ऊपर एक प्रायोजक पट्टी दिखती है। इसे केवल आपका ब्राउज़र GitHub ([ads-sponsor](https://github.com/githubflyideas/ads-sponsor)) से लोड करता है; traffic66 सर्वर इसके लिए कभी GitHub से संपर्क नहीं करता और कोई प्रति नहीं रखता। जो ब्राउज़र GitHub तक नहीं पहुँच पाता, वह इसकी जगह एक छोटा अंतर्निहित पाठ दिखाता है। आपके या इंस्टॉलेशन के बारे में कोई डेटा नहीं भेजा जाता। पंजीकरण करने पर यह हट जाती है।

<a id="11-security-sizing-troubleshooting"></a>

## 11. सुरक्षा, क्षमता का अनुमान, समस्या निवारण

web UI सादा HTTP है: अविश्वसनीय नेटवर्कों पर `-addr 127.0.0.1:8066` को किसी
TLS proxy (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) या SSH tunnel के पीछे इस्तेमाल करें। UDP ports को केवल अपने
डिवाइसों से अनुमति दें। SNMP communities सादे टेक्स्ट में रहती हैं; read-only
communities इस्तेमाल करें।

2 cores पर 5,000 flows/s पर: detail के हर दिन के लिए लगभग 12 GB disk (30 दिन के
लिए 360 GB), एक core का छठा हिस्सा, 0.6–0.8 GB memory। लंबी ranges के overviews
0.2 s से कम लेते हैं; सभी conversations का 1 घंटे का Top 66 लगभग 9 s।

| लक्षण | समाधान |
|---|---|
| "waiting for the sampling rate" | sampler options export करें, या डिवाइस की line पर `sampling=N` / `unsampled` |
| counters से कम | interfaces sample नहीं हो रहे, loss, या 60 s से लंबा active timeout |
| counters से ज़्यादा | वही ट्रैफ़िक दो interfaces या दो डिवाइसों पर sample हो रहा है |
| पासवर्ड भूल गए | traffic66 मशीन पर `traffic66 passwd` |
| `Conflicting lock is held` | कोई दूसरा traffic66 यही data directory इस्तेमाल कर रहा है |
| `address already in use` | `-addr` या `-listen` से दूसरे ports चुनें |
| Windows: "Windows protected your PC" (Windows ने आपके PC को सुरक्षित किया) | **More info** (अधिक जानकारी) → **Run anyway** (फिर भी चलाएँ) |

सोर्स से बिल्ड करें: Go 1.26 और एक C compiler, फिर `CGO_ENABLED=1 go build ./cmd/traffic66`।
