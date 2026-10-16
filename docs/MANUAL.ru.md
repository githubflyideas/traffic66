[English](MANUAL.md) | [中文](MANUAL.zh.md) | [हिन्दी](MANUAL.hi.md) | [Español](MANUAL.es.md) | [العربية](MANUAL.ar.md) | [Français](MANUAL.fr.md) | [বাংলা](MANUAL.bn.md) | [Português](MANUAL.pt.md) | **Русский** | [Bahasa Indonesia](MANUAL.id.md) | [اردو](MANUAL.ur.md) | [Deutsch](MANUAL.de.md) | [日本語](MANUAL.ja.md) | [Tiếng Việt](MANUAL.vi.md) | [한국어](MANUAL.ko.md)

# traffic66 — коллектор и анализатор трафика NetFlow, sFlow и IPFIX

Анализ потоков sFlow, NetFlow и IPFIX в одной программе: кто занимает
полосу, куда уходит трафик и сходятся ли цифры с собственными счётчиками
интерфейсов устройств — в веб-интерфейсе и в терминальном интерфейсе.

Самостоятельно размещаемая альтернатива ntopng, ElastiFlow, связке pmacct с
Grafana или модулям анализа потоков PRTG и SolarWinds NTA для мониторинга
сети (network monitoring), мониторинга полосы пропускания
(bandwidth monitoring), поиска самых активных узлов (top talkers),
обнаружения DDoS и сканирований и анализа pcap — без Elasticsearch, Kafka и
отдельной базы данных.

