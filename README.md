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
- 13 languages: English, 中文, हिन्दी, Español, العربية, Français, বাংলা,
  Português, Русский, Bahasa Indonesia, اردو, 日本語, 한국어.

## Try it

Download the archive for your system from the releases page, unpack it and
run:

```
traffic66 demo -password try66
```

Open http://127.0.0.1:8066 and sign in as `admin` / `try66`. The demo builds
a small company network with a day of history and live traffic from four
simulated devices, including two incidents to find.

On Windows, run `traffic66.exe demo -password try66` from a terminal.

## Collect from your network

```
traffic66 -password <choose one>
```

It listens on UDP 6343 (sFlow), 2055 (NetFlow) and 4739 (IPFIX); every port
accepts every protocol. Point your devices at this machine's address and
allow those UDP ports through any firewall in between. Check
**Sources** in the web UI: it shows each device, its protocol, sampling rate
and loss, and what to change when something is wrong.

Typical device settings (adjust names and addresses; check your device's
manual for its exact commands):

Cisco IOS / IOS-XE Flexible NetFlow:

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

Switches with sFlow:

```
sflow collector 192.0.2.50 port 6343
sflow sampling-rate 4096        (per interface: about 1:1000 for 1 Gb/s, 1:4096 for 10 Gb/s)
sflow counter-interval 30
```

Recommendations:

- Set the active flow timeout to 60 seconds.
- If the device samples NetFlow/IPFIX, let it export its sampler options so
  the rate is known; traffic66 holds records until the rate arrives rather
  than counting them 1:1.
- sFlow counter samples, or an `snmp` line in the inventory, let traffic66
  compare its numbers with the interface counters.

## Names and SNMP

**Sources → Names** in the web UI (or `inventory.txt` in the data directory)
takes one entry per line; see `inventory.txt.example`:

```
net    10.10.0.0/16  Office LAN
device 192.0.2.1     Core router
iface  192.0.2.1 3   ISP uplink speed=1000000000
host   10.10.3.27    Finance PC
snmp   192.0.2.1     public
```

An `snmp` line reads the device's interface counters every minute (SNMPv2c,
IF-MIB 64-bit counters). Add a management address when it differs from the
address the flows come from: `snmp 192.0.2.1 public 10.99.0.1:161`.
Interface descriptions read over SNMP are used as names unless you name the
interface yourself.

## Countries, networks and threat lists

Put an IP-to-ASN table in the data directory as `asn.tsv.gz` (tab separated:
first address, last address, AS number, country code, AS name — the format
published by iptoasn.com). Threat lists are plain text files with one
address or network per line in `<data>/threats/<name>.txt`. Restart after
adding files.

## Terminal UI

```
traffic66 tui                          # the traffic66 running on this machine
traffic66 tui -server http://10.0.5.9:8066 -user admin -password ...
traffic66 -tui                         # collect and show the terminal UI in one process
```

Keys: 1–8 pages, ↑↓ select, Enter actions, f show only, x exclude,
/ search, t time range, c clear filters, w open the same view in a browser,
q quit.

## Local capture

`traffic66 interfaces` lists the interfaces; `traffic66 -capture <name>`
captures on one (repeatable), for example on a mirror port. Linux and macOS
need root or the corresponding capabilities. Windows needs
[Npcap](https://npcap.com), installed separately.

## Options

| Option | Default | |
|---|---|---|
| `-addr` | `:8066` | web UI and API address |
| `-data` | `traffic66-data` | data directory |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | UDP collectors; empty disables |
| `-user`, `-password` | `admin`, generated | login; a generated password is printed once |
| `-retention-days` | 30 | days of flow detail kept; summaries are kept 400 days |
| `-memory` | 0.10 | share of physical memory the database may use |
| `-l2-overhead` | 18 | bytes per packet added to IP-layer NetFlow/IPFIX counts |
| `-sampling-wait` | 5m | how long records wait for a sampling rate |
| `-capture` | | capture on a local interface (repeatable) |
| `-dns-upstream` | system resolver | DNS server for showing host names |
| `-dns-rate`, `-dns-cache` | 20/s, 2m | reverse lookup rate limit and cache time |
| `-no-dns` | | no reverse lookups |
| `-tui` | | also open the terminal UI |

## Sizing

Measured at 5,000 flows per second on a 2-core machine: detail uses about
12 GB of disk per day plus about 1.5 GB for the current hour, the program
about 0.5 GB of memory and a sixth of one core. Overviews of long time
ranges come from summaries and take under 0.2 s. Queries over detail scan
about 22 million rows per hour: one host over 1 hour takes under 1 s, a
1-hour Top 66 of all conversations about 9 s; the time grows with the range
and shrinks with more cores.

## Build from source

Go 1.24 and a C compiler (gcc or clang; MinGW-w64 on Windows):

```
scripts/build.sh 0.1.0 traffic66
```
