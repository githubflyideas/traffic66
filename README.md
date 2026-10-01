**English** | [中文](docs/README.zh.md) | [हिन्दी](docs/README.hi.md) | [Español](docs/README.es.md) | [العربية](docs/README.ar.md) | [Français](docs/README.fr.md) | [বাংলা](docs/README.bn.md) | [Português](docs/README.pt.md) | [Русский](docs/README.ru.md) | [Bahasa Indonesia](docs/README.id.md) | [اردو](docs/README.ur.md) | [日本語](docs/README.ja.md) | [한국어](docs/README.ko.md)

# traffic66

Flow analytics for sFlow, NetFlow and IPFIX in a single program. It
collects flow exports from switches, routers and firewalls, stores them in
an embedded database, and shows who uses the bandwidth, where the traffic
goes and whether the numbers match the devices' own interface counters —
in a web UI and in a terminal UI.

- One executable for Windows, Linux and macOS. No database to install, no
  runtime, works offline.
- sFlow v5, NetFlow v5, NetFlow v9 and IPFIX on any UDP port; optional local
  capture from a network interface or mirror port.
- Checks its own numbers against interface counters (sFlow counters or
  SNMP) and says why they differ when they do.
- Top 66 lists, flow paths, countries and networks, threat list matches,
  flow records, encapsulation (GRE, IPIP, VXLAN, GENEVE, MPLS).
- 13 languages in the web UI and the terminal UI.

## Contents

