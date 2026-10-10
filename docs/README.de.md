[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | **Deutsch** | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow-, sFlow- und IPFIX-Collector und Traffic-Analyzer

Flow-Analyse für sFlow, NetFlow und IPFIX in einem einzigen Programm: wer
die Bandbreite nutzt, wohin der Verkehr geht und ob die Zahlen mit den
Schnittstellenzählern der Geräte selbst übereinstimmen, in einer
Weboberfläche und einer Terminaloberfläche.

Eine selbst gehostete Alternative zu ntopng, ElastiFlow, pmacct mit Grafana
oder den Flow-Modulen von PRTG und SolarWinds NTA, für Netzwerküberwachung,
Bandbreitenüberwachung, Top Talkers, DDoS- und Scan-Erkennung sowie
pcap-Analyse, ohne Elasticsearch, Kafka oder eine separate Datenbank.

- Eine einzige ausführbare Datei für Windows, Linux und macOS; keine Datenbank zu installieren, funktioniert offline.
- sFlow v5, NetFlow v5/v9 und IPFIX auf jedem UDP-Port oder lokale Erfassung an einer Schnittstelle.
- Prüft seine Zahlen gegen die Schnittstellenzähler (sFlow oder SNMP) und erklärt, warum sie abweichen.
- Erkennt Scans, Passwortraten, Lateral Movement, ungewöhnliche Uploads, Floods und Verkehr mit Adressen aus Bedrohungslisten, auch durch Sampling hindurch.
- `traffic66 capture.pcap` analysiert Paketmitschnitte ohne jede Einrichtung.
- 15 Sprachen. Kostenlos zur Evaluierung und für Organisationen mit weniger als 100 Personen ([Lizenz](#licence)).

![Übersicht: offene Befunde, Bandbreite nach Anwendung im Vergleich zur gleichen Zeit gestern, wichtigste Clients und Dienste](images/overview.png)

<sub>Alle Screenshots stammen aus `traffic66 demo`, einem simulierten Firmennetz.</sub>

<a id="contents"></a>

## Inhalt

1. [Demo ausprobieren](#1-try-the-demo)
2. [Installation](#2-install)
3. [Benutzer und Passwörter](#3-users-and-passwords)
4. [Flows von Ihren Geräten senden](#4-send-flows-from-your-devices)
5. [Prüfen, ob Flows ankommen](#5-check-that-flows-arrive)
6. [Schnittstellen und Zähler](#6-interfaces-and-counters)
7. [Namen, Länder und Bedrohungslisten](#7-names-countries-and-threat-lists)
8. [Die Weboberfläche verwenden](#8-using-the-web-ui)
9. [Offline-pcap, Terminaloberfläche, lokale Erfassung](#9-offline-pcap-terminal-ui-local-capture)
10. [Optionen und Daten](#10-options-and-data)
11. [Sicherheit, Dimensionierung, Fehlerbehebung](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. Demo ausprobieren

Laden Sie das Archiv für Ihr System von der
[Release-Seite](https://github.com/githubflyideas/traffic66/releases)
herunter (Windows x64, Linux x86-64/ARM64 mit Kernel 3.2+, macOS 11+),
entpacken Sie es und führen Sie aus:

```
./traffic66 demo          # Linux, macOS
.\traffic66.exe demo      # Windows
```

Unter macOS führen Sie zuerst `xattr -dr com.apple.quarantine <folder>` aus.
Öffnen Sie http://127.0.0.1:8066 als `admin` / `traffic66`: ein Tag Verlauf
und Live-Verkehr von vier simulierten Geräten, darunter ein Angriff, der
unter **Befunde** Schritt für Schritt gezeigt wird. Strg+C beendet die
Demo; löschen Sie `traffic66-demo`, um neu zu beginnen. Um sie neben einer
echten Installation zu betreiben: `-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. Installation

traffic66 ist eine einzige Datei. Ports: UDP 6343 (sFlow), 2055 (NetFlow),
4739 (IPFIX), TCP 8066 (Weboberfläche); jeder UDP-Port nimmt jedes
Protokoll an.

**Linux** (systemd):

```
sudo mkdir -p /opt/traffic66 && sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
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
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload
```

**Windows** (PowerShell als Administrator): Entpacken Sie nach
`C:\traffic66`, führen Sie `C:\traffic66\traffic66.exe passwd` aus, öffnen
Sie die Ports und lassen Sie das Programm beim Systemstart starten:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

Ein Doppelklick auf `traffic66.exe` funktioniert ebenfalls: Er öffnet die
Weboberfläche.

**macOS**: Entpacken Sie nach `/usr/local/traffic66`, entfernen Sie das
Quarantäne-Attribut, führen Sie `traffic66 passwd -data "/Library/Application Support/traffic66"`
aus und starten Sie das Programm über einen LaunchDaemon, dessen
`ProgramArguments` das Programm, `-data` und dieses Verzeichnis sind, mit
`RunAtLoad` und `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. Benutzer und Passwörter

Bei einer neuen Installation melden Sie sich als `admin` / `traffic66` an;
die erste Anmeldung verlangt ein neues Passwort, bevor irgendetwas anderes
angezeigt wird (die Demo behält `traffic66`). Danach ändern Sie unter
**Konto**, am Fuß des Menüs, Ihr Passwort; der Administrator (`admin`) legt
dort auch Benutzer an, löscht sie und setzt ihre Passwörter zurück. Die
Anmeldung über LDAP / Active Directory ist in Entwicklung.

Benutzer werden als gesalzene Hashes in `password` im Datenverzeichnis
gespeichert. Dasselbe geht auf dem traffic66-Rechner mit einem einzigen
Befehl (fügen Sie `-data …` hinzu, wenn traffic66 damit läuft):

| Ziel | Befehl |
|---|---|
| Passwort von `admin` ändern | `traffic66 passwd` |
| `alice` anlegen oder ihr Passwort ändern | `traffic66 passwd -user alice` |
| `alice` löschen | `traffic66 passwd -user alice -delete` |
| Benutzer auflisten | `traffic66 passwd -list` |

Änderungen gelten sofort. Abgesehen von der Benutzerverwaltung haben alle
Benutzer dieselben Rechte. Für Skripte und Container akzeptiert
`TRAFFIC66_PASSWORD=…` (oder `-password`) für diesen Lauf nur `-user` mit
diesem Passwort. Fünf falsche Passwörter innerhalb einer Minute sperren die
Adresse für eine Minute.

<a id="4-send-flows-from-your-devices"></a>

## 4. Flows von Ihren Geräten senden

`192.0.2.50` ist traffic66, `192.0.2.1` das Gerät. Setzen Sie den Active
Timeout auf 60 Sekunden, lassen Sie NetFlow/IPFIX-Geräte ihre
Sampler-Optionen exportieren und sampeln Sie **jede Schnittstelle
eingehend** (oder nur die Randschnittstellen): Dann wird jedes Paket genau
einmal gezählt. sFlow-Raten: etwa 1:1000 für 1 Gb/s, 1:4096 für 10 Gb/s,
1:8192 für 40/100 Gb/s.

```
! Cisco IOS-XE, NetFlow v9
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
# Huawei CloudEngine, sFlow (H3C Comware is similar)
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
# Juniper EX/QFX, sFlow
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow interfaces ge-0/0/0

# Arista EOS, sFlow
sflow sample 4096
sflow destination 192.0.2.50
sflow run

# MikroTik RouterOS 7
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Prüfen, ob Flows ankommen

**Einstellungen** listet innerhalb von Sekunden jedes Gerät auf, das
irgendetwas sendet: Protokoll, Sampling-Rate, Verluste, die gesampelten
Schnittstellen und was zu beheben ist, wenn der Status nicht grün ist.
sFlow-Verluste werden aufgeteilt in Verluste auf dem Weg (erhöhen Sie
`net.core.rmem_max`, wenn `netstat -su` Pufferfehler zeigt) und Samples,
die das Gerät selbst verworfen hat.

![Einstellungen: jedes Gerät mit Protokoll, Sampling, Verlusten und was zu beheben ist](images/sources.png)

Fehlt ein Gerät? Führen Sie `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739` aus: Kommt dort nichts an, liegt es am Routing, an einer
Firewall oder an der Gerätekonfiguration; kommen Pakete an, aber unter
**Einstellungen** erscheint nichts, liegt es an der lokalen Firewall oder an
`-listen`. `traffic66 simulate -to 192.0.2.50` von einem anderen Rechner
testet den Weg mit simulierten Geräten.

<a id="6-interfaces-and-counters"></a>

## 6. Schnittstellen und Zähler

Flow-Zahlen sind Schätzungen (Samples × Sampling-Rate). **Schnittstellenprüfung**
vergleicht sie mit den Schnittstellenzählern des Geräts (sFlow-Zähler oder
SNMP über eine `snmp`-Zeile unter Namen) und erklärt, warum sie abweichen:
nicht gesampelte Schnittstellen, derselbe Verkehr doppelt gesampelt,
Verluste auf dem Weg oder eine unbekannte Sampling-Rate. Jede Schnittstelle
hat ein Diagramm in Bit/s und eines in Paketen/s, eingehend grün und
ausgehend blau, Zähler gestrichelt.

In jeder Zeile vergibt **✎** einen Namen und ein kurzes Tag (etwa *uplink*),
und **☆** macht sie zur Standardschnittstelle (★), mit der die Seiten
geöffnet werden.

Ein Gerät, das nur einige Schnittstellen sampelt, zeigt auch die anderen
Enden dieser Flows. Diese **Gegenschnittstellen** stehen zuletzt in kleiner
grauer Schrift: Sie enthalten nur den Verkehr über die gesampelte
Schnittstelle. Die gesampelte Schnittstelle ergibt sich aus der
sFlow-Datenquelle oder dem Feld flowDirection (IPFIX 61); ohne diese
Angaben gilt eine Schnittstelle mit 90 % des Verkehrs eines Geräts als
gesampelt.

![Schnittstellenprüfung: Verkehr jeder Schnittstelle und die Flow-Schätzung neben dem Gerätezähler](images/interfaces.png)

traffic66 verwendet bereits die vom Gerät angewendete Rate, wartet auf
unbekannte Raten, gleicht Exportverluste aus, verteilt lange Flows auf ihre
Minuten und rechnet bei NetFlow/IPFIX 18 Byte Ethernet-Overhead pro Paket
hinzu (`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Namen, Länder und Bedrohungslisten

Klicken Sie auf eine beliebige Adresse und wählen Sie **Benennen…**, oder
verwenden Sie **Einstellungen → Namen**. Namen werden in `inventory.txt` im
Datenverzeichnis gespeichert:

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

Private Adressbereiche gelten immer als Ihre eigenen. Änderungen werden mit
**Speichern** wirksam, ohne Neustart.

Länder und Netzwerke (AS) funktionieren sofort mit den kostenlosen
Lite-Datenbanken von DB-IP ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **Einstellungen**
aktualisiert sie oder übernimmt stattdessen Dateien von MaxMind GeoLite2,
IPinfo Lite oder IPtoASN. Kartenumrisse: [Natural Earth](https://www.naturalearthdata.com).

![Geo & Netze: externer Traffic nach Ländern auf einer Weltkarte](images/geo.png)

Bedrohungslisten sind Textdateien mit einer Adresse oder einem Netz pro
Zeile in `<data>/threats/<name>.txt` (zum Beispiel Spamhaus DROP); starten
Sie nach Änderungen neu. Treffer erscheinen unter **Threat Intel**.

![Threat Intel: ein interner Host, der Daten an eine Adresse aus einer Bedrohungsliste sendet](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Die Weboberfläche verwenden

Jeder Wert auf jeder Seite ist anklickbar: **Nur dies anzeigen** / **Dies
ausschließen** (Filter gelten für alle Seiten), **Flow-Datensätze
anzeigen**, **Details anzeigen** (eine Seite zu einem Host oder Dienst),
**Benennen…**, **Online nachschlagen**, **Kopieren**.

| Seite | Was sie zeigt |
|---|---|
| Übersicht | Die Bandbreite der gewählten Schnittstelle; Verkehr nach Anwendung (gesamt, eingehend oder ausgehend) im Vergleich zu gestern oder letzter Woche; offene Befunde; wichtigste Clients und Dienste |
| Top 66 | Die 66 größten Verbindungen, nach jeder Spalte sortierbar oder gruppiert nach Anwendung, Netz, Segment, Gerät, Kapselung, VLAN; die 30 größten Talker |
| Traffic-Details | Ringdiagramme: Server und ihre Clients (oder umgekehrt) sowie Dienste |
| Flow-Pfade | Host → Anwendung → Land, oder Client → Dienst → Server, oder nach Netz |
| Schnittstellenprüfung | Jede Schnittstelle im Zeitverlauf gegen ihre Zähler; Namen, Tags, Standard |
| Flow-Datensätze | Die einzelnen Flows, live alle 5 Sekunden oder für einen beliebigen Zeitraum |
| Befunde, Threat Intel | Was Aufmerksamkeit erfordert ([unten](#findings)); Verkehr mit gelisteten Adressen |
| Geo & Netze | Eine Weltkarte nach Ländern, Netzwerke (AS) im Zeitverlauf |
| Einstellungen | Geräte, Sampling, Verluste, SNMP, Datenbanken, Logo, Namen |
| Offline-pcap-Analyse, Datenbereinigung | Mitschnittdateien ([unten](#9-offline-pcap-terminal-ui-local-capture)); Löschen alter Daten |

Über den Seiten: **Schnittstelle** (alle oder eine gesampelte
Schnittstelle; die Verkehrsseiten zeigen dann nur den Verkehr über diese),
der Zeitraum (15 Minuten bis 30 Tage oder benutzerdefiniert), Aktualisierung
alle 30 s und **Link kopieren** für genau diese Ansicht. Sprache und fünf
Farbschemata finden Sie am Fuß des Menüs. Diagramme enden dort, wo die
Daten vollständig sind: bei NetFlow/IPFIX so spät, wie die Geräte
exportieren (höchstens 2 Minuten). Zeiträume über 6 Stunden beginnen zur
vollen Stunde; eine Schnittstelle über 7 oder 30 Tage liest die
Flow-Details, ist daher langsamer und reicht nur so weit zurück, wie Details
aufbewahrt werden.

![Top 66: die 66 größten Verbindungen, nach jeder Spalte sortierbar](images/topn.png)

![Traffic-Details: Server mit ihren Clients und Dienste mit ihren Servern als Ringdiagramme](images/traffic.png)

![Details zu einem Host: die Befunde zu ihm, sein Verkehr, mit wem er kommuniziert, Dienste, Länder und neueste Flows](images/detail.png)

![Flow-Pfade: welcher Host welche Anwendung in Richtung welches Landes nutzt](images/paths.png)

![Übersicht auf Chinesisch](images/overview-zh.png)

<a id="findings"></a>

### Befunde

Alle 5 Minuten über die letzten 10 geprüft; etwas, das eine Stunde andauert,
ist ein einziger, wachsender Befund.

| Befund | Bedeutung |
|---|---|
| Scan, Portscan | Kleine Proben an viele Hosts auf einem Port oder an viele Ports eines Hosts |
| Passwortraten | Viele kurze Verbindungen zu einem Anmeldedienst |
| Lateral Movement | Dateifreigabe oder Fernverwaltung zu internen Hosts, die so etwas nie zuvor angeboten haben |
| Ungewöhnlicher Upload | 100 MB in 10 Minuten an eine neue Adresse, das Dreifache dessen, was zurückkam |
| Flood | Über 20.000 kleine Pakete/s an eine Adresse, das Zehnfache ihrer üblichen Rate |
| Bedrohungsliste | Verkehr mit einer gelisteten Adresse |

Aus dem eigenen Netz haben sie hohe Priorität, aus dem Internet niedrige.
**Erledigt** schließt einen Befund, **Kein Problem** blendet ihn dauerhaft
aus. Lateral Movement und Uploads benötigen einen Tag Verlauf. Durch ein
Sampling von 1:4096 wird der Angriff der Demo vollständig erkannt; sehr
kleine Scans können sich hinter dem Sampling verbergen.

![Befunde: jeder Schritt eines Angriffs, erkannt durch 1:4096-sFlow-Sampling](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Offline-pcap, Terminaloberfläche, lokale Erfassung

**Offline-pcap-Analyse** zeigt Paketmitschnitte (pcap, pcapng) mit denselben
Seiten, abgesehen von den Live-Daten: `traffic66 a.pcap b.pcapng` startet
auf 127.0.0.1 und öffnet den Browser (bis zu 3 Dateien, 3 GB; Strg+C löscht
die importierten Daten), oder laden Sie auf dieser Seite bis zu 3 Dateien
mit je 50 MB hoch. Es wird jeweils eine Datei analysiert, jede in einer
eigenen Datenbank: **Analysieren** in der Zeile einer Datei zeigt sie auf
allen Seiten, und die Leiste oben wechselt zu einer anderen. Ausgewertet
werden Flows, nicht Paketinhalte.

![Offline-Analyse: Mitschnittdateien mit ihren Paketen, Flows und Zeitraum](images/sandbox.png)

**Terminaloberfläche**: `traffic66 tui` auf dem traffic66-Rechner oder
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Tasten: 1–8 Seiten, Enter Aktionen, f nur dies anzeigen, x ausschließen,
t Zeitraum, w im Browser öffnen, q beenden; `-lang` wählt die Sprache.

![Terminaloberfläche: Übersicht](images/tui-overview.png)

![Terminaloberfläche: Top 66 Verbindungen](images/tui-topn.png)

**Lokale Erfassung** bildet Flows aus einer lokalen Schnittstelle, am besten
einem Port, der mit dem Mirror-Port eines Switches verbunden ist:
`traffic66 interfaces` listet sie auf, `-capture eth1` erfasst. Unter
Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux benötigt root oder `setcap cap_net_raw,cap_net_admin+ep`, macOS root,
Windows [Npcap](https://npcap.com). Erfasste Flows stammen vom Gerät
`127.0.0.1`. Die lokale Erfassung hat keine Geräteschnittstellen oder
-zähler, daher hat die **Schnittstellenprüfung** für sie nichts zu
vergleichen.

<a id="10-options-and-data"></a>

## 10. Optionen und Daten

`traffic66 -h` listet alles auf. Die meistgenutzten:

| Option | Standard | |
|---|---|---|
| `-data` | `traffic66-data` neben dem Programm | Datenverzeichnis |
| `-addr` | `:8066` | Weboberfläche; `127.0.0.1:8066` nur für diesen Rechner |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP-Collectors; leer deaktiviert sie |
| `-retention-days` | `30` | Tage mit Flow-Details; Zusammenfassungen werden 400 Tage aufbewahrt |
| `-memory` | `0.10` | Anteil des RAM für den Datenbank-Cache |
| `-sampling-wait` | `5m` | wie lange Datensätze auf eine Sampling-Rate warten |
| `-capture` | | lokale Schnittstelle (mehrfach angebbar) |
| `-no-dns` | | keine Reverse-Lookups |

Das Datenverzeichnis enthält `raw/` (Details, eine Datei pro Stunde),
`traffic66.duckdb` (Zusammenfassungen und Zähler), `password`,
`inventory.txt`, `license.json`, Ihr Logo und die Datenbanken. Zur
Sicherung stoppen Sie traffic66 und kopieren es; zum Upgrade ersetzen Sie
die Programmdatei. **Datenbereinigung** löscht Daten, die älter als 7–120
Tage sind, oder alle Daten.

<a id="licence"></a>

### Lizenz

Source-available unter der [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
und dem [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (die
englische Fassung ist verbindlich): kostenlos zur Evaluierung und für
Organisationen mit weniger als 100 Personen; größere Organisationen
registrieren sich nach 30 Tagen Produktivbetrieb; Verkauf, Hosting für
Dritte oder konkurrierende Produkte erfordern eine kommerzielle Lizenz. Es
wird nie etwas abgeschaltet. Am Fuß jeder Seite steht die 8-stellige
Installationsnummer; senden Sie sie an den Autor und legen Sie die
zurückgeschickte `license.json` in das Datenverzeichnis. Kontakt:
<https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Sicherheit, Dimensionierung, Fehlerbehebung

Die Weboberfläche ist reines HTTP: In nicht vertrauenswürdigen Netzen
verwenden Sie `-addr 127.0.0.1:8066` hinter einem TLS-Proxy (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) oder einem SSH-Tunnel. Erlauben Sie die UDP-Ports nur von
Ihren Geräten. SNMP-Communities werden im Klartext gespeichert; verwenden
Sie nur lesende.

Bei 5.000 Flows/s auf 2 Kernen: etwa 12 GB Festplatte pro Tag Details
(360 GB für 30 Tage), ein Sechstel eines Kerns, 0,6–0,8 GB Arbeitsspeicher.
Übersichten über lange Zeiträume dauern unter 0,2 s; ein Top 66 aller
Verbindungen über 1 Stunde etwa 9 s.

| Symptom | Abhilfe |
|---|---|
| "Warte auf die Sampling-Rate" | Sampler-Optionen exportieren oder `sampling=N` / `unsampled` in der Gerätezeile |
| Niedriger als die Zähler | Nicht gesampelte Schnittstellen, Verluste oder ein Active Timeout über 60 s |
| Höher als die Zähler | Derselbe Verkehr wird an zwei Schnittstellen oder Geräten gesampelt |
| Passwort vergessen | `traffic66 passwd` auf dem traffic66-Rechner |
| `Conflicting lock is held` | Eine andere traffic66-Instanz verwendet dieses Datenverzeichnis |
| `address already in use` | Andere Ports mit `-addr` oder `-listen` wählen |
| Windows: "Der Computer wurde durch Windows geschützt" | **Weitere Informationen** → **Trotzdem ausführen** |

Aus dem Quellcode bauen: Go 1.24 und ein C-Compiler, dann `scripts/build.sh 0.1.0 traffic66`.
