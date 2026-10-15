# Roadmap

Work goes top to bottom, one small item a day (split anything larger).
Tick an item in the same pull request that completes it. Releases are
separate: merged work reaches users with the next release.

## Benchmark (target 24 Oct)
- [ ] Gap list against ElastiFlow and Noction: what they have that we lack, marked "must have" or "not for us" (delivered privately, not in this repository)

## Reports (target 7 Nov)
- [ ] Make `TestAttackFound` deterministic (it fails now and then with sampled scans)
- [ ] `GET /api/report?from&to&if`: totals, average, peak and 95th percentile per interface, utilisation against interface speed, top hosts, applications and countries, findings by severity; with tests
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

## Japan (target 21 Nov)
- [ ] Japanese home page on the site, for company IT teams and system integrators
- [ ] Japanese product sheet: requirements, performance, tested devices, support policy (printable / PDF)
- [ ] Japanese how-to article draft: Zabbix shows a full uplink, traffic66 finds who (for Qiita / Zenn)
- [ ] Pricing page in yen and an enquiry form (waiting for prices)
- [ ] Enquiry → quote → licence issue, documented end to end
