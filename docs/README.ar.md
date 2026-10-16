[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | **العربية** | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

<div dir="rtl">

# traffic66 — مُجمِّع ومحلّل حركة NetFlow وsFlow وIPFIX

تحليل تدفقات sFlow وNetFlow وIPFIX في برنامج واحد: من يستهلك عرض النطاق،
وإلى أين تذهب الحركة، وهل تتطابق الأرقام مع عدّادات الواجهات في الأجهزة
نفسها، في واجهة ويب وواجهة طرفية.

## بدء سريع

```
# لينكس وmacOS: نزّله في ./traffic66 وابدأ الجمع
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# ويندوز، PowerShell: الشيء نفسه
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

تطبع الطرفية عنوان واجهة الويب وعنوان IP ومنافذ UDP التي توجَّه إليها أجهزتك ([إعداد الأجهزة](MANUAL.ar.md#device-configuration)). سجّل الدخول باسم `admin` / `traffic66` واختر كلمة مرور جديدة. للاطلاع فقط؟ العرض التوضيحي بأجهزة محاكاة:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

المنافذ: TCP 8066 لواجهة الويب، UDP 6343 لـ sFlow و2055 لـ NetFlow و4739 لـ IPFIX (كل منفذ UDP يقبل كل البروتوكولات). تطبع الطرفية العناوين التي توجَّه إليها الأجهزة، ويطلب أول تسجيل دخول كلمة مرور جديدة.

على macOS يدويًا: أولًا `xattr -dr com.apple.quarantine traffic66-darwin-*`. رسالة ويندوز «حمى جهاز الكمبيوتر»: **مزيد من المعلومات** ← **التشغيل على أي حال**.

## ملخص الأوامر

| المهمة | الأمر |
|---|---|
| جمع sFlow على 6343 وNetFlow v5/v9 على 2055 وIPFIX على 4739 (افتراضي) | `./traffic66` |
| NetFlow v5/v9 على 2055 وIPFIX على 4739 فقط | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| sFlow فقط، UDP 6343 | `./traffic66 -listen sflow=:6343` |
| واجهة الويب على IP ومنفذ محددين | `./traffic66 -addr 10.0.0.5:8080` |
| التقاط بطاقة هذا الجهاز (لينكس) | `sudo ./traffic66 -capture eth0 -listen ""` |
| تعيين كلمة مرور admin أو إعادة تعيينها (أضف ‎-data DIR إن استُخدم) | `./traffic66 passwd` |
| الواجهة الطرفية إلى خادم بعيد | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

## لقطات الشاشة

| | |
|---|---|
| <a href="images/overview.png"><img src="images/overview.png" alt="نظرة عامة"></a><br>نظرة عامة | <a href="images/topn.png"><img src="images/topn.png" alt="أعلى 66"></a><br>أعلى 66 |
| <a href="images/geo.png"><img src="images/geo.png" alt="الجغرافيا والشبكات"></a><br>الجغرافيا والشبكات | <a href="images/interfaces.png"><img src="images/interfaces.png" alt="مطابقة الواجهات"></a><br>مطابقة الواجهات |
| <a href="images/tui-top.png"><img src="images/tui-top.png" alt="TUI"></a><br>TUI | |

## الدليل الكامل

في واجهة الويب: **المساعدة** في القائمة؛ وعلى GitHub: [MANUAL.ar.md](MANUAL.ar.md)

15 لغة. مجاني للتقييم وللمؤسسات التي يقل عدد أفرادها عن 100 شخص ([الترخيص](../LICENSE.md)).

</div>
