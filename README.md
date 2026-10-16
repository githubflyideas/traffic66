**English** | [中文](docs/README.zh.md) | [हिन्दी](docs/README.hi.md) | [Español](docs/README.es.md) | [العربية](docs/README.ar.md) | [Français](docs/README.fr.md) | [বাংলা](docs/README.bn.md) | [Português](docs/README.pt.md) | [Русский](docs/README.ru.md) | [Bahasa Indonesia](docs/README.id.md) | [اردو](docs/README.ur.md) | [Deutsch](docs/README.de.md) | [日本語](docs/README.ja.md) | [Tiếng Việt](docs/README.vi.md) | [한국어](docs/README.ko.md)

# traffic66 — NetFlow, sFlow and IPFIX collector and traffic analyzer

Flow analytics for sFlow, NetFlow and IPFIX in a single program: who uses
the bandwidth, where the traffic goes, and whether the numbers match the
devices' own interface counters, in a web UI and a terminal UI.

## Quick start

```
# Linux, macOS: download into ./traffic66 and start collecting
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: the same
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

The terminal prints the web UI address and the IP and UDP ports to point your devices at ([device configuration](docs/MANUAL.md#device-configuration)). Sign in as `admin` / `traffic66` and choose a new password. Only looking? The demo, with simulated devices:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Ports: TCP 8066 web UI, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (every UDP port takes every protocol). The terminal prints the addresses to point your devices at; the first sign-in asks for a new password.

macOS by hand: `xattr -dr com.apple.quarantine traffic66-darwin-*` first. Windows "protected your PC": **More info** → **Run anyway**.

## Command cheat sheet

| Task | Command |
|---|---|
| Collect sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (default) | `./traffic66` |
| NetFlow v5/v9 on 2055 and IPFIX on 4739 only | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| sFlow only, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Web UI on a given IP and port | `./traffic66 -addr 10.0.0.5:8080` |
| Capture this machine's NIC (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Set or reset the admin password (add -data DIR if used) | `./traffic66 passwd` |
| Terminal UI to a remote server | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Overview: open findings, bandwidth by application compared with the same time yesterday, top clients and services](docs/images/overview.png)

## Full manual

In the web UI: **Help** in the menu. On GitHub: [MANUAL.md](docs/MANUAL.md)

15 languages. Free for evaluation and for organizations under 100 people ([licence](LICENSE.md)).
