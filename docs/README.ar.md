[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | **العربية** | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66 — مُجمِّع ومحلّل حركة NetFlow وsFlow وIPFIX

تحليل تدفقات sFlow وNetFlow وIPFIX في برنامج واحد: من يستهلك عرض النطاق،
وإلى أين تذهب الحركة، وهل تتطابق الأرقام مع عدّادات الواجهات في الأجهزة
نفسها، في واجهة ويب وواجهة طرفية.

## بدء سريع

1. نزّل أرشيف نظامك من: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. فك الضغط وشغّل البرنامج:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

ويندوز: فك ضغط `traffic66-windows-amd64.zip` وشغّل `traffic66.exe` (يكفي النقر المزدوج).

macOS: شغّل أولًا `xattr -dr com.apple.quarantine .` داخل المجلد.

ثم:

- واجهة الويب: `http://<IP الخادم>:8066`، وتستمع على كل واجهات الشبكة.
- تسجيل الدخول: المستخدم `admin` وكلمة المرور `traffic66`. يُطلب تعيين كلمة مرور جديدة عند أول دخول.
- وجّه أجهزتك إلى IP الخادم: sFlow على UDP 6343 وNetFlow v5/v9 على UDP 2055 وIPFIX على UDP 4739 (كل الواجهات؛ كل منفذ يقبل كل البروتوكولات).

## أوامر شائعة

- `./traffic66`  
  التشغيل بالإعدادات الافتراضية أعلاه.

- `./traffic66 -addr :9000`  
  واجهة الويب على المنفذ 9000 بدل 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  تغيير منافذ المجمّعات (هنا NetFlow على 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  الاستماع على عنوان IP واحد بدل كل الواجهات.

- `./traffic66 -data /var/lib/traffic66`  
  حفظ البيانات في مجلد آخر (الافتراضي: `traffic66-data` بجانب البرنامج).

- `./traffic66 passwd`  
  تعيين كلمة مرور admin أو إعادة تعيينها (أضف `-data DIR` إن كنت تستخدمه).

- `traffic66.exe interfaces`  
  ويندوز: سرد بطاقات الشبكة (الاسم والرقم).

- `traffic66.exe -capture Wi-Fi`  
  ويندوز: تحليل حركة هذا الحاسوب نفسه على تلك البطاقة (الاسم أو الرقم). يتطلب Npcap (npcap.com).

- `./traffic66 interfaces`  
  لينكس وmacOS: سرد بطاقات الشبكة.

- `sudo ./traffic66 -capture eth0`  
  لينكس: تحليل حركة هذا الجهاز على eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  الواجهة الطرفية على جهاز traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  الواجهة الطرفية متصلة بـ traffic66 بعيد.

- `./traffic66 demo`  
  التجربة بأجهزة محاكاة.

## لقطات الشاشة

<img src="images/overview.png" alt="نظرة عامة" width="49%"> <img src="images/topn.png" alt="أعلى 66" width="49%">

<img src="images/geo.png" alt="الجغرافيا والشبكات" width="49%"> <img src="images/interfaces.png" alt="مطابقة الواجهات" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## الدليل الكامل

في واجهة الويب: **المساعدة** في القائمة. وعلى GitHub: [MANUAL.ar.md](MANUAL.ar.md)

15 لغة. مجاني للتقييم وللمؤسسات التي يقل عدد أفرادها عن 100 شخص ([الترخيص](../LICENSE.md)).

</div>
