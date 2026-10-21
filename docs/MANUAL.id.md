[English](MANUAL.md) | [中文](MANUAL.zh.md) | [हिन्दी](MANUAL.hi.md) | [Español](MANUAL.es.md) | [العربية](MANUAL.ar.md) | [Français](MANUAL.fr.md) | [বাংলা](MANUAL.bn.md) | [Português](MANUAL.pt.md) | [Русский](MANUAL.ru.md) | **Bahasa Indonesia** | [اردو](MANUAL.ur.md) | [Deutsch](MANUAL.de.md) | [日本語](MANUAL.ja.md) | [Tiếng Việt](MANUAL.vi.md) | [한국어](MANUAL.ko.md)

# traffic66 — kolektor dan penganalisis trafik NetFlow, sFlow, dan IPFIX

Analitik flow untuk sFlow, NetFlow, dan IPFIX dalam satu program: siapa yang
memakai bandwidth, ke mana trafik mengalir, dan apakah angkanya cocok dengan
counter interface di perangkat itu sendiri, lewat antarmuka web maupun
antarmuka terminal.

Alternatif self-hosted untuk ntopng, ElastiFlow, pmacct dengan Grafana, atau
modul flow di PRTG dan SolarWinds NTA, untuk pemantauan jaringan
(network monitoring), pemantauan bandwidth (bandwidth monitoring), pemakai
trafik terbesar (top talkers), deteksi DDoS dan pemindaian, serta analisis
pcap, tanpa Elasticsearch, Kafka, atau database terpisah.