- Один исполняемый файл для Windows, Linux и macOS; не нужно ставить базу данных, работает без доступа в интернет.
- sFlow v5, NetFlow v5/v9 и IPFIX на любом UDP-порту или локальный захват с интерфейса.
- Сверяет свои цифры со счётчиками интерфейсов (sFlow или SNMP) и объясняет, почему они расходятся.
- Находит сканирования, подбор паролей, боковое перемещение, необычные выгрузки, флуд и трафик со списков угроз, в том числе при сэмплировании.
- `traffic66 capture.pcap` анализирует захваты пакетов без какой-либо настройки.
- 15 языков. Бесплатно для оценки и для организаций численностью менее 100 человек ([лицензия](#licence)).

![Обзор: открытые обнаружения, полоса по приложениям в сравнении с тем же временем вчера, основные клиенты и сервисы](images/overview.png)

<sub>Все скриншоты сделаны в `traffic66 demo` — смоделированной сети компании.</sub>

<a id="contents"></a>

## Содержание

1. [Быстрый старт](#1-quick-start)
2. [Примеры](#2-examples)
3. [Пользователи и пароли](#3-users-and-passwords)
4. [Отправка потоков с устройств](#4-send-flows-from-your-devices)
5. [Проверка поступления потоков](#5-check-that-flows-arrive)
6. [Интерфейсы и счётчики](#6-interfaces-and-counters)
7. [Названия, страны и списки угроз](#7-names-countries-and-threat-lists)
8. [Работа с веб-интерфейсом](#8-using-the-web-ui)
9. [Офлайн-анализ pcap, терминальный интерфейс, локальный захват](#9-offline-pcap-terminal-ui-local-capture)
10. [Параметры и данные](#10-options-and-data)
11. [Безопасность, оценка ресурсов, устранение неполадок](#11-security-sizing-troubleshooting)

<a id="1-quick-start"></a>

<a id="1-try-the-demo"></a>

## 1. Быстрый старт

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

<a id="2-examples"></a>

<a id="2-install"></a>

## 2. Примеры

<a id="command-cheat-sheet"></a>

### Шпаргалка по командам

| Задача | Команда |
|---|---|
| Сбор sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (по умолчанию) | `./traffic66` |
| Только sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Только NetFlow v5/v9 на 2055 и IPFIX на 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Другие UDP-порты | `./traffic66 -listen netflow=:9995,sflow=:6344` |
| Коллекторы на одном IP-адресе | `./traffic66 -listen sflow=10.0.0.5:6343,netflow=10.0.0.5:2055,ipfix=10.0.0.5:4739` |
| Веб-интерфейс на заданных IP и порту | `./traffic66 -addr 10.0.0.5:8080` |
| Веб-интерфейс только на этой машине | `./traffic66 -addr 127.0.0.1:8066` |
| Веб-интерфейс по HTTPS | `./traffic66 -tls` |
| Каталог данных | `./traffic66 -data /var/lib/traffic66` |
| Хранить детализацию 7 дней | `./traffic66 -retention-days 7` |
| Память: 20 % ОЗУ под базу | `./traffic66 -memory 0.20` |
| Список сетевых карт для локального захвата | `./traffic66 interfaces` |
| Захват своей сетевой карты (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Захват своей сетевой карты (macOS) | `sudo ./traffic66 -capture en0 -listen ""` |
| Захват своей сетевой карты (Windows, нужен Npcap) | `.\traffic66.exe -capture Wi-Fi -listen ""` |
| Задать или сбросить пароль admin (с -data DIR, если используется) | `./traffic66 passwd` |
| Случайный пароль admin с выводом | `./traffic66 passwd -generate` |
| Добавить пользователя или сменить его пароль | `./traffic66 passwd -user alice` |
| Список пользователей | `./traffic66 passwd -list` |
| Пароль только на этот запуск (скрипты, контейнеры) | `TRAFFIC66_PASSWORD='S3cret-pass' ./traffic66` |
| Терминальный интерфейс на машине traffic66 | `./traffic66 tui` |
| Терминальный интерфейс к удалённому серверу | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |
| Язык терминального интерфейса | `./traffic66 tui -lang ja` |
| Анализ файла захвата | `./traffic66 capture.pcap` |
| Отправить тестовые потоки на коллектор | `./traffic66 simulate -to 192.0.2.50` |
| Демо с имитацией устройств | `./traffic66 demo` |
| Все параметры | `./traffic66 serve -h` |

Язык веб-интерфейса — внизу меню (флага нет). Остальное: `traffic66 serve -h`.

<a id="device-configuration"></a>

### Настройка устройств

`192.0.2.50` — traffic66, `192.0.2.1` — устройство; таймауты и частота выборки: [раздел 4](#4-send-flows-from-your-devices).

```
! Cisco IOS-XE, NetFlow v9 (IPFIX: add "export-protocol ipfix" and use "transport udp 4739")
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

```
# Huawei CloudEngine, sFlow
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

```
# Huawei AR/NE, NetStream (NetFlow v9)
ip netstream export version 9
ip netstream export host 192.0.2.50 2055
ip netstream timeout active 1
interface GigabitEthernet0/0/1
 ip netstream inbound
 ip netstream sampler fix-packets 1000 inbound
```

```
# H3C Comware 7, sFlow
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

```
# Juniper EX/QFX, sFlow
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow interfaces ge-0/0/0

# Arista EOS, sFlow
sflow sample 4096
sflow destination 192.0.2.50
sflow run
```

```
# MikroTik RouterOS 7, NetFlow v9 (IPFIX: version=ipfix port=4739)
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

### Открыть брандмауэр

```
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload   # RHEL, Fedora
sudo ufw allow proto udp to any port 6343,2055,4739 && sudo ufw allow 8066/tcp                          # Ubuntu, Debian
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow   # Windows
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

### Веб-интерфейс только на этой машине, доступ через SSH

```
./traffic66 -addr 127.0.0.1:8066                 # on the server
ssh -L 8066:127.0.0.1:8066 user@192.0.2.50        # on your PC, then open http://127.0.0.1:8066
```

### Как служба: Linux (systemd)

```
sudo mkdir -p /opt/traffic66 && curl -fsSL https://github.com/githubflyideas/traffic66/releases/latest/download/traffic66-linux-amd64.tar.gz | sudo tar xz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

`/etc/systemd/system/traffic66.service`:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
MemoryMax=2G
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN   # only for local capture

[Install]
WantedBy=multi-user.target
```

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf && sudo sysctl --system
sudo systemctl daemon-reload && sudo systemctl enable --now traffic66
journalctl -u traffic66 -f
```

### Как служба: Windows

```
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

### Как служба: macOS

Распакуйте в `/usr/local/traffic66`, снимите атрибут карантина, выполните `traffic66 passwd -data "/Library/Application Support/traffic66"` и запускайте через LaunchDaemon, где `ProgramArguments` — программа, `-data` и этот каталог, с `RunAtLoad` и `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. Пользователи и пароли

В новой установке вход выполняется как `admin` / `traffic66`, и при первом
входе, прежде чем показать что-либо ещё, traffic66 просит задать новый
пароль (в демо остаётся `traffic66`). Затем пароль меняется в разделе
**Учётная запись** внизу меню; администратор (`admin`) там же добавляет и
удаляет пользователей и сбрасывает их пароли. Вход через LDAP / Active
Directory в разработке.

Пользователи хранятся в виде хешей с солью в файле `password` в каталоге
данных. То же самое можно сделать одной командой на машине с traffic66
(добавьте `-data …`, если traffic66 запущен с этим параметром):

| Задача | Команда |
|---|---|
| Сменить пароль `admin` | `traffic66 passwd` |
| Добавить пользователя `alice` или сменить её пароль | `traffic66 passwd -user alice` |
| Удалить пользователя `alice` | `traffic66 passwd -user alice -delete` |
| Вывести список пользователей | `traffic66 passwd -list` |

Изменения применяются сразу. Кроме управления пользователями, у всех
пользователей одинаковые права. Для скриптов и контейнеров `TRAFFIC66_PASSWORD=…` (или `-password`) на этот
запуск принимает только `-user` с этим паролем. Пять неверных паролей за
минуту блокируют адрес на минуту.

<a id="4-send-flows-from-your-devices"></a>

## 4. Отправка потоков с устройств

`192.0.2.50` — это traffic66, `192.0.2.1` — устройство. Установите активный
тайм-аут 60 секунд, разрешите устройствам NetFlow/IPFIX экспортировать
параметры сэмплера и сэмплируйте **каждый интерфейс на входе** (или только
граничные интерфейсы): тогда каждый пакет учитывается один раз. Частоты
sFlow: примерно 1:1000 для 1 Гбит/с, 1:4096 для 10 Гбит/с, 1:8192 для
40/100 Гбит/с.

Настройка Cisco, Huawei, H3C, Juniper, Arista, MikroTik и softflowd: [Примеры → Настройка устройств](#device-configuration).

<a id="5-check-that-flows-arrive"></a>

## 5. Проверка поступления потоков

**Настройки** в течение нескольких секунд показывают каждое устройство,
которое хоть что-то отправляет: протокол, частоту сэмплирования, потери,
интерфейсы с выборкой и что исправить, если статус не зелёный. Потери sFlow
разделяются на потери в пути (увеличьте `net.core.rmem_max`, если
`netstat -su` показывает ошибки буфера) и выборки, отброшенные самим
устройством.

![Настройки: каждое устройство с протоколом, сэмплированием, потерями и тем, что нужно исправить](images/sources.png)

Устройства нет? Выполните `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739`: если там пусто — дело в маршрутизации, межсетевом экране
или конфигурации устройства; если пакеты есть, а в **Настройках** ничего
нет — в локальном межсетевом экране или `-listen`. `traffic66 simulate -to 192.0.2.50`
с другой машины проверяет путь с имитируемыми устройствами.

<a id="6-interfaces-and-counters"></a>

## 6. Интерфейсы и счётчики

Цифры по потокам — это оценки (выборки × частота сэмплирования).
**Сверка интерфейсов** сравнивает их со счётчиками интерфейсов устройства
(счётчики sFlow или SNMP через строку `snmp` в Названиях) и объясняет,
почему они расходятся: интерфейсы без выборки, один и тот же трафик
сэмплирован дважды, потери в пути или неизвестная частота сэмплирования. У
каждого интерфейса есть график в бит/с и в пакетах/с: вход зелёный, выход
синий, счётчики пунктиром.

В каждой строке **✎** задаёт название и короткую метку (например, *uplink*),
а **☆** делает интерфейс основным (★) — на нём открываются страницы.

Устройство, сэмплирующее только часть интерфейсов, показывает и другие
концы этих потоков. Эти **встречные интерфейсы** перечислены последними
мелким серым шрифтом: в них только трафик через сэмплируемый интерфейс.
Сэмплируемый интерфейс известен из источника данных sFlow или поля
flowDirection (IPFIX 61); без него — это интерфейс, через который проходит
90% трафика устройства.

![Сверка интерфейсов: трафик каждого интерфейса и оценка по потокам рядом со счётчиком устройства](images/interfaces.png)

traffic66 уже использует частоту, применённую устройством, ждёт неизвестные
частоты, компенсирует потери экспорта, распределяет длинные потоки по их
минутам и добавляет к NetFlow/IPFIX 18 байт накладных расходов Ethernet на
пакет (`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Названия, страны и списки угроз

Щёлкните любой адрес и выберите **Дать название…** или используйте
**Настройки → Названия**. Названия хранятся в `inventory.txt` в каталоге
данных:

```
net    10.10.0.0/16    Office LAN                  # your networks
net    203.0.113.0/24  Public servers country=JP    # country: lines on the world map
device 192.0.2.1       Core router
device 192.0.2.9       Branch firewall unsampled    # exports every packet
device 192.0.2.20      Edge router sampling=1000    # rate it does not declare
iface  192.0.2.1 3     ISP uplink speed=1000000000 tag=uplink default
host   10.10.3.27      Finance PC
snmp   192.0.2.1       public                       # read counters over SNMPv2c
snmp   192.0.2.9       s3cret  10.99.0.9:161        # other management address
```

Частные диапазоны всегда считаются вашими. Изменения применяются по
**Сохранить**, без перезапуска.

Страны и сети (AS) работают сразу со свободными базами DB-IP Lite
([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **Настройки**
обновляют их или принимают вместо них файлы MaxMind GeoLite2, IPinfo Lite
или IPtoASN. Контуры карты: [Natural Earth](https://www.naturalearthdata.com).

![География и сети: внешний трафик по странам на карте мира](images/geo.png)

Списки угроз — это текстовые файлы с одним адресом или сетью на строку в
`<data>/threats/<name>.txt` (например, Spamhaus DROP); после изменения
перезапустите traffic66. Совпадения появляются на странице **Угрозы**.

![Угрозы: внутренний хост отправляет данные на адрес из списка угроз](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Работа с веб-интерфейсом

Любое значение на любой странице можно щёлкнуть: **Показать только это** /
**Исключить это** (фильтры действуют на всех страницах), **Показать записи
потоков**, **Подробнее** (страница об одном хосте или сервисе), **Дать
название…**, **Найти в интернете**, **Копировать**.

| Страница | Что показывает |
|---|---|
| Обзор | Полоса выбранного интерфейса; трафик по приложениям (всего, входящий или исходящий) в сравнении со вчера или прошлой неделей; открытые обнаружения; основные клиенты и сервисы |
| Топ 66 | Топ 66 сеансов с сортировкой по любому столбцу или с группировкой по приложению, сети, сегменту, устройству, инкапсуляции, VLAN; топ 30 собеседников |
| Детали трафика | Кольцевые диаграммы: серверы и их клиенты (или наоборот) и сервисы |
| Пути трафика | Узел → приложение → страна, или клиент → сервис → сервер, или по сетям |
| Сверка интерфейсов | Каждый интерфейс во времени в сравнении с его счётчиками; названия, метки, основной интерфейс |
| Записи потоков | Отдельные потоки, вживую каждые 5 секунд или за любой период |
| Обнаружения, Угрозы | Что требует внимания ([ниже](#findings)); трафик с адресами из списков |
| География и сети | Карта мира по странам, сети (AS) во времени |
| Настройки | Устройства, сэмплирование, потери, SNMP, базы, логотип, названия |
| Офлайн-анализ pcap, Очистка данных | Файлы захвата ([ниже](#9-offline-pcap-terminal-ui-local-capture)); удаление старых данных |

Над страницами: **Интерфейс** (все или один сэмплируемый интерфейс; тогда
страницы трафика показывают только трафик через него), период (от 15 минут
до 30 дней или произвольный), обновление каждые 30 с и **Скопировать
ссылку** на точный вид. Язык и пять цветовых тем — внизу меню. Графики
заканчиваются там, где данные полны: с NetFlow/IPFIX — с задержкой, с
которой экспортируют устройства (не более 2 минут). Периоды длиннее 6 часов
начинаются с целого часа; один интерфейс за 7 или 30 дней читается из
детальных данных, поэтому это медленнее и доступно лишь на глубину хранения
детальных данных.

![Топ 66: топ 66 сеансов, сортировка по любому столбцу](images/topn.png)

![Детали трафика: серверы с их клиентами и сервисы с их серверами в виде кольцевых диаграмм](images/traffic.png)

![Подробно об одном хосте: обнаружения по нему, его трафик, с кем он общается, сервисы, страны и последние потоки](images/detail.png)

![Пути трафика: какой узел к какому приложению в какой стране обращается](images/paths.png)

![Обзор на китайском](images/overview-zh.png)

<a id="findings"></a>

### Обнаружения

Проверка выполняется каждые 5 минут за последние 10; то, что длится час, —
это одно растущее обнаружение.

| Обнаружение | Что это значит |
|---|---|
| Сканирование, сканирование портов | Мелкие пробы на множество хостов на один порт или на множество портов одного хоста |
| Подбор паролей | Множество коротких подключений к службе входа |
| Боковое перемещение | Общий доступ к файлам или удалённое администрирование к внутренним хостам, которые раньше никогда этого не предоставляли |
| Необычная выгрузка | 100 MB за 10 минут на новый адрес, втрое больше полученного обратно |
| Флуд | 20,000+ мелких пакетов/с на один адрес, в десять раз больше обычного |
| Список угроз | Трафик с адресом из списка |

Изнутри вашей сети они высокой важности, из интернета — низкой.
**Обработано** закрывает обнаружение, **Ложная тревога** заглушает его
навсегда. Для бокового перемещения и выгрузок нужны сутки истории. При
сэмплировании 1:4096 атака в демо обнаруживается полностью; очень мелкие
сканирования могут скрыться за сэмплированием.

![Обнаружения: все шаги атаки, найденные через сэмплирование sFlow 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Офлайн-анализ pcap, терминальный интерфейс, локальный захват

**Офлайн-анализ pcap** показывает захваты пакетов (pcap, pcapng) на тех же
страницах, отдельно от живых данных: `traffic66 a.pcap b.pcapng` запускается
на 127.0.0.1 и открывает браузер (до 3 файлов, 3 ГБ; Ctrl+C удаляет
импортированные данные), или загрузите на этой странице до 3 файлов по
50 МБ. Файлы анализируются по одному, у каждого своя база данных: кнопка
**Анализ** в строке файла показывает его на всех страницах, а панель вверху
переключает на другой. Анализируются потоки, а не содержимое пакетов.

![Офлайн-анализ: файлы захвата с пакетами, потоками и временем](images/sandbox.png)

**Терминальный интерфейс**: `traffic66 tui` на машине с traffic66 или
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Три страницы: **1 Топ** (клиенты, серверы, приложения, порты, сети (AS) и страны/регионы по скорости; ← → переключают, Enter — действия), **2 Интерфейсы** (скорость, загрузка и сверка по устройствам и интерфейсам, а также выборка и потери каждого устройства; Enter раскрывает график и топ узлов/приложений) и **3 Поиск** (записи потоков по фильтрам вида `ip:10.0.0.5 net:10.0.0.0/8 asn:13335 port:443 cc:US dev:192.0.2.1`). Клавиши: `/` фильтр, Backspace/`c` убрать/очистить, `t` период (от 5 мин до 24 часов), `u` бит/с, байты или пакеты/с, `p` пауза, `+`/`-` интервал обновления, `w` открыть то же в браузере, `?` справка, `q` выход; `-lang` выбирает язык, `NO_COLOR` отключает цвета.

![Терминальный интерфейс: Топ](images/tui-top.png)

![Терминальный интерфейс: интерфейсы](images/tui-interfaces.png)

**Локальный захват** строит потоки с локального интерфейса, лучше всего с
порта, подключённого к зеркальному порту коммутатора: `traffic66 interfaces`
выводит их список, `-capture eth1` захватывает. В Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

В Linux нужны root или `setcap cap_net_raw,cap_net_admin+ep`, в macOS — root,
в Windows — [Npcap](https://npcap.com). Захваченные потоки приходят от
устройства `127.0.0.1`. У локального захвата нет интерфейсов и счётчиков
устройства, поэтому в **Сверке интерфейсов** для него нечего сравнивать.

<a id="10-options-and-data"></a>

## 10. Параметры и данные

`traffic66 -h` выводит всё. Самые используемые:

| Параметр | По умолчанию | |
|---|---|---|
| `-data` | `traffic66-data` рядом с программой | каталог данных |
| `-addr` | `:8066` | веб-интерфейс; `127.0.0.1:8066` — только для этой машины |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP-коллекторы; пустое значение отключает |
| `-retention-days` | `30` | сколько дней хранить детальные данные; сводки хранятся 400 дней |
| `-memory` | `0.10` | доля оперативной памяти под кеш базы данных |
| `-sampling-wait` | `5m` | сколько записи ждут частоту сэмплирования |
| `-capture` | | локальный интерфейс (можно повторять) |
| `-no-dns` | | без обратного разрешения имён |

В каталоге данных находятся `raw/` (детальные данные, один файл на час),
`traffic66.duckdb` (сводки и счётчики), `password`, `inventory.txt`,
`license.json`, ваш логотип и базы. Резервное копирование: остановите
traffic66 и скопируйте каталог; обновление: замените файл программы.
**Очистка данных** удаляет данные старше 7–120 дней или все данные.

<a id="licence"></a>

### Лицензия

Исходный код открыт по [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
и [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (юридическую
силу имеет английский текст): бесплатно для оценки и для организаций
численностью менее 100 человек; более крупные организации регистрируются
после 30 дней промышленного использования; для продажи, хостинга для других
или конкурирующих продуктов нужна коммерческая лицензия. Ничего никогда не
отключается. Внизу каждой страницы показан 8-значный номер установки;
отправьте его автору и положите полученный `license.json` в каталог данных.
Контакт: <https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Безопасность, оценка ресурсов, устранение неполадок

Веб-интерфейс работает по обычному HTTP: в недоверенных сетях используйте
`-addr 127.0.0.1:8066` за TLS-прокси (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) или SSH-туннель. Разрешайте UDP-порты только с ваших
устройств. SNMP-сообщества хранятся открытым текстом; используйте сообщества
только для чтения.

При 5,000 потоков/с на 2 ядрах: около 12 ГБ диска на сутки детальных данных
(360 ГБ за 30 дней), шестая часть ядра, 0,6–0,8 ГБ памяти. Обзоры за
длинные периоды строятся менее чем за 0,2 с; Топ 66 всех сеансов за 1 час —
около 9 с.

| Симптом | Решение |
|---|---|
| "ожидание частоты сэмплирования" | Экспортируйте параметры сэмплера или укажите `sampling=N` / `unsampled` в строке устройства |
| Меньше, чем счётчики | Интерфейсы без выборки, потери или активный тайм-аут больше 60 с |
| Больше, чем счётчики | Один и тот же трафик сэмплируется на двух интерфейсах или устройствах |
| Забыли пароль | `traffic66 passwd` на машине с traffic66 |
| `Conflicting lock is held` | Этот каталог данных использует другой traffic66 |
| `address already in use` | Выберите другие порты с помощью `-addr` или `-listen` |
| Windows: "Система Windows защитила ваш компьютер" | **Подробнее** → **Выполнить в любом случае** |

Сборка из исходников: Go 1.26 и компилятор C, затем `CGO_ENABLED=1 go build ./cmd/traffic66`.
