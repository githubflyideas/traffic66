[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | **Русский** | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

Анализ потоков sFlow, NetFlow и IPFIX в одной программе. traffic66
принимает экспорт потоков от коммутаторов, маршрутизаторов и межсетевых
экранов, хранит его во встроенной базе данных и показывает, кто занимает
полосу, куда уходит трафик и сходятся ли цифры с собственными счётчиками
интерфейсов устройств — в веб-интерфейсе и в терминальном интерфейсе.

- Один исполняемый файл для Windows, Linux и macOS. Не нужно ставить ни
  базу данных, ни среду выполнения; работает без доступа в интернет.
- sFlow v5, NetFlow v5, NetFlow v9 и IPFIX на любом UDP-порту; при желании —
  локальный захват с сетевого интерфейса или зеркального порта.
- Сверяет свои цифры со счётчиками интерфейсов (счётчики sFlow или SNMP) и,
  если они расходятся, объясняет почему.
- Списки Top 66, пути трафика, страны и сети, совпадения со списками угроз,
  записи потоков, инкапсуляция (GRE, IPIP, VXLAN, GENEVE, MPLS).
- 13 языков в веб-интерфейсе и в терминальном интерфейсе.

![Обзор: полоса по приложениям в сравнении с прошлой неделей, основные клиенты и сервисы](images/overview.png)

<sub>Все скриншоты сделаны в `traffic66 demo` — смоделированной сети компании, которую можно запустить самому (см. [Попробовать демо](#1-try-the-demo)).</sub>

<a id="contents"></a>

## Содержание

1. [Попробовать демо](#1-try-the-demo)
2. [Установка](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Пользователи и пароли](#3-users-and-passwords)
4. [Отправка потоков с устройств](#4-send-flows-from-your-devices)
5. [Проверка поступления потоков](#5-check-that-flows-arrive)
6. [Как добиться совпадения со счётчиками интерфейсов](#6-make-the-numbers-match-the-interface-counters)
7. [Названия, SNMP и собственные сети](#7-names-snmp-and-your-own-networks)
8. [Страны, сети и списки угроз](#8-countries-networks-and-threat-lists)
9. [Работа с веб-интерфейсом](#9-using-the-web-ui)
10. [Терминальный интерфейс](#10-terminal-ui)
11. [Локальный захват](#11-local-capture)
12. [Параметры](#12-options)
13. [Данные, резервное копирование, обновление, удаление](#13-data-backup-upgrade-uninstall)
14. [Безопасность](#14-security)
15. [Оценка ресурсов](#15-sizing)
16. [Устранение неполадок](#16-troubleshooting)
17. [Сборка из исходников](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. Попробовать демо

Скачайте архив для своей системы со
[страницы релизов](https://github.com/githubflyideas/traffic66/releases):

| Система | Архив |
|---|---|
| Windows 10/11, Server 2016 и новее (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: любой дистрибутив с ядром 3.2 или новее, включая CentOS 7 и Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: те же дистрибутивы | `traffic66-linux-arm64.tar.gz` |
| macOS 11 и новее, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 и новее, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (вторая строка разрешает macOS запускать скачанную из интернета
программу не из App Store):

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows (PowerShell):

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

Откройте http://127.0.0.1:8066 и войдите как `admin` / `try66`. Демо
разворачивает небольшую сеть компании с историей за сутки и живым трафиком
от четырёх имитируемых устройств; в ней спрятаны два инцидента. Начните со
страницы **Обзор**, щёлкните хост в **Основные клиенты**, выберите
**Подробнее** и дальше идите по щелчкам.
Остановка — Ctrl+C. Данные демо лежат в `traffic66-demo` рядом с
программой; удалите эту папку, чтобы начать демо с нуля.

Демо использует те же порты, что и рабочая установка (8066 и UDP 6343,
2055, 4739). Чтобы запустить его рядом с рабочей, задайте другие порты:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

В Windows можно также просто дважды щёлкнуть `traffic66.exe`. Так traffic66
запускается по-настоящему (не демо) и открывает веб-интерфейс в браузере;
пароль первого запуска показывается в чёрном окне, а закрытие окна
останавливает traffic66. Если Windows сообщает
"Система Windows защитила ваш компьютер", нажмите **Подробнее** →
**Выполнить в любом случае**.

<a id="2-install"></a>

## 2. Установка

traffic66 — это один файл. Установить его — значит положить его в нужное
место, выбрать каталог данных, задать пароль, открыть порты в межсетевом
экране и включить автозапуск. В примерах `192.0.2.50` — машина с
traffic66, а `192.0.2.1` — маршрутизатор; подставьте свои адреса.

Порты:

| Порт | Назначение |
|---|---|
| UDP 6343 | sFlow (по умолчанию) |
| UDP 2055 | NetFlow (по умолчанию) |
| UDP 4739 | IPFIX (по умолчанию) |
| TCP 8066 | веб-интерфейс и API |

Каждый UDP-порт принимает любой протокол, так что устройство может слать
NetFlow и на 6343, если так удобнее. Изменить или добавить порты можно
через `-listen`.

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

Последняя команда запросит пароль пользователя `admin`.

Создайте `/etc/systemd/system/traffic66.service`:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target
Wants=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
# only needed for local capture (-capture):
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

Запустите службу и увеличьте допустимый размер UDP-буферов, чтобы не
терять пакеты при всплесках:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Межсетевой экран, если это firewalld (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

или ufw (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

Распакуйте в `C:\traffic66` и задайте пароль (PowerShell от имени
администратора):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Данные пишутся в `C:\traffic66\traffic66-data`, рядом с программой.

Откройте порты в межсетевом экране:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

Чтобы попробовать в интерактивном режиме, запустите
`C:\traffic66\traffic66.exe` и остановите по Ctrl+C. Чтобы программа
работала в фоне с момента загрузки, даже когда никто не вошёл в систему,
зарегистрируйте её как задачу автозапуска:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` обязателен: без него Windows
останавливает задачу через три дня. Остановить задачу — `Stop-ScheduledTask -TaskName
traffic66`, удалить — `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Создайте `/Library/LaunchDaemons/traffic66.plist`:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>traffic66</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/traffic66/traffic66</string>
    <string>-data</string>
    <string>/Library/Application Support/traffic66</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardErrorPath</key><string>/Library/Logs/traffic66.log</string>
</dict>
</plist>
```

Запуск и остановка:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

Если в macOS включён брандмауэр, разрешите входящие подключения для
traffic66 в Системных настройках → Сеть → Брандмауэр → Параметры.

<a id="3-users-and-passwords"></a>

## 3. Пользователи и пароли

**Коротко:** пользователи и пароли хранятся в одном файле, `password`, в
каталоге данных. Вручную его не редактируют: команда `traffic66 passwd`
добавляет, меняет, выводит списком и удаляет пользователей. Откройте
`http://<traffic66 machine>:8066` и войдите под одним из них.

<a id="the-first-sign-in"></a>

### Первый вход

При первом запуске traffic66 создаёт пользователя `admin` со случайным
паролем и один раз показывает его:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Запуск двойным щелчком в Windows: в чёрном окне.
- Запуск в терминале: в этом же терминале.
- Служба Linux: `journalctl -u traffic66 | grep "first start"`
- Служба macOS: `grep "first start" /Library/Logs/traffic66.log`

Пропустили? Задайте новый командой `traffic66 passwd` (см. ниже). Если пароль
задан командой `traffic66 passwd` до первого запуска, как в шагах установки
выше, ничего не генерируется.

<a id="where-the-users-are-stored"></a>

### Где хранятся пользователи

В файле `password` в каталоге данных:

| Как запущен traffic66 | Файл |
|---|---|
| Распакован и запущен из своей папки (по умолчанию) | `traffic66-data/password` рядом с программой |
| Служба Linux (раздел 2) | `/var/lib/traffic66/password` |
| Задача автозапуска Windows (раздел 2) | `C:\traffic66\traffic66-data\password` |
| Служба macOS (раздел 2) | `/Library/Application Support/traffic66/password` |
| Демо | `traffic66-demo/password` рядом с программой |

Одна строка на пользователя. Пароли хранятся в виде хешей с солью, поэтому
прочитать их из файла не может никто, даже вы; если пароль забыт, задайте
новый. Читать файл может только его владелец.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### Управление пользователями

Выполняйте эти команды на машине с traffic66:

| Задача | Команда |
|---|---|
| Сменить пароль `admin` | `traffic66 passwd` |
| Добавить пользователя `alice` или сменить её пароль | `traffic66 passwd -user alice` |
| Удалить пользователя `alice` | `traffic66 passwd -user alice -delete` |
| Вывести список пользователей | `traffic66 passwd -list` |
| Задать случайный пароль и вывести его | `traffic66 passwd -generate` (с `-user` для других пользователей) |

- Команда дважды запрашивает новый пароль и не показывает вводимые символы.
  Используйте не меньше 8 символов.
- Если traffic66 запущен с `-data`, добавьте тот же `-data` к команде. Для
  службы Linux из раздела 2:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  В Windows (PowerShell от имени администратора):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- Изменения действуют сразу, без перезапуска: новый пароль работает при
  следующем входе, а удалённый пользователь выходит из открытых браузеров.
- Последнего оставшегося пользователя удалить нельзя; сначала добавьте
  другого.
- Все пользователи видят и могут менять одно и то же; ролей нет.

<a id="passwords-for-scripts-and-containers"></a>

### Пароли для скриптов и контейнеров

`TRAFFIC66_PASSWORD=…` в окружении или `-password …` в командной строке
заставляют traffic66 на этот запуск принимать ровно одного пользователя:
указанного в `-user` (по умолчанию `admin`) с этим паролем. Файл `password`
при этом игнорируется и не меняется. Лучше использовать переменную окружения:
командную строку видят другие пользователи машины.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

После пяти неверных паролей в течение минуты адрес блокируется на минуту.

<a id="4-send-flows-from-your-devices"></a>

## 4. Отправка потоков с устройств

Направьте каждое устройство на машину с traffic66. Команды зависят от
модели и версии ПО — сверяйтесь с документацией устройства. Во всех
примерах `192.0.2.50` — это traffic66, а `192.0.2.1` — собственный адрес
устройства.

Общие рекомендации:

- Задайте active flow timeout 60 секунд. При больших значениях трафик
  приходит с опозданием и крупными порциями.
- Если устройство сэмплирует NetFlow/IPFIX, включите экспорт sampler
  options, чтобы коэффициент был известен. traffic66 придерживает записи до
  получения коэффициента, а не считает их как 1:1.
- Сэмплируйте либо все интерфейсы, либо только граничные, и в одном
  направлении. Если один и тот же трафик сэмплируется на входе и на выходе,
  он считается дважды; страница **Сверка интерфейсов** на это укажет.
- Коэффициент сэмплирования sFlow: около 1:1000 для каналов 1 Гбит/с,
  1:4096 для 10 Гбит/с, 1:8192 для 40/100 Гбит/с.

Cisco IOS / IOS-XE (Flexible NetFlow):

```
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 export-protocol netflow-v9
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

Cisco NX-OS (sFlow):

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS (sFlow):

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX (sFlow):

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

Huawei CloudEngine (sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

H3C Comware (sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7 (NetFlow v9 / IPFIX):

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 и новее (NetFlow v9):

```
config system netflow
    config collectors
        edit 1
            set collector-ip 192.0.2.50
            set collector-port 2055
        next
    end
end
config system interface
    edit port1
        set netflow-sampler both
    next
end
```

Серверы и хосты Linux, через softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Проверка поступления потоков

Откройте **Источники**. Каждое устройство, которое что-либо присылает,
появляется там за несколько секунд — с протоколом, коэффициентом
сэмплирования, потерями, временем последнего пакета и статусом. Если
статус не зелёный, текст рядом с ним объясняет, что не так и что изменить.

![Источники: каждое устройство с протоколом, сэмплированием, потерями и тем, что нужно исправить](images/sources.png)

Если устройство не появилось:

1. Проверьте, приходят ли пакеты на машину с traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Если там пусто, пакеты до машины не доходят: проверьте конфигурацию
   устройства, маршрутизацию и межсетевые экраны на пути.
2. Пакеты приходят, но **Источники** остаются пустыми: их отбрасывает
   локальный межсетевой экран (см. [Установка](#2-install)) или traffic66
   слушает другие порты (`-listen`).
3. Чтобы проверить путь с другой машины, не трогая устройства, запустите
   там на несколько секунд `traffic66 simulate -to 192.0.2.50`. Команда
   шлёт sFlow, NetFlow и IPFIX от имитируемых устройств; они появятся в
   **Источники** и в данных, поэтому лучше делать это на тестовой установке.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Как добиться совпадения со счётчиками интерфейсов

Цифры по потокам — это оценки: число сэмплированных пакетов, умноженное на
коэффициент сэмплирования. traffic66 сравнивает их с собственными
счётчиками интерфейсов устройства и показывает разницу на странице
**Сверка интерфейсов**, а если она больше, чем можно объяснить одним
сэмплированием, — и вероятную причину.

![Сверка интерфейсов: оценка по потокам рядом со счётчиком устройства для каждого интерфейса](images/interfaces.png)

Откуда взять счётчики для сравнения:

- Устройства sFlow присылают их сами, если задан интервал счётчиков
  (`sflow counter interval 30` и аналоги).
- Для устройств NetFlow и IPFIX добавьте строку `snmp` в **Источники → Названия**
  (см. [Названия](#7-names-snmp-and-your-own-networks)). Тогда traffic66
  будет считывать счётчики интерфейсов каждую минуту.

Что traffic66 уже делает, чтобы цифры сходились: берёт коэффициент
сэмплирования, фактически применённый устройством; придерживает записи
NetFlow/IPFIX, пока коэффициент неизвестен; компенсирует экспортные
пакеты, потерянные по пути; распределяет длинные потоки по минутам, в
течение которых они шли; добавляет к байтам NetFlow/IPFIX по 18 байт
накладных расходов Ethernet на пакет (счётчики интерфейсов их включают,
а счётчики потоков на уровне IP — нет; меняется через `-l2-overhead`).

Типичные причины оставшейся разницы, и все они видны на странице
**Сверка интерфейсов**: часть интерфейсов не сэмплируется, один и тот же
трафик сэмплируется на двух интерфейсах, экспортные пакеты теряются, не
дойдя до traffic66, или коэффициент сэмплирования ещё неизвестен.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. Названия, SNMP и собственные сети

Быстрее всего дать название хосту или устройству так: щёлкните его адрес на
любой странице и выберите **Дать название…**. Введите название и нажмите
Enter; оно сразу сохраняется и показывается везде вместо голого адреса.

Для сетей, интерфейсов и SNMP раздел **Источники → Названия** в веб-интерфейсе
принимает по одной записи на строку. Содержимое сохраняется как `inventory.txt` в каталоге данных, так
что можно править и сам файл (см. `inventory.txt.example`). Все строки
необязательны.

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers

# device names; "unsampled" if it exports every packet (1:1)
device 192.0.2.1     Core router
device 192.0.2.9     Branch firewall unsampled

# interface names, by device address and ifIndex; speed in bits per second
iface  192.0.2.1 3   ISP uplink speed=1000000000

# host names shown instead of addresses
host   10.10.3.27    Finance PC

# read interface counters over SNMPv2c (IF-MIB 64-bit counters)
snmp   192.0.2.1     public
snmp   192.0.2.9     s3cret  10.99.0.9:161
```

- `net`: частные диапазоны (10/8, 172.16/12, 192.168/16, 100.64/10)
  всегда считаются вашими. Добавьте свои публичные диапазоны, чтобы трафик
  к ним и от них тоже считался вашим; название появится в
  **Top-N → Сегменты** и в путях трафика.
- `snmp <device> <community> [<management address>[:port]]`: устройство —
  это адрес, с которого приходят потоки. Укажите адрес управления, если
  устройство отвечает по SNMP на другом адресе. Описания интерфейсов,
  считанные по SNMP, используются как названия, если вы не задали название
  интерфейса через `iface`. Разрешите машину с traffic66 в списке доступа
  SNMP на устройстве.
- Изменения применяются по кнопке **Сохранить**; перезапуск не нужен.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Страны, сети и списки угроз

Для стран и названий сетей (AS) нужна база, сопоставляющая им адреса.
Загрузите её в веб-интерфейсе: **Источники → База стран и сетей → Загрузить файл базы…**.
Файл проверяется, сохраняется в каталоге данных и сразу используется для
нового трафика; перезапуск не нужен. Уже сохранённый трафик сохраняет ту
страну, с которой был записан.

Поддерживаемые файлы:

| Файл | Что даёт | Где взять |
|---|---|---|
| DB-IP Lite country или ASN, `.mmdb` | страну или номер и название AS | бесплатно, без регистрации: [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country или ASN, `.mmdb` | страну или номер и название AS | бесплатно, с учётной записью MaxMind: [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| Таблица IP-to-ASN, `.tsv` или `.tsv.gz` | номер AS, название AS и страну | бесплатно: [iptoasn.com](https://iptoasn.com) (`ip2asn-combined.tsv.gz`) |

Чтобы получить и то и другое, загрузите базу стран и базу ASN; если загружено
несколько, файлы `.mmdb` имеют приоритет в том, что они содержат. Новые
версии выходят ежемесячно: чтобы заменить старый файл, загрузите новый тем же
способом.

Без веб-интерфейса скопируйте файл в каталог данных как `country.mmdb`,
`asn.mmdb` или `asn.tsv.gz` и перезапустите traffic66.

Списки угроз — это текстовые файлы с одним адресом или сетью на строку
(текст после `#` или `;` игнорируется), сохранённые как
`<data directory>/threats/<name>.txt`, например:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

После добавления или изменения списков перезапустите traffic66.
Совпадения появляются на странице **Угрозы**, сгруппированные по имени
списка.

![Угрозы: внутренний хост отправляет данные на адрес из списка угроз](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Работа с веб-интерфейсом

Печатать почти не придётся. Любое значение на любой странице — адрес,
порт, приложение, страну, устройство — можно щёлкнуть:

- **Показать только это** / **Исключить это** добавляет фильтр. Фильтры
  отображаются под верхней панелью и действуют на всех страницах, пока вы
  их не уберёте.
- **Показать записи потоков** открывает соответствующие отдельные потоки.
- **Подробнее** (хосты, устройства и сервисы) открывает страницу об этом
  одном хосте или сервисе: его трафик во времени по приложениям, с кем он
  общается, какие сервисы или клиенты, страны и последние потоки. Любое
  значение там снова можно щёлкнуть и углубляться дальше; кнопка «Назад»
  в браузере возвращает обратно.
- **Дать название…** (хосты и устройства) даёт адресу название, которое
  с этого момента показывается везде.
- **Найти в интернете** открывает адрес или AS на публичном сайте-справочнике.
- **Копировать** копирует значение.

Страницы:

| Страница | На какой вопрос отвечает |
|---|---|
| Обзор | Сколько трафика сейчас и по сравнению с прошлой неделей, по приложениям; основные клиенты и сервисы |
| Top-N | Одна таблица топ 66: по умолчанию сеансы (клиент, сервер, сервис, страна). Любой заголовок сортирует; числовые столбцы (трафик, пакеты, средний пакет, потоки) заново выбирают топ 66 из всего трафика за период, поэтому наименьший средний пакет находит сканирования и флуд. **Группировать по** переключает на приложения, сети, сегменты, устройства, инкапсуляцию и VLAN |
| Пути трафика | Какой сегмент обращается к какому приложению в какой стране |
| География и сети | Трафик по странам и по сетям (AS) |
| Угрозы | Хосты, обращавшиеся к адресам из ваших списков угроз, и сколько они отправили |
| Записи потоков | Отдельные потоки, новые сверху, с выбором столбцов |
| Сверка интерфейсов | Цифры по потокам рядом со счётчиками интерфейсов, худшие сверху, с причинами |
| Источники | Устройства, сэмплирование, потери, коллекторы, SNMP, база стран и сетей и **Названия** |

Над страницами: временной диапазон (от 15 минут до 30 дней), необязательная
строка поиска, автообновление каждые 30 секунд и **Скопировать ссылку** —
копирует ссылку ровно на текущий вид (страница, диапазон и фильтры), чтобы
отправить коллеге. Язык берётся из браузера; сменить его можно внизу меню.

Диапазоны длиннее 6 часов начинаются с целого часа, поэтому все цифры на
странице считаются ровно за одно и то же время: «24 часа» — это последние 24
целых часа плюс текущий. Top-N за такие диапазоны строится по почасовым
сводкам; фильтры там недоступны, и страница об этом сообщает. Чтобы
фильтровать, выберите диапазон покороче. Сеансы всегда читаются из подробных
данных потоков, поэтому на длинных диапазонах при большом числе потоков
они могут строиться долго; быстрее всего — один час.

В боковом меню видно, сколько места на диске занимают данные и сколько
свободно; наведите указатель на свободное место, чтобы увидеть, сколько
нужно хранимым дням подробных данных при текущей нагрузке (оценка появляется,
когда накопятся данные за сутки).

![Top-N: топ 66 сеансов за последний час](images/topn.png)

![Подробно об одном хосте: его трафик, с кем он общается, сервисы, страны и последние потоки](images/detail.png)

![Пути трафика: какой сегмент к какому приложению в какой стране обращается](images/paths.png)

Тот же обзор на китайском; каждая страница доступна на 13 языках:

![Обзор на китайском](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. Терминальный интерфейс

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

На машине с traffic66 `traffic66 tui` входит сам, если может прочитать
каталог данных (укажите `-data`, если он не стандартный). Если traffic66
работает от другого пользователя, как в случае службы, используйте
`-user` и `-password`. `-lang` выбирает язык (`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

Клавиши: 1–8 страницы, ↑↓ выбор, Enter действия над выбранным значением,
f показать только это, x исключить, / поиск, t временной диапазон,
c сбросить фильтры, w открыть тот же вид в браузере, q выход.

![Терминальный интерфейс: обзор](images/tui-overview.png)

![Терминальный интерфейс: Top-N сеансов](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Локальный захват

Помимо приёма экспорта потоков, traffic66 может сам строить потоки из пакетов
на сетевом интерфейсе машины, на которой он работает. Что он увидит, зависит
от интерфейса:

| Интерфейс | Что видит traffic66 |
|---|---|
| Свободный сетевой порт, подключённый к зеркальному (SPAN) порту коммутатора | Весь трафик, который зеркалирует коммутатор: целую сеть или аплинк |
| Собственный Ethernet или Wi-Fi машины | Только трафик самой этой машины |

Адаптеры Wi-Fi не видят трафик других устройств. Чтобы видеть всю сеть Wi-Fi,
настройте экспорт потоков на маршрутизаторе или точке доступа (раздел 4) или
зеркалируйте порт коммутатора, к которому подключена точка доступа.

<a id="windows-1"></a>

### Windows

1. Установите [Npcap](https://npcap.com) с параметрами по умолчанию. Если
   отметить «Restrict Npcap driver's access to Administrators only»,
   запускайте traffic66 от имени администратора.
2. Выведите список интерфейсов (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Столбец Name — это имя подключения из сетевых параметров Windows; у
   используемого интерфейса есть адрес.
3. Запустите захват на Wi-Fi по имени или по номеру:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Имена с пробелами берите в кавычки: `-capture "Ethernet 2"`. Чтобы
   захватывать на нескольких интерфейсах, повторите `-capture`. Если нужен
   только захват без коллекторов потоков, добавьте `-listen=`. Для задачи
   автозапуска из раздела 2 добавьте параметр в `-Argument`:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

Для захвата нужен root или capabilities `CAP_NET_RAW` и `CAP_NET_ADMIN`:
строка `setcap` выше или строка `AmbientCapabilities` в unit-файле systemd из
раздела 2. Интерфейсы Wi-Fi обычно называются `wlan0` или `wlp…`.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

Для захвата нужен root; ничего устанавливать не надо. На MacBook `en0` — это
Wi-Fi.

<a id="checking-that-it-works"></a>

### Проверка работы

На странице **Источники** перечислены все интерфейсы захвата со способом
захвата и числом увиденных пакетов. Потоки отображаются как поступающие от
устройства `127.0.0.1` (эта машина) — на всех страницах, как и потоки любого
другого устройства. Пакеты, видимые дважды (например, на двух зеркальных
портах), и считаются дважды.

<a id="12-options"></a>

## 12. Параметры

`traffic66 -h` и `traffic66 <command> -h` выводят полный список.

Команды:

| Команда | |
|---|---|
| `traffic66` | собирать потоки и обслуживать веб-интерфейс |
| `traffic66 demo` | то же самое, но с имитируемой сетью |
| `traffic66 tui` | терминальный интерфейс к запущенному traffic66 |
| `traffic66 passwd` | добавить, изменить, вывести списком или удалить пользователей (см. [Пользователи и пароли](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | отправлять имитируемый экспорт на коллектор |
| `traffic66 interfaces` | список интерфейсов для локального захвата |
| `traffic66 version` | вывести версию |

Параметры `traffic66` и `traffic66 demo`:

| Параметр | По умолчанию | |
|---|---|---|
| `-addr` | `:8066` | адрес веб-интерфейса; `127.0.0.1:8066` — доступ только с этой машины |
| `-data` | `traffic66-data` рядом с программой | каталог данных |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP-коллекторы в виде `name=address` через запятую; пустое значение отключает |
| `-user` | `admin` | имя пользователя, создаваемого при первом запуске, и пользователя, к которому относится `-password` |
| `-password` | не задан | на этот запуск принимать только `-user` с этим паролем, игнорируя файл `password` (также `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | сколько дней хранить детальные данные; сводки хранятся 400 дней |
| `-memory` | `0.10` | доля физической памяти под кеш базы данных и столько же — мягкий лимит для остальной программы (каждый не меньше 256 МБ) |
| `-l2-overhead` | `18` | байт на пакет, добавляемых к счётчикам байт NetFlow/IPFIX |
| `-sampling-wait` | `5m` | сколько записи ждут коэффициент сэмплирования |
| `-capture` | | захват на локальном интерфейсе (можно повторять) |
| `-inventory` | `<data>/inventory.txt` | файл названий |
| `-asn` | `<data>/asn.tsv.gz` | таблица IP-to-ASN (файлы `.mmdb`: загрузите их или положите как `<data>/country.mmdb` и `<data>/asn.mmdb`) |
| `-threat` | `<data>/threats/*.txt` | дополнительный список угроз в виде `name=path` (можно повторять) |
| `-dns-upstream` | системный резолвер | DNS-сервер для отображения имён хостов |
| `-dns-rate` | `20` | максимум обратных DNS-запросов в секунду |
| `-dns-cache` | `2m` | сколько хранятся в кеше имена хостов |
| `-no-dns` | | не делать обратных DNS-запросов |
| `-tui` | | заодно открыть терминальный интерфейс |

Пример: второй порт коллектора, детальные данные за год и веб-интерфейс
только на локальной машине:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. Данные, резервное копирование, обновление, удаление

Всё хранится в каталоге данных:

| | |
|---|---|
| `raw/` | детальные данные потоков, по одному сжатому файлу на час |
| `traffic66.duckdb` | сводки, счётчики интерфейсов и текущий час |
| `password` | пароли для входа (в виде хешей) |
| `inventory.txt` | названия (**Источники → Названия**) |
| `country.mmdb`, `asn.mmdb`, `asn.tsv.gz`, `threats/` | добавленные вами базы стран и сетей и списки угроз |

- **Резервное копирование**: остановите traffic66 и скопируйте каталог. Без
  остановки копируйте `raw/`, `password` и `inventory.txt`; текущего часа и
  сводок в копии тогда не будет.
- **Перенос**: остановите traffic66, перенесите каталог и запустите с
  `-data`, указывающим на новое место.
- **Обновление**: остановите traffic66, замените файл программы и запустите
  снова. Данные сохраняются. Например, в Linux:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Удаление**: остановите и удалите службу или задачу автозапуска (см.
  [Установка](#2-install)), затем удалите папку программы и каталог данных.

<a id="14-security"></a>

## 14. Безопасность

- Веб-интерфейс работает по обычному HTTP: пароли и данные идут по сети без
  шифрования. В сетях, которым вы не доверяете полностью, слушайте только
  на этой машине (`-addr 127.0.0.1:8066`) и поставьте перед traffic66
  обратный прокси с TLS, например [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  Или подключайтесь через VPN или SSH-туннель:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, затем откройте
  http://127.0.0.1:8066.
- Открывайте UDP-порты коллекторов только для адресов своих устройств.
- SNMP community в `inventory.txt` хранятся открытым текстом; используйте
  community только для чтения.

<a id="15-sizing"></a>

## 15. Оценка ресурсов

Замер при 5000 потоков в секунду на 2-ядерной машине: детальные данные
занимают около 12 ГБ диска в сутки плюс около 1,5 ГБ на текущий час,
программа — шестую часть одного ядра. Обзоры за
длинные диапазоны строятся по сводкам и выполняются быстрее 0,2 с. Запросы
по детальным данным просматривают около 22 млн строк на час: один хост за
1 час — меньше 1 с, Top 66 всех сессий за 1 час — около 9 с; время растёт
с диапазоном и сокращается с числом ядер.

Значит, на 30 дней при 5000 потоков/с нужно около 360 ГБ диска;
пересчитайте под свою интенсивность потоков (видна на странице
**Источники**) и `-retention-days`.

Память: `-memory` (по умолчанию 10 % ОЗУ, не меньше 256 МБ) ограничивает
кеш базы данных, а остальная программа получает мягкий лимит того же
размера. При 5000 потоков в секунду собственные данные программы
(декодирование, обнаружение дубликатов, пачки записей) занимают около
90 МБ; всего рассчитывайте на 0,6–0,8 ГБ, так что машины с 2 ГБ ОЗУ
достаточно. Замерено за 10 минут непрерывного сбора (пик 0,58 ГБ на машине
с 8 ГБ) и при загрузке часа потоков с интенсивностью в одиннадцать раз
выше с ограничениями машины с 2 ГБ (пик 0,74 ГБ).

`-memory` — это бюджет, а не жёсткий предел: лимит Go мягкий, и база данных
может ненадолго превысить свою долю. Для жёсткого предела используйте
средства операционной системы: `MemoryMax=` в unit-файле systemd (раздел 2)
или лимит памяти контейнера. Закладывайте примерно в 2,5 раза больше доли
`-memory` и не меньше 1 ГБ; при доле по умолчанию `MemoryMax=2G` подходит
для машин с ОЗУ до 8 ГБ. Тогда перезапускается traffic66, а не у машины
заканчивается память.

<a id="16-troubleshooting"></a>

## 16. Устранение неполадок

| Симптом | Причина и решение |
|---|---|
| Устройства нет на странице **Источники** | Пакеты не доходят: см. [Проверка поступления потоков](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | Устройство ещё не прислало sampler options; большинство устройств повторяют их в течение нескольких минут. Если так и не пришлёт, включите их экспорт (`option sampler-table` на Cisco) или пометьте устройство как `unsampled` в разделе «Названия», если оно действительно экспортирует 1:1 |
| Цифры ниже счётчиков интерфейсов | См. **Сверка интерфейсов**: потери по пути, несэмплируемые интерфейсы или потоки ещё в кеше устройства (active timeout больше 60 с) |
| Цифры выше счётчиков интерфейсов | Один и тот же трафик сэмплируется на двух интерфейсах или двух устройствах |
| Нет стран и сетей («Неизвестно») | Не загружена база: загрузите её на странице **Источники**, см. [Страны](#8-countries-networks-and-threat-lists) |
| "База данных упёрлась в лимит памяти и не смогла ответить" на странице | Выберите период покороче или запустите с бо́льшим `-memory`; подробности в журнале |
| Забыт пароль | `traffic66 passwd` на машине с traffic66 (добавьте `-data`, если traffic66 запущен с ним) |
| `Conflicting lock is held` | Этот каталог данных уже использует другой экземпляр traffic66 |
| `receive buffer is only … KB` | Linux ограничивает размер UDP-буферов: задайте `net.core.rmem_max=16777216` (см. [Linux](#linux)) |
| `cannot create the data directory` | У этого пользователя нет прав на запись в папку программы: укажите `-data` |
| macOS: "cannot be opened" или "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Система Windows защитила ваш компьютер" | **Подробнее** → **Выполнить в любом случае**; программа пока не подписана |
| Захват в Windows: Npcap не найден | Установите [Npcap](https://npcap.com) |
| `address already in use` | Порт занят другой программой: выберите другие через `-addr` или `-listen` |

<a id="17-build-from-source"></a>

## 17. Сборка из исходников

Go 1.24 и компилятор C (gcc или clang; в Windows — MinGW-w64):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
