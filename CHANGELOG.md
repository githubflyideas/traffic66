# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## Unreleased

- Licence: the Additional Use Grant is now version 1.1. Production use is free for organizations of fewer than 50 people (before, 100); larger organizations register after 30 days of production use, as before.

## 2.10.0

- Traffic statistics (registered installations): a new page in the sidebar for the traffic of one VIP or of a whole product line — the VIPs a line bills together are written in the inventory as a product line (`product Shop vip=203.0.113.10,203.0.113.11`; one address belongs to one line only). Inbound and outbound are added together as the headline figure, with the split kept underneath, the members listed per address, and a CSV export with the same figures. Traffic between two VIPs of the same line is counted once. The figures say they come from the sampled flow records and cover the settled part of the window.
- Read-only accounts (registered installations): the administrator can give a business an account of its own, bound to one VIP or one product line. It signs in with a password like any other account, counts toward the installation's account limit like any other account, and then sees only that object's traffic statistics — a window it picks of at most fourteen days, a per-minute query limit, and no other page and no writes. Accounts are made, rebound, disabled, reset and removed beside the platform links on the Integrations page.
- Top over long ranges is much faster: a range that reaches past the flow records is answered from the summaries as a whole, and the answer says where it came from. The records themselves are kept for seven days by default (before, thirty) and the summaries for 180 days (before, 400); one query can span at most thirty days. Past the records, the record list says that only summaries remain.
- The traffic statistics no longer depend on how long the flow records are kept: they have hourly summaries of their own, kept for 180 days. When an installation is upgraded, the summaries are first built from all the records on hand, and records are deleted only after that, so the billing figures do not change.

## 2.8.0

- Interface counter snapshots dropped because processing fell behind are now counted (status, metrics and the self page), like dropped flow records already were. A gap in the counters no longer looks exactly like an interface disagreeing with its flows.
- Interfaces page: each interface now has a checkbox. Untick interfaces to subtract them and the page adds a panel for what remains — the selected interfaces' combined load (in and out, counted the way the device counters count, so the two can be compared), their totals, and their top applications and clients. One button narrows to the interface being inspected, another brings everything back.
- The web interface's font sizes, corner radii and monospace stack are now defined once, as variables the whole stylesheet and the embed page share, instead of being repeated rule by rule.

## 2.7.0

- Settings: the IP databases (country/region and operator) have a tab of their own, before registration and purchase.
- The version and registration under the logo in the sidebar are centred.
- Interface check: the bandwidth chart at the top has the same total / inbound / outbound switch as the overview, and the note about the dashed line (yesterday at the same time) sits next to the chart title.
- Pages on a 7-day or 30-day range refresh every 30 seconds (before, up to 5 minutes).
- Export CSV: the hourly device counters of the range on show as a spreadsheet file — one row per hour, device and interface, with average bit rates. The counters are kept for the whole rollup retention (400 days), long after the flow detail is gone.
- API version 1: every answer carries an Api-Version header, /api/v1/ works next to /api/, the machine-readable description is at /api/openapi.json, errors are JSON with stable codes, and a wrong method is answered 405 with the methods that are allowed.
- API tokens: a program can be given its own token, read-only or read-write, on the account page or with `traffic66 gen-token` — no browser session needed.
- Security: sign-ins and sign-outs, account changes and token changes are written to the log; password hashing is strengthened (a stored password is re-hashed at its next sign-in); account limits are hardened; HTTPS answers carry HSTS.
- API: an invalid time range is answered with an error (bad_range) instead of silently using the default window.
- SECURITY.md added. Release checksums and the SBOM are covered by the build attestation, like the packages themselves.

## 2.6.6

- Top 66 is now called TOP; it still shows 33 servers and 33 clients.
- New time ranges: 30 minutes and 12 hours, in the web UI, the terminal UI and the API.
- Shorter range buttons, the same in every language: 15M, 30M, 1H, 6H, 12H, 1D, 1W, 30D (M minutes, H hours, D days, W weeks); the full name shows as a tooltip and is read by screen readers.
- sFlow: switches that share one sample sequence across interfaces are no longer counted several times too high; reordered or repeated datagrams are not taken as loss; a sampling-rate change is followed at once; heavy loss is compensated up to 20×, also on quiet interfaces, and agent restarts do not cause spikes.
- sFlow samples over 512 KB are refused, counted and logged.
- Settings: an observation domain that is waiting for its template is no longer hidden.
- Licence install errors are shown in your language.
- A page with a panel that scrolls sideways (for example the overview in Japanese) no longer ends in a script error.
- Right-to-left languages: the evaluation badge no longer covers Sign out.
- Manual in all 15 languages updated for the current pages, themes and ranges; release packages no longer contain Go source in docs/.

## 2.6.5

- sFlow: traffic is scaled by the rate the switch declares, raised only by samples that were lost; some switches were counted about a fifth too high.

## 2.6.4

- Interfaces and data sources: clearer lists, fewer false warnings.
- Interface reconciliation shows total bandwidth on top.
- Header: version and registration on two lines.

## 2.6.3

- Top 66 and traffic details: new views for registered installations.
- Overview: simplified.

## 2.6.2

- Navigation and settings: layout improvements.
- Registration status is shown under the logo.

## 2.6.1

- Settings: layout improvements.

## 2.6.0

- Reverse proxy: default ports are ignored when matching the address, the standard `Forwarded` header is understood, and the new `-public-url` names the public address. A refused request says why instead of "wrong password".
- Byte counts: the link-layer header is taken from the sampled frame, so switch VLAN records, QinQ and MPLS are no longer over- or under-counted; sampled frames over 64 KB are accepted.
- After an unclean stop, summaries of the last hours are rebuilt from the flow detail at the next start.
- Flow records show the number of flows; ICMP is shown by name instead of "0/1".
- No false peak in the charts right after older data arrives.
- Dark themes: flow lines have five clearly different shades.
- Settings shows how many records were refused or had their time corrected.
- New `/metrics` endpoint for Prometheus (sign-in required).
- `traffic66 help <command>`; every command has `-h` with examples.
- A data directory written by a newer data format is refused with a clear message.
- TLS certificate errors are reported before anything else starts; the startup banner shows the address actually used.
- Clearer error messages in all languages; accessibility fixes in the web UI.
- Data cleanup can delete one device's stored data.
- Manual in all 15 languages: install, upgrade and roll back, reverse proxy (nginx, Caddy, Traefik), monitoring.
- Releases include `SHA256SUMS`, an SBOM and build provenance.

## 2.5.15

- Stricter checks on incoming samples; refused records are logged and counted.
- Works behind a reverse proxy that passes the host name on (see the manual).
- Account changes from `passwd` and the web UI at the same time are all kept.
- Smaller fixes to sign-in and user names.

## 2.5.14

- Build with Go 1.27.4.
- Robustness fixes for flow records with unusual timestamps.
- `-l2-overhead 0` is honoured; byte counts for VLAN-tagged samples corrected.
- `passwd` accepts a piped password; IPv6 /128 filters; per-direction interface report.
- Country/region wording in all languages; map line colours on dark themes; smaller fixes to messages and logs.

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
