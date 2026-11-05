# Roadmap

Work goes top to bottom, one small item a day (split anything larger).
Tick an item in the same pull request that completes it. Releases are
separate: merged work reaches users with the next release.


## Asked for since (done)
- [x] Top 66 opens on the two ends of each conversation
- [x] Sponsor banner as a strip across the top, built from the 728x90 banners
- [x] The evaluation mark on every page, a little denser
- [x] Live pages refresh themselves on a beat that follows the range
- [x] Network charts count traffic whose AS is unknown, so short ranges are not drawn as blocks
- [x] 95th-percentile billing draws the line its figure was taken across
- [x] Narrower top end for the map's line widths
- [x] A licence says whether it is made out to a person or an organisation
- [x] The ledger's addresses go; a device carries the networks written against it

## Reports (target 7 Nov)
- [x] Make `TestAttackFound` deterministic (it fails now and then with sampled scans)
- [x] `GET /api/report?from&to&if`: totals, average, peak and 95th percentile per interface, utilisation against interface speed, top hosts, applications and countries, findings by severity; with tests
- [ ] Report page: printable layout (print CSS), period and interface picker, in all 15 languages
- [ ] Report: CSV export of the same tables
- [ ] Report: comparison with the previous period (change in %, new top hosts)

## Alerts (target 28 Nov)
- [ ] Alert rules: bandwidth over a threshold on an interface for N minutes; new high-severity finding; stored in the data directory; with tests
- [ ] Alert delivery: webhook (JSON, retries)
- [ ] Alert delivery: e-mail over SMTP
- [ ] Alert delivery: syslog (RFC 5424)
- [ ] Settings page: alert rules and destinations, with a "send test" button
- [ ] Zabbix template (HTTP agent polling traffic66) and a page in the README

## Enterprise basics (target 19 Dec)
- [ ] Built-in HTTPS: `-tls-cert`/`-tls-key`, or a self-signed certificate on request
- [ ] Read-only user role
- [ ] Audit log of sign-ins and setting changes
- [ ] LDAP / Active Directory sign-in
