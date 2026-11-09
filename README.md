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

Needs: a ~73 MB program, nothing to install; about 0.1 GB of memory idle and 0.6–0.8 GB at 5,000 flows/s, so give it 1 GB or more. Keep the web UI on an internal network or behind a reverse proxy with HTTPS; don't expose it directly to the internet.

## Screenshots

<img src="docs/images/topn.png" alt="Top 66" width="49%">

<img src="docs/images/geo.png" alt="Geo & networks" width="49%"> <img src="docs/images/interfaces.png" alt="Interface check" width="49%">

<img src="docs/images/tui-top.png" alt="TUI" width="49%">

## Parameters

```bash
./traffic66 -capture eth0                     # capture a local interface (name or number)
./traffic66 -addr :9000                       # web UI port / bind address (default :8066)
./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739   # collector ports
./traffic66 -data /var/lib/traffic66          # data directory (default: traffic66-data next to the program)
./traffic66 interfaces                        # list capturable interfaces
./traffic66 passwd                            # set the admin password
./traffic66 tui                               # terminal UI (remote: -server http://IP:8066)
./traffic66 demo                              # simulated devices; the first start builds sample history (a few seconds, longer on slow disks), then opens the web UI
```

macOS: run `xattr -dr com.apple.quarantine .` in the folder first. PowerShell: use `.\traffic66.exe`.

## FAQ

**No data?** Check the device exports to this host and port, and the firewall allows that UDP port. NetFlow v9/IPFIX need a template first and long flows appear at the active timeout: set both to 60 s on the device. Right after start, expect a short wait: a device shows in **Settings** within seconds, charts trail by about a minute, and records from a device that has not sent its sampling rate are held up to 5 minutes (`-sampling-wait`), then counted as 1:1.

**Interface name?** `./traffic66 interfaces`

**Forgot the password?** Stop traffic66 and run `./traffic66 passwd` (add `-data DIR` if used).

## Manual

Web UI: **Help** in the menu. GitHub: [MANUAL.md](docs/MANUAL.md)

## Licence

- **Free**: evaluation, testing and demos for any organization, without time limit; production use by organizations under 100 people (employees plus contractors) and noncommercial use. Larger organizations: 30 days of production use, then registration. Details: [LICENSE.md](LICENSE.md), [Additional Use Grant](ADDITIONAL-USE-GRANT.md).
- **Unregistered**: reports and charts carry an EVALUATION COPY mark; up to 3 web accounts.
- **Registered**: no mark. Contact route66traffic66@gmail.com.
