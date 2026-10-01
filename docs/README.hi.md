[English](../README.md) | [中文](README.zh.md) | **हिन्दी** | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

sFlow, NetFlow और IPFIX के लिए flow analytics, एक ही प्रोग्राम में। यह
switches, routers और firewalls से flow exports इकट्ठा करता है, उन्हें एक
embedded database में रखता है, और दिखाता है कि bandwidth कौन इस्तेमाल कर
रहा है, ट्रैफ़िक कहाँ जा रहा है और क्या आँकड़े डिवाइस के अपने interface
counters से मेल खाते हैं — web UI में भी और terminal UI में भी।

- Windows, Linux और macOS के लिए एक ही executable। कोई database इंस्टॉल
  नहीं करना, कोई runtime नहीं, offline भी चलता है।
- किसी भी UDP port पर sFlow v5, NetFlow v5, NetFlow v9 और IPFIX; चाहें तो
  किसी network interface या mirror port से local capture भी।
- अपने आँकड़ों को interface counters (sFlow counters या SNMP) से मिलाकर
  जाँचता है, और फ़र्क़ होने पर बताता है कि क्यों।
- Top 66 सूचियाँ, flow के रास्ते, देश और नेटवर्क, threat list के matches,
  flow records, encapsulation (GRE, IPIP, VXLAN, GENEVE, MPLS)।
- web UI और terminal UI में 13 भाषाएँ।

<a id="contents"></a>

## विषय-सूची

