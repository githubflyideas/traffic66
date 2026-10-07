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
- flows में scans, पासवर्ड का अनुमान, lateral movement, असामान्य uploads,
  floods और threat list वाला ट्रैफ़िक ढूँढता है, sampling के बावजूद भी, और
  उन्हें निपटाने के लिए संदिग्ध गतिविधियों की सूची में रखता है।
- Top 66 सूचियाँ, कौन किससे बात करता है यह ring charts में (servers और उनके
  clients, services और उनके servers), interface और नेटवर्क (AS) के हिसाब से
  समय के साथ ट्रैफ़िक, flow के रास्ते, विश्व मानचित्र पर देश, threat list के
  matches, flow records, encapsulation (GRE, IPIP, VXLAN, GENEVE, MPLS)।
- `traffic66 capture.pcap` अधिकतम 3 पैकेट कैप्चर (कुल 3 GB) वेब UI में खोलता है: पूरे कैप्चर के फ़्लो, निष्कर्ष, देश और फ़्लो रिकॉर्ड, बिना किसी सेटअप के।
- web UI और terminal UI में 13 भाषाएँ।
- Source available: evaluation के लिए और 100 से कम लोगों वाले संगठनों के लिए
  मुफ़्त; बड़े संगठन production use के 30 दिन बाद रजिस्टर करते हैं। कभी कुछ
  बंद नहीं होता ([ट्रायल और लाइसेंस](#trial-and-licence) देखें)।

![सारांश: खुली संदिग्ध गतिविधियाँ, कल इसी समय की तुलना में application के हिसाब से bandwidth, top clients और services](images/overview.png)

<sub>सभी स्क्रीनशॉट `traffic66 demo` से लिए गए हैं, एक simulated कंपनी नेटवर्क जिसे आप ख़ुद चला सकते हैं ([डेमो चलाकर देखें](#1-try-the-demo) देखें)।</sub>

<a id="contents"></a>

## विषय-सूची

1. [डेमो चलाकर देखें](#1-try-the-demo)
2. [इंस्टॉल करें](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [यूज़र और पासवर्ड](#3-users-and-passwords)
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
| Linux x86-64: kernel 3.2 या उसके बाद वाला कोई भी distribution, जिसमें CentOS 7 और Alpine शामिल हैं | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: वही distributions | `traffic66-linux-arm64.tar.gz` |
| macOS 11 या नया, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 या नया, Intel | `traffic66-darwin-amd64.tar.gz` |

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

http://127.0.0.1:8066 खोलें और `admin` / `try66` से साइन इन करें। डेमो एक छोटी
कंपनी का नेटवर्क बनाता है, जिसमें एक दिन का इतिहास और चार simulated डिवाइसों
से live ट्रैफ़िक होता है, और उसमें एक हमला भी है: **संदिग्ध गतिविधि** उसका हर
चरण दिखाता है (एक scan, एक port scan, पासवर्ड का अनुमान, lateral movement, एक
control server पर upload) और public website पर एक flood भी। सूची की किसी
प्रविष्टि पर **विवरण** क्लिक करें, या **सारांश** से शुरू करें,
**शीर्ष क्लाइंट** में किसी host पर क्लिक करें, **विवरण देखें** चुनें, और वहाँ
से क्लिक करते हुए आगे बढ़ें। Ctrl+C से बंद करें। डेमो का डेटा प्रोग्राम के बगल में
`traffic66-demo` में रहता है; डेमो नए सिरे से शुरू करने के लिए वह फ़ोल्डर
हटा दें।

डेमो वही ports इस्तेमाल करता है जो असली installation करता है (8066, और UDP
6343, 2055, 4739)। असली installation के साथ-साथ चलाने के लिए उसे दूसरे ports
दें: `traffic66 demo -password try66 -addr :8067 -listen ""`।

Windows पर आप सीधे `traffic66.exe` पर double-click भी कर सकते हैं। इससे
traffic66 असली रूप में (डेमो नहीं) शुरू होता है और आपके browser में web UI
खुल जाता है; पहली बार शुरू होने का पासवर्ड काली window में दिखता है, और
window बंद करने से traffic66 रुक जाता है। अगर Windows
"Windows protected your PC" (Windows ने आपके PC को सुरक्षित किया) दिखाए, तो
**More info** (अधिक जानकारी) → **Run anyway** (फिर भी चलाएँ) पर क्लिक करें।

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
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
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

<a id="3-users-and-passwords"></a>

## 3. यूज़र और पासवर्ड

**संक्षेप में:** यूज़र और पासवर्ड data directory की एक ही फ़ाइल, `password`, में
रहते हैं। इसे कभी हाथ से edit न करें: `traffic66 passwd` command यूज़र जोड़ता,
बदलता, उनकी सूची दिखाता और उन्हें हटाता है। `http://<traffic66 machine>:8066`
खोलें और इनमें से किसी एक यूज़र से साइन इन करें।

<a id="the-first-sign-in"></a>

### पहली बार साइन इन

पहली बार शुरू होने पर traffic66 एक random पासवर्ड के साथ यूज़र `admin` बनाता
है और उसे एक बार दिखाता है:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Windows पर double-click से शुरू किया: काली window में।
- terminal में शुरू किया: उसी terminal में।
- Linux service: `journalctl -u traffic66 | grep "first start"`
- macOS service: `grep "first start" /Library/Logs/traffic66.log`

छूट गया? `traffic66 passwd` (नीचे देखें) से नया सेट करें। अगर आपने पहली बार
शुरू करने से पहले `traffic66 passwd` से पासवर्ड सेट कर दिया है, जैसा ऊपर के
install steps करते हैं, तो कुछ भी generate नहीं होता।

<a id="where-the-users-are-stored"></a>

### यूज़र कहाँ रखे जाते हैं

data directory की फ़ाइल `password` में:

| traffic66 कैसे चलता है | फ़ाइल |
|---|---|
| unpack करके उसके folder से शुरू किया गया (default) | प्रोग्राम के बगल में `traffic66-data/password` |
| Linux service (अनुभाग 2) | `/var/lib/traffic66/password` |
| Windows startup task (अनुभाग 2) | `C:\traffic66\traffic66-data\password` |
| macOS service (अनुभाग 2) | `/Library/Application Support/traffic66/password` |
| डेमो | प्रोग्राम के बगल में `traffic66-demo/password` |

हर यूज़र की एक line। पासवर्ड salted hash के रूप में रखे जाते हैं, इसलिए कोई भी
उन्हें फ़ाइल से वापस नहीं पढ़ सकता, आप भी नहीं; पासवर्ड भूल जाएँ तो नया सेट
करें। फ़ाइल को केवल उसका owner पढ़ सकता है।

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### यूज़र्स का प्रबंधन

ये commands traffic66 मशीन पर चलाएँ:

| काम | Command |
|---|---|
| `admin` का पासवर्ड बदलें | `traffic66 passwd` |
| यूज़र `alice` जोड़ें, या उसका पासवर्ड बदलें | `traffic66 passwd -user alice` |
| यूज़र `alice` को हटाएँ | `traffic66 passwd -user alice -delete` |
| यूज़र्स की सूची देखें | `traffic66 passwd -list` |
| random पासवर्ड सेट करके छापें | `traffic66 passwd -generate` (दूसरे यूज़र्स के लिए `-user` के साथ) |

- command नया पासवर्ड दो बार पूछता है और आप जो टाइप करते हैं उसे नहीं
  दिखाता। कम से कम 8 अक्षर रखें।
- जब traffic66 `-data` के साथ चलता है, तो command में भी वही `-data` जोड़ें।
  अनुभाग 2 वाली Linux service के लिए:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Windows पर (PowerShell, Administrator के रूप में):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- बदलाव तुरंत लागू होते हैं, restart की ज़रूरत नहीं: नया पासवर्ड अगले साइन इन
  पर काम करता है, और हटाया गया यूज़र खुले browsers से sign out हो जाता है।
- आख़िरी बचा यूज़र हटाया नहीं जा सकता; पहले कोई दूसरा यूज़र जोड़ें।
- सभी यूज़र एक जैसी चीज़ें देखते और बदल सकते हैं; कोई roles नहीं हैं।

<a id="passwords-for-scripts-and-containers"></a>

### scripts और containers के लिए पासवर्ड

environment में `TRAFFIC66_PASSWORD=…`, या command line पर `-password …`, से
traffic66 उस run के लिए ठीक एक ही यूज़र स्वीकार करता है: `-user` में दिया गया
यूज़र (default `admin`), उसी पासवर्ड के साथ। तब `password` फ़ाइल को अनदेखा किया
जाता है और बदला नहीं जाता। environment variable को प्राथमिकता दें: command lines
मशीन के दूसरे यूज़र्स को दिखती हैं।

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

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

**सेटिंग्स** खोलें। जो भी डिवाइस कुछ भी भेजता है, वह कुछ सेकंड में दिख जाता है,
अपने protocol, sampling rate, loss, आख़िरी packet और status के साथ। जब status
हरा न हो, तो उसके बगल का टेक्स्ट बताता है कि क्या गड़बड़ है और क्या बदलना है।

**खोए** उन samples या records को गिनता है जो कभी पहुँचे ही नहीं। sFlow के लिए
टेक्स्ट बताता है कि वे कहाँ खोए: यहाँ आते हुए रास्ते में (sequence numbers में
gaps: नेटवर्क, या इस मशीन का UDP receive buffer; अगर `netstat -su` में receive
buffer errors बढ़ते दिखें, तो `net.core.rmem_max` बढ़ाएँ), या ख़ुद डिवाइस में
(sFlow उन samples की रिपोर्ट देता है जिन्हें डिवाइस ने छोड़ दिया: उसका sFlow
export rate-limited है, इसलिए कम बार sample करें या डिवाइस की सीमा बढ़ाएँ)।
दोनों ही स्थितियों में कुल आँकड़ों की भरपाई की जाती है; हर host का ब्योरा नहीं।

![सेटिंग्स: हर डिवाइस, उसका protocol, sampling, loss और क्या ठीक करना है](images/sources.png)

अगर कोई डिवाइस नहीं दिखता:

1. traffic66 मशीन पर packets देखें (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`।
   वहाँ कुछ नहीं है तो packets मशीन तक पहुँच ही नहीं रहे: डिवाइस का
   configuration, routing और रास्ते के firewalls जाँचें।
2. packets आ रहे हैं पर **सेटिंग्स** ख़ाली है: local firewall उन्हें drop कर रहा है
   ([इंस्टॉल करें](#2-install) देखें), या traffic66 दूसरे ports पर सुन रहा है
   (`-listen`)।
3. किसी डिवाइस को छुए बिना दूसरी मशीन से रास्ता टेस्ट करने के लिए, वहाँ कुछ
   सेकंड के लिए `traffic66 simulate -to 192.0.2.50` चलाएँ। यह simulated
   डिवाइसों से sFlow, NetFlow और IPFIX भेजता है; वे फिर **सेटिंग्स** में और डेटा
   में दिखते हैं, इसलिए इसके लिए test installation बेहतर है।

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. आँकड़ों को interface counters से मिलाएँ

Flow के आँकड़े अनुमान होते हैं: sampled packets गुणा sampling rate।
traffic66 इन्हें डिवाइस के अपने interface counters से मिलाता है और फ़र्क़
**इंटरफ़ेस मिलान** पर दिखाता है; जब फ़र्क़ अकेले sampling से समझ में आने
लायक से ज़्यादा हो, तो संभावित कारण भी बताता है। हर interface का एक bits/s
और एक packets/s chart होता है जिसमें ingress (हरा) और egress (नीला) हैं;
डिवाइस के अपने counters bits/s chart पर dashed lines में दिखते हैं। सूची में
कोई interface चुनने पर उसके charts दिखते हैं।

![इंटरफ़ेस मिलान: हर interface का ट्रैफ़िक, और डिवाइस के counter के बगल में flow का अनुमान](images/interfaces.png)

तुलना के लिए counters पाने के लिए:

- sFlow डिवाइस counter interval सेट होने पर उन्हें ख़ुद भेजते हैं
  (`sflow counter interval 30` और इसी तरह की कमांड)।
- NetFlow और IPFIX डिवाइसों के लिए **सेटिंग्स → नाम** में एक `snmp` लाइन जोड़ें
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

किसी host या डिवाइस को नाम देने का सबसे तेज़ तरीका: किसी भी पेज पर उसके address
पर क्लिक करें और **नाम दें…** चुनें। नाम टाइप करके Enter दबाएँ; यह तुरंत सेव हो
जाता है और हर जगह खाली address की जगह दिखता है।

नेटवर्क, interfaces और SNMP के लिए **सेटिंग्स → नाम** का इस्तेमाल करें: प्रकार चुनें
(host, नेटवर्क, डिवाइस, interface, SNMP), address और नाम भरें, और **जोड़ें**
पर क्लिक करें। नीचे की table में हर नाम **बदलें** और **हटाएँ** के साथ दिखता
है; वही address फिर से जोड़ने पर पुरानी entry बदल जाती है। सेव करने से पहले
addresses और नेटवर्क जाँचे जाते हैं।

नाम data directory में `inventory.txt` के रूप में सेव होते हैं, हर लाइन में एक
entry। **टेक्स्ट के रूप में बदलें (उन्नत)** वह फ़ाइल दिखाता है, और आप उसे सीधे भी
edit कर सकते हैं (`inventory.txt.example` देखें)। हर लाइन वैकल्पिक है।

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers country=JP

# device names; "unsampled" if it exports every packet (1:1),
# sampling=N if it samples 1:N but does not say so in its export
device 192.0.2.1     Core router
device 192.0.2.9     Branch firewall unsampled
device 192.0.2.20    Edge router sampling=1000

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
  आपका गिना जाए; नाम **शीर्ष 66** में segment के अनुसार समूह बनाने पर और network
  के अनुसार flow के रास्तों में दिखता है। `country=JP` (दो अक्षरों का देश कोड)
  बताता है कि नेटवर्क कहाँ है; तब विश्व मानचित्र उससे उन देशों तक लाइनें
  खींचता है जिनसे वह बात करता है।
- `snmp <device> <community> [<management address>[:port]]`: device वह address
  है जिससे flows आते हैं। जब डिवाइस SNMP का जवाब किसी दूसरे address पर देता हो,
  तो management address जोड़ें। SNMP से पढ़े गए interface descriptions नाम के
  रूप में इस्तेमाल होते हैं, जब तक आप `iface` से interface का नाम न दें।
  डिवाइस की SNMP access list में traffic66 मशीन को अनुमति दें।
- बदलाव **सहेजें** पर क्लिक करते ही लागू होते हैं; restart की ज़रूरत नहीं।

<a id="8-countries-networks-and-threat-lists"></a>

## 8. देश, नेटवर्क और threat lists

देश और नेटवर्क (AS) शुरू से ही काम करते हैं: traffic66 में DB-IP के मुफ़्त **IP to Country Lite** और **IP to ASN Lite** डेटाबेस अंतर्निहित हैं (लाइसेंस [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/); "IP Geolocation by DB-IP", [db-ip.com](https://db-ip.com))। देश और नेटवर्क दिखाने वाले पेज डेटा का स्रोत बताते हैं।

अंतर्निहित प्रति आपके चलाए जा रहे रिलीज़ के समय की है। DB-IP हर महीने नया संस्करण जारी करता है; **सेटिंग्स → देश और नेटवर्क डेटाबेस → DB-IP Lite अभी अपडेट करें** db-ip.com से नवीनतम डाउनलोड करता है (traffic66 चलाने वाले सर्वर को इंटरनेट चाहिए; विफल होने पर वेब UI बताता है)।

आप कोई दूसरा मुफ़्त डेटाबेस भी इस्तेमाल कर सकते हैं। उसे डाउनलोड करें, फिर उसी पेज पर **डेटाबेस फ़ाइल अपलोड करें…** से अपलोड करें। फ़ाइल जाँची जाती है, डेटा डायरेक्टरी में सहेजी जाती है और नए ट्रैफ़िक के लिए तुरंत इस्तेमाल होती है; रीस्टार्ट की ज़रूरत नहीं। पहले से सहेजा गया ट्रैफ़िक वही देश रखता है जिसके साथ वह सहेजा गया था।

| डेटाबेस | क्या देता है | लाइसेंस | कहाँ से मिलेगा |
|---|---|---|---|
| DB-IP Lite (अंतर्निहित) | देश; नेटवर्क | CC BY 4.0, खाता नहीं चाहिए | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country और ASN, `.mmdb` | देश; नेटवर्क | GeoLite2 EULA, मुफ़्त खाता | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite, `ipinfo_lite.mmdb` | देश और नेटवर्क एक ही फ़ाइल में | CC BY-SA 4.0, मुफ़्त खाता | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN, `ip2asn-combined.tsv.gz` | नेटवर्क और उनका देश | PDDL 1.0, खाता नहीं चाहिए | [iptoasn.com](https://iptoasn.com) |

आपकी अपनी फ़ाइलें पहले इस्तेमाल होती हैं; जो उनमें नहीं है उसका जवाब अंतर्निहित DB-IP Lite देता है। फ़ाइल के पास **हटाएँ** बाक़ी पर लौटा देता है। पेज दिखाता है कि क्या इस्तेमाल हो रहा है और हर डेटाबेस की तारीख़।

वेब UI के बिना, फ़ाइल को डेटा डायरेक्टरी में `country.mmdb`, `asn.mmdb`, `both.mmdb` (देश और नेटवर्क वाली एक फ़ाइल, जैसे IPinfo Lite) या `asn.tsv.gz` नाम से कॉपी करें और traffic66 रीस्टार्ट करें।

**भूगोल और नेटवर्क** दूसरे देशों के साथ ट्रैफ़िक को विश्व मानचित्र पर दिखाता है: देश जितना गहरा, ट्रैफ़िक उतना ज़्यादा। किसी देश पर पॉइंटर ले जाएँ तो उसका ट्रैफ़िक दिखता है; क्लिक करके फ़िल्टर करें या उसके flow records खोलें। जब आपके नेटवर्कों का देश दिया गया हो (`net` लाइन पर `country=`, [नाम](#7-names-snmp-and-your-own-networks) देखें), तो उस देश से उन देशों तक लाइनें जाती हैं जिनसे वे ट्रैफ़िक का आदान-प्रदान करते हैं; ज़्यादा ट्रैफ़िक के लिए मोटी लाइन। देशों की सीमाएँ [Natural Earth](https://www.naturalearthdata.com) (सार्वजनिक डोमेन) से हैं।

![भूगोल और नेटवर्क: विश्व मानचित्र पर देश के अनुसार बाहरी ट्रैफ़िक](images/geo.png)

Threat lists सादी text फ़ाइलें होती हैं, हर लाइन में एक address या नेटवर्क
(`#` या `;` के बाद का टेक्स्ट नज़रअंदाज़ होता है), जो
`<data directory>/threats/<name>.txt` के रूप में सेव होती हैं, उदाहरण के लिए:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Lists जोड़ने या बदलने के बाद traffic66 restart करें। Matches
**ख़तरे की जानकारी** पर list के नाम के हिसाब से दिखते हैं।

![ख़तरे की जानकारी: एक internal host जो threat list के किसी address को डेटा भेज रहा है](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. web UI का इस्तेमाल

आपको शायद ही कुछ टाइप करना पड़े। हर पेज की हर value — address, port,
application, देश, डिवाइस — पर क्लिक किया जा सकता है:

- **केवल यही दिखाएँ** / **इसे हटाएँ** एक filter जोड़ता है। Filters top bar के
  नीचे दिखते हैं और हटाए जाने तक हर पेज पर लागू रहते हैं।
- **इसके फ़्लो रिकॉर्ड देखें** मेल खाते अलग-अलग flows खोलता है।
- **विवरण देखें** (hosts, डिवाइस और services) उस एक host या service के बारे में
  एक पेज खोलता है: समय के साथ application के हिसाब से उसका ट्रैफ़िक, वह किससे
  बात करता है, कौन-सी services या clients, देश और उसके नवीनतम flows। वहाँ की हर
  value पर फिर से क्लिक किया जा सकता है, ताकि आप और गहराई में जा सकें; browser
  का Back बटन वापस ले जाता है।
- **नाम दें…** (hosts और डिवाइस) address को एक नाम देता है, जो उसके बाद हर जगह
  दिखता है।
- **ऑनलाइन खोजें** address या AS को किसी public lookup साइट पर खोलता है।
- **कॉपी करें** value कॉपी करता है।

पेज:

| पेज | किस सवाल का जवाब देता है |
|---|---|
| सारांश | अभी कितना ट्रैफ़िक है, application के हिसाब से (**कुल**, या आपके नेटवर्क का सिर्फ़ **अंदर आने वाला** या **बाहर जाने वाला** ट्रैफ़िक), कल इसी समय (एक दिन तक की ranges), पिछले हफ़्ते (एक हफ़्ते तक) या पिछले दिनों (लंबी ranges) की तुलना में, अगर उस समय का डेटा हो; खुली संदिग्ध गतिविधियाँ; दिशा और protocol; top clients और services |
| शीर्ष 66 | **तालिका** पर खुलता है, top 66 की एक table: डिफ़ॉल्ट रूप से conversations (client, server, service, देश)। हर column heading से sort होता है; संख्या वाले columns (ट्रैफ़िक, पैकेट, औसत पैकेट, flows) समय-सीमा के पूरे ट्रैफ़िक से top 66 फिर से चुनते हैं, इसलिए सबसे छोटे औसत पैकेट से scan और flood पकड़े जाते हैं। **इसके अनुसार समूह** से applications, नेटवर्क, segments, डिवाइस, encapsulation और VLAN पर जाएँ। **सबसे सक्रिय होस्ट** top 30 clients और servers साथ-साथ दिखाता है, ट्रैफ़िक, पैकेट और flow records के साथ, पूरे ट्रैफ़िक वाली एक row के ऊपर |
| ट्रैफ़िक विवरण | दो ring charts। **सर्वर और क्लाइंट**: अंदर का घेरा सबसे व्यस्त 8 servers, बाहर का घेरा हर server के clients; **क्लाइंट अंदर** इसे उलट देता है (clients अंदर, हर client जिन servers का इस्तेमाल करता है वे बाहर), क्योंकि अक्सर एक तरफ़ दूसरी से ज़्यादा समझाती है। **सेवाएँ**: सबसे व्यस्त services का एक घेरा। किसी हिस्से पर पॉइंटर ले जाएँ तो उसका ट्रैफ़िक दिखता है; किसी भी value की तरह उस पर क्लिक करें |
| ट्रैफ़िक के रास्ते | कौन-सा host किस देश की ओर किस application का इस्तेमाल करता है: सबसे व्यस्त 8 hosts, बाकी "अन्य" के रूप में। **क्लाइंट → सर्वर** client → service → server दिखाता है; **सेगमेंट के अनुसार** hosts की जगह नेटवर्क दिखाता है। लंबे नाम 22 अक्षरों तक छोटे किए जाते हैं; पूरा नाम देखने के लिए उस पर पॉइंटर ले जाएँ |
| संदिग्ध गतिविधि | किस पर ध्यान देना है: scans, पासवर्ड का अनुमान, lateral movement, असामान्य uploads, floods और threat list वाला ट्रैफ़िक ([और जानें](#findings)) |
| ख़तरे की जानकारी | वे hosts जिन्होंने आपकी threat lists के addresses से बात की, और उन्होंने कितना भेजा |
| भूगोल और नेटवर्क | देश के अनुसार ट्रैफ़िक का विश्व मानचित्र, आपके नेटवर्कों से लाइनों के साथ; वे नेटवर्क (AS) जहाँ से ट्रैफ़िक आया और जहाँ गया, समय के साथ bits/s और packets/s में; देश के हिसाब से और नेटवर्क के हिसाब से ट्रैफ़िक |
| सेटिंग्स | डिवाइस, sampling, loss, collectors, SNMP, देश और नेटवर्क डेटाबेस, लोगो, और **नाम** |
| इंटरफ़ेस मिलान | समय के साथ हर interface का ट्रैफ़िक bits/s और packets/s में, ingress (हरा) और egress (नीला), डिवाइस counters dashed lines के रूप में; flow के आँकड़े counters से कितने दूर हैं, सबसे ख़राब पहले, कारणों के साथ |
| फ़्लो रिकॉर्ड | कितने flow records थे और कब (हर interval के लिए एक bar), और ख़ुद records, सबसे नए पहले, पेज-दर-पेज, चुने जा सकने वाले columns के साथ। पिछले 15 मिनट पर खुलता है और हर 5 सेकंड में अपडेट होता है; किसी दूसरे पेज की value से (**इसके फ़्लो रिकॉर्ड देखें**) खोलने पर उस पेज की समय-सीमा बनी रहती है, और **लाइव पर लौटें** वापस ले आता है |
| डेटा सफ़ाई | 120, 90, 60, 30 या 7 दिन से पुराना डेटा, या पूरा डेटा, हटाता है, और बताता है कि हर विकल्प से कितनी जगह खाली होगी ([और](#13-data-backup-upgrade-uninstall)) |
| ऑफ़लाइन pcap विश्लेषण | pcap, pcapng कैप्चर का लाइव डेटा से अलग विश्लेषण ([और](#ऑफ़लाइन-pcap-विश्लेषण)) |

साइड मेनू पेजों को चार समूहों में दिखाता है: ट्रैफ़िक (सारांश, शीर्ष 66, ट्रैफ़िक
विवरण, ट्रैफ़िक के रास्ते, इंटरफ़ेस मिलान), सुरक्षा (संदिग्ध गतिविधि, ख़तरे की
जानकारी, भूगोल और नेटवर्क), सेटअप और डेटा (सेटिंग्स, फ़्लो रिकॉर्ड, डेटा सफ़ाई)
और ऑफ़लाइन pcap विश्लेषण। लोगो के नीचे version और server की तारीख़ और समय हैं।

पेजों के ऊपर: time range (15 मिनट से 30 दिन, या किसी भी शुरुआत और अंत के लिए
**कस्टम…**, 30 दिन से पहले का भी), हर 30 सेकंड पर automatic refresh, और
**लिंक कॉपी करें**, जो ठीक मौजूदा view (पेज, time range और filters) का link
कॉपी करता है ताकि आप उसे किसी सहकर्मी को भेज सकें। **शीर्ष 66** और **ट्रैफ़िक
विवरण** पर, **डिवाइस**, **क्लाइंट**, **सर्वर** और **सेवा** समय-सीमा की सबसे
व्यस्त values दिखाते हैं: filter करने के लिए एक चुनें या टाइप
करें; फिर box ख़ाली करने तक filter हर पेज पर लागू रहता है। भाषा browser के
हिसाब से चुनी जाती है; menu के नीचे, **लॉग आउट** के ऊपर से बदलें।
सेटिंग्स, फ़्लो रिकॉर्ड (लाइव रहते हुए), डेटा सफ़ाई और ऑफ़लाइन pcap विश्लेषण
में कोई समय-सीमा नहीं होती।

भाषा के बगल में रंग की थीम है, iOS के system रंगों के आधार पर: **हल्का**
(default), **स्लेटी**, **काला** (दीवार पर लगी screens के लिए), **फ़िरोज़ी** और
**नारंगी**। उस पर हर click अगली थीम पर ले जाता है; चुनाव browser में रखा जाता है।

समय वाले charts सबसे बड़ी 8 values तय रंगों में और बाकी को "अन्य" के रूप में
दिखाते हैं; legend हर value का कुल देता है और उस पर किसी भी दूसरी value की तरह
click किया जा सकता है। clients और servers के charts बाकी को drawing से बाहर
रखते हैं, क्योंकि हज़ारों hosts के साथ वह top 8 को चपटा कर देता; legend फिर भी
उसका कुल देता है।

charts वहीं तक जाते हैं जहाँ तक डेटा पूरा है: sFlow के साथ मौजूदा मिनट तक,
NetFlow और IPFIX के साथ थोड़ा पहले तक, उतना जितना डिवाइसों को अपने flows
export करने में लगता है (traffic66 इसे मापता है; ज़्यादा से ज़्यादा 2 मिनट)।

6 घंटे से लंबी ranges पूरे घंटे से शुरू होती हैं, ताकि पेज का हर आँकड़ा ठीक
एक जैसा समय गिने: "24 घंटे" में पिछले 24 पूरे घंटे और मौजूदा घंटा शामिल हैं।
इन ranges पर शीर्ष 66 घंटेवार summaries से आता है; वहाँ filters उपलब्ध नहीं
हैं, और पेज यह बता देता है। filter करने के लिए छोटी range चुनें। बातचीत (conversations) हमेशा flow detail पढ़ती है, इसलिए
ऊँची flow दरों पर लंबी ranges में इसमें कुछ समय लग सकता है; एक घंटा सबसे तेज़ है।

side menu दिखाता है कि data कितनी disk इस्तेमाल करता है और कितनी ख़ाली है;
ख़ाली जगह पर hover करें तो दिखता है कि मौजूदा दर पर रखे गए दिनों के detail को
कितनी जगह चाहिए (एक दिन का data होने के बाद अनुमान लगाया जाता है)।

साइन-इन पेज पर और menu के ऊपर अपना लोगो दिखाने के लिए
**सेटिंग्स → लोगो → लोगो अपलोड करें…** का इस्तेमाल करें: PNG, SVG, JPEG, WebP या
GIF, 1 MB तक, 272 × 92 pixels पर सबसे अच्छा (दूसरे आकार फ़िट होने के लिए scale
किए जाते हैं)। **बिल्ट-इन लोगो वापस लाएँ** से traffic66 का लोगो वापस आ जाता है।

<a id="findings"></a>

### संदिग्ध गतिविधि

**संदिग्ध गतिविधि** वह सब दिखाता है जो traffic66 ने flows में पाया, सबसे गंभीर
पहले। यह हर 5 मिनट पर पिछले 10 मिनट जाँचता है; जो चीज़ एक घंटे तक चलती है वह
एक ही प्रविष्टि है जो बढ़ती जाती है, हर जाँच पर नई नहीं।

| गतिविधि | इसका मतलब | गंभीरता |
|---|---|---|
| स्कैन | एक address ने एक ही port पर कई addresses को छोटे probes भेजे (TCP या ping) | आपके नेटवर्क के अंदर से हो तो उच्च, इंटरनेट से हो तो निम्न |
| पोर्ट स्कैन | एक address ने एक host के कई ports पर छोटे probes भेजे | अंदर से उच्च, इंटरनेट से निम्न |
| पासवर्ड का अनुमान | किसी login service (SSH, RDP, SMB, databases और अन्य) से कई छोटे connections | अंदर से उच्च, इंटरनेट से निम्न |
| लेटरल मूवमेंट | आपके नेटवर्क के अंदर, file sharing या remote administration sessions (SMB, RDP, SSH, WinRM, VNC) ऐसे hosts से जिन्होंने पहले कभी वह service नहीं दी | उच्च |
| असामान्य अपलोड | एक internal host ने जितना पाया उससे कहीं ज़्यादा भेजा (10 मिनट में 100 MB, पाए गए डेटा का तीन गुना) ऐसे address को जिससे उसने पहले कभी डेटा का लेन-देन नहीं किया | उच्च |
| फ़्लड | एक address पर प्रति सेकंड 20,000 या उससे ज़्यादा छोटे packets, उसकी सामान्य दर का दस गुना | मध्यम |
| ख़तरा सूची | आपकी किसी threat list के address के साथ ट्रैफ़िक | उच्च जब आपके host ने उससे connect किया, निम्न जब सूची वाला address बाहर से दस्तक दे रहा था |

हर प्रविष्टि बताती है कि किसने किसके साथ क्या किया, कब और कितनी देर तक, इसके
पीछे के आँकड़ों के साथ, और यह भी कि data कैसे sample किया गया था। **विवरण**
host का पेज खोलता है, जिसमें उसके बारे में मिली गतिविधियाँ भी दिखती हैं।
**निपटा दिया** प्रविष्टि को बंद करता है; अगर वही फिर होता है तो नई प्रविष्टि
खुलती है। **कोई समस्या नहीं** उसे हमेशा के लिए बंद करता है: वह फिर कभी report
नहीं होती। side menu में **संदिग्ध गतिविधि** के बगल का लाल नंबर पिछले 24 घंटों
की खुली उच्च और मध्यम प्रविष्टियाँ गिनता है।

लेटरल मूवमेंट और असामान्य uploads के लिए यह जानना ज़रूरी है कि सामान्य क्या
है, इसलिए वे एक दिन का इतिहास होने के बाद ही report होते हैं। पहली बार शुरू
होने पर traffic66 पहले से मौजूद इतिहास से सीखता है।

Sampled data (sFlow, sampled NetFlow) के साथ नियम वही गिनते हैं जो samples
दिखाते हैं और कम संख्या माँगते हैं, लेकिन तब हर sample एक छोटे probe जैसा
दिखना चाहिए, ताकि व्यस्त सामान्य hosts इन्हें trigger न करें। जो sampling छिपा
देता है वह पकड़ा नहीं जा सकता: 1:4096 sampling के पीछे, कुछ दर्जन hosts का
scan इतने कम packets भेजता है कि दिखता नहीं। डेमो का हमला एक ऐसे switch से
गुज़रता है जो 1:4096 पर sample करता है, और पूरा पकड़ा जाता है; डेमो के एक दिन
के सामान्य ट्रैफ़िक से कोई गतिविधि नहीं निकलती, सिवाय website पर दस्तक देते
internet scanner के।

![संदिग्ध गतिविधि: एक हमले का हर चरण, 1:4096 sFlow sampling के बावजूद पकड़ा गया](images/findings.png)

![शीर्ष 66: top 66 conversations, किसी भी column से sort होने वाली](images/topn.png)

![ट्रैफ़िक विवरण: ring charts के रूप में servers अपने clients के साथ, और services अपने servers के साथ](images/traffic.png)

![एक host का विवरण: उसके बारे में मिली संदिग्ध गतिविधियाँ, उसका ट्रैफ़िक, वह किससे बात करता है, services, देश और नवीनतम flows](images/detail.png)

![ट्रैफ़िक के रास्ते: कौन-सा host किस देश की ओर किस application का इस्तेमाल करता है](images/paths.png)

यही सारांश चीनी में; हर पेज 13 भाषाओं में उपलब्ध है:

![चीनी में सारांश](images/overview-zh.png)

<a id="10-terminal-ui"></a>

### ऑफ़लाइन pcap विश्लेषण

**ऑफ़लाइन pcap विश्लेषण** Wireshark या tcpdump के कैप्चर को लाइव डेटा वाले पेजों पर ही दिखाता है, बिना उन्हें मिलाए।

यह सभी पैकेटों को फ़्लो में समेटता है: किसने किससे, कितनी, कब बात की, और क्या हमले जैसा दिखता है। यह प्रोटोकॉल डिकोड नहीं करता और पैकेट की सामग्री नहीं दिखाता; एक पैकेट या एक TCP स्ट्रीम देखने के लिए Wireshark इस्तेमाल करें।

कमांड लाइन से, बिना कोई सेटअप किए:

```
traffic66 office.pcap
traffic66 a.pcap b.pcapng c.pcap
```

traffic66 सिर्फ़ इसी कंप्यूटर पर (127.0.0.1, कोई खाली पोर्ट) शुरू होता है, पता, पासवर्ड और एक बार चलने वाला साइन-इन लिंक छापता है, और ब्राउज़र में कैप्चर खोल देता है। अधिकतम 3 फ़ाइलें, कुल 3 GB; फ़ाइलें अपनी जगह से पढ़ी जाती हैं और कभी बदली नहीं जातीं। कुछ भी इकट्ठा या भेजा नहीं जाता, और होस्ट नाम नहीं खोजे जाते (`-dns` से चालू करें)। Ctrl+C रोकता है और आयात किया डेटा मिटा देता है। 2-कोर मशीन पर 1 GB का कैप्चर लगभग 5 सेकंड (12 लाख पूरे आकार के पैकेट) से 30 सेकंड (1.4 करोड़ छोटे पैकेट) में तैयार होता है।

```
$ traffic66 office.pcap

traffic66 0.3.1: analysing 1 capture file(s); nothing is collected or sent
  Web UI    http://127.0.0.1:38217  (port 38217, this computer only)
  Sign in   user admin, password gfhfhbuutz2e
  Open      http://127.0.0.1:38217/auto?t=b9388f…  (signs in once)
  Stop      Ctrl+C; the imported data is deleted, your files are kept
```

चल रहे traffic66 के वेब UI में:

1. **कैप्चर फ़ाइलें अपलोड करें…**: `.pcap` या `.pcapng`, संपीड़ित नहीं। अधिकतम 3 फ़ाइलें, हर एक 50 MB तक। फ़ाइलें फ़्लो में बदलकर अलग डेटाबेस (`<data>/sandbox/`) में जाती हैं; लाइव डेटा, उसके आँकड़े और निष्कर्ष नहीं बदलते।
2. **विश्लेषण करें**: सभी पेज (अवलोकन, Top 66, ट्रैफ़िक विवरण, निष्कर्ष, फ़्लो पथ, मानचित्र, फ़्लो रिकॉर्ड) कैप्चर फ़ाइलों का पूरा समय दिखाते हैं। नारंगी पट्टी फ़ाइलों के नाम बताती है; **लाइव डेटा पर लौटें** से लौटें। हर फ़ाइल एक डिवाइस की तरह दिखती है, इसलिए **डिवाइस** बॉक्स से एक-एक फ़ाइल देखी जा सकती है।
3. पहचान के नियम कैप्चर पर भी चलते हैं: स्कैन, पोर्ट स्कैन और पासवर्ड अनुमान **संदिग्ध गतिविधि** में दिखते हैं। जिन नियमों को एक दिन का इतिहास चाहिए (लेटरल मूवमेंट, असामान्य अपलोड) वे कैप्चर पर लागू नहीं होते।
4. **हटाएँ** एक फ़ाइल और उसका डेटा हटाता है; **सब हटाएँ** सब कुछ हटाता है।

डेमो में हमले वाला एक उदाहरण कैप्चर शामिल है।

![ऑफ़लाइन विश्लेषण: कैप्चर फ़ाइलें, उनके पैकेट, फ़्लो और समय](images/sandbox.png)

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

![Terminal UI: सारांश](images/tui-overview.png)

![Terminal UI: शीर्ष 66 conversations](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Local capture

Flow exports पाने के अलावा, traffic66 जिस मशीन पर चलता है उसके किसी network
interface के packets से ख़ुद भी flows बना सकता है। उसे क्या दिखता है, यह
interface पर निर्भर करता है:

| Interface | traffic66 को क्या दिखता है |
|---|---|
| switch के mirror (SPAN) port से जुड़ा एक ख़ाली network port | वह सारा ट्रैफ़िक जो switch mirror करता है: पूरा नेटवर्क या uplink |
| मशीन का अपना Ethernet या Wi-Fi | केवल इसी मशीन का अपना ट्रैफ़िक |

Wi-Fi adapters दूसरे डिवाइसों का ट्रैफ़िक नहीं देख सकते। पूरे Wi-Fi नेटवर्क को
देखने के लिए router या access point से flows export करवाएँ (अनुभाग 4), या उस
switch port को mirror करें जिससे access point जुड़ा है।

<a id="windows-1"></a>

### Windows

1. [Npcap](https://npcap.com) को उसके default options के साथ इंस्टॉल करें। अगर
   आप "Restrict Npcap driver's access to Administrators only" पर tick लगाते
   हैं, तो traffic66 को Administrator के रूप में चलाएँ।
2. Interfaces की सूची देखें (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Name column में Windows की network settings वाला connection का नाम होता है;
   जो interface इस्तेमाल में है, उसका एक address होता है।
3. Wi-Fi पर capture करें, नाम से या नंबर से:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   जिन नामों में spaces हों, उन्हें quotes में रखें: `-capture "Ethernet 2"`।
   कई interfaces पर capture करने के लिए `-capture` दोहराएँ। अगर आपको केवल
   capture चाहिए और कोई flow collector नहीं, तो `-listen=` जोड़ें। अनुभाग 2 के
   startup task के लिए option को `-Argument` में जोड़ें:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`।

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

Capture के लिए root चाहिए, या capabilities `CAP_NET_RAW` और `CAP_NET_ADMIN`:
ऊपर वाली `setcap` लाइन, या अनुभाग 2 की systemd unit में `AmbientCapabilities`
वाली लाइन। Wi-Fi interfaces के नाम आमतौर पर `wlan0` या `wlp…` होते हैं।

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

Capture के लिए root चाहिए; कुछ इंस्टॉल नहीं करना। MacBooks पर `en0` Wi-Fi है।

<a id="checking-that-it-works"></a>

### जाँचें कि यह काम कर रहा है

**सेटिंग्स** हर capture किए जा रहे interface को capture method और देखे गए packets
की संख्या के साथ दिखाता है। Flows डिवाइस `127.0.0.1` (यह मशीन) से आते हुए
दिखते हैं, हर पेज पर, किसी भी दूसरे डिवाइस के flows की तरह। दो बार दिखे
packets (जैसे दो mirror ports पर) दो बार गिने जाते हैं।

<a id="12-options"></a>

## 12. विकल्प

`traffic66 -h` और `traffic66 <command> -h` सब कुछ सूचीबद्ध करते हैं।

कमांड:

| कमांड | |
|---|---|
| `traffic66` | flows इकट्ठा करता है और web UI चलाता है |
| `traffic66 demo` | वही, एक simulated नेटवर्क के साथ |
| `traffic66 tui` | चल रहे traffic66 के लिए terminal UI |
| `traffic66 passwd` | यूज़र जोड़ता, बदलता, उनकी सूची दिखाता या उन्हें हटाता है (देखें [यूज़र और पासवर्ड](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | किसी collector को simulated exports भेजता है |
| `traffic66 interfaces` | local capture के लिए interfaces की सूची |
| `traffic66 version` | version छापता है |

`traffic66` और `traffic66 demo` के विकल्प:

| विकल्प | डिफ़ॉल्ट | |
|---|---|---|
| `-addr` | `:8066` | web UI का address; केवल इसी मशीन के लिए `127.0.0.1:8066` |
| `-data` | प्रोग्राम के बगल में `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors, `name=address` के रूप में, comma से अलग; ख़ाली हो तो बंद |
| `-user` | `admin` | पहली बार शुरू होने पर बनाए गए यूज़र का नाम, और उस यूज़र का जिस पर `-password` लागू होता है |
| `-password` | सेट नहीं | इस run में केवल इसी पासवर्ड के साथ `-user` को स्वीकार करता है, `password` फ़ाइल को अनदेखा करके (`TRAFFIC66_PASSWORD` भी) |
| `-retention-days` | `30` | flow detail कितने दिन रखा जाए; summaries 400 दिन रखी जाती हैं |
| `-memory` | `0.10` | physical memory का कितना हिस्सा database cache के लिए, और उतना ही बाकी प्रोग्राम के लिए soft limit (हर एक कम से कम 256 MB) |
| `-l2-overhead` | `18` | NetFlow/IPFIX byte counts में हर packet पर जोड़े जाने वाले bytes |
| `-sampling-wait` | `5m` | records sampling rate का कितनी देर इंतज़ार करें |
| `-capture` | | local interface पर capture (दोहराया जा सकता है) |
| `-inventory` | `<data>/inventory.txt` | नामों की फ़ाइल |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table (`.mmdb` फ़ाइलें: उन्हें अपलोड करें, या `<data>/country.mmdb` और `<data>/asn.mmdb`, `<data>/both.mmdb`) |
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
| `inventory.txt` | नाम (**सेटिंग्स → नाम**) |
| `license.json` | installation number और लाइसेंस ([ट्रायल और लाइसेंस](#trial-and-licence) देखें) |
| `logo.png` (या `.svg`, `.jpg`, `.webp`, `.gif`) | आपका लोगो (**सेटिंग्स → लोगो**), अगर आपने अपलोड किया है |
| `country.mmdb`, `asn.mmdb`, `both.mmdb`, `asn.tsv.gz`, `dbip-country.mmdb`, `dbip-asn.mmdb`, `threats/`, `sandbox/` | आपके जोड़े हुए देश और नेटवर्क databases और threat lists |

**डेटा कितने समय तक रखा जाता है**: flow detail 30 दिन, summaries (overview और लंबी समय-सीमाएँ) 400 दिन।
इससे पुराना डेटा अपने आप हटता है, हर 5 मिनट में जाँच होती है; इसके अलावा कुछ नहीं हटता और कोई और सीमा नहीं है।
detail की अवधि `-retention-days` से बदलें, कितने भी दिन, उदाहरण के लिए `-retention-days 365`। डिस्क का उपयोग
उसी के साथ बढ़ता है: रखे गए दिन न समाएँ तो साइड मेनू में **खाली** लाल हो जाता है। डिस्क भर जाए तो जगह खाली होने तक
नए flows सहेजे नहीं जा सकते।

साइड मेनू में **डेटा सफ़ाई** ज़रूरत पड़ने से पहले ही डेटा हटाता है: 120, 90, 60,
30 या 7 दिन से पुराना, या पूरा डेटा। हर विकल्प के लिए यह दिखाता है कि कितने
flow records हटेंगे और लगभग कितनी डिस्क जगह खाली होगी, और हटाने से पहले
पूछता है। flow records, घंटेवार और दैनिक summaries, interface counters और
संदिग्ध गतिविधियाँ हटती हैं; पूरा डेटा हटाने पर detection rules ने जो सीखा है
वह भी रीसेट हो जाता है। इसे वापस नहीं किया जा सकता।

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

<a id="trial-and-licence"></a>

### ट्रायल और लाइसेंस

traffic66 source available है,
[PolyForm Noncommercial License 1.0.0](../LICENSE.md) और
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) के तहत; दोनों का
अंग्रेज़ी पाठ ही बाध्यकारी है। संक्षेप में:

- **Evaluation**, testing, development और demonstration: सबके लिए मुफ़्त,
  बिना समय-सीमा के।
- 100 से कम कर्मचारियों और contractors वाले संगठन द्वारा **production use**
  (असली traffic, संगठन के कामकाज के लिए): मुफ़्त।
- **बड़े संगठनों** द्वारा production use: 30 दिन मुफ़्त, फिर लेखक से
  registration licence लेना ज़रूरी है।
- Contractors और service providers इसे किसी customer के लिए, उसी customer के
  अपने deployment में चला सकते हैं; customer का आकार तय करता है।
- Commercial licence के बिना अनुमति नहीं: इसे बेचना या किसी product में
  जोड़ना, इसे दूसरों को hosted या multi-tenant service के रूप में देना, या
  प्रतिस्पर्धी product।

Registration licence की फ़ीस, दायरा और अवधि हर मामले में अलग से तय होते हैं,
और यह मुफ़्त भी हो सकता है। संपर्क: <https://github.com/githubflyideas/traffic66>।

हर installation ट्रायल दिखाता है, वहाँ भी जहाँ लाइसेंस की ज़रूरत नहीं है। पहली
बार शुरू होने पर traffic66 data
directory में 8 अंकों के installation number के साथ `license.json` लिखता है।
हर पेज के नीचे दिखता है कि ट्रायल के कितने दिन बचे हैं, और फिर यह कि ट्रायल
ख़त्म हो गया है। दोनों ही हालत में कुछ बंद नहीं होता: हर feature चलता रहता है।

रजिस्टर करने के लिए लेखक को installation number भेजें (यह हर पेज के नीचे भी
दिखता है)। लाइसेंस एक नई `license.json` के रूप में वापस आता है; उसे पुरानी की
जगह data directory में रखें। traffic66 शुरू होने पर और हर 4 घंटे में इसकी जाँच
होती है, इसलिए restart की ज़रूरत नहीं; फिर पेज के नीचे दिखता है कि लाइसेंस
किसके नाम है और कितने दिन बचे हैं।

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
disk लेता है, साथ में मौजूदा घंटे के लिए लगभग 1.5 GB; प्रोग्राम एक core का
छठा हिस्सा लेता है। लंबी time ranges के overviews
summaries से आते हैं और 0.2 s से कम लेते हैं। Detail पर queries हर घंटे के
लगभग 2.2 करोड़ rows scan करती हैं: 1 घंटे में एक host 1 s से कम लेता है, सभी
conversations का 1 घंटे का Top 66 लगभग 9 s; समय range के साथ बढ़ता है और
ज़्यादा cores के साथ घटता है।

इसलिए 5,000 flows/s पर 30 दिनों के लिए disk लगभग 360 GB है; इसे अपने flow rate
(**सेटिंग्स** पर दिखता है) और `-retention-days` के अनुपात में बढ़ाएँ-घटाएँ।

Memory: `-memory` (डिफ़ॉल्ट RAM का 10%, कम से कम 256 MB) database cache को
सीमित करता है, और बाकी प्रोग्राम को उतने ही आकार की soft limit मिलती है।
5,000 flows प्रति सेकंड पर प्रोग्राम का अपना data (decoding, duplicate
detection, batches) लगभग 90 MB लेता है; कुल मिलाकर 0.6–0.8 GB मानकर चलें,
इसलिए 2 GB RAM वाली मशीन काफ़ी है। लगातार 10 मिनट collection के दौरान
(8 GB मशीन पर peak 0.58 GB) और 2 GB मशीन की limits के साथ इससे ग्यारह गुना
rate पर एक घंटे के flows load करते समय (peak 0.74 GB) मापा गया।

`-memory` एक budget है, hard cap नहीं: Go की limit soft है और database
थोड़ी देर के लिए अपने हिस्से से ज़्यादा ले सकता है। Hard cap के लिए
operating system की limit इस्तेमाल करें: systemd unit में `MemoryMax=`
(अनुभाग 2) या container की memory limit। `-memory` वाले हिस्से का लगभग
2.5 गुना और कम से कम 1 GB रखें; डिफ़ॉल्ट हिस्से पर 8 GB तक की मशीनों के
लिए `MemoryMax=2G` ठीक है। तब मशीन की memory खत्म होने के बजाय traffic66
restart हो जाता है।

<a id="16-troubleshooting"></a>

## 16. समस्या निवारण

| लक्षण | कारण और समाधान |
|---|---|
| डिवाइस **सेटिंग्स** में नहीं है | Packets नहीं पहुँच रहे: [जाँचें कि flows पहुँच रहे हैं](#5-check-that-flows-arrive) देखें |
| "waiting for the sampling rate" | डिवाइस ने अभी तक अपने sampler options नहीं भेजे; ज़्यादातर कुछ मिनटों में दोबारा भेज देते हैं। अगर कभी न भेजे, तो उन्हें export करवाएँ (Cisco पर `option sampler-table`) या अगर वह सच में 1:1 है तो नाम में उसे `unsampled` लिखें, या उसकी `device` line पर `sampling=N` से rate दें। फिर **सेटिंग्स** में डिवाइस द्वारा भेजे गए templates दिखते हैं, जिससे पता चलता है कि वह क्या घोषित करता है |
| आँकड़े interface counters से कम | **इंटरफ़ेस मिलान** देखें: रास्ते में loss, interfaces sample नहीं हो रहे, या flows अभी डिवाइस के cache में हैं (active timeout 60 s से लंबा) |
| आँकड़े interface counters से ज़्यादा | वही ट्रैफ़िक दो interfaces या दो डिवाइसों पर sample हो रहा है |
| कोई देश या नेटवर्क नहीं ("अज्ञात") | कोई database लोड नहीं है: **सेटिंग्स** पर एक अपलोड करें, [देश](#8-countries-networks-and-threat-lists) देखें |
| पेज पर "डेटाबेस अपनी मेमोरी सीमा तक पहुँच गया और जवाब नहीं दे सका" | छोटी समय सीमा चुनें, या बड़े `-memory` के साथ शुरू करें; विवरण log में है |
| पासवर्ड भूल गए | traffic66 मशीन पर `traffic66 passwd` (अगर traffic66 `-data` के साथ चलता है तो `-data` जोड़ें) |
| `Conflicting lock is held` | कोई दूसरा traffic66 पहले से यही data directory इस्तेमाल कर रहा है |
| `receive buffer is only … KB` | Linux UDP buffers सीमित रखता है: `net.core.rmem_max=16777216` सेट करें ([Linux](#linux) देखें) |
| `cannot create the data directory` | इस यूज़र के लिए प्रोग्राम फ़ोल्डर writable नहीं है: `-data` दें |
| macOS: "cannot be opened" या "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" (Windows ने आपके PC को सुरक्षित किया) | **More info** (अधिक जानकारी) → **Run anyway** (फिर भी चलाएँ); प्रोग्राम अभी signed नहीं है |
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
