[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | **বাংলা** | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow, sFlow ও IPFIX কালেক্টর এবং ট্রাফিক অ্যানালাইজার

sFlow, NetFlow ও IPFIX-এর জন্য flow analytics, একটিমাত্র প্রোগ্রামে: কে
bandwidth ব্যবহার করছে, ট্রাফিক কোথায় যাচ্ছে, আর সংখ্যাগুলো ডিভাইসের নিজের
interface counter-এর সাথে মেলে কি না — web UI ও terminal UI-তে।

## দ্রুত শুরু

1. আপনার সিস্টেমের আর্কাইভ ডাউনলোড করুন: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. খুলে প্রোগ্রামটি চালান:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: `traffic66-windows-amd64.zip` খুলে `traffic66.exe` চালান (ডাবল-ক্লিকেও হয়)।

macOS: আগে ফোল্ডারে `xattr -dr com.apple.quarantine .` চালান।

তারপর:

- Web UI: `http://<সার্ভার IP>:8066`। এটি সব নেটওয়ার্ক ইন্টারফেসে শোনে।
- সাইন ইন: ইউজার `admin`, পাসওয়ার্ড `traffic66`। প্রথম সাইন-ইনে নতুন পাসওয়ার্ড দিতে হয়।
- ডিভাইসগুলোকে সার্ভার IP-তে পাঠাতে দিন: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (সব ইন্টারফেস; প্রতিটি পোর্ট সব প্রোটোকল নেয়)।

## সাধারণ কমান্ড

- `./traffic66`  
  উপরের ডিফল্টে চালু করুন।

- `./traffic66 -addr :9000`  
  Web UI 8066-এর বদলে পোর্ট 9000-এ।

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  collector পোর্ট বদলান (এখানে NetFlow 9995-এ)।

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  সব ইন্টারফেসের বদলে শুধু একটি IP-তে শুনুন।

- `./traffic66 -data /var/lib/traffic66`  
  ডেটা অন্য ডিরেক্টরিতে রাখুন (ডিফল্ট: প্রোগ্রামের পাশে `traffic66-data`)।

- `./traffic66 passwd`  
  admin পাসওয়ার্ড সেট বা রিসেট (`-data` থাকলে `-data DIR` যোগ করুন)।

- `traffic66.exe interfaces`  
  Windows: নেটওয়ার্ক কার্ডের তালিকা (নাম ও নম্বর)।

- `traffic66.exe -capture Wi-Fi`  
  Windows: ওই কার্ডে এই PC-র নিজের ট্রাফিক বিশ্লেষণ (নাম বা নম্বর)। Npcap (npcap.com) লাগে।

- `./traffic66 interfaces`  
  Linux, macOS: নেটওয়ার্ক কার্ডের তালিকা।

- `sudo ./traffic66 -capture eth0`  
  Linux: eth0-তে এই মেশিনের নিজের ট্রাফিক বিশ্লেষণ (macOS: `-capture en0`)।

- `./traffic66 tui`  
  traffic66 মেশিনে Terminal UI।

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  দূরের traffic66-এর সাথে যুক্ত Terminal UI।

- `./traffic66 demo`  
  সিমুলেটেড ডিভাইস দিয়ে চেষ্টা করুন।

## স্ক্রিনশট

<img src="images/overview.png" alt="সারসংক্ষেপ" width="49%"> <img src="images/topn.png" alt="শীর্ষ 66" width="49%">

<img src="images/geo.png" alt="ভূগোল ও নেটওয়ার্ক" width="49%"> <img src="images/interfaces.png" alt="ইন্টারফেস মিলানো" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## সম্পূর্ণ ম্যানুয়াল

web UI-এর মেনুতে **সহায়তা**। GitHub-এ: [MANUAL.bn.md](MANUAL.bn.md)

15টি ভাষা। মূল্যায়নের জন্য এবং 100 জনের কম মানুষের প্রতিষ্ঠানের জন্য বিনামূল্যে ([লাইসেন্স](../LICENSE.md))।
