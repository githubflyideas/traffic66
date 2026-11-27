# Changes

The release notes on GitHub are taken from this file: the section whose
heading is the version number.

## 2.15.1

- Strengthened sign-in handling behind reverse proxies: failed sign-ins are counted per account and per client address, so one person's failures no longer lock anyone else out. A new `-trusted-proxies` option names the proxies (addresses or CIDR ranges) whose `X-Forwarded-For` header names the client; from every other peer the header is still ignored. The manual's reverse-proxy section explains it.
- Sign-in responses no longer reveal whether an account exists: an unknown user is answered exactly like a wrong password.

## 2.15.0

- A non-GET request to `/logo`, `/auto` or `/tls.crt` now answers 405 JSON with an `Allow: GET` header and the same `API-Version` and security headers as `/healthz`, instead of the static server's plain-text 404.
- `GET /tls.crt` when no certificate is configured, or the certificate file is missing, answers JSON 404 (`not_found`). A missing built-in logo file does the same.
- Traditional Chinese (繁體中文) is listed immediately after Korean in the web interface and embed language menus.
- Tests: the command package covers `serve` starting and stopping, `passwd` and `gen-token`; the terminal UI covers keyboard navigation, filters, the help overlay and window sizing.

## 2.14.0

- **Breaking (API):** every error answer now has the one shape the product family uses: `{"error": "short message", "code": "stable_code"}`. The stable code moved from `error` to `code`, and `error` now carries a short English message; programs must read `code` for the stable value (the web UI does). The version header is spelled `API-Version`, as in fogping and pingping.
- The OpenAPI description at `/api/openapi.json` can be fetched without signing in, and it now also documents `/healthz`, `/auto`, `/logo` and `/tls.crt`.
- Errors that used to answer plain text now answer JSON with a stable code: a non-GET request to `/healthz` gets 405 with an `Allow: GET` header, an unhealthy `/healthz` gets a JSON 503 (`storage_unhealthy`), a missing manual image gets a JSON 404, and uploading a file that is not a capture — or too many files — to the sandbox gets `not_capture` / `too_many_files` instead of a sentence.
- An error the API does not classify no longer echoes its internal text to the client; the detail is written to the server log only, and the client gets a generic code.
- `/healthz` now carries the same headers as every other answer (`API-Version` and the security headers), and the evaluation header of an unregistered installation is short ("evaluation copy - not registered"); the full evaluation notice still appears as the page watermark.
- Documentation: the interface language count is 16 everywhere, Traditional Chinese included. The Linux install steps now make raising `net.core.rmem_max` a required step, with the command and how to verify it.
- Terminal UI: Traditional Chinese locales (zh_TW, zh_HK, zh_MO, zh-Hant) select the Traditional Chinese dictionary instead of Simplified.
- Test coverage where there was little or none: the flow record types, the traffic simulator, the wire reader and the packet encoders are now covered, and the local capture, terminal UI and command packages gained substantial tests.

## 2.13.0

- Traditional Chinese (繁體中文) is a sixteenth interface language, written in Taiwan usage and a formal register, available in the web interface and in embedded pages.
- Interface text revised in Simplified Chinese, English, Japanese, German and Spanish: sentences that were still shown in English are translated, mistranslations are corrected, and the core terms (traffic, sampling rate, VIP groups and others) now read the same everywhere in each language. The wording is more formal throughout.
- Build checks: the pipeline now fails on unformatted code (gofmt), on staticcheck findings and on the race detector, beside vet, the vulnerability scan and the tests; staticcheck and govulncheck are pinned to a version. The route table that answers 405 and builds the OpenAPI description is checked against the registered routes by a test.
- The manual's 404 answers JSON like every other error.

## 2.12.0

