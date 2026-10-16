---
title: My Boss Asked "Who's Hogging the Bandwidth?" — I Handed Him One EXE and Got the Answer in 3 Minutes 😎
date: 2026-10-13
lede: Single file · No database to install · Up and running in 30 seconds
summary: The network is slow and the boss wants to know who is hogging the bandwidth. One executable that reads sFlow, NetFlow and IPFIX gives the answer in three minutes.
image: assets/img/overview.png
---

![The web overview page: bandwidth graph and top clients](assets/img/overview.png)

---

I've been doing network operations for five years now. What I fear most isn't outages. It's the message in the team chat: "The network has been slow lately. Who's hogging the bandwidth?" 🙃

My old routine went like this: log into the core switch → `show interface` → can't tell who's responsible → set up a mirror port and capture packets → spend two hours staring at captures...

Now my routine is just one thing: open the traffic66 web page.

## What is it?

In one sentence: it collects sFlow, NetFlow, and IPFIX flow data and tells you who is using the bandwidth, where the traffic is going, and whether it matches your interface counters.

## Why I keep it running on a server 👇

* ✅ **One executable.** Single-file builds for Windows, Linux, and macOS. No Elasticsearch, no Kafka, no separate database. It runs offline too.
* ✅ **Running in 30 seconds.** Unzip → run → open `127.0.0.1:8066` → log in with `admin / traffic66`. Done.
* ✅ **Want to see it first?** One command: `traffic66 demo` simulates a company network with a day of historical data, 4 devices, and a full attack scenario, so you can explore right away.
* ✅ **Two interfaces.** A web UI in the browser, plus a terminal UI on the server (`traffic66 tui`). Connect over SSH: Top, Interfaces and Search are on keys 1–3. No browser needed.
* ✅ **Interface reconciliation (this one is rare).** Flow statistics are estimates by nature. traffic66 places those estimates side by side with your device's interface counters (sFlow or SNMP). When they don't match, it tells you why: an interface isn't being sampled, traffic is being counted twice, or packets are being dropped along the way.
* ✅ **Geo and ASN out of the box.** Ships with the free DB-IP database. The world map shows which countries your traffic goes to. You can swap in MaxMind GeoLite2 if you prefer.
* ✅ **13 languages**, with native Chinese UI. No guessing at English menus.

## Deployment warnings (important)

* The web interface is plain HTTP. Don't expose it to the public internet. Bind to `-addr 127.0.0.1:8066` and put a TLS proxy in front, or access it through an SSH tunnel.
* Only allow UDP ports from the devices you're collecting from (6343 / 2055 / 4739).
* Capacity reference: on a 2-core machine at 5,000 flows/s, detailed records come to about 12 GB per day, roughly 360 GB over 30 days. Memory use is 0.6–0.8 GB.

## Licensing, stated plainly

Evaluation is free. Organizations with fewer than 100 people can use it in production for free. Larger organizations need to register for production use after 30 days. Registration doesn't disable any features.

---

**Let's talk in the comments:** What are you using to look at traffic today? ntopng or PRTG?
