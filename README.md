**English** | [中文](docs/README.zh.md) | [हिन्दी](docs/README.hi.md) | [Español](docs/README.es.md) | [العربية](docs/README.ar.md) | [Français](docs/README.fr.md) | [বাংলা](docs/README.bn.md) | [Português](docs/README.pt.md) | [Русский](docs/README.ru.md) | [Bahasa Indonesia](docs/README.id.md) | [اردو](docs/README.ur.md) | [Deutsch](docs/README.de.md) | [日本語](docs/README.ja.md) | [Tiếng Việt](docs/README.vi.md) | [한국어](docs/README.ko.md)

# traffic66 — NetFlow, sFlow and IPFIX collector and traffic analyzer

Flow analytics for sFlow, NetFlow and IPFIX in a single program: who uses
the bandwidth, where the traffic goes, and whether the numbers match the
devices' own interface counters, in a web UI and a terminal UI.

## Quick start

1. Download the archive for your system from [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Unzip it and run the program:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: unzip `traffic66-windows-amd64.zip` and run `traffic66.exe` (double-click works).

macOS: first run `xattr -dr com.apple.quarantine .` in the folder.

Then:

- Web UI: `http://<server IP>:8066`. It listens on all network interfaces.
- Sign in: user `admin`, password `traffic66`. The first sign-in makes you set a new password.
- Point your devices at the server IP: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (all interfaces; every port accepts every protocol).

## Common commands

- `./traffic66`  
  Start with the defaults above.

- `./traffic66 -addr :9000`  
  Web UI on port 9000 instead of 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Change the collector ports (here NetFlow on 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Listen on one IP address only, instead of all interfaces.

- `./traffic66 -data /var/lib/traffic66`  
  Keep the data in another directory (default: `traffic66-data` next to the program).

- `./traffic66 passwd`  
  Set or reset the admin password (add `-data DIR` if you use one).

- `traffic66.exe interfaces`  
  Windows: list the network cards (name and number).

- `traffic66.exe -capture Wi-Fi`  
  Windows: analyse this PC's own traffic on that card (name or number). Needs Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: list the network cards.

- `sudo ./traffic66 -capture eth0`  
  Linux: analyse this machine's own traffic on eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Terminal UI on the traffic66 machine.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Terminal UI connected to a remote traffic66.

- `./traffic66 demo`  
  Try it with simulated devices.

## Screenshots

<img src="docs/images/overview.png" alt="Overview" width="49%"> <img src="docs/images/topn.png" alt="Top 66" width="49%">

<img src="docs/images/geo.png" alt="Geo & networks" width="49%"> <img src="docs/images/interfaces.png" alt="Interface check" width="49%">

<img src="docs/images/tui-top.png" alt="TUI" width="49%">

## Full manual

In the web UI: **Help** in the menu. On GitHub: [MANUAL.md](docs/MANUAL.md)

15 languages. Free for evaluation and for organizations under 100 people ([licence](LICENSE.md)).
