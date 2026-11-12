# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## 2.5.14

- Build with Go 1.27.4.

## 2.5.13

- Security and robustness fixes for incoming data and account management.
- Better contrast in all themes; text fixes.

## 2.5.12

- Users: resetting a user's password signs that user out everywhere; DELETE /api/users also takes the name in a JSON body.
- Startup: an option error now says that older versions ignored it, so an upgrade that stops at start is easy to explain.
- Text fixes.

## 2.5.11

- Startup: out-of-range options and a bad -listen entry are reported and stop the start instead of being ignored.
- New GET /healthz for probes and containers.
- Users: adding a name that already exists no longer changes that user's password; use Reset password instead.
- The password given at start (-password, TRAFFIC66_PASSWORD) is explained in the help, the log, the web UI and the manual.
- `passwd` says whether it added a user or changed a password; `simulate` reports failed sends.
- Docs: notes on IP fragments and time ranges in offline analysis.

## 2.5.10

- Single-binary sFlow, NetFlow v5/v9 and IPFIX collector with a web UI and a terminal UI; embedded database and country/ASN data, nothing to install.
- Overview, Top 66 (top 33 clients and 33 servers), traffic details, flow records, flow paths, interface check, findings, threat intel, and a geo & networks map; five themes and 15 languages.
- Offline analysis of up to 3 pcap/pcapng files, with a one-time login link and temporary data removed on exit; local capture (`-capture`), a demo mode and a built-in `simulate` sender.
- Demo: the first-start history is much smaller and quicker to generate, and its progress is logged every 10%; the web UI opens when it is done.
- Web UI: pages are served with a Content-Security-Policy (scripts only from the program itself, nothing loaded from other hosts; embed pages can still be framed).
- Web UI: the first-data hints give the real waits: templates and sampling rates are waited for up to 5 minutes by default (-sampling-wait).
- Startup: a bad or busy -addr is reported before the collectors start.
- HTTPS (-tls) works on every installation; its help text now says so.
- Docs: READMEs give the real `-data` default, brief system requirements, how long the first data takes to appear, a note not to expose the web UI directly to the internet, and the licence terms in three lines.
