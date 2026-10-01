# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## 0.1.2

Using it
- New **Conversations** page in the web UI and the terminal UI (key 3): who
  talks to whom, with client, server, service and traffic. The terminal UI's
  pages are now 1–9.
- The side menu no longer shows group headings that looked like buttons but
  did nothing; groups are separated by lines.
- Windows local capture by name: `-capture Wi-Fi`, `-capture Ethernet` or
  the number from `traffic66 interfaces`, instead of Npcap's
  `\Device\NPF_{…}` names. `traffic66 interfaces` now shows each adapter's
  Windows connection name and address.

Documentation
- README section "Local capture" rewritten with step-by-step instructions
  for Windows, Linux and macOS, including what a Wi-Fi adapter can and
  cannot see (13 languages).
- This changelog; releases now carry these notes instead of a list of pull
  requests.

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
