[English](../README.md) | [中文](README.zh.md) | **हिन्दी** | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow, sFlow और IPFIX कलेक्टर तथा ट्रैफ़िक एनालाइज़र

sFlow, NetFlow और IPFIX के लिए flow analytics, एक ही प्रोग्राम में: bandwidth
कौन इस्तेमाल कर रहा है, ट्रैफ़िक कहाँ जा रहा है, और क्या आँकड़े डिवाइसों के
अपने interface counters से मेल खाते हैं — web UI में भी और terminal UI में भी।

## क्विक स्टार्ट

1. अपने सिस्टम का आर्काइव डाउनलोड करें: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. खोलें और प्रोग्राम चलाएँ:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: `traffic66-windows-amd64.zip` खोलें और `traffic66.exe` चलाएँ (डबल-क्लिक भी चलेगा)।

macOS: पहले उस फ़ोल्डर में `xattr -dr com.apple.quarantine .` चलाएँ।

डाउनलोड की जाँच: `sha256sum -c SHA256SUMS --ignore-missing`; अपग्रेड, वापसी और रिवर्स प्रॉक्सी के लिए [मैनुअल](MANUAL.hi.md#12-install-upgrade-reverse-proxy-monitoring) देखें।

फिर:

- Web UI: `http://<सर्वर IP>:8066`। यह सभी नेटवर्क इंटरफ़ेस पर सुनता है।
- साइन इन: यूज़र `admin`, पासवर्ड `traffic66`। पहली बार साइन इन पर नया पासवर्ड सेट करना होता है।
- डिवाइसों को सर्वर IP पर भेजने को कहें: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (सभी इंटरफ़ेस; हर पोर्ट हर प्रोटोकॉल लेता है)।
- शुरू करने के तुरंत बाद कुछ देर डेटा न दिखना सामान्य है: डिवाइस अपने पहले पैकेट के कुछ सेकंड बाद **सेटिंग्स** में दिखता है, और चार्ट लगभग एक मिनट पीछे चलते हैं। NetFlow v9/IPFIX को पहले डिवाइस का टेम्पलेट चाहिए; जिस डिवाइस ने अभी सैंपलिंग दर नहीं भेजी, उसके रिकॉर्ड अधिकतम 5 मिनट (`-sampling-wait`) रोके जाते हैं, फिर 1:1 गिने जाते हैं।
- आवश्यकताएँ: प्रोग्राम लगभग 73 MB, कुछ इंस्टॉल नहीं करना; मेमोरी खाली बैठे लगभग 0.1 GB और 5,000 फ़्लो/सेकंड पर 0.6–0.8 GB, इसलिए 1 GB या अधिक रखें। वेब UI को सीधे इंटरनेट पर न खोलें: इसे आंतरिक नेटवर्क में या HTTPS वाले रिवर्स प्रॉक्सी के पीछे रखें।

## आम कमांड

- `./traffic66`  
  ऊपर के डिफ़ॉल्ट के साथ शुरू करें।

- `./traffic66 -addr :9000`  
  Web UI 8066 की जगह पोर्ट 9000 पर।

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  collector पोर्ट बदलें (यहाँ NetFlow 9995 पर)।

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  सभी इंटरफ़ेस की जगह सिर्फ़ एक IP पते पर सुनें।

- `./traffic66 -data /var/lib/traffic66`  
  डेटा दूसरी डायरेक्टरी में रखें (डिफ़ॉल्ट: प्रोग्राम के पास `traffic66-data`)।

- `./traffic66 passwd`  
  admin पासवर्ड सेट या रीसेट करें (`-data` इस्तेमाल हो तो `-data DIR` जोड़ें)।

- `traffic66.exe interfaces`  
  Windows: नेटवर्क कार्ड की सूची (नाम और नंबर)।

- `traffic66.exe -capture Wi-Fi`  
  Windows: इस PC का अपना ट्रैफ़िक उस कार्ड पर देखें (नाम या नंबर)। Npcap (npcap.com) चाहिए।

- `./traffic66 interfaces`  
  Linux, macOS: नेटवर्क कार्ड की सूची।

- `sudo ./traffic66 -capture eth0`  
  Linux: इस मशीन का अपना ट्रैफ़िक eth0 पर देखें (macOS: `-capture en0`)।

- `./traffic66 tui`  
  traffic66 मशीन पर Terminal UI।

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  रिमोट traffic66 से जुड़ा Terminal UI।

- `./traffic66 demo`  
  सिम्युलेटेड डिवाइसों के साथ आज़माएँ। पहली बार चलाने पर यह पहले एक दिन का सिम्युलेटेड इतिहास बनाता है (कुछ सेकंड, धीमी डिस्क पर ज़्यादा), फिर वेब UI खोलता है।

## स्क्रीनशॉट

<img src="images/overview.png" alt="सारांश" width="49%"> <img src="images/topn.png" alt="शीर्ष 66" width="49%">

<img src="images/geo.png" alt="भूगोल और नेटवर्क" width="49%"> <img src="images/interfaces.png" alt="इंटरफ़ेस मिलान" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## पूरा मैनुअल

web UI के मेनू में **सहायता**। GitHub पर: [MANUAL.hi.md](MANUAL.hi.md)

15 भाषाएँ। evaluation के लिए और 100 से कम लोगों वाले संगठनों के लिए मुफ़्त ([लाइसेंस](../LICENSE.md))।

- **मुफ़्त**: किसी भी संगठन के लिए मूल्यांकन, परीक्षण और डेमो बिना समय सीमा; 100 से कम लोगों (कर्मचारी और ठेकेदार मिलाकर) वाले संगठनों और गैर-व्यावसायिक उपयोग के लिए प्रोडक्शन उपयोग भी। बड़े संगठन: 30 दिन प्रोडक्शन उपयोग, फिर पंजीकरण। विवरण: [LICENSE.md](../LICENSE.md), [Additional Use Grant](../ADDITIONAL-USE-GRANT.md)।
- **अपंजीकृत**: रिपोर्ट और चार्ट पर "मूल्यांकन प्रति" का निशान; अधिकतम 3 वेब खाते।
- **पंजीकृत**: निशान हट जाता है। संपर्क: route66traffic66@gmail.com।
