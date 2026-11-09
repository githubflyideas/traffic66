[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | **اردو** | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66 — NetFlow، sFlow اور IPFIX کلیکٹر اور ٹریفک اینالائزر

sFlow، NetFlow اور IPFIX کے لیے flow analytics، ایک ہی پروگرام میں: bandwidth
کون استعمال کر رہا ہے، ٹریفک کہاں جا رہی ہے، اور کیا اعداد ڈیوائسز کے اپنے
interface counters سے میل کھاتے ہیں — web UI اور terminal UI میں۔

## فوری آغاز

1. اپنے سسٹم کا آرکائیو ڈاؤن لوڈ کریں: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. کھول کر پروگرام چلائیں:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: `traffic66-windows-amd64.zip` کھولیں اور `traffic66.exe` چلائیں (ڈبل کلک بھی کافی ہے)۔

macOS: پہلے اس فولڈر میں `xattr -dr com.apple.quarantine .` چلائیں۔

پھر:

- ویب UI: `http://<سرور IP>:8066`۔ یہ تمام نیٹ ورک انٹرفیسز پر سنتا ہے۔
- سائن اِن: صارف `admin`، پاس ورڈ `traffic66`۔ پہلی بار سائن اِن پر نیا پاس ورڈ رکھنا ہوتا ہے۔
- آلات کو سرور IP پر بھیجنے دیں: sFlow UDP 6343، NetFlow v5/v9 UDP 2055، IPFIX UDP 4739 (تمام انٹرفیسز؛ ہر پورٹ ہر پروٹوکول لیتا ہے)۔
- شروع کرنے کے فوراً بعد کچھ دیر ڈیٹا نہ ہونا معمول ہے: ڈیوائس اپنے پہلے پیکٹ کے چند سیکنڈ بعد **ترتیبات** میں نظر آتی ہے، اور چارٹ تقریباً ایک منٹ پیچھے رہتے ہیں۔ NetFlow v9/IPFIX کو پہلے ڈیوائس کا ٹیمپلیٹ چاہیے؛ جس ڈیوائس نے ابھی سیمپلنگ ریٹ نہیں بھیجا اس کے ریکارڈ زیادہ سے زیادہ 5 منٹ (`-sampling-wait`) روکے جاتے ہیں، پھر 1:1 شمار ہوتے ہیں۔
- ضروریات: پروگرام تقریباً 73 MB، کچھ انسٹال نہیں کرنا؛ میموری خالی حالت میں تقریباً 0.1 GB اور 5,000 فلو فی سیکنڈ پر 0.6–0.8 GB، اس لیے 1 GB یا زیادہ رکھیں۔ ویب UI کو براہِ راست انٹرنیٹ پر نہ کھولیں: اسے اندرونی نیٹ ورک میں یا HTTPS والے ریورس پراکسی کے پیچھے رکھیں۔

## عام کمانڈز

- `./traffic66`  
  اوپر کی ڈیفالٹ ترتیب سے شروع کریں۔

- `./traffic66 -addr :9000`  
  ویب UI پورٹ 8066 کی جگہ 9000 پر۔

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  کلیکٹر پورٹس بدلیں (یہاں NetFlow کو 9995 پر)۔

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  تمام انٹرفیسز کی جگہ صرف ایک IP پتے پر سنیں۔

- `./traffic66 -data /var/lib/traffic66`  
  ڈیٹا دوسری ڈائریکٹری میں رکھیں (ڈیفالٹ: پروگرام کے ساتھ `traffic66-data`)۔

- `./traffic66 passwd`  
  admin پاس ورڈ سیٹ یا ری سیٹ کریں (`-data` استعمال ہو تو `-data DIR` لگائیں)۔

- `traffic66.exe interfaces`  
  Windows: نیٹ ورک کارڈز کی فہرست (نام اور نمبر)۔

- `traffic66.exe -capture Wi-Fi`  
  Windows: اس PC کی اپنی ٹریفک اس کارڈ پر دیکھیں (نام یا نمبر)۔ Npcap (npcap.com) درکار۔

- `./traffic66 interfaces`  
  Linux، macOS: نیٹ ورک کارڈز کی فہرست۔

- `sudo ./traffic66 -capture eth0`  
  Linux: اس مشین کی اپنی ٹریفک eth0 پر دیکھیں (macOS: `-capture en0`)۔

- `./traffic66 tui`  
  traffic66 مشین پر ٹرمینل UI۔

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  دور کے traffic66 سے جڑا ٹرمینل UI۔

- `./traffic66 demo`  
  فرضی آلات کے ساتھ آزمائیں۔ پہلی بار چلانے پر پہلے ایک دن کی فرضی تاریخ بنتی ہے (چند سیکنڈ، سست ڈسک پر زیادہ)، پھر ویب UI کھلتا ہے۔

## اسکرین شاٹس

<img src="images/overview.png" alt="جائزہ" width="49%"> <img src="images/topn.png" alt="ٹاپ 66" width="49%">

<img src="images/geo.png" alt="مقامات اور نیٹ ورک" width="49%"> <img src="images/interfaces.png" alt="انٹرفیس جانچ" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## مکمل دستی

ویب UI کے مینو میں **مدد**۔ GitHub پر: [MANUAL.ur.md](MANUAL.ur.md)

15 زبانیں۔ جانچ کے لیے اور 100 سے کم افراد والی تنظیموں کے لیے مفت ([لائسنس](../LICENSE.md))۔

- **مفت**: ہر تنظیم کے لیے جانچ، ٹیسٹ اور ڈیمو بغیر وقت کی حد کے؛ 100 سے کم افراد (ملازمین اور ٹھیکیدار ملا کر) والی تنظیموں اور غیر تجارتی استعمال کے لیے پروڈکشن استعمال بھی۔ بڑی تنظیمیں: 30 دن پروڈکشن استعمال، پھر رجسٹریشن۔ تفصیل: [LICENSE.md](../LICENSE.md)، [Additional Use Grant](../ADDITIONAL-USE-GRANT.md)۔
- **غیر رجسٹرڈ**: رپورٹوں اور چارٹس پر "تشخیصی نسخہ" کا نشان؛ زیادہ سے زیادہ 3 ویب اکاؤنٹس۔
- **رجسٹرڈ**: نشان ہٹ جاتا ہے۔ رابطہ: route66traffic66@gmail.com۔

</div>
