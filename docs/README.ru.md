[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | **Русский** | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — коллектор и анализатор трафика NetFlow, sFlow и IPFIX

Анализ потоков sFlow, NetFlow и IPFIX в одной программе: кто занимает
полосу, куда уходит трафик и сходятся ли цифры с собственными счётчиками
интерфейсов устройств — в веб-интерфейсе и в терминальном интерфейсе.

## Быстрый старт

1. Скачайте архив для своей системы: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Распакуйте и запустите программу:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: распакуйте `traffic66-windows-amd64.zip` и запустите `traffic66.exe` (можно двойным щелчком).

macOS: сначала выполните в папке `xattr -dr com.apple.quarantine .`.

Затем:

- Веб-интерфейс: `http://<IP сервера>:8066`. Слушает на всех сетевых интерфейсах.
- Вход: пользователь `admin`, пароль `traffic66`. При первом входе нужно задать новый пароль.
- Направьте устройства на IP сервера: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (все интерфейсы; каждый порт принимает любой протокол).

## Частые команды

- `./traffic66`  
  Запуск с настройками по умолчанию.

- `./traffic66 -addr :9000`  
  Веб-интерфейс на порту 9000 вместо 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Сменить порты коллекторов (здесь NetFlow на 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Слушать только на одном IP-адресе вместо всех интерфейсов.

- `./traffic66 -data /var/lib/traffic66`  
  Хранить данные в другом каталоге (по умолчанию `traffic66-data` рядом с программой).

- `./traffic66 passwd`  
  Задать или сбросить пароль admin (с `-data DIR`, если используется).

- `traffic66.exe interfaces`  
  Windows: список сетевых карт (имя и номер).

- `traffic66.exe -capture Wi-Fi`  
  Windows: анализ трафика самого ПК на этой карте (имя или номер). Нужен Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: список сетевых карт.

- `sudo ./traffic66 -capture eth0`  
  Linux: анализ трафика этой машины на eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Терминальный интерфейс на машине traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Терминальный интерфейс к удалённому traffic66.

- `./traffic66 demo`  
  Попробовать с имитацией устройств.

## Скриншоты

<img src="images/overview.png" alt="Обзор" width="49%"> <img src="images/topn.png" alt="Топ 66" width="49%">

<img src="images/geo.png" alt="География и сети" width="49%"> <img src="images/interfaces.png" alt="Сверка интерфейсов" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Полное руководство

В веб-интерфейсе: **Справка** в меню. На GitHub: [MANUAL.ru.md](MANUAL.ru.md)

15 языков. Бесплатно для оценки и для организаций численностью менее 100 человек ([лицензия](../LICENSE.md)).
