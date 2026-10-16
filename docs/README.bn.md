[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | **বাংলা** | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow, sFlow ও IPFIX কালেক্টর এবং ট্রাফিক অ্যানালাইজার

sFlow, NetFlow ও IPFIX-এর জন্য flow analytics, একটিমাত্র প্রোগ্রামে: কে
bandwidth ব্যবহার করছে, ট্রাফিক কোথায় যাচ্ছে, আর সংখ্যাগুলো ডিভাইসের নিজের
interface counter-এর সাথে মেলে কি না — web UI ও terminal UI-তে।

## দ্রুত শুরু

```
# Linux, macOS: ./traffic66-এ ডাউনলোড করে সংগ্রহ শুরু করুন
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: একই
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

টার্মিনালে web UI-এর ঠিকানা আর ডিভাইসের জন্য IP ও UDP পোর্ট দেখা যায় ([ডিভাইস কনফিগারেশন](MANUAL.bn.md#device-configuration))। `admin` / `traffic66` দিয়ে সাইন ইন করে নতুন পাসওয়ার্ড দিন। শুধু দেখতে চান? সিমুলেটেড ডিভাইসের ডেমো:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

পোর্ট: TCP 8066 web UI, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (প্রতিটি UDP পোর্ট সব প্রোটোকল নেয়)। ডিভাইস কোন ঠিকানায় পাঠাবে তা টার্মিনালে দেখা যায়; প্রথম সাইন-ইনে নতুন পাসওয়ার্ড চাওয়া হয়।

macOS-এ হাতে: আগে `xattr -dr com.apple.quarantine traffic66-darwin-*`। Windows "protected your PC": **More info** → **Run anyway**।

## কমান্ড চিট শিট

| কাজ | কমান্ড |
|---|---|
| sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 সংগ্রহ (ডিফল্ট) | `./traffic66` |
| শুধু NetFlow v5/v9 (2055) ও IPFIX (4739) | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| শুধু sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Web UI নির্দিষ্ট IP ও পোর্টে | `./traffic66 -addr 10.0.0.5:8080` |
| এই মেশিনের NIC ক্যাপচার (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin পাসওয়ার্ড সেট/রিসেট (-data DIR থাকলে যোগ করুন) | `./traffic66 passwd` |
| রিমোট সার্ভারে Terminal UI | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![সারসংক্ষেপ: খোলা সন্দেহজনক কার্যকলাপ, গতকাল একই সময়ের তুলনায় application অনুযায়ী bandwidth, শীর্ষ client ও service](images/overview.png)

## সম্পূর্ণ ম্যানুয়াল

web UI-এর মেনুতে **সহায়তা**; GitHub-এ: [MANUAL.bn.md](MANUAL.bn.md)

15টি ভাষা। মূল্যায়নের জন্য এবং 100 জনের কম মানুষের প্রতিষ্ঠানের জন্য বিনামূল্যে ([লাইসেন্স](../LICENSE.md))।
