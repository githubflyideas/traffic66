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
- Finds scans, password guessing, lateral movement, unusual uploads, floods
  and threat list traffic in the flows, also through sampling, and lists
  them as findings to deal with.
- Top 66 lists, who talks to whom as ring charts (servers and their
  clients, services and their servers), traffic over time by interface and
  network (AS), flow paths, countries on a world map, threat list matches,
  flow records, encapsulation (GRE, IPIP, VXLAN, GENEVE, MPLS).
- `traffic66 capture.pcap` opens up to 3 packet captures (3 GB in all) in the web UI: flows, findings, countries and flow records over the whole capture, with nothing to set up.
- 13 languages in the web UI and the terminal UI.
- Free to try for 30 days with every feature; it keeps working after that
  (see [Trial and licence](#trial-and-licence)).

![Overview: open findings, bandwidth by application compared with last week, top clients and services](docs/images/overview.png)

<sub>All screenshots come from `traffic66 demo`, a simulated company network that you can run yourself (see [Try the demo](#1-try-the-demo)).</sub>

## Contents

1. [Try the demo](#1-try-the-demo)
2. [Install](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Users and passwords](#3-users-and-passwords)
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
| Linux x86-64: any distribution with kernel 3.2 or later, including CentOS 7 and Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: same distributions | `traffic66-linux-arm64.tar.gz` |
| macOS 11 or later, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 or later, Intel | `traffic66-darwin-amd64.tar.gz` |

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
simulated devices, including an attack: **Findings** shows each step of it
(a scan, a port scan, password guessing, lateral movement, an upload to a
control server) and a flood on the public website. Click **Details** on a
finding, or start on **Overview**, click a host in **Top clients**, choose
**Show details**, and keep clicking from there. Stop it with Ctrl+C.
Demo data is kept in `traffic66-demo` next to the program; delete that
folder to start the demo afresh.

The demo uses the same ports as a real installation (8066, and UDP 6343,
2055, 4739). To run it next to a real one, give it other ports:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

On Windows you can also simply double-click `traffic66.exe`. That starts
traffic66 for real (not the demo) and opens the web UI in your browser; the
first-start password is shown in the black window, and closing the window
stops traffic66. If Windows says "Windows protected your PC", click
**More info** → **Run anyway**.

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
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
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

## 3. Users and passwords

**In short:** users and passwords live in one file, `password`, in the data
directory. You never edit it by hand: the `traffic66 passwd` command adds,
changes, lists and deletes users. Open `http://<traffic66 machine>:8066`
and sign in with one of them.

### The first sign-in

On its first start traffic66 creates the user `admin` with a random
password and shows it once:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Started by double-click on Windows: in the black window.
- In a terminal: in the terminal.
- Linux service: `journalctl -u traffic66 | grep "first start"`
- macOS service: `grep "first start" /Library/Logs/traffic66.log`

Missed it? Set a new one with `traffic66 passwd` (below). If you set a
password with `traffic66 passwd` before the first start, as the install
steps above do, nothing is generated.

### Where the users are stored

The file `password` in the data directory:

| How traffic66 runs | File |
|---|---|
| Unpacked and started from its folder (default) | `traffic66-data/password` next to the program |
| Linux service (section 2) | `/var/lib/traffic66/password` |
| Windows startup task (section 2) | `C:\traffic66\traffic66-data\password` |
| macOS service (section 2) | `/Library/Application Support/traffic66/password` |
| Demo | `traffic66-demo/password` next to the program |

One line per user. Passwords are stored as salted hashes, so nobody can
read them back from the file, not even you; if a password is forgotten, set
a new one. The file is readable by its owner only.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

### Managing users

Run these on the traffic66 machine:

| To | Command |
|---|---|
| Change the password of `admin` | `traffic66 passwd` |
| Add the user `alice`, or change her password | `traffic66 passwd -user alice` |
| Delete the user `alice` | `traffic66 passwd -user alice -delete` |
| List the users | `traffic66 passwd -list` |
| Set a random password and print it | `traffic66 passwd -generate` (with `-user` for other users) |

- The command asks for the new password twice and does not show what you
  type. Use at least 8 characters.
- When traffic66 runs with `-data`, add the same `-data` to the command.
  For the Linux service from section 2:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  On Windows (PowerShell as Administrator):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- Changes apply immediately, without a restart: a new password works at the
  next sign-in, and a deleted user is signed out of open browsers.
- The last remaining user cannot be deleted; add another one first.
- All users see and can change the same things; there are no roles.

### Passwords for scripts and containers

`TRAFFIC66_PASSWORD=…` in the environment, or `-password …` on the command
line, makes traffic66 accept exactly one user for that run: the one named by
`-user` (default `admin`) with that password. The `password` file is then
ignored and not changed. Prefer the environment variable: command lines are
visible to other users of the machine.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

After five wrong passwords within a minute, an address is blocked for a
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

![Sources: each device with protocol, sampling, loss and what to fix](docs/images/sources.png)

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
larger than sampling alone explains. Each interface has one chart with
ingress (green) and egress (blue); picking an interface in the list shows
its charts.

![Interface check: traffic of every interface, and the flow estimate next to the device counter](docs/images/interfaces.png)

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

The quickest way to name a host or a device: click its address on any page
and choose **Name it…**. Type the name and press Enter; it is saved at
once and shown everywhere instead of the bare address.

For networks, interfaces and SNMP, use **Sources → Names**: choose the type
(host, network, device, interface, SNMP), fill in the address and the name,
and click **Add**. The table below lists every name with **Edit** and
**Delete**; adding the same address again replaces the old entry. Addresses
and networks are checked before saving.

The names are saved as `inventory.txt` in the data directory, one entry per
line. **Edit as text (advanced)** shows that file, and you can also edit it
directly (see `inventory.txt.example`). Every line is optional.

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers country=JP

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
  as yours too; the name shows up in **Top 66** grouped by segment and in
  the flow paths by network. `country=JP` (a two-letter country code)
  says where the network is; the world map then draws lines from it to the
  countries it talks to.
- `snmp <device> <community> [<management address>[:port]]`: the device
  is the address flows come from. Add the management address when the
  device answers SNMP on another address. Interface descriptions read over
  SNMP are used as names unless you name the interface with `iface`.
  Allow the traffic66 machine in the device's SNMP access list.
- Changes apply when you click **Save**; no restart is needed.

## 8. Countries, networks and threat lists

Countries and networks (AS) work out of the box: traffic66 has DB-IP's free
**IP to Country Lite** and **IP to ASN Lite** databases built in (licensed
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/); "IP Geolocation
by DB-IP", [db-ip.com](https://db-ip.com)). The pages that show countries
and networks name the data's source.

The built-in copy is from the release you run. DB-IP publishes a new one
every month; **Sources → Countries and networks database → Update DB-IP
Lite now** downloads the latest from db-ip.com (the server running
traffic66 needs internet access for this; the web UI says so if it fails).

You can also use another free database. Download it, then upload it on the
same page with **Upload a database file…**. It is checked, saved in the data
directory and used for new traffic at once; no restart is needed. Traffic
already stored keeps the country it was saved with.

| Database | Gives | Licence | Where to get it |
|---|---|---|---|
| DB-IP Lite (built in) | countries; networks | CC BY 4.0, no account | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country and ASN, `.mmdb` | countries; networks | GeoLite2 EULA, free account | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite, `ipinfo_lite.mmdb` | countries and networks in one file | CC BY-SA 4.0, free account | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN, `ip2asn-combined.tsv.gz` | networks with their country | PDDL 1.0, no account | [iptoasn.com](https://iptoasn.com) |

Your own files are used first; the built-in DB-IP Lite answers what they
do not cover. **Remove** next to a file goes back to the rest. The page
lists what is in use and the date of each database.

Without the web UI, copy the file into the data directory as
`country.mmdb`, `asn.mmdb`, `both.mmdb` (a file with countries and
networks, such as IPinfo Lite) or `asn.tsv.gz` and restart traffic66.

**Geo & networks** shows the traffic to and from other countries on a
world map: the darker a country, the more traffic. Point at a country for
its traffic; click it to filter or open its flow records. When your
networks have a country (`country=` on a `net` line, see
[Names](#7-names-snmp-and-your-own-networks)), lines run from that country
to the countries they exchange traffic with, thicker for more traffic. Country outlines
come from [Natural Earth](https://www.naturalearthdata.com) (public domain).

![Geo & networks: remote traffic by country on a world map](docs/images/geo.png)

Threat lists are plain text files with one address or network per line
(text after `#` or `;` is ignored), saved as `<data directory>/threats/<name>.txt`, for
example:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Restart traffic66 after adding or changing lists. Matches appear on
**Threat intel**, by list name.

![Threat intel: an internal host sending data to an address on a threat list](docs/images/threats.png)

## 9. Using the web UI

You should rarely need to type. Every value on every page — an address, a
port, an application, a country, a device — can be clicked:

- **Show only this** / **Exclude this** adds a filter. Filters appear
  under the top bar and apply to every page until you remove them.
- **Show its flow records** opens the matching individual flows.
- **Show details** (hosts, devices and services) opens a page about that
  one host or service: its traffic over time by application, who it talks
  to, which services or clients, countries and its latest flows. Every
  value there can be clicked again, so you can keep drilling down; the
  browser's Back button returns.
- **Name it…** (hosts and devices) gives the address a name, shown
  everywhere from then on.
- **Look it up online** opens the address or AS in a public lookup site.
- **Copy** copies the value.

Pages:

| Page | What it answers |
|---|---|
| Overview | How much traffic now and compared with last week, by application; open findings; direction and protocol; top clients and services |
| Top 66 | Opens on **Talkers**: the top 30 clients and servers side by side with traffic, packets and flow records, above a row for all traffic. **Table** is one table of the top 66: by default conversations (client, server, service, country). Every column heading sorts; number columns (traffic, packets, average packet size, flows) rank all traffic in the range, so the smallest average packet size finds scanners and floods. **Group by** switches to applications, networks, segments, devices, encapsulation and VLAN |
| Traffic details | Two ring charts. **Servers and clients**: the inner ring is the 8 busiest servers, the outer ring the clients of each; **Clients inside** turns it round (clients inside, the servers each uses outside), since one side often explains more than the other. **Services and servers**: services inside and the servers offering them outside, or the other way round. Point at a segment for its traffic; click it like any value |
| Flow paths | Which host uses which application towards which country: the 8 busiest hosts, the rest as Other. **Client → server** shows client → service → server; **By network** shows networks instead of hosts. Long names are shortened to 22 characters; point at one for the full name |
| Findings | What needs attention: scans, password guessing, lateral movement, unusual uploads, floods and threat list traffic ([more](#findings)) |
| Threat intel | Hosts that talked to addresses on your threat lists, and how much they sent |
| Geo & networks | A world map of traffic by country, with lines from your networks; the networks (AS) traffic came from and went to, over time in bits/s and packets/s; traffic by country and by network |
| Sources | Devices, sampling, loss, collectors, SNMP, the countries and networks database, the logo, and **Names** |
| Interface check | Traffic of every interface over time, ingress (green) and egress (blue) in one chart, in bits/s and packets/s, and flow numbers next to the interface counters, worst first, with reasons |
| Flow records | How many flow records there were and when (a bar per interval), and the records themselves, newest first, page by page, with selectable columns |
| Data cleanup | Deletes data older than 120, 90, 60, 30 or 7 days, or all of it, with how much each frees ([more](#13-data-backup-upgrade-uninstall)) |
| Offline pcap analysis | Packet captures (pcap, pcapng) analysed apart from the live data ([more](#offline-pcap-analysis)) |

The side menu lists the pages in four groups: traffic (Overview, Top 66,
Traffic details, Flow paths), security (Findings, Threat intel, Geo &
networks), setup and data (Sources, Interface check, Flow records, Data
cleanup) and Offline pcap analysis. Under the logo are the version and the
server's date and time.

Above the pages: time range (15 minutes to 30 days, or **Custom…** for any
start and end, also further back than 30 days), automatic refresh every 30
seconds, and **Copy link**, which copies a link to exactly the current view
(page, time range and filters) to send to a colleague. On **Top 66** and
**Traffic details**, a search box and **Device**, **Client**, **Server**
and **Service** list the busiest values of the time range: pick one, or
type one, to filter; the filter then applies to every page until you empty
the box. The language follows the browser; change it at the bottom of the
menu, above **Log out**.

Charts over time show the 8 largest values in fixed colours and the rest as
Other; the legend gives each value's total and can be clicked like any
other value. Charts of clients and servers leave the rest out of the
drawing, since with thousands of hosts it would flatten the top 8; the
legend still gives its total.

Ranges longer than 6 hours start on a whole hour, so every number on the
page counts exactly the same time: "24 hours" covers the last 24 whole
hours plus the current one. Top 66 over these ranges comes from hourly
summaries; filters are not available there, and the page says so. Choose a
shorter range to filter. Conversations always read the flow detail, so over long ranges
at high flow rates they can take a while; one hour is fastest.

The side menu shows how much disk the data uses and how much is free;
hover over the free space to see how much the kept days of detail need at
the current rate (estimated once there is a day of data).

To show your own logo on the sign-in page and at the top of the menu, use
**Sources → Logo → Upload a logo…**: PNG, SVG, JPEG, WebP or GIF, up to
1 MB, best at 272 × 92 pixels (other sizes are scaled to fit). **Use the
built-in logo** goes back to traffic66's.

### Findings

**Findings** lists what traffic66 found in the flows, most serious first. It
checks the last 10 minutes every 5 minutes; something that goes on for an
hour is one finding that grows, not a new one at every check.

| Finding | What it means | Severity |
|---|---|---|
| Scan | One address sent small probes to many addresses on one port (TCP or ping) | High from inside your network, low from the internet |
| Port scan | One address sent small probes to many ports of one host | High from inside, low from the internet |
| Password guessing | Many short connections to a login service (SSH, RDP, SMB, databases and others) | High from inside, low from the internet |
| Lateral movement | Inside your network, file sharing or remote administration sessions (SMB, RDP, SSH, WinRM, VNC) to hosts that never offered that service before | High |
| Unusual upload | An internal host sent much more than it received (100 MB in 10 minutes, three times what it received) to an address it had not exchanged data with before | High |
| Flood | 20,000 or more small packets per second to one address, ten times its usual rate | Medium |
| Threat list | Traffic with an address on one of your threat lists | High when your host connected to it, low when the listed address knocked from outside |

Each finding says who did what to whom, when and for how long, with the
numbers behind it and how the data was sampled. **Details** opens the
host's page, which also lists the findings about it. **Dealt with** closes
a finding; if it happens again, a new one opens. **Not a problem** closes it
for good: it is never reported again. The red number next to **Findings** in
the side menu counts the open high and medium findings of the last 24 hours.

Lateral movement and unusual uploads need to know what is normal, so they
are reported once there is a day of history. On first start traffic66
learns from the history it already has.

With sampled data (sFlow, sampled NetFlow) the rules count what the samples
show and ask for fewer of them, but then each must look like one short
probe, so busy normal hosts do not trigger them. What sampling hides cannot
be found: behind 1:4096 sampling, a scan of a few dozen hosts sends too few
packets to be seen. The demo's attack goes through a switch that samples
1:4096 and is found completely; a day of the demo's normal traffic produces
no findings except the internet scanner knocking on the website.

![Findings: every step of an attack, found through 1:4096 sFlow sampling](docs/images/findings.png)

![Top 66, Talkers: the top 30 clients and servers with a row for all traffic](docs/images/topn.png)

![Traffic details: servers with their clients, and services with their servers, as ring charts](docs/images/traffic.png)

![Details of one host: the findings about it, its traffic, who it talks to, services, countries and latest flows](docs/images/detail.png)

![Flow paths: which host uses which application towards which country](docs/images/paths.png)

The same overview in Chinese; every page is available in 13 languages:

![Overview in Chinese](docs/images/overview-zh.png)

### Offline pcap analysis

**Offline pcap analysis** looks at packet captures from Wireshark or tcpdump with the same pages as the live data, without mixing them in.

It summarizes all the packets into flows: who talked to whom, how much, when, and what looks like an attack. It does not decode protocols or show packet contents; for one packet or one TCP stream, use Wireshark.

From the command line, without setting anything up:

```
traffic66 office.pcap
traffic66 a.pcap b.pcapng c.pcap
```

traffic66 starts on this computer only (127.0.0.1, a free port), prints the address, the password and a one-time sign-in link, and opens the browser on the capture. Up to 3 files, 3 GB in all; they are read where they are and never changed. Nothing is collected or sent, and host names are not looked up (`-dns` turns that on). Ctrl+C stops and deletes the imported data. On a 2-core machine a 1 GB capture is ready in about 5 seconds (1.2 million full-size packets) to 30 seconds (14 million small packets).

```
$ traffic66 office.pcap

traffic66 0.3.1: analysing 1 capture file(s); nothing is collected or sent
  Web UI    http://127.0.0.1:38217  (port 38217, this computer only)
  Sign in   user admin, password gfhfhbuutz2e
  Open      http://127.0.0.1:38217/auto?t=b9388f…  (signs in once)
  Stop      Ctrl+C; the imported data is deleted, your files are kept
```

In the web UI of a running traffic66:

1. **Upload capture files…**: `.pcap` or `.pcapng`, not compressed. Up to 3 files, each at most 50 MB. The files are turned into flows in a database of their own (`<data>/sandbox/`); the live data, its numbers and findings are not touched.
2. **Analyse**: every page (overview, Top 66, traffic details, findings, flow paths, map, flow records) now shows the capture files over their whole time. An orange bar names the files; **Back to live data** returns. Each file appears as a device, so the **Device** box shows one file at a time.
3. The detection rules run over the capture: scans, port scans and password guessing are listed under **Findings**. Rules that need a day of history (lateral movement, unusual uploads) do not apply to a capture.
4. **Delete** removes a file and its data; **Delete all** removes everything.

The demo includes an example capture with an attack in it.

![Offline analysis: capture files with their packets, flows and time](docs/images/sandbox.png)

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

![Terminal UI: overview](docs/images/tui-overview.png)

![Terminal UI: Top 66 conversations](docs/images/tui-topn.png)

## 11. Local capture

Besides receiving flow exports, traffic66 can build flows itself from the
packets on a network interface of the machine it runs on. What it sees
depends on the interface:

| Interface | What traffic66 sees |
|---|---|
| A spare network port connected to a switch's mirror (SPAN) port | All traffic the switch mirrors: a whole network or uplink |
| The machine's own Ethernet or Wi-Fi | Only this machine's own traffic |

Wi-Fi adapters cannot see other devices' traffic. To see a whole Wi-Fi
network, let the router or access point export flows (section 4), or mirror
the switch port the access point is connected to.

### Windows

1. Install [Npcap](https://npcap.com) with its default options. If you tick
   "Restrict Npcap driver's access to Administrators only", run traffic66 as
   Administrator.
2. List the interfaces (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   The Name column is the connection name from Windows' network settings;
   the interface in use has an address.
3. Capture on Wi-Fi, by name or by number:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Put names with spaces in quotes: `-capture "Ethernet 2"`. Repeat
   `-capture` to capture on several interfaces. Add `-listen=` if you only
   want capture and no flow collectors. For the startup task in section 2,
   add the option to `-Argument`:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

Capture needs root or the capabilities `CAP_NET_RAW` and `CAP_NET_ADMIN`:
the `setcap` line above, or the `AmbientCapabilities` line in the systemd
unit in section 2. Wi-Fi interfaces are usually named `wlan0` or `wlp…`.

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

Capture needs root; nothing to install. On MacBooks `en0` is the Wi-Fi.

### Checking that it works

**Sources** lists each captured interface with the capture method and the
number of packets seen. The flows appear as coming from the device
`127.0.0.1` (this machine), on every page, like any other device's. Packets
seen twice (for example on two mirror ports) are counted twice.

## 12. Options

`traffic66 -h` and `traffic66 <command> -h` list everything.

Commands:

| Command | |
|---|---|
| `traffic66` | collect flows and serve the web UI |
| `traffic66 demo` | the same, with a simulated network |
| `traffic66 tui` | terminal UI for a running traffic66 |
| `traffic66 passwd` | add, change, list or delete users (see [Users and passwords](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | send simulated exports to a collector |
| `traffic66 interfaces` | list interfaces for local capture |
| `traffic66 version` | print the version |

Options of `traffic66` and `traffic66 demo`:

| Option | Default | |
|---|---|---|
| `-addr` | `:8066` | web UI address; `127.0.0.1:8066` for this machine only |
| `-data` | `traffic66-data` next to the program | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors as `name=address`, comma separated; empty disables |
| `-user` | `admin` | name of the user created on first start, and of the user `-password` applies to |
| `-password` | not set | accept only `-user` with this password for this run, ignoring the `password` file (also `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | days of flow detail kept; summaries are kept 400 days |
| `-memory` | `0.10` | share of physical memory for the database cache, and the same again as a soft limit for the rest of the program (each at least 256 MB) |
| `-l2-overhead` | `18` | bytes per packet added to NetFlow/IPFIX byte counts |
| `-sampling-wait` | `5m` | how long records wait for a sampling rate |
| `-capture` | | capture on a local interface (repeatable) |
| `-inventory` | `<data>/inventory.txt` | names file |
| `-asn` | `<data>/asn.tsv.gz` | IP-to-ASN table (`.mmdb` files: upload them, or `<data>/country.mmdb`, `<data>/asn.mmdb`, `<data>/both.mmdb`) |
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
| `license.json` | installation number and licence (see [Trial and licence](#trial-and-licence)) |
| `logo.png` (or `.svg`, `.jpg`, `.webp`, `.gif`) | your logo (**Sources → Logo**), if you uploaded one |
| `country.mmdb`, `asn.mmdb`, `both.mmdb`, `asn.tsv.gz`, `dbip-country.mmdb`, `dbip-asn.mmdb`, `threats/`, `sandbox/` | countries and networks databases you added or downloaded, and threat lists |

**How long data is kept**: flow detail 30 days, summaries (overview and long
time ranges) 400 days. Older data is deleted automatically, checked every 5
minutes; nothing else is deleted and there is no other limit. Change the
detail period with `-retention-days`, any number of days, for example
`-retention-days 365`. Disk use grows with it: **Free** in the side menu
turns red when the kept days will not fit. If the disk fills up, new flows
cannot be stored until space is freed.

**Data cleanup** in the side menu deletes data before you need to: older
than 120, 90, 60, 30 or 7 days, or all data. It shows for each choice how
many flow records go and about how much disk it frees, and asks before
deleting. Flow records, the hourly and daily summaries, interface counters
and findings are deleted; deleting all data also resets what the detection
rules have learned. It cannot be undone.

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

### Trial and licence

traffic66 can be tried for 30 days. On first start it writes `license.json`
to the data directory with an 8-digit installation number. The foot of
every page shows how many days of the trial are left, then that the trial
has ended. Nothing is switched off either way: every feature keeps working.

To register, send the author the installation number (also shown at the
foot of every page). The licence comes back as a new `license.json`; put
it in the data directory in place of the old one. It is checked when
traffic66 starts and every 4 hours, so no restart is needed; the foot of the
page then shows who it is licensed to and how many days are left.

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
12 GB of disk per day plus about 1.5 GB for the current hour, and the
program a sixth of one core. Overviews of long time
ranges come from summaries and take under 0.2 s. Queries over detail scan
about 22 million rows per hour: one host over 1 hour takes under 1 s, a
1-hour Top 66 of all conversations about 9 s; the time grows with the range
and shrinks with more cores.

Disk for 30 days at 5,000 flows/s is therefore about 360 GB; scale it with
your flow rate (shown on **Sources**) and `-retention-days`.

Memory: `-memory` (default 10% of RAM, at least 256 MB) limits the
database cache, and the rest of the program gets a soft limit of the same
size. At 5,000 flows per second the program's own data (decoding, duplicate
detection, batches) takes about 90 MB; in total expect 0.6–0.8 GB, so a
machine with 2 GB of RAM is enough. Measured over 10 minutes of continuous
collection (peak 0.58 GB on an 8 GB machine) and while loading an hour of
flows at eleven times that rate with the limits of a 2 GB machine (peak
0.74 GB).

`-memory` is a budget, not a hard cap: the Go limit is soft and the
database can briefly exceed its share. For a hard cap use the operating
system's: `MemoryMax=` in the systemd unit (section 2) or a container's
memory limit. Allow about 2.5 times the `-memory` share and at least 1 GB;
`MemoryMax=2G` suits machines with up to 8 GB at the default share.
traffic66 then restarts instead of the machine running out of memory.

## 16. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Device missing from **Sources** | Packets do not arrive: see [Check that flows arrive](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | The device has not sent its sampler options yet; most resend within minutes. If it never does, export them (`option sampler-table` on Cisco) or mark it `unsampled` in Names if it really is 1:1 |
| Numbers lower than the interface counters | See **Interface check**: loss on the way, interfaces not sampled, or flows still in the device cache (active timeout longer than 60 s) |
| Numbers higher than the interface counters | The same traffic sampled on two interfaces or two devices |
| No countries or networks ("Unknown") | No database loaded: upload one on **Sources**, see [Countries](#8-countries-networks-and-threat-lists) |
| "The database reached its memory limit" on a page | Choose a shorter time range, or start with a larger `-memory`; details are in the log |
| Forgot the password | `traffic66 passwd` on the traffic66 machine (add `-data` if traffic66 runs with it) |
| `Conflicting lock is held` | Another traffic66 already uses this data directory |
| `receive buffer is only … KB` | Linux limits UDP buffers: set `net.core.rmem_max=16777216` (see [Linux](#linux)) |
| `cannot create the data directory` | The program folder is not writable for this user: give `-data` |
| macOS: "cannot be opened" or "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" | **More info** → **Run anyway**; the program is not signed yet |
| Windows capture: Npcap not found | Install [Npcap](https://npcap.com) |
| `address already in use` | Another program uses the port: choose others with `-addr` or `-listen` |

## 17. Build from source

Go 1.24 and a C compiler (gcc or clang; MinGW-w64 on Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
