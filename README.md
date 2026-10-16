**English** | [中文](docs/README.zh.md) | [हिन्दी](docs/README.hi.md) | [Español](docs/README.es.md) | [العربية](docs/README.ar.md) | [Français](docs/README.fr.md) | [বাংলা](docs/README.bn.md) | [Português](docs/README.pt.md) | [Русский](docs/README.ru.md) | [Bahasa Indonesia](docs/README.id.md) | [اردو](docs/README.ur.md) | [Deutsch](docs/README.de.md) | [日本語](docs/README.ja.md) | [Tiếng Việt](docs/README.vi.md) | [한국어](docs/README.ko.md)

# traffic66 — NetFlow, sFlow and IPFIX collector and traffic analyzer

Who uses the bandwidth, where the traffic goes, and whether the numbers match the devices' own interface counters. One program, web UI and terminal UI.

## Quick start

Download: [Releases](https://github.com/githubflyideas/traffic66/releases/latest)  
Run: `traffic66.exe` (Windows) or `./traffic66` (Linux, macOS)  
Open: `http://<this computer's IP>:8066` (on the same computer: http://localhost:8066)

**Login: user `admin`, password `traffic66` (you set a new password at first sign-in)**

Devices send to this computer's IP: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739.

### Windows

1. Download `traffic66-windows-amd64.zip` and unzip it, for example to `C:\traffic66`.
2. To capture this PC's own traffic, install [Npcap](https://npcap.com/#download) first (keep the defaults). Not needed for sFlow/NetFlow/IPFIX.
3. Double-click `traffic66.exe` to start with the defaults (no options). For any command with options, open a terminal:
   - Normal: open the folder, type `cmd` in the address bar and press Enter.
   - Local capture needs an admin terminal: right-click Start > **Terminal (Admin)**, then `cd C:\traffic66`.
4. Run `traffic66.exe` (in PowerShell: `.\traffic66.exe`).
5. Open http://localhost:8066 and sign in.

### Linux

1. Download and unzip:
   ```
   tar xzf traffic66-linux-amd64.tar.gz
   cd traffic66-linux-amd64
   ```
   (ARM: `traffic66-linux-arm64.tar.gz`)
2. Run `./traffic66` (local capture needs `sudo`).
3. Open `http://<server IP>:8066` and sign in.

### macOS

1. Download `traffic66-darwin-arm64.tar.gz` (Apple silicon) or `traffic66-darwin-amd64.tar.gz` (Intel), then:
   ```
   tar xzf traffic66-darwin-arm64.tar.gz
   cd traffic66-darwin-arm64
   xattr -dr com.apple.quarantine .
   ```
2. Run `./traffic66` (local capture needs `sudo`).
3. Open http://localhost:8066 and sign in.

## Screenshots

<img src="docs/images/overview.png" alt="Overview" width="49%"> <img src="docs/images/topn.png" alt="Top 66" width="49%">

<img src="docs/images/geo.png" alt="Geo & networks" width="49%"> <img src="docs/images/interfaces.png" alt="Interface check" width="49%">

<img src="docs/images/tui-top.png" alt="TUI" width="49%">

## What do you want to do?

Capture this computer's traffic (Windows, admin terminal, Npcap installed):  
`traffic66.exe -capture Wi-Fi`  
Use the card name or number from `traffic66.exe interfaces`, e.g. `-capture Ethernet`.

Capture this computer's traffic (Linux / macOS):  
`sudo ./traffic66 -capture eth0`  
On macOS usually `en0`. Names come from `./traffic66 interfaces`.

Receive sFlow from a switch:  
`./traffic66`  
Nothing to add: configure the switch to send sFlow to this computer's IP, UDP 6343.

Receive NetFlow or IPFIX from a router:  
`./traffic66`  
Configure the router to export to this computer's IP, UDP 2055 (NetFlow) or 4739 (IPFIX).

Use another collector port:  
`./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
Here NetFlow is received on 9995. Every port accepts every protocol.

Change the web port:  
`./traffic66 -addr :9000`  
Then open `http://<IP>:9000`.

Bind to one IP address only:  
`./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
Web UI and collectors listen only on 192.168.1.10.

Set or change the admin password:  
`./traffic66 passwd`  
Prompts for a new password (Windows: `traffic66.exe passwd`).

Keep data in another directory:  
`./traffic66 -data /var/lib/traffic66`  
Default: `traffic66-data` next to the program.

Terminal UI:  
`./traffic66 tui`  
Run on the traffic66 machine; for a remote one add `-server http://192.168.1.10:8066`.

Try without devices:  
`./traffic66 demo`  
Starts with simulated devices.

On Windows, write `traffic66.exe` instead of `./traffic66` (PowerShell: `.\traffic66.exe`).

## FAQ

**No data showing.** Check that the device exports to this computer's IP and the right UDP port (6343 / 2055 / 4739), that the firewall allows those UDP ports inbound, and wait a minute: NetFlow/IPFIX devices often send records only after flows end or time out.
NetFlow v9 and IPFIX need the device's template before records can be decoded, and long flows are exported only at the active timeout: set template refresh and active timeout to 60 s on the device. NetFlow v5, sFlow and local capture show up immediately.

**Can't find the interface name.** Run `traffic66.exe interfaces` (Windows) or `./traffic66 interfaces` (Linux, macOS); it lists the names and numbers `-capture` accepts.

**Forgot the password.** Stop traffic66 and run `./traffic66 passwd` (add `-data DIR` if you use one) to set a new one.

## Full manual

In the web UI: **Help** in the menu. On GitHub: [MANUAL.md](docs/MANUAL.md)

Free for evaluation and for organizations under 100 people ([licence](LICENSE.md)).
