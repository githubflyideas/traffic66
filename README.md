**English** | [中文](docs/README.zh.md) | [हिन्दी](docs/README.hi.md) | [Español](docs/README.es.md) | [العربية](docs/README.ar.md) | [Français](docs/README.fr.md) | [বাংলা](docs/README.bn.md) | [Português](docs/README.pt.md) | [Русский](docs/README.ru.md) | [Bahasa Indonesia](docs/README.id.md) | [اردو](docs/README.ur.md) | [Deutsch](docs/README.de.md) | [日本語](docs/README.ja.md) | [Tiếng Việt](docs/README.vi.md) | [한국어](docs/README.ko.md)

# traffic66

A single-binary NetFlow / sFlow / IPFIX collector and traffic analyzer, with a web UI and a terminal UI.

<img src="docs/images/overview.png" alt="Overview" width="100%">

Just download and run

```bash
# Linux x86_64 (also: linux-arm64, darwin-amd64, darwin-arm64 .tar.gz, windows-amd64.zip)
wget https://github.com/githubflyideas/traffic66/releases/latest/download/traffic66-linux-amd64.tar.gz
tar -zxvf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64

# run, then open http://localhost:8066  (admin / traffic66, change at first login)
./traffic66

# capture this machine's traffic (names from: ./traffic66 interfaces)
sudo ./traffic66 -capture eth0

# run in background
nohup ./traffic66 > traffic66.log 2>&1 &

# change the admin password
./traffic66 passwd
```

Windows: unzip `traffic66-windows-amd64.zip`, install [Npcap](https://npcap.com/#download) for local capture, then in an admin `cmd` in that folder run `traffic66.exe -capture Wi-Fi` (names from `traffic66.exe interfaces`).

Devices: send sFlow to UDP 6343, NetFlow v5/v9 to UDP 2055, IPFIX to UDP 4739 on this host.

## Screenshots

<img src="docs/images/overview.png" alt="Overview" width="49%"> <img src="docs/images/topn.png" alt="Top 66" width="49%">

<img src="docs/images/geo.png" alt="Geo & networks" width="49%"> <img src="docs/images/interfaces.png" alt="Interface check" width="49%">

<img src="docs/images/tui-top.png" alt="TUI" width="49%">

## Parameters

```bash
./traffic66 -capture eth0                     # capture a local interface (name or number)
./traffic66 -addr :9000                       # web UI port / bind address (default :8066)
./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739   # collector ports
./traffic66 -data /var/lib/traffic66          # data directory (default ./traffic66-data)
./traffic66 interfaces                        # list capturable interfaces
./traffic66 passwd                            # set the admin password
./traffic66 tui                               # terminal UI (remote: -server http://IP:8066)
./traffic66 demo                              # try with simulated devices
```

macOS: run `xattr -dr com.apple.quarantine .` in the folder first. PowerShell: use `.\traffic66.exe`.

## FAQ

**No data?** Check the device exports to this host and port, and the firewall allows that UDP port. NetFlow v9/IPFIX need a template first and long flows appear at the active timeout: set both to 60 s on the device.

**Interface name?** `./traffic66 interfaces`

**Forgot the password?** Stop traffic66 and run `./traffic66 passwd` (add `-data DIR` if used).

## Manual

Web UI: **Help** in the menu. GitHub: [MANUAL.md](docs/MANUAL.md)

Free for evaluation and for organizations under 100 people ([licence](LICENSE.md)).
