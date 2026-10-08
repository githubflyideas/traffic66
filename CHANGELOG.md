# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## 1.6.0

Sign-in and users
- A new installation and the demo sign in as admin / traffic66; the
  sign-in page shows it while it is still in use. A new installation asks
  for a new password at the first sign-in, before anything else.
  Installations that already have a password file keep it.
- **Account** at the foot of the menu: change your password. The
  administrator (admin) also adds and deletes users and resets passwords;
  the password file is the same one `traffic66 passwd` writes.
- LDAP / Active Directory sign-in is shown as in development.
- With -password or TRAFFIC66_PASSWORD, passwords are not changed from the
  web UI, and the page says so.

## 1.5.2

Fixes
- Client and server were swapped for half of many TCP connections: exported
  records (NetFlow, IPFIX, local capture, pcap files) carry the flags of all
  their packets together, so a client's SYN followed by ACKs looked like a
  server's SYN+ACK and its upload was counted the wrong way round. The flags
  now decide only for single packets (sFlow samples, lone SYNs); local
  capture and pcap files use the first packet of each connection; other
  records go by the ports. Data stored before stays as it is; pcap files are
  right once imported again.

Offline pcap analysis
- One file at a time, each in a database of its own: **Analyse** on a
  file's row shows that file on every page; the bar at the top switches to
  another. Deleting a file no longer re-imports the others.

Interfaces
- Interface check explains that local capture has nothing to compare (no
  device interfaces or counters), and the interface picker is hidden when
  there are no interfaces.

Docs
- README: the Windows commands for local capture (`traffic66.exe interfaces`,
  `traffic66.exe -capture Wi-Fi`).

## 1.5.1

