[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | **Русский** | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — коллектор и анализатор трафика NetFlow, sFlow и IPFIX

Анализ потоков sFlow, NetFlow и IPFIX в одной программе: кто занимает
полосу, куда уходит трафик и сходятся ли цифры с собственными счётчиками
интерфейсов устройств — в веб-интерфейсе и в терминальном интерфейсе.

## Быстрый старт

```
# Linux, macOS: скачать в ./traffic66 и начать сбор
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: то же
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

Терминал показывает адрес веб-интерфейса, а также IP и UDP-порты для ваших устройств ([настройка устройств](MANUAL.ru.md#device-configuration)). Войдите как `admin` / `traffic66` и задайте новый пароль. Только посмотреть? Демо с имитацией устройств:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Порты: TCP 8066 веб-интерфейс, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (каждый UDP-порт принимает любой протокол). Терминал показывает адреса для устройств; при первом входе нужно задать новый пароль.

macOS вручную: сначала `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows «защитила ваш компьютер»: **Подробнее** → **Выполнить в любом случае**.

## Шпаргалка по командам

| Задача | Команда |
|---|---|
| Сбор sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (по умолчанию) | `./traffic66` |
| Только NetFlow v5/v9 на 2055 и IPFIX на 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Только sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Веб-интерфейс на заданных IP и порту | `./traffic66 -addr 10.0.0.5:8080` |
| Захват своей сетевой карты (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Задать или сбросить пароль admin (с -data DIR, если используется) | `./traffic66 passwd` |
| Терминальный интерфейс к удалённому серверу | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Обзор: открытые обнаружения, полоса по приложениям в сравнении с тем же временем вчера, основные клиенты и сервисы](images/overview.png)

## Полное руководство

в веб-интерфейсе: **Справка** в меню; на GitHub: [MANUAL.ru.md](MANUAL.ru.md)

15 языков. Бесплатно для оценки и для организаций численностью менее 100 человек ([лицензия](../LICENSE.md)).