- Satu file executable untuk Windows, Linux, dan macOS; tanpa database yang perlu dipasang, bisa berjalan offline.
- sFlow v5, NetFlow v5/v9, dan IPFIX di port UDP mana pun, atau capture lokal dari sebuah interface.
- Mencocokkan angkanya dengan counter interface (sFlow atau SNMP) dan menjelaskan penyebab selisihnya.
- Menemukan pemindaian, tebakan kata sandi, pergerakan lateral, unggahan tidak biasa, flood, dan trafik daftar ancaman, juga melalui sampling.
- `traffic66 capture.pcap` menganalisis tangkapan paket tanpa pengaturan apa pun.
- 15 bahasa. Gratis untuk evaluasi dan untuk organisasi dengan kurang dari 100 orang ([lisensi](#licence)).

![Ringkasan: temuan terbuka, bandwidth per aplikasi dibanding waktu yang sama kemarin, klien dan layanan teratas](images/overview.png)

<sub>Semua tangkapan layar berasal dari `traffic66 demo`, jaringan perusahaan simulasi.</sub>

<a id="contents"></a>

## Daftar isi

1. [Mulai cepat](#1-quick-start)
2. [Contoh](#2-examples)
3. [User dan kata sandi](#3-users-and-passwords)
4. [Kirim flow dari perangkat Anda](#4-send-flows-from-your-devices)
5. [Pastikan flow masuk](#5-check-that-flows-arrive)
6. [Interface dan counter](#6-interfaces-and-counters)
7. [Nama, negara, dan daftar ancaman](#7-names-countries-and-threat-lists)
8. [Memakai antarmuka web](#8-using-the-web-ui)
9. [Pcap offline, antarmuka terminal, capture lokal](#9-offline-pcap-terminal-ui-local-capture)
10. [Opsi dan data](#10-options-and-data)
11. [Keamanan, kebutuhan sumber daya, pemecahan masalah](#11-security-sizing-troubleshooting)

<a id="1-quick-start"></a>

<a id="1-try-the-demo"></a>

## 1. Mulai cepat

1. Unduh arsip untuk sistem Anda: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Ekstrak lalu jalankan programnya:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: ekstrak `traffic66-windows-amd64.zip` lalu jalankan `traffic66.exe` (klik dua kali bisa).

macOS: jalankan dulu `xattr -dr com.apple.quarantine .` di folder itu.

Lalu:

- Antarmuka web: `http://<IP server>:8066`. Mendengarkan di semua antarmuka jaringan.
- Masuk: pengguna `admin`, kata sandi `traffic66`. Saat pertama masuk Anda wajib membuat kata sandi baru.
- Arahkan perangkat ke IP server: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (semua antarmuka; setiap port menerima semua protokol).

<a id="2-examples"></a>

<a id="2-install"></a>

## 2. Contoh

<a id="command-cheat-sheet"></a>

### Ringkasan perintah

| Tugas | Perintah |
|---|---|
| Kumpulkan sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (bawaan) | `./traffic66` |
| Hanya sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Hanya NetFlow v5/v9 di 2055 dan IPFIX di 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Port UDP lain | `./traffic66 -listen netflow=:9995,sflow=:6344` |
| Kolektor di satu alamat IP | `./traffic66 -listen sflow=10.0.0.5:6343,netflow=10.0.0.5:2055,ipfix=10.0.0.5:4739` |
| Antarmuka web di IP dan port tertentu | `./traffic66 -addr 10.0.0.5:8080` |
| Antarmuka web hanya di mesin ini | `./traffic66 -addr 127.0.0.1:8066` |
| Antarmuka web lewat HTTPS | `./traffic66 -tls` |
| Direktori data | `./traffic66 -data /var/lib/traffic66` |
| Simpan detail flow 7 hari | `./traffic66 -retention-days 7` |
| Memori: 20% RAM untuk basis data | `./traffic66 -memory 0.20` |
| Daftar kartu jaringan untuk tangkapan lokal | `./traffic66 interfaces` |
| Tangkap kartu jaringan mesin ini (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Tangkap kartu jaringan mesin ini (macOS) | `sudo ./traffic66 -capture en0 -listen ""` |
| Tangkap kartu jaringan mesin ini (Windows, perlu Npcap) | `.\traffic66.exe -capture Wi-Fi -listen ""` |
| Atur atau reset kata sandi admin (tambah -data DIR bila dipakai) | `./traffic66 passwd` |
| Kata sandi admin acak, ditampilkan | `./traffic66 passwd -generate` |
| Tambah pengguna atau ubah kata sandinya | `./traffic66 passwd -user alice` |
| Daftar pengguna | `./traffic66 passwd -list` |
| Kata sandi untuk sekali jalan (skrip, kontainer) | `TRAFFIC66_PASSWORD='S3cret-pass' ./traffic66` |
| Antarmuka terminal di mesin traffic66 | `./traffic66 tui` |
| Antarmuka terminal ke server jarak jauh | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |
| Bahasa antarmuka terminal | `./traffic66 tui -lang ja` |
| Analisis file tangkapan | `./traffic66 capture.pcap` |
| Kirim flow uji ke kolektor | `./traffic66 simulate -to 192.0.2.50` |
| Demo dengan perangkat simulasi | `./traffic66 demo` |
| Semua opsi | `./traffic66 serve -h` |

Bahasa antarmuka web: di bawah menu (tanpa flag). Selebihnya: `traffic66 serve -h`.

<a id="device-configuration"></a>

### Konfigurasi perangkat

`192.0.2.50` adalah traffic66, `192.0.2.1` perangkatnya; timeout dan laju sampling: [bagian 4](#4-send-flows-from-your-devices).

```
! Cisco IOS-XE, NetFlow v9 (IPFIX: add "export-protocol ipfix" and use "transport udp 4739")
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

```
# Huawei CloudEngine, sFlow
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

```
# Huawei AR/NE, NetStream (NetFlow v9)
ip netstream export version 9
ip netstream export host 192.0.2.50 2055
ip netstream timeout active 1
interface GigabitEthernet0/0/1
 ip netstream inbound
 ip netstream sampler fix-packets 1000 inbound
```

```
# H3C Comware 7, sFlow
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

```
# Juniper EX/QFX, sFlow
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow interfaces ge-0/0/0

# Arista EOS, sFlow
sflow sample 4096
sflow destination 192.0.2.50
sflow run
```

```
# MikroTik RouterOS 7, NetFlow v9 (IPFIX: version=ipfix port=4739)
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

### Buka firewall

```
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload   # RHEL, Fedora
sudo ufw allow proto udp to any port 6343,2055,4739 && sudo ufw allow 8066/tcp                          # Ubuntu, Debian
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow   # Windows
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

### Antarmuka web hanya di mesin ini, lewat SSH

```
./traffic66 -addr 127.0.0.1:8066                 # on the server
ssh -L 8066:127.0.0.1:8066 user@192.0.2.50        # on your PC, then open http://127.0.0.1:8066
```

### Sebagai layanan: Linux (systemd)

```
sudo mkdir -p /opt/traffic66 && curl -fsSL https://github.com/githubflyideas/traffic66/releases/latest/download/traffic66-linux-amd64.tar.gz | sudo tar xz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

`/etc/systemd/system/traffic66.service`:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
MemoryMax=2G
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN   # only for local capture

[Install]
WantedBy=multi-user.target
```

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf && sudo sysctl --system
sudo systemctl daemon-reload && sudo systemctl enable --now traffic66
journalctl -u traffic66 -f
```

### Sebagai layanan: Windows

```
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

### Sebagai layanan: macOS

Ekstrak ke `/usr/local/traffic66`, hapus atribut karantina, jalankan `traffic66 passwd -data "/Library/Application Support/traffic66"`, lalu mulai dari LaunchDaemon dengan `ProgramArguments` berisi program, `-data` dan direktori itu, serta `RunAtLoad` dan `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. User dan kata sandi

Instalasi baru masuk sebagai `admin` / `traffic66`, dan login pertama
meminta kata sandi baru sebelum menampilkan apa pun (demo tetap memakai
`traffic66`). Setelah itu, **Akun** di bagian bawah menu mengganti kata
sandi Anda; administrator (`admin`) juga menambah dan menghapus user serta
me-reset kata sandi mereka di sana. Login dengan LDAP / Active Directory
sedang dikembangkan.

User disimpan sebagai salted hash di `password` di direktori data. Hal yang
sama bisa dilakukan di mesin traffic66 dengan satu perintah (tambahkan
`-data …` jika traffic66 berjalan dengannya):

| Untuk | Perintah |
|---|---|
| Mengganti kata sandi `admin` | `traffic66 passwd` |
| Menambah `alice` atau mengganti kata sandinya | `traffic66 passwd -user alice` |
| Menghapus `alice` | `traffic66 passwd -user alice -delete` |
| Menampilkan daftar user | `traffic66 passwd -list` |

Perubahan langsung berlaku. Selain pengelolaan user, semua user punya hak
yang sama. Untuk skrip dan container, `TRAFFIC66_PASSWORD=…` (atau `-password`) hanya menerima `-user`
dengan kata sandi itu untuk run tersebut. Lima kali salah kata sandi dalam
satu menit memblokir alamat itu selama satu menit.

<a id="4-send-flows-from-your-devices"></a>

## 4. Kirim flow dari perangkat Anda

`192.0.2.50` adalah traffic66, `192.0.2.1` perangkatnya. Atur active timeout
ke 60 detik, biarkan perangkat NetFlow/IPFIX mengekspor sampler options-nya,
dan lakukan sampling **di setiap interface pada arah masuk** (atau hanya di
interface tepi): dengan begitu setiap paket dihitung sekali. Sampling rate
sFlow: sekitar 1:1000 untuk 1 Gb/s, 1:4096 untuk 10 Gb/s, 1:8192 untuk
40/100 Gb/s.

Konfigurasi untuk Cisco, Huawei, H3C, Juniper, Arista, MikroTik dan softflowd: [Contoh → Konfigurasi perangkat](#device-configuration).

<a id="5-check-that-flows-arrive"></a>

## 5. Pastikan flow masuk

**Pengaturan** mencantumkan dalam hitungan detik setiap perangkat yang
mengirim sesuatu: protokol, sampling rate, loss, interface yang disampelnya,
dan apa yang perlu diperbaiki bila status tidak hijau. Loss sFlow dipisah
menjadi loss di perjalanan (naikkan `net.core.rmem_max` jika `netstat -su`
menunjukkan error buffer) dan sampel yang dibuang oleh perangkat itu sendiri.

![Pengaturan: setiap perangkat beserta protokol, sampling, loss, dan apa yang perlu diperbaiki](images/sources.png)

Ada perangkat yang tidak muncul? Jalankan `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739`: jika tidak ada apa-apa, masalahnya di routing, firewall,
atau konfigurasi perangkat; jika paket ada tetapi **Pengaturan** kosong,
masalahnya di firewall lokal atau `-listen`. `traffic66 simulate -to 192.0.2.50`
dari mesin lain menguji jalurnya dengan perangkat simulasi.

<a id="6-interfaces-and-counters"></a>

## 6. Interface dan counter

Angka flow adalah estimasi (sampel × sampling rate). **Pencocokan antarmuka**
membandingkannya dengan counter interface perangkat (counter sFlow, atau
SNMP lewat baris `snmp` di Nama) dan menjelaskan penyebab selisihnya:
interface tidak disampel, trafik yang sama disampel dua kali, loss di
perjalanan, atau sampling rate yang tidak diketahui. Setiap interface punya
grafik bit/s dan grafik paket/s, ingress hijau dan egress biru, counter
berupa garis putus-putus.

Di setiap baris, **✎** mengatur nama dan label pendek (misalnya *uplink*),
dan **☆** menjadikannya interface bawaan (★), tempat halaman dibuka.

Perangkat yang hanya menyampel sebagian interface juga menampilkan ujung lain
dari flow tersebut. **Antarmuka lawan** ini dicantumkan paling akhir dengan
huruf abu-abu kecil: isinya hanya trafik yang melewati interface yang
disampel. Interface yang disampel diketahui dari sumber data sFlow atau field
flowDirection (IPFIX 61); tanpa itu, interface yang ada pada 90% trafik
sebuah perangkat.

![Pencocokan antarmuka: trafik setiap interface, dan estimasi flow di samping counter perangkat](images/interfaces.png)

traffic66 sudah memakai rate yang diterapkan perangkat, menunggu rate yang
belum diketahui, mengompensasi loss ekspor, menyebar flow panjang ke
menit-menitnya, dan menambahkan overhead Ethernet 18 byte per paket ke
NetFlow/IPFIX (`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Nama, negara, dan daftar ancaman

Klik alamat mana pun lalu pilih **Beri nama…**, atau gunakan
**Pengaturan → Nama**. Nama disimpan di `inventory.txt` di direktori data:

```
net    10.10.0.0/16    Office LAN                  # your networks
net    203.0.113.0/24  Public servers country=JP    # country: lines on the world map
device 192.0.2.1       Core router
device 192.0.2.9       Branch firewall unsampled    # exports every packet
device 192.0.2.20      Edge router sampling=1000    # rate it does not declare
iface  192.0.2.1 3     ISP uplink speed=1000000000 tag=uplink default
host   10.10.3.27      Finance PC
snmp   192.0.2.1       public                       # read counters over SNMPv2c
snmp   192.0.2.9       s3cret  10.99.0.9:161        # other management address
```

Rentang privat selalu dianggap milik Anda. Perubahan berlaku saat
**Simpan**, tanpa restart.

Negara dan jaringan (AS) langsung berfungsi dengan basis data Lite gratis
dari DB-IP ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **Pengaturan**
memperbaruinya, atau menerima file MaxMind GeoLite2, IPinfo Lite, atau
IPtoASN sebagai gantinya. Garis batas peta: [Natural Earth](https://www.naturalearthdata.com).

![Geografi & jaringan: trafik luar per negara di peta dunia](images/geo.png)

Daftar ancaman adalah file teks berisi satu alamat atau jaringan per baris
di `<data>/threats/<name>.txt` (misalnya Spamhaus DROP); restart setelah
mengubahnya. Kecocokan muncul di **Intel ancaman**.

![Intel ancaman: host internal yang mengirim data ke alamat di daftar ancaman](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Memakai antarmuka web

Setiap nilai di setiap halaman bisa diklik: **Tampilkan hanya ini** /
**Kecualikan ini** (filter berlaku di semua halaman), **Lihat catatan
flow-nya**, **Lihat detail** (halaman tentang satu host atau layanan),
**Beri nama…**, **Cari secara online**, **Salin**.

| Halaman | Yang ditampilkan |
|---|---|
| Ringkasan | Bandwidth interface yang dipilih; trafik per aplikasi (total, masuk, atau keluar) dibanding kemarin atau minggu lalu; temuan terbuka; klien dan layanan teratas |
| Top 66 | 66 percakapan teratas, bisa diurutkan menurut kolom mana pun, atau dikelompokkan per aplikasi, jaringan, segmen, perangkat, enkapsulasi, VLAN; 30 pihak teratas |
| Detail trafik | Diagram cincin: server dan kliennya (atau sebaliknya), serta layanan |
| Jalur trafik | Host → aplikasi → negara, atau klien → layanan → server, atau per jaringan |
| Pencocokan antarmuka | Setiap interface dari waktu ke waktu dibanding counter-nya; nama, label, bawaan |
| Catatan flow | Flow individual, live setiap 5 detik atau untuk rentang waktu mana pun |
| Temuan, Intel ancaman | Apa yang perlu diperhatikan ([di bawah](#findings)); trafik dengan alamat yang terdaftar |
| Geografi & jaringan | Peta dunia per negara, jaringan (AS) dari waktu ke waktu |
| Pengaturan | Perangkat, sampling, loss, SNMP, basis data, logo, nama |
| Analisis pcap offline, Pembersihan data | File tangkapan ([di bawah](#9-offline-pcap-terminal-ui-local-capture)); menghapus data lama |

Di atas halaman: **Antarmuka** (semua, atau satu interface yang disampel;
halaman trafik lalu hanya menampilkan trafik yang melewatinya), rentang
waktu (15 menit sampai 30 hari, atau kustom), refresh setiap 30 detik, dan
**Salin tautan** untuk tampilan yang persis sama. Bahasa dan lima tema warna
ada di bagian bawah menu. Grafik berakhir di titik data sudah lengkap: dengan
NetFlow/IPFIX, selambat perangkat mengekspor (paling lama 2 menit). Rentang
lebih dari 6 jam dimulai pada jam penuh; satu interface selama 7 atau 30 hari
membaca detail flow, sehingga lebih lambat dan hanya menjangkau sejauh detail
disimpan.

![Top 66: 66 percakapan teratas, bisa diurutkan menurut kolom mana pun](images/topn.png)

![Detail trafik: server dengan kliennya, dan layanan dengan servernya, sebagai diagram cincin](images/traffic.png)

![Detail satu host: temuan tentangnya, trafiknya, dengan siapa ia berbicara, layanan, negara, dan flow terbaru](images/detail.png)

![Jalur trafik: host mana memakai aplikasi apa menuju negara mana](images/paths.png)

![Ringkasan dalam bahasa Tionghoa](images/overview-zh.png)

<a id="findings"></a>

### Temuan

Diperiksa setiap 5 menit atas 10 menit terakhir; sesuatu yang berlangsung
satu jam adalah satu temuan yang terus bertambah.

| Temuan | Artinya |
|---|---|
| Pemindaian, pemindaian port | Probe kecil ke banyak host pada satu port, atau ke banyak port pada satu host |
| Tebak kata sandi | Banyak koneksi singkat ke layanan login |
| Pergerakan lateral | Berbagi file atau administrasi jarak jauh ke host internal yang sebelumnya tidak pernah menyediakannya |
| Unggahan tidak biasa | 100 MB dalam 10 menit ke alamat baru, tiga kali yang kembali |
| Flood | 20,000+ paket kecil/detik ke satu alamat, sepuluh kali laju biasanya |
| Daftar ancaman | Trafik dengan alamat yang terdaftar |

Dari dalam jaringan Anda tingkatnya tinggi, dari internet rendah. **Sudah
ditangani** menutup temuan, **Bukan masalah** membungkamnya untuk selamanya.
Pergerakan lateral dan unggahan butuh riwayat satu hari. Melalui sampling
1:4096, serangan dalam demo ditemukan seluruhnya; pemindaian yang sangat
kecil bisa tersembunyi di balik sampling.

![Temuan: setiap langkah serangan, ditemukan melalui sampling sFlow 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Pcap offline, antarmuka terminal, capture lokal

**Analisis pcap offline** menampilkan tangkapan paket (pcap, pcapng) dengan
halaman yang sama, terpisah dari data langsung: `traffic66 a.pcap b.pcapng`
berjalan di 127.0.0.1 dan membuka browser (maksimal 3 file, 3 GB; Ctrl+C
menghapus data yang diimpor), atau unggah maksimal 3 file berukuran 50 MB di
halaman itu. Satu file dianalisis pada satu waktu, masing-masing dalam
database tersendiri: **Analisis** pada baris sebuah file menampilkannya di
semua halaman, dan bilah di bagian atas beralih ke file lain. Analisis
bekerja pada flow, bukan isi paket.

![Analisis offline: file tangkapan beserta paket, flow, dan waktunya](images/sandbox.png)

**Antarmuka terminal**: `traffic66 tui` di mesin traffic66, atau
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Tiga halaman: **1 Teratas** (klien, server, aplikasi, port, jaringan (AS), dan negara/wilayah menurut laju; ← → berganti, Enter membuka aksi), **2 Antarmuka** (laju, utilisasi, dan rekonsiliasi per perangkat dan antarmuka, beserta sampling dan kehilangan tiap perangkat; Enter membentangkan tren dan host/aplikasi teratas), dan **3 Cari** (catatan flow untuk filter seperti `ip:10.0.0.5 net:10.0.0.0/8 asn:13335 port:443 cc:US dev:192.0.2.1`). Tombol: `/` filter, Backspace/`c` hapus/bersihkan, `t` periode (5 menit hingga 24 jam), `u` bit/dtk, byte, atau paket/dtk, `p` jeda, `+`/`-` interval penyegaran, `w` buka tampilan yang sama di peramban, `?` bantuan, `q` keluar; `-lang` memilih bahasa dan `NO_COLOR` mematikan warna.

![Antarmuka terminal: Teratas](images/tui-top.png)

![Antarmuka terminal: antarmuka](images/tui-interfaces.png)

**Capture lokal** membuat flow dari interface lokal, paling baik port yang
terhubung ke port mirror sebuah switch: `traffic66 interfaces` menampilkan
daftarnya, `-capture eth1` melakukan capture. Di Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux butuh root atau `setcap cap_net_raw,cap_net_admin+ep`, macOS butuh
root, Windows butuh [Npcap](https://npcap.com). Flow hasil capture berasal
dari perangkat `127.0.0.1`. Capture lokal tidak punya interface atau counter
perangkat, jadi **Pencocokan antarmuka** tidak punya apa pun untuk
dibandingkan.

<a id="10-options-and-data"></a>

## 10. Opsi dan data

`traffic66 -h` menampilkan semuanya. Yang paling sering dipakai:

| Opsi | Default | |
|---|---|---|
| `-data` | `traffic66-data` di sebelah program | direktori data |
| `-addr` | `:8066` | antarmuka web; `127.0.0.1:8066` hanya untuk mesin ini |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collector UDP; kosong berarti nonaktif |
| `-retention-days` | `30` | berapa hari detail flow disimpan; ringkasan disimpan 400 hari |
| `-memory` | `0.10` | porsi RAM untuk cache database |
| `-sampling-wait` | `5m` | berapa lama record menunggu sampling rate |
| `-capture` | | interface lokal (bisa diulang) |
| `-no-dns` | | tanpa reverse lookup |

Direktori data berisi `raw/` (detail, satu file per jam),
`traffic66.duckdb` (ringkasan dan counter), `password`, `inventory.txt`,
`license.json`, logo Anda, dan basis data. Backup dengan menghentikan
traffic66 lalu menyalinnya; upgrade dengan mengganti file program.
**Pembersihan data** menghapus data yang lebih lama dari 7–120 hari, atau
semuanya.

<a id="licence"></a>

### Lisensi

Kode sumber tersedia di bawah [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
dan [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (teks bahasa
Inggris yang mengikat): gratis untuk evaluasi dan untuk organisasi dengan
kurang dari 100 orang; organisasi yang lebih besar mendaftar setelah 30 hari
penggunaan produksi; menjual, meng-host untuk pihak lain, atau produk pesaing
memerlukan lisensi komersial. Tidak ada yang pernah dimatikan. Bagian bawah
setiap halaman menampilkan nomor instalasi 8 digit; kirim nomor itu kepada
pembuatnya, lalu letakkan `license.json` yang Anda terima di direktori data.
Kontak: <route66traffic66@gmail.com>.

Instalasi yang belum terdaftar menampilkan strip sponsor di bagian atas halaman. Hanya browser Anda yang memuatnya, dari GitHub ([ads-sponsor](https://github.com/githubflyideas/ads-sponsor)); server traffic66 tidak pernah menghubungi GitHub untuk itu dan tidak menyimpan salinan. Browser yang tidak dapat menjangkau GitHub menampilkan teks bawaan singkat sebagai gantinya. Tidak ada data tentang Anda atau instalasi yang dikirim. Mendaftar akan menghapusnya.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Keamanan, kebutuhan sumber daya, pemecahan masalah

Antarmuka web memakai HTTP biasa: di jaringan yang tidak tepercaya, gunakan
`-addr 127.0.0.1:8066` di belakang proxy TLS (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) atau tunnel SSH. Izinkan port UDP hanya dari perangkat
Anda. Community SNMP disimpan sebagai teks biasa; gunakan community
read-only.

Pada 5.000 flow/detik di 2 core: sekitar 12 GB disk per hari detail
(360 GB untuk 30 hari), seperenam satu core, memori 0,6–0,8 GB. Ringkasan
rentang waktu panjang butuh di bawah 0,2 detik; Top 66 semua percakapan
selama 1 jam sekitar 9 detik.

| Gejala | Solusi |
|---|---|
| "waiting for the sampling rate" | Ekspor sampler options, atau `sampling=N` / `unsampled` di baris perangkat |
| Lebih rendah dari counter | Interface tidak disampel, loss, atau active timeout lebih dari 60 detik |
| Lebih tinggi dari counter | Trafik yang sama disampel di dua interface atau dua perangkat |
| Lupa kata sandi | `traffic66 passwd` di mesin traffic66 |
| `Conflicting lock is held` | traffic66 lain memakai direktori data ini |
| `address already in use` | Pilih port lain dengan `-addr` atau `-listen` |
| Windows: "Windows protected your PC" (Windows melindungi PC Anda) | **More info** (info selengkapnya) → **Run anyway** (tetap jalankan) |

Build dari source: Go 1.26 dan compiler C, lalu `CGO_ENABLED=1 go build ./cmd/traffic66`.