- The web UI's scripts and styles are no longer kept by the browser across
  an upgrade: after replacing the program, a normal reload shows the new
  pages (before, an old app.js could stay in the browser's cache).

## 1.5.0

Includes the changes listed under 1.2.3: the 1.2.3 release was built from
the same code as 1.2.2.

Interfaces
- **Interface** at the top of the traffic pages (Overview, Top 66, Traffic
  details, Flow paths, Geo & networks, Flow records, detail): all
  interfaces, or one, and the pages show only the traffic through it. It
  starts on the default interface and is kept in the link. Findings,
  Threat intel, Interface check and Settings stay on all traffic. One
  interface over 7 or 30 days is read from the flow records (slower,
  as far back as they are kept).
- Interface check: each interface can be named and tagged (✎) and one made
  the default (★); saved to its iface line in Names as tag=… and default.
  The page opens on the default interface.
- Interface check: the bits/s and packets/s charts are full width, one
  above the other.
- Overview: the bandwidth of the chosen (or default, or busiest) interface
  in bits/s, ingress green and egress blue, above the bandwidth by
  application.
- The interfaces a device samples are told from the ones only seen as the
  other end of its flows: by the sFlow data source, by flowDirection
  (IPFIX 61) in NetFlow v9 and IPFIX, or else, without that field, as the
  interface on at least 90% of the device's traffic.
- Interface check lists those other interfaces last, in smaller grey type,
  under "Peer interfaces": their numbers hold only the traffic through the
  sampled interface. Their charts say so.
- **Interface** above the pages lists only sampled interfaces, grouped by
  device.
- Settings shows for each device the interfaces it samples, and whether its
  NetFlow/IPFIX templates carry flowDirection.

Menu
- Flow records moved up to the traffic pages, under Interface check.
  Offline pcap analysis follows Settings; Data cleanup is last.

## 1.2.3

Built from the same code as 1.2.2; its changes are in 1.5.0.

## 1.2.2

NetFlow and IPFIX sampling
- Sampling rates are found in more of the ways devices declare them:
  samplingProbability (IPFIX 311); a rate per interface that is set on the
  egress interface of a flow; a single sampler whose id the flow records do
  not name; a rate declared under another observation domain of the same
  device (line cards export under their own ids); a samplingInterval of 0
  next to samplerRandomInterval.
- A rate can be given by hand for a device that does not declare one:
  sampling=N on its device line in Names (Settings). A rate the device
  declares still wins; flows waiting for a rate go on with it at once.
- Settings shows the templates a device sent (template and field ids) when
  its sampling rate is not declared, to see what it does send.
- Export sessions silent for an hour while the same device sends in another
  way (a changed version or domain) are listed apart as stopped, not as
  faults.

## 1.2.1

Pages
- Overview: **Total**, **Inbound** and **Outbound** above the bandwidth
  chart; the title says which (total is inbound plus outbound, so it is the
  sum of an uplink's two directions on Interface check). The numbers above
  follow the choice.
- Findings are at the foot of the overview.
- Themes: the pages and the cards are one colour (white, grey, teal or
  cream), set off by thin lines; Black stays as it was.

Fixes
- Charts lagged 2 to 4 minutes behind the clock. They now end where the
  data is complete, measured from the flows: with sFlow at the current
  minute, with NetFlow and IPFIX as long as the devices take to export
  (at most 2 minutes).

## 1.0.3

Pages
- The comparison on the overview says what it compares with: the same time
  yesterday (ranges up to a day), the same time last week (up to a week)
  or the days before (longer ranges). Without data then there is none,
  instead of falling back to "the previous period".
- Top 66 opens on the table; Talkers is the second tab.
- Traffic details: the second chart is one ring of services.
- Flow paths is a quarter taller, and the middle column (application,
  service) sits two thirds of the way across, on the server side.
- Interface check moved up, under Flow paths.
- No search box on Top 66 and Traffic details; the Device, Client, Server
  and Service boxes stay.
- Themes: Light, Grey, Black, Teal and Orange; one button, each click goes
  to the next.
- Settings: sFlow loss is split into loss on the way (sequence gaps: the
  network or this machine's receive buffer) and samples the device dropped
  itself (its sFlow export is rate-limited), each with what to do.

## 1.0.2

Pages
- **Sources** is now **Settings**, with a gear icon (13 languages and the
  READMEs).
- Five colour themes after iOS's system colours, chosen next to the
  language and kept per browser: **Light** (default), **Bright** (white,
  stronger contrast), **Grey**, **Dim** and **Dark**, or **Follow
  system**. Charts, map and logo follow the theme.
- The side menu is a shade darker than the pages, so the two are easy to
  tell apart.
- Settings, Data cleanup and Offline pcap analysis no longer show the time
  range, refresh and Copy link; they do not use them.
- Flow records opens on the last 15 minutes and updates every 5 seconds,
  without the time range bar. Opened from a value on another page (Show
  its flow records) it keeps that page's time range; **Back to live**
  returns.
- Interface check: the bits/s and packets/s charts again (one chart in
  1.0.1); when the device sends counters, they are dashed lines on the
  bits/s chart. The separate comparison chart is gone, its verdict stays.

Fixes
- Charts no longer drop at the right end: the newest minute (or 5 or 15
  minutes) is not drawn while its flows are still arriving.
- Last week's line is left out where there was no data then, instead of
  falling to zero.
- After signing in on the sign-in page, pages did not refresh on their own
  until the browser was reloaded.
- Pages no longer refresh while you type in a form field (the names form
  in Settings).

## 1.0.1

- Interface check: one chart, Interface bandwidth, with a bits/s |
  packets/s switch.

## 1.0.0

Licence terms
- traffic66 is source available under the PolyForm Noncommercial License
  1.0.0 and the Traffic66 Additional Use Grant 1.0 (LICENSE.md,
  ADDITIONAL-USE-GRANT.md, both in every download; the grant has reference
  translations in French, Spanish, Chinese and Japanese, the English text
  is binding). Evaluation is free for
  everyone; production use is free for organizations with fewer than 100
  people; larger organizations need a registration licence after 30 days.

Licence
- Traffic66 can be tried for 30 days. The foot of every page shows the days
  left, then that the trial has ended; it keeps working with every feature
  either way. A licence from the author (a signed license.json in the data
  directory, checked every 4 hours) shows who it is licensed to and the days
  left. On first start license.json is created with an 8-digit installation
  number, which is what a licence is issued for.

Offline analysis
- New page **Offline pcap analysis**: upload packet captures (.pcap, .pcapng
  from Wireshark or tcpdump; up to 3 files of at most 50 MB). They are
  turned into flows in a database of their own, apart from the live data.
  **Analyse** shows them on every page (overview, Top 66, traffic details,
  findings, flow paths, map, flow records) with an orange bar naming the
  files; **Back to live data** returns. **Delete** removes the files and
  their data.
- The detection rules run over the capture: scans, port scans and password
  guessing are found in it.
- The demo includes an example capture with an attack.
- `traffic66 capture.pcap` (up to 3 files, 3 GB in all) starts a private
  traffic66 on 127.0.0.1 with a free port, prints the address, a password
  and a one-time sign-in link, opens the browser on the capture, and
  deletes the imported data on Ctrl+C. Files are read in place; nothing is
  collected or sent and host names are not looked up unless `-dns` is
  given. A 1 GB capture takes about 5 s (1.2 million full-size packets) to
  30 s (14 million small packets) on 2 cores.

Fixes
- Traffic details over 6 and 24 hours showed one spike an hour instead of
  a continuous chart: 6 hours was read from the hourly summaries by
  mistake, and 24 hours drew the hourly summaries in 5-minute steps. 6
  hours now uses 5-minute steps from the flows, 24 hours and longer hourly
  steps.

- Interface check drew an interface's traffic too low where another device
  reported the same traffic: those flows were left out as duplicates. A
  query kept to one device or interface now counts all of that device's
  flows (traffic across devices still counts once).
- The demo no longer leaves a gap in the charts where its live exporters
  take over from the generated history, and restarting it fills the time
  it was stopped.

Pages
- Side menu in four groups: traffic (Overview, Top 66, Traffic details,
  Flow paths), security (Findings, Threat intel, Geo & networks), setup and
  data (Sources, Interface check, Flow records, Data cleanup) and Offline
  pcap analysis; bolder, with icons. The version and the server's date and
  time are under the logo; Log out has a line of its own.
- The search box and the Device, Client, Server and Service boxes are only
  on Top 66 and Traffic details, where they are used; a filter set there
  still applies to every page.
- Time range **Custom…**: any start and end, also further back than 30
  days; copied links keep it.
- Top 66 Talkers without the traffic-by-service chart: the two tables.
- Traffic details is two ring charts: servers with their clients, and
  services with their servers. A switch puts either side inside.
- Flow paths shortens long names to 22 characters; the full name shows on
  hover.
- Interface check: ingress (green) and egress (blue) in one chart per
  measure, for the interface picked in the list; the comparison with the
  device counters shows both directions by default.
- Names: a form (type, address, name) and a table with Edit and Delete,
  with addresses and networks checked; adding the same address again
  replaces it. The text file is still there under Edit as text (advanced).
- `net` lines take `country=XX`; the world map then draws lines from that
  country to the countries your networks talk to.
- New page **Data cleanup**: delete data older than 120, 90, 60, 30 or 7
  days, or all, with how much each frees. Summaries, interface counters and
  findings go with it.

## 0.3.1

- Traffic details opens on servers only; tabs switch to clients, both ends
  side by side, and services (13 languages).

## 0.3.0

Findings
- traffic66 now looks through the flows every 5 minutes and lists what
  needs attention on a new **Findings** page: scans, port scans, password
  guessing, lateral movement inside the network, unusual uploads to new
  destinations, floods and traffic with addresses on threat lists. Each
  finding is a sentence (who did what to whom, when, for how long) with the
  numbers behind it and how the data was sampled.
- The rules work on sampled sFlow and NetFlow. Tested with the demo's
  attack sent through a switch sampling 1:4096: every step is found, each
  as one finding; a day of normal traffic gives no findings apart from the
  internet scanner.
- **Dealt with** and **Not a problem** close a finding; "not a problem" is
  never reported again. The overview shows the open findings first, a
  host's details page lists the findings about it, and the side menu shows
  how many high and medium findings are open.

Charts and filters (compared with ElastiFlow's dashboards)
- **Device**, **Client**, **Server** and **Service** filter boxes above every
  page list the busiest values of the time range.
- Top 66 opens on **Talkers**: traffic by service over time and the top 30
  clients and servers with traffic, packets and flow records, and a row for
  all traffic. The regroupable table is one click away.
- New page **Traffic details**: clients, servers and services over time in
  bits/s and packets/s.
- Interface check shows every interface's traffic over time (ingress and
  egress, bits/s and packets/s); Geo & networks shows source and
  destination networks (AS) over time.
- Flow records shows how many records there were and when, and pages
  through all of them.
- Flow paths can show client → service → server.
- Charts use 8 fixed colours checked for colour-blind readers; the rest is
  Other.

Pages
- Flow paths start from each host by default (the 10 busiest, the rest as
  Other); **By network** switches back to network segments.
- The overview shows the findings below the traffic chart, and no longer
  the "remote location" chart or the change column in Top clients.
- Top-N is called Top 66 and comes right after Overview in the menu.
- Your own logo: **Sources → Logo** takes a PNG, SVG, JPEG, WebP or GIF
  (best at 272 × 92 pixels) for the sign-in page and the top of the menu.

Fixes
- A host's details page counted only part of the internal hosts it talked
  to (the servers of internal conversations were missed).
- Arabic and Urdu: ports read backwards ("tcp/445") and names ran into
  their addresses.

Drill-down and naming
- **Show details** on any host, device or service opens a page about it:
  traffic over time by application, who it talks to, services or clients,
  countries and the latest flows. Everything on it can be clicked again to
  keep drilling down; the browser's Back button returns.
- **Name it…** on any host or device names it on the spot; the name is
  saved and shown everywhere. The Names box on Sources now shows examples.

Countries and networks
- Works out of the box: DB-IP's free country and ASN databases (CC BY 4.0)
  are built in, so countries and networks show without uploading anything.
  **Update DB-IP Lite now** on the Sources page downloads this month's
  version.
- Sources lists the free databases (DB-IP Lite, MaxMind GeoLite2, IPinfo
  Lite, IPtoASN) with their licences and where to get them. Uploaded files
  are used first and the built-in DB-IP Lite answers the rest; **Remove**
  goes back. IPinfo Lite, which holds countries and networks in one file,
  is now understood.
- The `.mmdb` reader was checked against MaxMind's own reader on 60,000
  lookups in six DB-IP databases with no difference.
- **Geo & networks** shows remote traffic on a world map; point at a country
  for its traffic, click it to filter.

Simpler pages
- Top-N is one table: conversations (client, server, service, country) by
  default; **Group by** switches the grouping. Every column sorts, and
  the number columns (traffic, packets, average packet size, flows) rank
  all traffic in the range rather than re-ordering the rows shown: the
  smallest average packet size finds scanners and floods. The row of
  eleven tabs is gone.
- "Who grew" is gone from the overview (web and terminal UI).
- Narrow windows: the menu, time range and tables no longer push the page
  wider than the window, and "Log out" stays inside the menu in every
  language.
- Flow paths: labels, bars and flows can all be clicked; before, only the
  thin bars could.
- The side menu no longer shows group headings that looked like buttons;
  "Log out" no longer wraps.
- Disk in the side menu shows used and free space; hovering over the free
  space shows how much the kept days of detail need at the current rate.
  The old "days left" figure was an unreliable extrapolation.

Capture
- Windows local capture by name: `-capture Wi-Fi`, `-capture Ethernet` or
  the number from `traffic66 interfaces`, which now shows each adapter's
  Windows connection name and address.
- Commands copied from a web page or chat work: full-width spaces, quotes,
  full-width dashes and a trailing full stop around words are ignored, and
  a mistyped command gets a "did you mean" hint.

Documentation
- README (13 languages): drill-down, naming, the database upload, local
  capture step by step for Windows, Linux and macOS including what Wi-Fi
  can and cannot see, new screenshots.
- This changelog; releases take their notes from it.

## 0.1.1

Accuracy
- For time ranges longer than 6 hours every number on a page now covers
  the same window. Before, Top-N and the other breakdowns dropped the
  first, partial hour, so right after each full hour they disagreed with
  the totals on the same page (in the demo one host showed 206.8 GB on the
  overview and 186.9 GB in Top-N). Ranges over 6 hours now start on a whole
  hour.

Memory
- Duplicate detection keeps hashes instead of full flow keys and a fixed
  8-minute window of data time, capped at 1,000,000 flows per minute. At
  5,000 flows per second the peak over 10 minutes fell from 1,326 MB (and
  rising) to 577 MB. Loading an hour of flows at eleven times that rate
  with the limits of a 1.9 GB machine now peaks at 738 MB instead of being
  killed at about 1.7 GB.
- `-memory` now also sets a soft memory limit for the rest of the program,
  not only for the database. For a hard limit, the README's systemd unit
  uses `MemoryMax=`.
- At most 10,000 exporters are tracked and silent ones are forgotten after
  a day, so datagrams from many spoofed addresses cannot grow memory.
- A long flow spread over many minutes no longer pushes the current minute
  out of duplicate detection.

Errors
- Pages no longer show raw database errors. When the database reaches its
  memory limit or cannot answer, the page says so in plain words (13
  languages) and the details go to the log.
- Clear startup messages when the data directory is used by another
  traffic66 or its temporary directory cannot be created.

Users
- `traffic66 passwd -list` and `traffic66 passwd -user NAME -delete`. A
  deleted user is signed out of open browsers at once; the last user cannot
  be deleted.

Releases and tests
- One-click releases: Actions → ci → Run workflow with a version.
- Load tests apply the program's memory rules and skip on machines with
  less than 1.5 GB of memory.

Documentation
- README with screenshots of the web UI and the terminal UI, a section on
  users and passwords, and corrected memory figures (13 languages).

## 0.1.0

First release.

- Collects sFlow v5, NetFlow v5/v9 and IPFIX on any UDP port; optional
  local capture.
- Embedded database; one program for Windows, Linux (any distribution with
  kernel 3.2 or later, including CentOS 7 and Alpine) and macOS 11 or later.
- Web UI and terminal UI in 13 languages: overview, Top 66, flow paths,
  countries and networks, threat lists, flow records, interface check,
  sources.
- Checks flow numbers against interface counters (sFlow counters or
  SNMP) and explains differences.
- The login password is kept, hashed, in the data directory.
