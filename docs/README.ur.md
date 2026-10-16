[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | **اردو** | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66 — NetFlow، sFlow اور IPFIX کلیکٹر اور ٹریفک اینالائزر

sFlow، NetFlow اور IPFIX کے لیے flow analytics، ایک ہی پروگرام میں: bandwidth
کون استعمال کر رہا ہے، ٹریفک کہاں جا رہی ہے، اور کیا اعداد ڈیوائسز کے اپنے
interface counters سے میل کھاتے ہیں — web UI اور terminal UI میں۔

## فوری آغاز

```
# Linux، macOS: ./traffic66 میں ڈاؤن لوڈ کر کے جمع کرنا شروع کریں
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows، PowerShell: یہی
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

ٹرمینل ویب UI کا پتا اور آلات کے لیے IP اور UDP پورٹس دکھاتا ہے ([آلات کی ترتیب](MANUAL.ur.md#device-configuration))۔ `admin` / `traffic66` سے سائن اِن کر کے نیا پاس ورڈ چنیں۔ صرف دیکھنا ہے؟ فرضی آلات والا ڈیمو:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

پورٹس: TCP 8066 ویب UI، UDP 6343 sFlow، 2055 NetFlow، 4739 IPFIX (ہر UDP پورٹ ہر پروٹوکول لیتا ہے)۔ ٹرمینل وہ پتے دکھاتا ہے جن پر آلات بھیجیں؛ پہلی سائن اِن پر نیا پاس ورڈ مانگا جاتا ہے۔

macOS پر ہاتھ سے: پہلے `xattr -dr com.apple.quarantine traffic66-darwin-*`۔ Windows "protected your PC": **More info** ← **Run anyway**۔

## کمانڈز کا خلاصہ

| کام | کمانڈ |
|---|---|
| sFlow 6343، NetFlow v5/v9 2055، IPFIX 4739 جمع کریں (ڈیفالٹ) | `./traffic66` |
| صرف NetFlow v5/v9 (2055) اور IPFIX (4739) | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| صرف sFlow، UDP 6343 | `./traffic66 -listen sflow=:6343` |
| ویب UI مخصوص IP اور پورٹ پر | `./traffic66 -addr 10.0.0.5:8080` |
| اس مشین کا NIC کیپچر کریں (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin پاس ورڈ سیٹ/ری سیٹ کریں (‎-data DIR ہو تو شامل کریں) | `./traffic66 passwd` |
| ریموٹ سرور سے ٹرمینل UI | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![جائزہ: کھلی مشتبہ سرگرمیاں، کل اسی وقت کے مقابلے میں application کے حساب سے bandwidth، سرفہرست clients اور services](images/overview.png)

## مکمل دستی

ویب UI کے مینو میں **مدد**؛ GitHub پر: [MANUAL.ur.md](MANUAL.ur.md)

15 زبانیں۔ جانچ کے لیے اور 100 سے کم افراد والی تنظیموں کے لیے مفت ([لائسنس](../LICENSE.md))۔

</div>
