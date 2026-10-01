# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## 0.1.2

Drill-down and naming
- **Show details** on any host, device or service opens a page about it:
  traffic over time by application, who it talks to, services or clients,
  countries and the latest flows. Everything on it can be clicked again to
  keep drilling down; the browser's Back button returns.
- **Name it…** on any host or device names it on the spot; the name is
  saved and shown everywhere. The Names box on Sources now shows examples.

Countries and networks database
- Upload a database on the Sources page: MaxMind GeoLite2 or DB-IP Lite
  `.mmdb` files (country or ASN) or an IP-to-ASN table (`.tsv`, `.tsv.gz`).
  It is checked, saved and used for new traffic at once, without a
  restart. The `.mmdb` reader was checked against MaxMind's own reader on
  60,000 lookups in six DB-IP databases with no difference.

Simpler pages
- Top-N is one table: conversations (client, server, service, country) by
  default; click a column heading to group by it, a number heading to
  sort. The row of eleven tabs is gone; the less common groupings are in
  **Group by**.
- "Who grew" is gone from the overview (web and terminal UI).
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