1. [Try the demo](#1-try-the-demo)
2. [Install](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Sign in and passwords](#3-sign-in-and-passwords)
4. [Send flows from your devices](#4-send-flows-from-your-devices)
5. [Check that flows arrive](#5-check-that-flows-arrive)
6. [Make the numbers match the interface counters](#6-make-the-numbers-match-the-interface-counters)
7. [Names, SNMP and your own networks](#7-names-snmp-and-your-own-networks)
8. [Countries, networks and threat lists](#8-countries-networks-and-threat-lists)
9. [Using the web UI](#9-using-the-web-ui)
10. [Terminal UI](#10-terminal-ui)
11. [Local capture](#11-local-capture)
12. [Options](#12-options)
13. [Data, backup, upgrade, uninstall](#13-data-backup-upgrade-uninstall)
14. [Security](#14-security)
15. [Sizing](#15-sizing)
16. [Troubleshooting](#16-troubleshooting)
17. [Build from source](#17-build-from-source)

## 1. Try the demo

Download the archive for your system from the
[releases page](https://github.com/githubflyideas/traffic66/releases):

| System | Archive |
|---|---|
| Windows 10/11, Server 2016 or later (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64 | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64 | `traffic66-linux-arm64.tar.gz` |
| macOS Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (the second line lets macOS run a program downloaded from the
internet that is not from the App Store):

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

Open http://127.0.0.1:8066 and sign in as `admin` / `try66`. The demo builds
a small company network with a day of history and live traffic from four
simulated devices, including two incidents to find: start on **Overview**,
look at **Who grew**, and click your way from there. Stop it with Ctrl+C.
Demo data is kept in `traffic66-demo` next to the program; delete that
folder to start the demo afresh.

The demo uses the same ports as a real installation (8066, and UDP 6343,
2055, 4739). To run it next to a real one, give it other ports:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

## 2. Install

traffic66 is a single file. Installing means putting it somewhere, choosing
a data directory, setting a password, opening the firewall and starting it
at boot. The examples use `192.0.2.50` for the traffic66 machine and
`192.0.2.1` for a router; replace them with your addresses.

Ports:

| Port | Used for |
|---|---|
| UDP 6343 | sFlow (default) |
| UDP 2055 | NetFlow (default) |
| UDP 4739 | IPFIX (default) |
| TCP 8066 | web UI and API |

Every UDP port accepts every protocol, so a device can send NetFlow to 6343
if that is easier. Change or add ports with `-listen`.

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

The last command asks for the password of the user `admin`.

Create `/etc/systemd/system/traffic66.service`:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target
Wants=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
# only needed for local capture (-capture):
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

Start it and allow larger UDP buffers so bursts are not lost:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, with firewalld (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

or with ufw (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

### Windows

Unpack to `C:\traffic66` and set the password (PowerShell as
Administrator):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Data goes to `C:\traffic66\traffic66-data`, next to the program.

Open the firewall:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

To try it in the foreground, run `C:\traffic66\traffic66.exe` and stop it
with Ctrl+C. To run it in the background from boot, without anyone signed
in, register it as a startup task:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` matters: without it Windows stops
the task after three days. Stop it with `Stop-ScheduledTask -TaskName
traffic66`, remove it with `Unregister-ScheduledTask -TaskName traffic66`.

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Create `/Library/LaunchDaemons/traffic66.plist`:

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

Start it, and stop it again:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

If the macOS firewall is on, allow incoming connections for traffic66 in
System Settings → Network → Firewall → Options.

## 3. Sign in and passwords

Open `http://<traffic66 machine>:8066` and sign in. The user is `admin`
unless you chose another.

- If you did not set a password before the first start, traffic66 makes one
  and prints it once in its log:
  `first start: sign in as user "admin" with password "…"`.
  On Linux find it with `journalctl -u traffic66 | grep "first start"`.
- The password is kept, hashed, in the file `password` in the data
  directory. It stays the same across restarts.
- Change it, or after forgetting it set a new one, on the traffic66 machine:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` makes a random one and prints it. A running
  traffic66 accepts the new password at the next sign-in; no restart is
  needed.
- More users: `traffic66 passwd -data <data directory> -user alice`. All
  users see the same thing.
- For scripts and containers, `TRAFFIC66_PASSWORD=…` in the environment or
  `-password …` on the command line sets the password for that run instead
  of the stored one. Prefer the environment: command lines are visible to
  other users of the machine.

After five wrong passwords within a minute an address is blocked for a
minute.

## 4. Send flows from your devices

Point each device at the traffic66 machine. Commands differ between models
and software versions; check your device's manual. In all examples,
`192.0.2.50` is traffic66 and `192.0.2.1` the device's own address.

General advice:

- Set the active flow timeout to 60 seconds. Longer timeouts make traffic
  arrive in late, large lumps.
- If the device samples NetFlow/IPFIX, let it export its sampler options so
  the rate is known. traffic66 holds records until the rate arrives rather
  than counting them 1:1.
- Sample either all interfaces or only the edge interfaces, in one
  direction. Sampling the same traffic on the way in and on the way out
  counts it twice; **Interface check** points this out.
- sFlow sampling rate: about 1:1000 for 1 Gb/s links, 1:4096 for 10 Gb/s,
  1:8192 for 40/100 Gb/s.

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

FortiGate FortiOS 7.4.2 or later (NetFlow v9):

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

Linux servers and hosts, with softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

## 5. Check that flows arrive

Open **Sources**. Each device that sends anything appears within seconds,
with its protocol, sampling rate, loss, last packet and a status. When the
status is not green, the text next to it says what is wrong and what to
change.

If a device does not appear:

1. Watch for packets on the traffic66 machine (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Nothing there means the packets do not reach the machine: check the
   device configuration, routing and firewalls on the way.
2. Packets arrive but **Sources** stays empty: the local firewall drops
   them (see [Install](#2-install)), or traffic66 listens on other ports
   (`-listen`).
3. To test the path from another machine without touching a device, run
   `traffic66 simulate -to 192.0.2.50` there for a few seconds. It sends
   sFlow, NetFlow and IPFIX from simulated devices; they then appear in
   **Sources** and in the data, so prefer a test installation for this.

## 6. Make the numbers match the interface counters

Flow numbers are estimates: sampled packets times the sampling rate.
traffic66 compares them with the device's own interface counters and shows
the difference on **Interface check**, with the likely cause when it is
larger than sampling alone explains.

To get counters to compare with:

- sFlow devices send them on their own when a counter interval is set
  (`sflow counter interval 30` and similar).
- For NetFlow and IPFIX devices, add an `snmp` line in **Sources → Names**
  (see [Names](#7-names-snmp-and-your-own-networks)). traffic66 then reads
  the interface counters every minute.

What traffic66 already does so the numbers match: it uses the sampling
rate the device actually applied, holds NetFlow/IPFIX records until the
sampling rate is known, compensates export packets lost on the way, spreads
long flows over the minutes they lasted, and adds 18 bytes per packet of
Ethernet overhead to NetFlow/IPFIX byte counts (interface counters include
it, IP-layer flow counts do not; change with `-l2-overhead`).

Common reasons for a remaining difference, all reported on **Interface
check**: some interfaces are not sampled, the same traffic is sampled on
two interfaces, export packets are lost before they reach traffic66, or
the sampling rate is not known yet.

## 7. Names, SNMP and your own networks

**Sources → Names** in the web UI takes one entry per line. It is saved as
`inventory.txt` in the data directory, so you can also edit that file (see
`inventory.txt.example`). Every line is optional.

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

- `net`: private ranges (10/8, 172.16/12, 192.168/16, 100.64/10) are
  always yours. Add your public ranges so traffic to and from them counts
  as yours too; the name shows up in **Top-N → Segment** and in the flow
  paths.
- `snmp <device> <community> [<management address>[:port]]`: the device
  is the address flows come from. Add the management address when the
  device answers SNMP on another address. Interface descriptions read over
  SNMP are used as names unless you name the interface with `iface`.
  Allow the traffic66 machine in the device's SNMP access list.
- Changes apply when you click **Save**; no restart is needed.

## 8. Countries, networks and threat lists

Countries and network (AS) names need an IP-to-ASN table. Download the free
one from [iptoasn.com](https://iptoasn.com):

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

Any file in the same format works (tab separated: first address, last
address, AS number, country code, AS name; plain or gzip). Restart
traffic66 after replacing it; download a new one every month or so.

Threat lists are plain text files with one address or network per line
(text after `#` or `;` is ignored), saved as `<data directory>/threats/<name>.txt`, for
example:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Restart traffic66 after adding or changing lists. Matches appear on
**Threat intel**, by list name.

## 9. Using the web UI

You should rarely need to type. Every value on every page — an address, a
port, an application, a country, a device — can be clicked:

- **Show only this** / **Exclude this** adds a filter. Filters appear
  under the top bar and apply to every page until you remove them.
- **Show its flow records** opens the matching individual flows.
- **Look it up online** opens the address or AS in a public lookup site.
- **Copy** copies the value.

Pages:

| Page | What it answers |
|---|---|
| Overview | How much traffic now and compared with last week, by application; what grew; top clients and services |
| Top-N | The top 66 of clients, servers, conversations, applications, ports, countries, networks, segments, devices, encapsulation or VLAN |
| Flow paths | Which segment talks to which application in which country |
| Geo & networks | Traffic by country and by network (AS) |
| Threat intel | Hosts that talked to addresses on your threat lists, and how much they sent |
| Flow records | Individual flows, newest first, with selectable columns |
| Interface check | Flow numbers next to the interface counters, worst first, with reasons |
| Sources | Devices, sampling, loss, collectors, SNMP, and **Names** |

Above the pages: time range (15 minutes to 30 days), an optional search
box, automatic refresh every 30 seconds, and **Copy link**, which copies a
link to exactly the current view (page, time range and filters) to send to
a colleague. The language follows the browser; change it at the bottom of
the menu.

Top-N over long time ranges comes from hourly summaries; filters are not
available there, and the page says so. Choose a shorter range to filter.

## 10. Terminal UI

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

On the traffic66 machine, `traffic66 tui` signs in by itself when it can
read the data directory (give `-data` when it is not the default). If
traffic66 runs as another user, as a service does, use `-user` and
`-password` instead. `-lang` picks the language (`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

Keys: 1–8 pages, ↑↓ select, Enter actions on the selected value, f show
only, x exclude, / search, t time range, c clear filters, w open the same
view in a browser, q quit.

## 11. Local capture

Besides flow exports, traffic66 can make flows itself from packets on a
local network interface, for example a mirror (SPAN) port:

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: needs root, or the capabilities `CAP_NET_RAW` and `CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`, or
  the `AmbientCapabilities` line in the systemd unit above).
- macOS: needs root (BPF devices); nothing to install.
- Windows: install [Npcap](https://npcap.com) first.

Captured interfaces are listed on **Sources**. Packets seen
twice (for example on two mirror ports) are counted twice.

## 12. Options

`traffic66 -h` and `traffic66 <command> -h` list everything.

Commands:

| Command | |
|---|---|
| `traffic66` | collect flows and serve the web UI |
| `traffic66 demo` | the same, with a simulated network |
| `traffic66 tui` | terminal UI for a running traffic66 |
| `traffic66 passwd` | set a login password |
| `traffic66 simulate -to HOST` | send simulated exports to a collector |
| `traffic66 interfaces` | list interfaces for local capture |
| `traffic66 version` | print the version |

Options of `traffic66` and `traffic66 demo`:

| Option | Default | |
|---|---|---|
| `-addr` | `:8066` | web UI address; `127.0.0.1:8066` for this machine only |
| `-data` | `traffic66-data` next to the program | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors as `name=address`, comma separated; empty disables |
| `-user` | `admin` | user for the first generated password and for `-password` |
| `-password` | stored password | password for this run only (also `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | days of flow detail kept; summaries are kept 400 days |
| `-memory` | `0.10` | share of physical memory the database may use |
| `-l2-overhead` | `18` | bytes per packet added to NetFlow/IPFIX byte counts |
| `-sampling-wait` | `5m` | how long records wait for a sampling rate |
| `-capture` | | capture on a local interface (repeatable) |
| `-inventory` | `<data>/inventory.txt` | names file |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table |
| `-threat` | `<data>/threats/*.txt` | extra threat list as `name=path` (repeatable) |
| `-dns-upstream` | system resolver | DNS server for showing host names |
| `-dns-rate` | `20` | reverse lookups per second at most |
| `-dns-cache` | `2m` | how long host names are cached |
| `-no-dns` | | no reverse lookups |
| `-tui` | | also open the terminal UI |

Example: a second collector port, a year of detail, and the web UI only on
the local machine:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

## 13. Data, backup, upgrade, uninstall

The data directory holds everything:

| | |
|---|---|
| `raw/` | flow detail, one compressed file per hour |
| `traffic66.duckdb` | summaries, interface counters and the current hour |
| `password` | login passwords (hashed) |
| `inventory.txt` | names (**Sources → Names**) |
| `asn.tsv.gz`, `threats/` | lookup tables you added |

- **Backup**: stop traffic66 and copy the directory. Without stopping,
  copy `raw/`, `password` and `inventory.txt`; the current hour and the
  summaries are then missing.
- **Move**: stop traffic66, move the directory, start with `-data` pointing
  to the new place.
- **Upgrade**: stop traffic66, replace the program file, start it again.
  The data stays. On Linux, for example:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Uninstall**: stop and remove the service or startup task (see
  [Install](#2-install)), then delete the program folder and the data
  directory.

## 14. Security

- The web UI uses plain HTTP: passwords and data cross the network
  unencrypted. On networks you do not fully trust, listen on this machine
  only (`-addr 127.0.0.1:8066`) and put a TLS reverse proxy in front, for
  example with [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  Or reach it over a VPN or SSH tunnel:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, then open
  http://127.0.0.1:8066.
- Allow the UDP collector ports only from your devices' addresses.
- SNMP communities in `inventory.txt` are stored in clear text; use a
  read-only community.

## 15. Sizing

Measured at 5,000 flows per second on a 2-core machine: detail uses about
12 GB of disk per day plus about 1.5 GB for the current hour, the program
about 0.5 GB of memory and a sixth of one core. Overviews of long time
ranges come from summaries and take under 0.2 s. Queries over detail scan
about 22 million rows per hour: one host over 1 hour takes under 1 s, a
1-hour Top 66 of all conversations about 9 s; the time grows with the range
and shrinks with more cores.

Disk for 30 days at 5,000 flows/s is therefore about 360 GB; scale it with
your flow rate (shown on **Sources**) and `-retention-days`.

## 16. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Device missing from **Sources** | Packets do not arrive: see [Check that flows arrive](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | The device has not sent its sampler options yet; most resend within minutes. If it never does, export them (`option sampler-table` on Cisco) or mark it `unsampled` in Names if it really is 1:1 |
| Numbers lower than the interface counters | See **Interface check**: loss on the way, interfaces not sampled, or flows still in the device cache (active timeout longer than 60 s) |
| Numbers higher than the interface counters | The same traffic sampled on two interfaces or two devices |
| No countries or networks | No IP-to-ASN table: see [Countries](#8-countries-networks-and-threat-lists) |
| Forgot the password | `traffic66 passwd -data <data directory>` on the traffic66 machine |
| `Conflicting lock is held` | Another traffic66 already uses this data directory |
| `receive buffer is only … KB` | Linux limits UDP buffers: set `net.core.rmem_max=16777216` (see [Linux](#linux)) |
| `cannot create the data directory` | The program folder is not writable for this user: give `-data` |
| macOS: "cannot be opened" or "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows capture: Npcap not found | Install [Npcap](https://npcap.com) |
| `address already in use` | Another program uses the port: choose others with `-addr` or `-listen` |

## 17. Build from source

Go 1.24 and a C compiler (gcc or clang; MinGW-w64 on Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
