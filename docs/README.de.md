[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | **Deutsch** | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — NetFlow-, sFlow- und IPFIX-Collector und Traffic-Analyzer

Flow-Analyse für sFlow, NetFlow und IPFIX in einem einzigen Programm: wer
die Bandbreite nutzt, wohin der Verkehr geht und ob die Zahlen mit den
Schnittstellenzählern der Geräte selbst übereinstimmen, in einer
Weboberfläche und einer Terminaloberfläche.

## Schnellstart

```
# Linux, macOS: nach ./traffic66 laden und mit dem Erfassen beginnen
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: dasselbe
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

Das Terminal zeigt die Adresse der Weboberfläche sowie IP und UDP-Ports für Ihre Geräte ([Gerätekonfiguration](MANUAL.de.md#device-configuration)). Als `admin` / `traffic66` anmelden und ein neues Passwort wählen. Nur anschauen? Die Demo mit simulierten Geräten:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Ports: TCP 8066 Weboberfläche, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (jeder UDP-Port nimmt jedes Protokoll). Das Terminal zeigt die Adressen für Ihre Geräte; die erste Anmeldung verlangt ein neues Passwort.

macOS von Hand: zuerst `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows „Der Computer wurde geschützt“: **Weitere Informationen** → **Trotzdem ausführen**.

## Befehlsübersicht

| Aufgabe | Befehl |
|---|---|
| sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 erfassen (Standard) | `./traffic66` |
| Nur NetFlow v5/v9 auf 2055 und IPFIX auf 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Nur sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Weboberfläche auf IP und Port | `./traffic66 -addr 10.0.0.5:8080` |
| Eigene Netzwerkkarte erfassen (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| admin-Passwort setzen oder zurücksetzen (ggf. -data DIR) | `./traffic66 passwd` |
| Terminaloberfläche zu einem entfernten Server | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Übersicht: offene Befunde, Bandbreite nach Anwendung im Vergleich zur gleichen Zeit gestern, wichtigste Clients und Dienste](images/overview.png)

## Vollständiges Handbuch

in der Weboberfläche: **Hilfe** im Menü; auf GitHub: [MANUAL.de.md](MANUAL.de.md)

15 Sprachen. Kostenlos zur Evaluierung und für Organisationen mit weniger als 100 Personen ([Lizenz](../LICENSE.md)).
