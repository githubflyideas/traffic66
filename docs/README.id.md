[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | **Bahasa Indonesia** | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — kolektor dan penganalisis trafik NetFlow, sFlow, dan IPFIX

Analitik flow untuk sFlow, NetFlow, dan IPFIX dalam satu program: siapa yang
memakai bandwidth, ke mana trafik mengalir, dan apakah angkanya cocok dengan
counter interface di perangkat itu sendiri, lewat antarmuka web maupun
antarmuka terminal.

## Mulai cepat

```
# Linux, macOS: unduh ke ./traffic66 lalu mulai mengumpulkan
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: sama
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

Terminal menampilkan alamat antarmuka web serta IP dan port UDP tujuan perangkat Anda ([konfigurasi perangkat](MANUAL.id.md#device-configuration)). Masuk sebagai `admin` / `traffic66` lalu pilih kata sandi baru. Hanya ingin melihat? Demo dengan perangkat simulasi:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Port: TCP 8066 antarmuka web, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (setiap port UDP menerima semua protokol). Terminal menampilkan alamat tujuan untuk perangkat; masuk pertama kali meminta kata sandi baru.

macOS manual: lebih dulu `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows "melindungi PC Anda": **Info selengkapnya** → **Tetap jalankan**.

## Ringkasan perintah

| Tugas | Perintah |
|---|---|
| Kumpulkan sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (bawaan) | `./traffic66` |
| Hanya NetFlow v5/v9 di 2055 dan IPFIX di 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Hanya sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Antarmuka web di IP dan port tertentu | `./traffic66 -addr 10.0.0.5:8080` |
| Tangkap kartu jaringan mesin ini (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Atur atau reset kata sandi admin (tambah -data DIR bila dipakai) | `./traffic66 passwd` |
| Antarmuka terminal ke server jarak jauh | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

## Tangkapan layar

| | |
|---|---|
| <a href="images/overview.png"><img src="images/overview.png" alt="Ringkasan"></a><br>Ringkasan | <a href="images/topn.png"><img src="images/topn.png" alt="Top 66"></a><br>Top 66 |
| <a href="images/geo.png"><img src="images/geo.png" alt="Geografi & jaringan"></a><br>Geografi & jaringan | <a href="images/interfaces.png"><img src="images/interfaces.png" alt="Pencocokan antarmuka"></a><br>Pencocokan antarmuka |
| <a href="images/tui-top.png"><img src="images/tui-top.png" alt="TUI"></a><br>TUI | |

## Manual lengkap

di antarmuka web: **Bantuan** di menu; di GitHub: [MANUAL.id.md](MANUAL.id.md)

15 bahasa. Gratis untuk evaluasi dan untuk organisasi dengan kurang dari 100 orang ([lisensi](../LICENSE.md)).