1. [डेमो चलाकर देखें](#1-try-the-demo)
2. [इंस्टॉल करें](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [साइन इन और पासवर्ड](#3-sign-in-and-passwords)
4. [अपने डिवाइसों से flows भेजें](#4-send-flows-from-your-devices)
5. [जाँचें कि flows पहुँच रहे हैं](#5-check-that-flows-arrive)
6. [आँकड़ों को interface counters से मिलाएँ](#6-make-the-numbers-match-the-interface-counters)
7. [नाम, SNMP और आपके अपने नेटवर्क](#7-names-snmp-and-your-own-networks)
8. [देश, नेटवर्क और threat lists](#8-countries-networks-and-threat-lists)
9. [web UI का इस्तेमाल](#9-using-the-web-ui)
10. [Terminal UI](#10-terminal-ui)
11. [Local capture](#11-local-capture)
12. [विकल्प](#12-options)
13. [डेटा, बैकअप, अपग्रेड, अनइंस्टॉल](#13-data-backup-upgrade-uninstall)
14. [सुरक्षा](#14-security)
15. [क्षमता का अनुमान](#15-sizing)
16. [समस्या निवारण](#16-troubleshooting)
17. [सोर्स से बिल्ड करें](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. डेमो चलाकर देखें

अपने सिस्टम के लिए archive
[releases पेज](https://github.com/githubflyideas/traffic66/releases) से डाउनलोड करें:

| सिस्टम | Archive |
|---|---|
| Windows 10/11, Server 2016 या नया (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64 | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64 | `traffic66-linux-arm64.tar.gz` |
| macOS Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (दूसरी लाइन macOS को इंटरनेट से डाउनलोड किया गया ऐसा प्रोग्राम चलाने
देती है जो App Store से नहीं आया):

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

http://127.0.0.1:8066 खोलें और `admin` / `try66` से साइन इन करें। डेमो एक
छोटी कंपनी का नेटवर्क बनाता है, जिसमें एक दिन का इतिहास और चार simulated
डिवाइसों से live ट्रैफ़िक होता है, और उसमें ढूँढने के लिए दो incidents भी हैं:
**सारांश** से शुरू करें, **किसका ट्रैफ़िक बढ़ा** देखें, और वहाँ से क्लिक करते
हुए आगे बढ़ें। Ctrl+C से बंद करें। डेमो का डेटा प्रोग्राम के बगल में
`traffic66-demo` में रहता है; डेमो नए सिरे से शुरू करने के लिए वह फ़ोल्डर
हटा दें।

डेमो वही ports इस्तेमाल करता है जो असली installation करता है (8066, और UDP
6343, 2055, 4739)। असली installation के साथ-साथ चलाने के लिए उसे दूसरे ports
दें: `traffic66 demo -password try66 -addr :8067 -listen ""`।

<a id="2-install"></a>

## 2. इंस्टॉल करें

traffic66 एक ही फ़ाइल है। इंस्टॉल करने का मतलब है: उसे कहीं रखना, एक data
directory चुनना, पासवर्ड सेट करना, firewall खोलना और boot पर उसे शुरू करवाना।
उदाहरणों में traffic66 मशीन के लिए `192.0.2.50` और router के लिए `192.0.2.1`
है; इन्हें अपने addresses से बदलें।

Ports:

| Port | किसलिए |
|---|---|
| UDP 6343 | sFlow (डिफ़ॉल्ट) |
| UDP 2055 | NetFlow (डिफ़ॉल्ट) |
| UDP 4739 | IPFIX (डिफ़ॉल्ट) |
| TCP 8066 | web UI और API |

हर UDP port हर protocol स्वीकार करता है, इसलिए आसान हो तो डिवाइस NetFlow भी
6343 पर भेज सकता है। Ports बदलने या जोड़ने के लिए `-listen` इस्तेमाल करें।

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

आख़िरी कमांड यूज़र `admin` का पासवर्ड पूछती है।

`/etc/systemd/system/traffic66.service` बनाएँ:

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

इसे शुरू करें और बड़े UDP buffers की अनुमति दें ताकि bursts में packets न खोएँ:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, firewalld के साथ (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

या ufw के साथ (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

`C:\traffic66` में unpack करें और पासवर्ड सेट करें (PowerShell, Administrator
के रूप में):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

डेटा प्रोग्राम के बगल में `C:\traffic66\traffic66-data` में जाता है।

Firewall खोलें:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

foreground में आज़माने के लिए `C:\traffic66\traffic66.exe` चलाएँ और Ctrl+C से
बंद करें। boot से ही background में चलाने के लिए, बिना किसी के साइन इन किए,
इसे startup task के रूप में register करें:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` ज़रूरी है: इसके बिना Windows तीन दिन
बाद task रोक देता है। रोकने के लिए `Stop-ScheduledTask -TaskName
traffic66`, हटाने के लिए `Unregister-ScheduledTask -TaskName traffic66`।

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

`/Library/LaunchDaemons/traffic66.plist` बनाएँ:

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

शुरू करें, और फिर से बंद करें:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

अगर macOS firewall चालू है, तो System Settings → Network → Firewall → Options
में traffic66 के लिए incoming connections की अनुमति दें।

<a id="3-sign-in-and-passwords"></a>

## 3. साइन इन और पासवर्ड

`http://<traffic66 machine>:8066` खोलें और साइन इन करें। यूज़र `admin` है,
जब तक आपने कोई और न चुना हो।

- अगर पहली बार शुरू करने से पहले आपने पासवर्ड सेट नहीं किया, तो traffic66
  ख़ुद एक बनाता है और उसे एक बार अपने log में छापता है:
  `first start: sign in as user "admin" with password "…"`।
  Linux पर इसे `journalctl -u traffic66 | grep "first start"` से ढूँढें।
- पासवर्ड hash करके data directory की `password` फ़ाइल में रखा जाता है।
  restart के बाद भी वही रहता है।
- इसे बदलने के लिए, या भूल जाने पर नया सेट करने के लिए, traffic66 मशीन पर:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` एक random पासवर्ड बनाकर छापता है। चल रहा
  traffic66 अगले साइन इन पर नया पासवर्ड स्वीकार कर लेता है; restart की ज़रूरत
  नहीं।
- और यूज़र: `traffic66 passwd -data <data directory> -user alice`। सभी
  यूज़र एक जैसा ही देखते हैं।
- scripts और containers के लिए, environment में `TRAFFIC66_PASSWORD=…` या
  command line पर `-password …` उस run के लिए stored पासवर्ड की जगह पासवर्ड
  सेट करता है। environment को प्राथमिकता दें: command lines मशीन के दूसरे
  यूज़र्स को दिखती हैं।

एक मिनट के भीतर पाँच ग़लत पासवर्ड के बाद उस address को एक मिनट के लिए block
कर दिया जाता है।

<a id="4-send-flows-from-your-devices"></a>

## 4. अपने डिवाइसों से flows भेजें

हर डिवाइस को traffic66 मशीन की ओर point करें। कमांड models और software
versions के हिसाब से अलग होती हैं; अपने डिवाइस का manual देखें। सभी उदाहरणों में
`192.0.2.50` traffic66 है और `192.0.2.1` डिवाइस का अपना address।

सामान्य सलाह:

- active flow timeout 60 सेकंड रखें। लंबे timeouts से ट्रैफ़िक देर से और बड़े
  ढेलों में आता है।
- अगर डिवाइस NetFlow/IPFIX को sample करता है, तो उसे sampler options export
  करने दें ताकि rate पता रहे। traffic66 records को 1:1 गिनने के बजाय rate
  आने तक रोके रखता है।
- या तो सभी interfaces sample करें या केवल edge interfaces, एक ही दिशा में।
  एक ही ट्रैफ़िक को अंदर आते और बाहर जाते दोनों समय sample करने से वह दो बार
  गिना जाता है; **इंटरफ़ेस मिलान** यह बता देता है।
- sFlow sampling rate: 1 Gb/s links के लिए लगभग 1:1000, 10 Gb/s के लिए
  1:4096, 40/100 Gb/s के लिए 1:8192।

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

FortiGate FortiOS 7.4.2 या नया (NetFlow v9):

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

Linux servers और hosts, softflowd के साथ (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. जाँचें कि flows पहुँच रहे हैं

**स्रोत** खोलें। जो भी डिवाइस कुछ भी भेजता है, वह कुछ सेकंड में दिख जाता है,
अपने protocol, sampling rate, loss, आख़िरी packet और status के साथ। जब status
हरा न हो, तो उसके बगल का टेक्स्ट बताता है कि क्या गड़बड़ है और क्या बदलना है।

अगर कोई डिवाइस नहीं दिखता:

1. traffic66 मशीन पर packets देखें (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`।
   वहाँ कुछ नहीं है तो packets मशीन तक पहुँच ही नहीं रहे: डिवाइस का
   configuration, routing और रास्ते के firewalls जाँचें।
2. packets आ रहे हैं पर **स्रोत** ख़ाली है: local firewall उन्हें drop कर रहा है
   ([इंस्टॉल करें](#2-install) देखें), या traffic66 दूसरे ports पर सुन रहा है
   (`-listen`)।
3. किसी डिवाइस को छुए बिना दूसरी मशीन से रास्ता टेस्ट करने के लिए, वहाँ कुछ
   सेकंड के लिए `traffic66 simulate -to 192.0.2.50` चलाएँ। यह simulated
   डिवाइसों से sFlow, NetFlow और IPFIX भेजता है; वे फिर **स्रोत** में और डेटा
   में दिखते हैं, इसलिए इसके लिए test installation बेहतर है।

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. आँकड़ों को interface counters से मिलाएँ

Flow के आँकड़े अनुमान होते हैं: sampled packets गुणा sampling rate।
traffic66 इन्हें डिवाइस के अपने interface counters से मिलाता है और फ़र्क़
**इंटरफ़ेस मिलान** पर दिखाता है; जब फ़र्क़ अकेले sampling से समझ में आने
लायक से ज़्यादा हो, तो संभावित कारण भी बताता है।

तुलना के लिए counters पाने के लिए:

- sFlow डिवाइस counter interval सेट होने पर उन्हें ख़ुद भेजते हैं
  (`sflow counter interval 30` और इसी तरह की कमांड)।
- NetFlow और IPFIX डिवाइसों के लिए **स्रोत → नाम** में एक `snmp` लाइन जोड़ें
  ([नाम](#7-names-snmp-and-your-own-networks) देखें)। फिर traffic66 हर मिनट
  interface counters पढ़ता है।

आँकड़े मिलें, इसके लिए traffic66 पहले से ये करता है: डिवाइस ने असल में जो
sampling rate लगाया वही इस्तेमाल करता है, sampling rate पता चलने तक
NetFlow/IPFIX records रोके रखता है, रास्ते में खोए export packets की भरपाई करता
है, लंबे flows को उतने मिनटों में बाँटता है जितने वे चले, और NetFlow/IPFIX
byte counts में हर packet पर 18 bytes का Ethernet overhead जोड़ता है (interface
counters में यह शामिल होता है, IP-layer flow counts में नहीं; `-l2-overhead`
से बदलें)।

बचे हुए फ़र्क़ के आम कारण, जो सब **इंटरफ़ेस मिलान** पर बताए जाते हैं: कुछ
interfaces sample नहीं हो रहे, एक ही ट्रैफ़िक दो interfaces पर sample हो रहा है,
export packets traffic66 तक पहुँचने से पहले खो जाते हैं, या sampling rate अभी
पता नहीं है।

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. नाम, SNMP और आपके अपने नेटवर्क

web UI में **स्रोत → नाम** हर लाइन में एक entry लेता है। यह data directory में
`inventory.txt` के रूप में सेव होता है, इसलिए आप वह फ़ाइल सीधे भी edit कर सकते
हैं (`inventory.txt.example` देखें)। हर लाइन वैकल्पिक है।

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

- `net`: private ranges (10/8, 172.16/12, 192.168/16, 100.64/10) हमेशा आपके
  माने जाते हैं। अपने public ranges जोड़ें ताकि उनसे आने-जाने वाला ट्रैफ़िक भी
  आपका गिना जाए; नाम **Top-N → सेगमेंट** में और flow के रास्तों में दिखता है।
- `snmp <device> <community> [<management address>[:port]]`: device वह address
  है जिससे flows आते हैं। जब डिवाइस SNMP का जवाब किसी दूसरे address पर देता हो,
  तो management address जोड़ें। SNMP से पढ़े गए interface descriptions नाम के
  रूप में इस्तेमाल होते हैं, जब तक आप `iface` से interface का नाम न दें।
  डिवाइस की SNMP access list में traffic66 मशीन को अनुमति दें।
- बदलाव **सहेजें** पर क्लिक करते ही लागू होते हैं; restart की ज़रूरत नहीं।

<a id="8-countries-networks-and-threat-lists"></a>

## 8. देश, नेटवर्क और threat lists

देशों और नेटवर्क (AS) के नामों के लिए IP-to-ASN table चाहिए।
[iptoasn.com](https://iptoasn.com) से मुफ़्त table डाउनलोड करें:

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

इसी format की कोई भी फ़ाइल चलेगी (tab से अलग: पहला address, आख़िरी address,
AS number, country code, AS name; plain या gzip)। इसे बदलने के बाद traffic66
restart करें; लगभग हर महीने नई फ़ाइल डाउनलोड करें।

Threat lists सादी text फ़ाइलें होती हैं, हर लाइन में एक address या नेटवर्क
(`#` या `;` के बाद का टेक्स्ट नज़रअंदाज़ होता है), जो
`<data directory>/threats/<name>.txt` के रूप में सेव होती हैं, उदाहरण के लिए:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Lists जोड़ने या बदलने के बाद traffic66 restart करें। Matches
**ख़तरे की जानकारी** पर list के नाम के हिसाब से दिखते हैं।

<a id="9-using-the-web-ui"></a>

## 9. web UI का इस्तेमाल

आपको शायद ही कुछ टाइप करना पड़े। हर पेज की हर value — address, port,
application, देश, डिवाइस — पर क्लिक किया जा सकता है:

- **केवल यही दिखाएँ** / **इसे हटाएँ** एक filter जोड़ता है। Filters top bar के
  नीचे दिखते हैं और हटाए जाने तक हर पेज पर लागू रहते हैं।
- **इसके फ़्लो रिकॉर्ड देखें** मेल खाते अलग-अलग flows खोलता है।
- **ऑनलाइन खोजें** address या AS को किसी public lookup साइट पर खोलता है।
- **कॉपी करें** value कॉपी करता है।

पेज:

| पेज | किस सवाल का जवाब देता है |
|---|---|
| सारांश | अभी कितना ट्रैफ़िक है और पिछले हफ़्ते की तुलना में कितना, application के हिसाब से; क्या बढ़ा; top clients और services |
| Top-N | clients, servers, conversations, applications, ports, देशों, नेटवर्कों, segments, डिवाइसों, encapsulation या VLAN के top 66 |
| ट्रैफ़िक के रास्ते | कौन-सा segment किस देश में किस application से बात करता है |
| भूगोल और नेटवर्क | देश के हिसाब से और नेटवर्क (AS) के हिसाब से ट्रैफ़िक |
| ख़तरे की जानकारी | वे hosts जिन्होंने आपकी threat lists के addresses से बात की, और उन्होंने कितना भेजा |
| फ़्लो रिकॉर्ड | अलग-अलग flows, सबसे नए पहले, चुने जा सकने वाले columns के साथ |
| इंटरफ़ेस मिलान | interface counters के बगल में flow के आँकड़े, सबसे ख़राब पहले, कारणों के साथ |
| स्रोत | डिवाइस, sampling, loss, collectors, SNMP, और **नाम** |

पेजों के ऊपर: time range (15 मिनट से 30 दिन), एक वैकल्पिक search box, हर 30
सेकंड पर automatic refresh, और **लिंक कॉपी करें**, जो ठीक मौजूदा view (पेज,
time range और filters) का link कॉपी करता है ताकि आप उसे किसी सहकर्मी को भेज
सकें। भाषा browser के हिसाब से चुनी जाती है; menu के नीचे से बदलें।

लंबी time ranges पर Top-N घंटेवार summaries से आता है; वहाँ filters उपलब्ध नहीं
हैं, और पेज यह बता देता है। filter करने के लिए छोटी range चुनें।

<a id="10-terminal-ui"></a>

## 10. Terminal UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

traffic66 मशीन पर, `traffic66 tui` ख़ुद ही साइन इन कर लेता है जब वह data
directory पढ़ सकता है (डिफ़ॉल्ट न हो तो `-data` दें)। अगर traffic66 किसी दूसरे
यूज़र के रूप में चलता है, जैसे service चलती है, तो इसके बजाय `-user` और
`-password` इस्तेमाल करें। `-lang` भाषा चुनता है (`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`)।

Keys: 1–8 पेज, ↑↓ चुनें, Enter चुनी गई value पर actions, f केवल यही दिखाएँ,
x हटाएँ, / search, t time range, c filters साफ़ करें, w यही view browser में
खोलें, q बाहर निकलें।

<a id="11-local-capture"></a>

## 11. Local capture

Flow exports के अलावा, traffic66 किसी local network interface, जैसे mirror
(SPAN) port, के packets से ख़ुद भी flows बना सकता है:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: root चाहिए, या capabilities `CAP_NET_RAW` और `CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`, या
  ऊपर दी गई systemd unit में `AmbientCapabilities` वाली लाइन)।
- macOS: root चाहिए (BPF devices); कुछ इंस्टॉल नहीं करना।
- Windows: पहले [Npcap](https://npcap.com) इंस्टॉल करें।

Capture किए जा रहे interfaces **स्रोत** पर दिखते हैं। दो बार दिखे packets
(जैसे दो mirror ports पर) दो बार गिने जाते हैं।

<a id="12-options"></a>

## 12. विकल्प

`traffic66 -h` और `traffic66 <command> -h` सब कुछ सूचीबद्ध करते हैं।

कमांड:

| कमांड | |
|---|---|
| `traffic66` | flows इकट्ठा करता है और web UI चलाता है |
| `traffic66 demo` | वही, एक simulated नेटवर्क के साथ |
| `traffic66 tui` | चल रहे traffic66 के लिए terminal UI |
| `traffic66 passwd` | login पासवर्ड सेट करता है |
| `traffic66 simulate -to HOST` | किसी collector को simulated exports भेजता है |
| `traffic66 interfaces` | local capture के लिए interfaces की सूची |
| `traffic66 version` | version छापता है |

`traffic66` और `traffic66 demo` के विकल्प:

| विकल्प | डिफ़ॉल्ट | |
|---|---|---|
| `-addr` | `:8066` | web UI का address; केवल इसी मशीन के लिए `127.0.0.1:8066` |
| `-data` | प्रोग्राम के बगल में `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors, `name=address` के रूप में, comma से अलग; ख़ाली हो तो बंद |
| `-user` | `admin` | पहले generated पासवर्ड और `-password` के लिए यूज़र |
| `-password` | stored पासवर्ड | केवल इस run के लिए पासवर्ड (`TRAFFIC66_PASSWORD` भी) |
| `-retention-days` | `30` | flow detail कितने दिन रखा जाए; summaries 400 दिन रखी जाती हैं |
| `-memory` | `0.10` | physical memory का कितना हिस्सा database इस्तेमाल कर सकता है |
| `-l2-overhead` | `18` | NetFlow/IPFIX byte counts में हर packet पर जोड़े जाने वाले bytes |
| `-sampling-wait` | `5m` | records sampling rate का कितनी देर इंतज़ार करें |
| `-capture` | | local interface पर capture (दोहराया जा सकता है) |
| `-inventory` | `<data>/inventory.txt` | नामों की फ़ाइल |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table |
| `-threat` | `<data>/threats/*.txt` | अतिरिक्त threat list, `name=path` के रूप में (दोहराया जा सकता है) |
| `-dns-upstream` | system resolver | host names दिखाने के लिए DNS server |
| `-dns-rate` | `20` | प्रति सेकंड अधिकतम reverse lookups |
| `-dns-cache` | `2m` | host names कितनी देर cache रहें |
| `-no-dns` | | कोई reverse lookup नहीं |
| `-tui` | | साथ में terminal UI भी खोलें |

उदाहरण: एक दूसरा collector port, एक साल का detail, और web UI केवल local मशीन
पर:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. डेटा, बैकअप, अपग्रेड, अनइंस्टॉल

सब कुछ data directory में रहता है:

| | |
|---|---|
| `raw/` | flow detail, हर घंटे की एक compressed फ़ाइल |
| `traffic66.duckdb` | summaries, interface counters और मौजूदा घंटा |
| `password` | login पासवर्ड (hashed) |
| `inventory.txt` | नाम (**स्रोत → नाम**) |
| `asn.tsv.gz`, `threats/` | आपकी जोड़ी हुई lookup tables |

- **बैकअप**: traffic66 रोकें और directory कॉपी करें। बिना रोके
  `raw/`, `password` और `inventory.txt` कॉपी करें; तब मौजूदा घंटा और
  summaries छूट जाते हैं।
- **दूसरी जगह ले जाना**: traffic66 रोकें, directory को move करें, और नई जगह की
  ओर इशारा करते `-data` के साथ शुरू करें।
- **अपग्रेड**: traffic66 रोकें, प्रोग्राम फ़ाइल बदलें, फिर से शुरू करें।
  डेटा बना रहता है। उदाहरण के लिए Linux पर:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **अनइंस्टॉल**: service या startup task रोकें और हटाएँ
  ([इंस्टॉल करें](#2-install) देखें), फिर प्रोग्राम फ़ोल्डर और data directory
  हटा दें।

<a id="14-security"></a>

## 14. सुरक्षा

- web UI सादा HTTP इस्तेमाल करता है: पासवर्ड और डेटा बिना encryption के नेटवर्क
  पर जाते हैं। जिन नेटवर्कों पर आपको पूरा भरोसा नहीं, वहाँ केवल इसी मशीन पर
  सुनें (`-addr 127.0.0.1:8066`) और आगे एक TLS reverse proxy लगाएँ, उदाहरण के
  लिए [Caddy](https://caddyserver.com) से:
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`।
  या इसे VPN या SSH tunnel से एक्सेस करें:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, फिर
  http://127.0.0.1:8066 खोलें।
- UDP collector ports को केवल अपने डिवाइसों के addresses से अनुमति दें।
- `inventory.txt` में SNMP communities सादे टेक्स्ट में रहती हैं; read-only
  community इस्तेमाल करें।

<a id="15-sizing"></a>

## 15. क्षमता का अनुमान

2-core मशीन पर 5,000 flows प्रति सेकंड पर मापा गया: detail रोज़ लगभग 12 GB
disk लेता है, साथ में मौजूदा घंटे के लिए लगभग 1.5 GB; प्रोग्राम लगभग 0.5 GB
memory और एक core का छठा हिस्सा लेता है। लंबी time ranges के overviews
summaries से आते हैं और 0.2 s से कम लेते हैं। Detail पर queries हर घंटे के
लगभग 2.2 करोड़ rows scan करती हैं: 1 घंटे में एक host 1 s से कम लेता है, सभी
conversations का 1 घंटे का Top 66 लगभग 9 s; समय range के साथ बढ़ता है और
ज़्यादा cores के साथ घटता है।

इसलिए 5,000 flows/s पर 30 दिनों के लिए disk लगभग 360 GB है; इसे अपने flow rate
(**स्रोत** पर दिखता है) और `-retention-days` के अनुपात में बढ़ाएँ-घटाएँ।

<a id="16-troubleshooting"></a>

## 16. समस्या निवारण

| लक्षण | कारण और समाधान |
|---|---|
| डिवाइस **स्रोत** में नहीं है | Packets नहीं पहुँच रहे: [जाँचें कि flows पहुँच रहे हैं](#5-check-that-flows-arrive) देखें |
| "waiting for the sampling rate" | डिवाइस ने अभी तक अपने sampler options नहीं भेजे; ज़्यादातर कुछ मिनटों में दोबारा भेज देते हैं। अगर कभी न भेजे, तो उन्हें export करवाएँ (Cisco पर `option sampler-table`) या अगर वह सच में 1:1 है तो नाम में उसे `unsampled` लिखें |
| आँकड़े interface counters से कम | **इंटरफ़ेस मिलान** देखें: रास्ते में loss, interfaces sample नहीं हो रहे, या flows अभी डिवाइस के cache में हैं (active timeout 60 s से लंबा) |
| आँकड़े interface counters से ज़्यादा | वही ट्रैफ़िक दो interfaces या दो डिवाइसों पर sample हो रहा है |
| कोई देश या नेटवर्क नहीं | IP-to-ASN table नहीं है: [देश](#8-countries-networks-and-threat-lists) देखें |
| पासवर्ड भूल गए | traffic66 मशीन पर `traffic66 passwd -data <data directory>` |
| `Conflicting lock is held` | कोई दूसरा traffic66 पहले से यही data directory इस्तेमाल कर रहा है |
| `receive buffer is only … KB` | Linux UDP buffers सीमित रखता है: `net.core.rmem_max=16777216` सेट करें ([Linux](#linux) देखें) |
| `cannot create the data directory` | इस यूज़र के लिए प्रोग्राम फ़ोल्डर writable नहीं है: `-data` दें |
| macOS: "cannot be opened" या "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows capture: Npcap नहीं मिला | [Npcap](https://npcap.com) इंस्टॉल करें |
| `address already in use` | कोई दूसरा प्रोग्राम वह port इस्तेमाल कर रहा है: `-addr` या `-listen` से दूसरे ports चुनें |

<a id="17-build-from-source"></a>

## 17. सोर्स से बिल्ड करें

Go 1.24 और एक C compiler (gcc या clang; Windows पर MinGW-w64):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
