[English](../README.md) | [中文](README.zh.md) | **हिन्दी** | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow, sFlow और IPFIX कलेक्टर तथा ट्रैफ़िक एनालाइज़र

sFlow, NetFlow और IPFIX के लिए flow analytics, एक ही प्रोग्राम में: bandwidth
कौन इस्तेमाल कर रहा है, ट्रैफ़िक कहाँ जा रहा है, और क्या आँकड़े डिवाइसों के
अपने interface counters से मेल खाते हैं — web UI में भी और terminal UI में भी।

## क्विक स्टार्ट

```
# Linux, macOS: ./traffic66 में डाउनलोड करें और flows लेना शुरू करें
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: वही
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

टर्मिनल web UI का पता और डिवाइसों के लिए IP व UDP पोर्ट दिखाता है ([डिवाइस कॉन्फ़िगरेशन](MANUAL.hi.md#device-configuration))। `admin` / `traffic66` से साइन इन करके नया पासवर्ड चुनें। सिर्फ़ देखना है? सिम्युलेटेड डिवाइसों वाला डेमो:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

पोर्ट: TCP 8066 web UI, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (हर UDP पोर्ट हर प्रोटोकॉल लेता है)। टर्मिनल बताता है कि डिवाइसों को किस पते पर भेजना है; पहले साइन-इन पर नया पासवर्ड माँगा जाता है।

macOS पर हाथ से: पहले `xattr -dr com.apple.quarantine traffic66-darwin-*`। Windows "protected your PC": **More info** → **Run anyway**।

## कमांड चीट शीट

| काम | कमांड |
|---|---|
| sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 लें (डिफ़ॉल्ट) | `./traffic66` |
| सिर्फ़ NetFlow v5/v9 (2055) और IPFIX (4739) | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| सिर्फ़ sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Web UI किसी IP और पोर्ट पर | `./traffic66 -addr 10.0.0.5:8080` |
| इसी मशीन का NIC कैप्चर करें (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin पासवर्ड सेट/रीसेट करें (-data DIR हो तो जोड़ें) | `./traffic66 passwd` |
| रिमोट सर्वर से Terminal UI | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

## स्क्रीनशॉट

| | |
|---|---|
| <a href="images/overview.png"><img src="images/overview.png" alt="सारांश"></a><br>सारांश | <a href="images/topn.png"><img src="images/topn.png" alt="शीर्ष 66"></a><br>शीर्ष 66 |
| <a href="images/geo.png"><img src="images/geo.png" alt="भूगोल और नेटवर्क"></a><br>भूगोल और नेटवर्क | <a href="images/interfaces.png"><img src="images/interfaces.png" alt="इंटरफ़ेस मिलान"></a><br>इंटरफ़ेस मिलान |
| <a href="images/tui-top.png"><img src="images/tui-top.png" alt="TUI"></a><br>TUI | |

## पूरा मैनुअल

web UI के मेनू में **सहायता**; GitHub पर: [MANUAL.hi.md](MANUAL.hi.md)

15 भाषाएँ। evaluation के लिए और 100 से कम लोगों वाले संगठनों के लिए मुफ़्त ([लाइसेंस](../LICENSE.md))।
