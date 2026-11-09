# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

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