- Registered pages: traffic statistics, interface check, findings and the inventory are registered features. Unregistered, they stay in the menu with a Pro tag and open on a page that says what they do and leads to registration; their API answers not_registered, reads and writes alike. Naming an address from its menu needs the inventory, so it is offered on registered installations. What the inventory already holds keeps applying, and the detection rules keep running. The demo shows every page.
- Reverse DNS: lookups stay in every installation; tuning them (-dns-upstream, -dns-rate, -dns-cache) is a registered feature. Unregistered, the flags are ignored for the system resolver, 20 lookups a second and a 2-minute cache, and the log says which were ignored. -no-dns works everywhere.
- Traffic statistics are a registered feature as a whole: an unregistered installation is refused every query, one VIP as much as a group or a subnet, on the page, its CSV and the read-only accounts alike. The three-VIP allowance for unregistered installations is gone.
- Inventory: `sampling=` on device lines is gone. The sampling rate is always the one the device declares (sFlow in every sample, NetFlow v9 and IPFIX in their sampler options); a device that exports every packet is still marked `unsampled`. An older inventory with `sampling=` still loads: the word is ignored.

## 2.11.4

- Inventory: product lines are called VIP groups on every page. Picking the type VIP group, the form asks for the group's name (say "Product A public") and its VIPs; commas, full-width commas, 、 and spaces all separate them, and the line is saved in the standard form. The table shows the group's name and its VIPs.
- Traffic statistics: one query covers a month at most (it was 186 days), so a many-VIP group or a wide subnet never ties the database up; a longer custom range says so.

## 2.11.2

- Traffic statistics: the window is picked from presets matched to how the data is kept: today, 1 day, 3 days and 1 week (from the flow records, a point every 5 or 15 minutes, or every hour), 30 days, this month and last month (from the hourly summaries), or a custom range. A read-only account has today to 2 weeks.
- Traffic statistics: the 95th percentile of the total rate, with the point width it was taken over, beside the average and the peak; the peak and the percentile leave out buckets still filling and buckets from before there was any data.
- Traffic statistics: usage per day as bars, under the bandwidth chart, for windows of two days or more. The days are added up from the same buckets as the total, so they always add up to it exactly, and they are the viewer's calendar days.
- Traffic statistics: an empty window says there is no data in it, instead of a message about waiting for a device.

## 2.11.1

- Interface check: the Inbound and Outbound views of the bandwidth chart show the traffic into and out of the interface being inspected, as the overview does. Before, they split by the inventory's own networks and stayed on "not yet" on a network whose ranges were not in the inventory.
- Traffic statistics: the object list offers every address the inventory names (the product lines' VIPs and the named hosts), and several VIPs can be billed together, separated by commas (`203.0.113.10, 203.0.113.11`).
- Traffic statistics: this month's usage of the object, inbound plus outbound in GB, as one figure above the window's.

## 2.11.0

- Traffic statistics: a subnet (`10.0.0.0/24`) can be billed as a whole, beside one VIP and a product line; traffic inside the subnet is counted once (registered installations).
- Traffic statistics on an unregistered installation: one VIP, or a product line of at most three VIPs, per query; a larger line is answered with the limit.

- Licences are signed with ECDSA P-521 (message v4); Ed25519 licences are no longer accepted. A licence issued before this release must be issued again; the installation number does not change.
- A new installation gets a 128-bit random installation number instead of 8 digits; an existing installation keeps its number.

## 2.10.1

- Licence: the Additional Use Grant is now version 1.1. Production use is free for organizations of fewer than 50 people (before, 100); larger organizations register after 30 days of production use, as before.
- API: a from/to span of centuries no longer slips past the thirty-day limit (it overflowed when converted and came out negative). The rollup export and the traffic statistics, which had no limit, take at most 186 days.
- Export CSV: an interface name that starts like a spreadsheet formula (=, +, -, @) is written as text.
- Inventory: `speed=` takes 100M, 10G and the like; a speed that cannot be read is refused instead of quietly read as 0, which dropped the interface's utilisation.
- Manual: traffic statistics, product lines and read-only accounts are described in all 15 languages; the example inventory has a product line.

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
