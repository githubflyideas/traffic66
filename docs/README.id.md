[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | **Bahasa Indonesia** | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

Analitik flow untuk sFlow, NetFlow, dan IPFIX dalam satu program.
traffic66 menerima ekspor flow dari switch, router, dan firewall,
menyimpannya di database tertanam, lalu menunjukkan siapa yang memakai
bandwidth, ke mana trafik mengalir, dan apakah angkanya cocok dengan
counter interface di perangkat itu sendiri — lewat antarmuka web maupun
antarmuka terminal.

- Satu file executable untuk Windows, Linux, dan macOS. Tanpa database
  yang perlu dipasang, tanpa runtime, bisa berjalan offline.
- sFlow v5, NetFlow v5, NetFlow v9, dan IPFIX di port UDP mana pun;
  opsional capture lokal dari interface jaringan atau port mirror.
- Mencocokkan angkanya sendiri dengan counter interface (counter sFlow atau
  SNMP) dan menjelaskan penyebabnya bila berbeda.
- Menemukan pemindaian, tebakan kata sandi, pergerakan lateral, unggahan tidak
  biasa, flood, dan trafik daftar ancaman di dalam flow, juga melalui
  sampling, lalu mencantumkannya sebagai temuan untuk ditangani.
- Daftar Top 66, trafik dari waktu ke waktu per klien, server, layanan,
  interface, dan jaringan (AS), jalur trafik, negara, kecocokan dengan daftar
  ancaman, catatan flow, enkapsulasi (GRE, IPIP, VXLAN, GENEVE, MPLS).
- 13 bahasa di antarmuka web dan antarmuka terminal.

![Ringkasan: temuan terbuka, bandwidth per aplikasi dibanding minggu lalu, klien dan layanan teratas](images/overview.png)

<sub>Semua tangkapan layar berasal dari `traffic66 demo`, jaringan perusahaan simulasi yang bisa Anda jalankan sendiri (lihat [Coba demo](#1-try-the-demo)).</sub>

<a id="contents"></a>

## Daftar isi

1. [Coba demo](#1-try-the-demo)
2. [Instalasi](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [User dan kata sandi](#3-users-and-passwords)
4. [Kirim flow dari perangkat Anda](#4-send-flows-from-your-devices)
5. [Pastikan flow masuk](#5-check-that-flows-arrive)
6. [Menyamakan angka dengan counter interface](#6-make-the-numbers-match-the-interface-counters)
7. [Nama, SNMP, dan jaringan Anda sendiri](#7-names-snmp-and-your-own-networks)
8. [Negara, jaringan, dan daftar ancaman](#8-countries-networks-and-threat-lists)
9. [Memakai antarmuka web](#9-using-the-web-ui)
10. [Antarmuka terminal](#10-terminal-ui)
11. [Capture lokal](#11-local-capture)
12. [Opsi](#12-options)
13. [Data, backup, upgrade, uninstall](#13-data-backup-upgrade-uninstall)
14. [Keamanan](#14-security)
15. [Kebutuhan sumber daya](#15-sizing)
16. [Pemecahan masalah](#16-troubleshooting)
17. [Build dari source](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. Coba demo

Unduh arsip untuk sistem Anda dari
[halaman rilis](https://github.com/githubflyideas/traffic66/releases):

| Sistem | Arsip |
|---|---|
| Windows 10/11, Server 2016 atau lebih baru (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: distribusi apa pun dengan kernel 3.2 atau lebih baru, termasuk CentOS 7 dan Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: distribusi yang sama | `traffic66-linux-arm64.tar.gz` |
| macOS 11 atau lebih baru, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 atau lebih baru, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (baris kedua mengizinkan macOS menjalankan program yang diunduh dari
internet dan bukan dari App Store):

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows (PowerShell):

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

Buka http://127.0.0.1:8066 dan login sebagai `admin` / `try66`. Demo ini
membangun jaringan kantor kecil dengan riwayat satu hari dan trafik live dari
empat perangkat simulasi, lengkap dengan sebuah serangan: **Temuan**
menampilkan setiap langkahnya (pemindaian, pemindaian port, tebakan kata
sandi, pergerakan lateral, unggahan ke server kendali) serta flood ke situs
web publik. Klik **Detail** pada sebuah temuan, atau mulai dari **Ringkasan**,
klik sebuah host di **Klien teratas**, pilih **Lihat detail**, lalu terus
telusuri dengan mengeklik. Hentikan dengan Ctrl+C. Data demo disimpan di
`traffic66-demo` di sebelah program; hapus folder itu untuk memulai demo
dari awal.

Demo memakai port yang sama dengan instalasi sungguhan (8066, serta UDP
6343, 2055, 4739). Untuk menjalankannya berdampingan dengan instalasi
sungguhan, beri port lain:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

Di Windows, Anda juga bisa cukup mengklik dua kali `traffic66.exe`. Itu
menjalankan traffic66 sungguhan (bukan demo) dan membuka antarmuka web di
browser Anda; kata sandi start pertama ditampilkan di jendela hitam, dan
menutup jendela itu menghentikan traffic66. Jika Windows menampilkan
"Windows protected your PC" (Windows melindungi PC Anda), klik
**More info** (info selengkapnya) → **Run anyway** (tetap jalankan).

<a id="2-install"></a>

## 2. Instalasi

traffic66 hanya satu file. Menginstalnya berarti menaruhnya di suatu
tempat, memilih direktori data, mengatur kata sandi, membuka firewall, dan
menjalankannya saat boot. Contoh-contoh di sini memakai `192.0.2.50` untuk
mesin traffic66 dan `192.0.2.1` untuk router; ganti dengan alamat Anda.

Port:

| Port | Kegunaan |
|---|---|
| UDP 6343 | sFlow (default) |
| UDP 2055 | NetFlow (default) |
| UDP 4739 | IPFIX (default) |
| TCP 8066 | antarmuka web dan API |

Setiap port UDP menerima semua protokol, jadi perangkat boleh mengirim
NetFlow ke 6343 kalau itu lebih mudah. Ubah atau tambah port dengan
`-listen`.

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

Perintah terakhir meminta kata sandi untuk user `admin`.

Buat `/etc/systemd/system/traffic66.service`:

```
[Unit]
Description=traffic66 flow analytics
After=network-online.target
Wants=network-online.target

[Service]
User=traffic66
ExecStart=/opt/traffic66/traffic66 -data /var/lib/traffic66
Restart=on-failure
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
# only needed for local capture (-capture):
#AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

Jalankan, dan izinkan buffer UDP yang lebih besar agar lonjakan trafik
tidak hilang:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, dengan firewalld (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

atau dengan ufw (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

Ekstrak ke `C:\traffic66` dan atur kata sandi (PowerShell sebagai
Administrator):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Data disimpan di `C:\traffic66\traffic66-data`, di sebelah program.

Buka firewall:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

Untuk mencobanya di foreground, jalankan `C:\traffic66\traffic66.exe` dan
hentikan dengan Ctrl+C. Agar berjalan di background sejak boot, tanpa perlu
ada yang login, daftarkan sebagai startup task:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` itu penting: tanpanya Windows
menghentikan task setelah tiga hari. Hentikan dengan `Stop-ScheduledTask -TaskName
traffic66`, hapus dengan `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Buat `/Library/LaunchDaemons/traffic66.plist`:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>traffic66</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/traffic66/traffic66</string>
    <string>-data</string>
    <string>/Library/Application Support/traffic66</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardErrorPath</key><string>/Library/Logs/traffic66.log</string>
</dict>
</plist>
```

Menjalankan, lalu menghentikannya lagi:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

Jika firewall macOS aktif, izinkan koneksi masuk untuk traffic66 di
System Settings → Network → Firewall → Options.

<a id="3-users-and-passwords"></a>

## 3. User dan kata sandi

**Singkatnya:** user dan kata sandi disimpan di satu file, `password`, di
direktori data. Jangan pernah mengeditnya secara manual: perintah
`traffic66 passwd` menambah, mengubah, menampilkan daftar, dan menghapus user.
Buka `http://<traffic66 machine>:8066` dan login dengan salah satunya.

<a id="the-first-sign-in"></a>

### Login pertama

Pada start pertama, traffic66 membuat user `admin` dengan kata sandi acak
dan menampilkannya sekali:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Dijalankan dengan klik dua kali di Windows: di jendela hitam.
- Dijalankan di terminal: di terminal itu.
- Service Linux: `journalctl -u traffic66 | grep "first start"`
- Service macOS: `grep "first start" /Library/Logs/traffic66.log`

Terlewat? Atur yang baru dengan `traffic66 passwd` (lihat di bawah). Jika
Anda sudah mengatur kata sandi dengan `traffic66 passwd` sebelum start
pertama, seperti pada langkah instalasi di atas, tidak ada yang dibuat
otomatis.

<a id="where-the-users-are-stored"></a>

### Tempat user disimpan

Di file `password` di direktori data:

| Cara traffic66 berjalan | File |
|---|---|
| Diekstrak dan dijalankan dari foldernya (default) | `traffic66-data/password` di samping program |
| Service Linux (bagian 2) | `/var/lib/traffic66/password` |
| Startup task Windows (bagian 2) | `C:\traffic66\traffic66-data\password` |
| Service macOS (bagian 2) | `/Library/Application Support/traffic66/password` |
| Demo | `traffic66-demo/password` di samping program |

Satu baris per user. Kata sandi disimpan sebagai salted hash, jadi tidak ada
yang bisa membacanya kembali dari file, termasuk Anda; jika lupa kata sandi,
atur yang baru. File ini hanya bisa dibaca oleh pemiliknya.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### Mengelola user

Jalankan perintah berikut di mesin traffic66:

| Untuk | Perintah |
|---|---|
| Mengganti kata sandi `admin` | `traffic66 passwd` |
| Menambah user `alice`, atau mengganti kata sandinya | `traffic66 passwd -user alice` |
| Menghapus user `alice` | `traffic66 passwd -user alice -delete` |
| Menampilkan daftar user | `traffic66 passwd -list` |
| Mengatur kata sandi acak dan mencetaknya | `traffic66 passwd -generate` (dengan `-user` untuk user lain) |

- Perintah ini meminta kata sandi baru dua kali dan tidak menampilkan apa
  yang Anda ketik. Gunakan minimal 8 karakter.
- Jika traffic66 berjalan dengan `-data`, tambahkan `-data` yang sama ke
  perintah. Untuk service Linux dari bagian 2:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Di Windows (PowerShell sebagai Administrator):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- Perubahan langsung berlaku tanpa restart: kata sandi baru berlaku pada
  login berikutnya, dan user yang dihapus otomatis logout dari browser yang
  sedang terbuka.
- User terakhir yang tersisa tidak bisa dihapus; tambahkan user lain dulu.
- Semua user melihat dan bisa mengubah hal yang sama; tidak ada role.

<a id="passwords-for-scripts-and-containers"></a>

### Kata sandi untuk skrip dan container

`TRAFFIC66_PASSWORD=…` di environment, atau `-password …` di command line,
membuat traffic66 hanya menerima satu user untuk run itu: user yang disebut
di `-user` (default `admin`) dengan kata sandi tersebut. File `password`
lalu diabaikan dan tidak diubah. Utamakan environment variable: command line
bisa dilihat user lain di mesin yang sama.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

Setelah lima kali salah kata sandi dalam satu menit, alamat tersebut
diblokir selama satu menit.

<a id="4-send-flows-from-your-devices"></a>

## 4. Kirim flow dari perangkat Anda

Arahkan setiap perangkat ke mesin traffic66. Perintahnya berbeda antar
model dan versi software; periksa manual perangkat Anda. Di semua contoh,
`192.0.2.50` adalah traffic66 dan `192.0.2.1` alamat perangkat itu sendiri.

Saran umum:

- Atur active flow timeout ke 60 detik. Timeout yang lebih lama membuat
  trafik datang terlambat dalam gumpalan besar.
- Jika perangkat melakukan sampling NetFlow/IPFIX, aktifkan ekspor sampler
  options agar rate-nya diketahui. traffic66 menahan record sampai rate
  tersebut datang, bukan menghitungnya 1:1.
- Lakukan sampling di semua interface atau hanya di interface tepi, dalam
  satu arah. Sampling trafik yang sama saat masuk dan saat keluar akan
  menghitungnya dua kali; **Pencocokan antarmuka** akan menunjukkan hal ini.
- Sampling rate sFlow: sekitar 1:1000 untuk link 1 Gb/s, 1:4096 untuk
  10 Gb/s, 1:8192 untuk 40/100 Gb/s.

Cisco IOS / IOS-XE (Flexible NetFlow):

```
flow exporter T66
 destination 192.0.2.50
 transport udp 2055
 export-protocol netflow-v9
 option sampler-table
flow monitor T66
 exporter T66
 record netflow ipv4 original-input
 cache timeout active 60
interface GigabitEthernet0/0/0
 ip flow monitor T66 input
```

Cisco NX-OS (sFlow):

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS (sFlow):

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX (sFlow):

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

Huawei CloudEngine (sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50
interface 10GE1/0/1
 sflow sampling collector 1
 sflow sampling rate 4096
 sflow sampling inbound
 sflow counter collector 1
 sflow counter interval 30
```

H3C Comware (sFlow):

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7 (NetFlow v9 / IPFIX):

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 atau lebih baru (NetFlow v9):

```
config system netflow
    config collectors
        edit 1
            set collector-ip 192.0.2.50
            set collector-port 2055
        next
    end
end
config system interface
    edit port1
        set netflow-sampler both
    next
end
```

Server dan host Linux, dengan softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Pastikan flow masuk

Buka **Sumber**. Setiap perangkat yang mengirim sesuatu muncul dalam
hitungan detik, beserta protokol, sampling rate, loss, paket terakhir, dan
statusnya. Jika status tidak hijau, teks di sebelahnya menjelaskan apa yang
salah dan apa yang perlu diubah.

![Sumber: setiap perangkat beserta protokol, sampling, loss, dan apa yang perlu diperbaiki](images/sources.png)

Jika sebuah perangkat tidak muncul:

1. Pantau paket di mesin traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Kalau tidak ada apa-apa, paket tidak sampai ke mesin: periksa
   konfigurasi perangkat, routing, dan firewall di sepanjang jalur.
2. Paket datang tetapi **Sumber** tetap kosong: firewall lokal membuangnya
   (lihat [Instalasi](#2-install)), atau traffic66 mendengarkan di port
   lain (`-listen`).
3. Untuk menguji jalur dari mesin lain tanpa menyentuh perangkat, jalankan
   `traffic66 simulate -to 192.0.2.50` di sana selama beberapa detik.
   Perintah ini mengirim sFlow, NetFlow, dan IPFIX dari perangkat simulasi;
   perangkat itu lalu muncul di **Sumber** dan di data, jadi sebaiknya
   lakukan ini di instalasi uji.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Menyamakan angka dengan counter interface

Angka flow adalah estimasi: jumlah paket sampel dikali sampling rate.
traffic66 membandingkannya dengan counter interface milik perangkat dan
menampilkan selisihnya di **Pencocokan antarmuka**, beserta kemungkinan
penyebabnya bila selisih itu lebih besar daripada yang bisa dijelaskan oleh
sampling saja.

![Pencocokan antarmuka: trafik setiap interface, dan estimasi flow di samping counter perangkat](images/interfaces.png)

Agar ada counter untuk dibandingkan:

- Perangkat sFlow mengirimkannya sendiri bila interval counter diatur
  (`sflow counter interval 30` dan sejenisnya).
- Untuk perangkat NetFlow dan IPFIX, tambahkan baris `snmp` di
  **Sumber → Nama** (lihat [Nama](#7-names-snmp-and-your-own-networks)).
  traffic66 lalu membaca counter interface setiap menit.

Yang sudah dilakukan traffic66 agar angkanya cocok: memakai sampling rate
yang benar-benar diterapkan perangkat, menahan record NetFlow/IPFIX sampai
sampling rate diketahui, mengompensasi paket ekspor yang hilang di jalan,
menyebar flow panjang ke menit-menit selama flow itu berlangsung, dan
menambahkan overhead Ethernet 18 byte per paket ke hitungan byte
NetFlow/IPFIX (counter interface menyertakannya, hitungan flow di layer IP
tidak; ubah dengan `-l2-overhead`).

Penyebab umum selisih yang tersisa, semuanya dilaporkan di
**Pencocokan antarmuka**: sebagian interface tidak di-sampling, trafik
yang sama di-sampling di dua interface, paket ekspor hilang sebelum
mencapai traffic66, atau sampling rate belum diketahui.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. Nama, SNMP, dan jaringan Anda sendiri

Cara tercepat memberi nama host atau perangkat: klik alamatnya di halaman mana
pun lalu pilih **Beri nama…**. Ketik namanya dan tekan Enter; nama langsung
disimpan dan ditampilkan di mana-mana menggantikan alamat polosnya.

Untuk jaringan, interface, dan SNMP, **Sumber → Nama** di antarmuka web
menerima satu entri per baris. Isinya
disimpan sebagai `inventory.txt` di direktori data, jadi Anda juga bisa
mengedit file itu langsung (lihat `inventory.txt.example`). Setiap baris
bersifat opsional.

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers

# device names; "unsampled" if it exports every packet (1:1)
device 192.0.2.1     Core router
device 192.0.2.9     Branch firewall unsampled

# interface names, by device address and ifIndex; speed in bits per second
iface  192.0.2.1 3   ISP uplink speed=1000000000

# host names shown instead of addresses
host   10.10.3.27    Finance PC

# read interface counters over SNMPv2c (IF-MIB 64-bit counters)
snmp   192.0.2.1     public
snmp   192.0.2.9     s3cret  10.99.0.9:161
```

- `net`: rentang privat (10/8, 172.16/12, 192.168/16, 100.64/10) selalu
  dianggap milik Anda. Tambahkan rentang publik Anda agar trafik ke dan
  dari rentang itu juga dihitung sebagai milik Anda; namanya muncul di
  **Top 66** saat dikelompokkan per segmen dan di jalur trafik per jaringan.
- `snmp <device> <community> [<management address>[:port]]`: device adalah
  alamat asal flow. Tambahkan alamat manajemen bila perangkat menjawab SNMP
  di alamat lain. Deskripsi interface yang dibaca lewat SNMP dipakai
  sebagai nama, kecuali Anda menamai interface itu dengan `iface`.
  Izinkan mesin traffic66 di access list SNMP perangkat.
- Perubahan berlaku saat Anda mengeklik **Simpan**; tidak perlu restart.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Negara, jaringan, dan daftar ancaman

Negara dan nama jaringan (AS) membutuhkan basis data yang memetakan alamat ke
keduanya. Unggah di antarmuka web: **Sumber → Basis data negara dan jaringan → Unggah file basis data…**.
File diperiksa, disimpan di direktori data, dan langsung dipakai untuk trafik
baru; tidak perlu restart. Trafik yang sudah tersimpan tetap memakai negara
saat ia disimpan.

File yang diterima:

| File | Memberikan | Tempat mendapatkannya |
|---|---|---|
| DB-IP Lite country atau ASN, `.mmdb` | negara, atau nomor dan nama AS | gratis, tanpa akun: [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country atau ASN, `.mmdb` | negara, atau nomor dan nama AS | gratis dengan akun MaxMind: [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| Tabel IP-to-ASN, `.tsv` atau `.tsv.gz` | nomor AS, nama AS, dan negara | gratis: [iptoasn.com](https://iptoasn.com) (`ip2asn-combined.tsv.gz`) |

Unggah basis data negara dan basis data ASN untuk mendapatkan keduanya; bila
beberapa dimuat, file `.mmdb` didahulukan untuk isi yang dimilikinya. Versi
baru terbit setiap bulan: unggah file baru dengan cara yang sama untuk
mengganti yang lama.

Tanpa antarmuka web, salin file ke direktori data sebagai `country.mmdb`,
`asn.mmdb`, atau `asn.tsv.gz` lalu restart traffic66.

Daftar ancaman adalah file teks biasa berisi satu alamat atau jaringan per
baris (teks setelah `#` atau `;` diabaikan), disimpan sebagai
`<data directory>/threats/<name>.txt`, misalnya:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Restart traffic66 setelah menambah atau mengubah daftar. Kecocokan muncul
di **Intel ancaman**, dikelompokkan per nama daftar.

![Intel ancaman: host internal yang mengirim data ke alamat di daftar ancaman](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Memakai antarmuka web

Anda jarang perlu mengetik. Setiap nilai di setiap halaman — alamat, port,
aplikasi, negara, perangkat — bisa diklik:

- **Tampilkan hanya ini** / **Kecualikan ini** menambahkan filter. Filter
  muncul di bawah bar atas dan berlaku di semua halaman sampai Anda
  menghapusnya.
- **Lihat catatan flow-nya** membuka flow-flow individual yang cocok.
- **Lihat detail** (host, perangkat, dan layanan) membuka halaman tentang satu
  host atau layanan itu: trafiknya dari waktu ke waktu per aplikasi, dengan
  siapa ia berbicara, layanan atau klien mana, negara, dan flow terbarunya.
  Setiap nilai di sana bisa diklik lagi, jadi Anda bisa terus menelusuri;
  tombol Back di browser membawa Anda kembali.
- **Beri nama…** (host dan perangkat) memberi alamat itu sebuah nama, yang
  sejak itu ditampilkan di mana-mana.
- **Cari secara online** membuka alamat atau AS tersebut di situs lookup
  publik.
- **Salin** menyalin nilainya.

Halaman:

| Halaman | Pertanyaan yang dijawab |
|---|---|
| Ringkasan | Berapa trafik sekarang dan dibanding minggu lalu, per aplikasi; temuan terbuka; arah dan protokol; klien dan layanan teratas |
| Top 66 | Terbuka di **Pihak teratas**: trafik per layanan dari waktu ke waktu, dan 30 klien dan server teratas berdampingan dengan trafik, paket, dan catatan flow, di atas satu baris untuk semua trafik. **Tabel** adalah satu tabel 66 teratas: secara default percakapan (klien, server, layanan, negara). Setiap judul kolom mengurutkan; kolom angka (lalu lintas, paket, rata-rata paket, flow) memilih ulang 66 teratas dari semua lalu lintas dalam rentang, sehingga rata-rata paket terkecil menemukan pemindaian dan banjir. **Kelompokkan menurut** beralih ke aplikasi, jaringan, segmen, perangkat, enkapsulasi, dan VLAN |
| Detail trafik | Klien, server, dan layanan dari waktu ke waktu, dalam bit/s dan paket/s: 8 teratas dari masing-masing, dan berapa jumlahnya |
| Temuan | Apa yang perlu diperhatikan: pemindaian, tebakan kata sandi, pergerakan lateral, unggahan tidak biasa, flood, dan trafik daftar ancaman ([selengkapnya](#findings)) |
| Jalur trafik | Host mana memakai aplikasi apa menuju negara mana: 8 host tersibuk, sisanya sebagai Lainnya. **Klien → server** menampilkan klien → layanan → server; **Per segmen** menampilkan jaringan, bukan host |
| Geografi & jaringan | Jaringan (AS) asal dan tujuan trafik, dari waktu ke waktu dalam bit/s dan paket/s; trafik per negara dan per jaringan |
| Intel ancaman | Host yang berkomunikasi dengan alamat di daftar ancaman Anda, dan berapa banyak yang mereka kirim |
| Catatan flow | Berapa banyak catatan flow dan kapan (satu batang per interval), dan catatannya sendiri, terbaru di atas, per halaman, dengan kolom yang bisa dipilih |
| Pencocokan antarmuka | Trafik setiap interface dari waktu ke waktu (masuk dan keluar, bit/s dan paket/s), dan angka flow di samping counter interface, yang paling buruk di atas, beserta alasannya |
| Sumber | Perangkat, sampling, loss, collector, SNMP, basis data negara dan jaringan, logo, dan **Nama** |

Di atas halaman: rentang waktu (15 menit sampai 30 hari), kotak pencarian
opsional, refresh otomatis setiap 30 detik, dan **Salin tautan**, yang
menyalin tautan ke tampilan saat ini secara persis (halaman, rentang waktu,
dan filter) untuk dikirim ke rekan kerja. Di bawahnya, **Perangkat**,
**Klien**, **Server**, dan **Layanan** mencantumkan nilai tersibuk dalam
rentang waktu: pilih satu, atau ketik satu, untuk memfilter setiap halaman;
kosongkan kotaknya untuk menghapus filter. Bahasa mengikuti browser; ubah di
bagian bawah menu.

Grafik dari waktu ke waktu menampilkan 8 nilai terbesar dengan warna tetap
dan sisanya sebagai Lainnya; legenda memberikan total setiap nilai dan bisa
diklik seperti nilai lainnya. Grafik klien dan server tidak menggambar
sisanya, karena dengan ribuan host sisanya akan meratakan 8 teratas; legenda
tetap memberikan totalnya.

Rentang yang lebih panjang dari 6 jam dimulai pada jam penuh, sehingga
setiap angka di halaman menghitung waktu yang persis sama: "24 jam" mencakup
24 jam penuh terakhir ditambah jam yang sedang berjalan. Top 66 untuk rentang
ini diambil dari ringkasan per jam; filter tidak tersedia di sana, dan
halaman memberi tahu hal itu. Pilih rentang yang lebih pendek untuk
memfilter. Percakapan selalu dibaca dari detail flow, jadi untuk rentang
panjang dengan laju flow tinggi bisa butuh waktu; satu jam paling
cepat.

Menu samping menunjukkan berapa banyak disk yang dipakai data dan berapa yang
masih kosong; arahkan kursor ke ruang kosong untuk melihat berapa yang
dibutuhkan detail untuk hari-hari yang disimpan pada laju saat ini
(diperkirakan setelah ada data satu hari).

Untuk menampilkan logo Anda sendiri di halaman masuk dan di bagian atas menu,
gunakan **Sumber → Logo → Unggah logo…**: PNG, SVG, JPEG, WebP, atau GIF, hingga
1 MB, paling baik 272 × 92 piksel (ukuran lain diskalakan agar pas).
**Pakai logo bawaan** mengembalikan logo traffic66.

<a id="findings"></a>

### Temuan

**Temuan** mencantumkan apa yang ditemukan traffic66 di dalam flow, yang
paling serius lebih dulu. Ia memeriksa 10 menit terakhir setiap 5 menit;
sesuatu yang berlangsung satu jam adalah satu temuan yang terus bertambah,
bukan temuan baru di setiap pemeriksaan.

| Temuan | Artinya | Tingkat |
|---|---|---|
| Pemindaian | Satu alamat mengirim probe kecil ke banyak alamat pada satu port (TCP atau ping) | Tinggi dari dalam jaringan Anda, rendah dari internet |
| Pemindaian port | Satu alamat mengirim probe kecil ke banyak port pada satu host | Tinggi dari dalam, rendah dari internet |
| Tebak kata sandi | Banyak koneksi singkat ke layanan login (SSH, RDP, SMB, database, dan lainnya) | Tinggi dari dalam, rendah dari internet |
| Pergerakan lateral | Di dalam jaringan Anda, sesi berbagi file atau administrasi jarak jauh (SMB, RDP, SSH, WinRM, VNC) ke host yang sebelumnya tidak pernah menyediakan layanan itu | Tinggi |
| Unggahan tidak biasa | Host internal mengirim jauh lebih banyak daripada yang diterimanya (100 MB dalam 10 menit, tiga kali yang diterimanya) ke alamat yang belum pernah bertukar data dengannya | Tinggi |
| Flood | 20,000 paket kecil atau lebih per detik ke satu alamat, sepuluh kali laju biasanya | Sedang |
| Daftar ancaman | Trafik dengan alamat di salah satu daftar ancaman Anda | Tinggi bila host Anda terhubung ke alamat itu, rendah bila alamat dalam daftar mengetuk dari luar |

Setiap temuan menyebutkan siapa melakukan apa terhadap siapa, kapan dan berapa
lama, beserta angka di baliknya dan cara data di-sampling. **Detail** membuka
halaman host, yang juga mencantumkan temuan tentang host itu.
**Sudah ditangani** menutup temuan; jika terjadi lagi, temuan baru dibuka.
**Bukan masalah** menutupnya untuk selamanya: temuan itu tidak akan pernah
dilaporkan lagi. Angka merah di samping **Temuan** di menu samping menghitung
temuan terbuka bertingkat tinggi dan sedang dalam 24 jam terakhir.

Pergerakan lateral dan unggahan tidak biasa perlu tahu apa yang normal, jadi
keduanya baru dilaporkan setelah ada riwayat satu hari. Saat pertama kali
dijalankan, traffic66 belajar dari riwayat yang sudah ada.

Dengan data hasil sampling (sFlow, NetFlow dengan sampling) aturan menghitung
apa yang terlihat di sampel dan meminta jumlah yang lebih sedikit, tetapi
setiap sampel harus tampak seperti satu probe singkat, sehingga host normal
yang sibuk tidak memicunya. Apa yang disembunyikan sampling tidak bisa
ditemukan: di balik sampling 1:4096, pemindaian terhadap beberapa puluh host
mengirim terlalu sedikit paket untuk terlihat. Serangan dalam demo melewati
switch yang melakukan sampling 1:4096 dan ditemukan seluruhnya; satu hari
trafik normal demo tidak menghasilkan temuan apa pun kecuali pemindai internet
yang mengetuk situs web.

![Temuan: setiap langkah serangan, ditemukan melalui sampling sFlow 1:4096](images/findings.png)

![Top 66, Pihak teratas: trafik per layanan, dan 30 klien dan server teratas dengan satu baris untuk semua trafik](images/topn.png)

![Detail trafik: klien, server, dan layanan dari waktu ke waktu, dalam bit/s dan paket/s](images/traffic.png)

![Detail satu host: temuan tentangnya, trafiknya, dengan siapa ia berbicara, layanan, negara, dan flow terbaru](images/detail.png)

![Jalur trafik: host mana memakai aplikasi apa menuju negara mana](images/paths.png)

Ringkasan yang sama dalam bahasa Tionghoa; setiap halaman tersedia dalam 13 bahasa:

![Ringkasan dalam bahasa Tionghoa](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. Antarmuka terminal

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

Di mesin traffic66, `traffic66 tui` login sendiri bila bisa membaca
direktori data (beri `-data` jika bukan direktori default). Jika traffic66
berjalan sebagai user lain, seperti halnya service, gunakan `-user` dan
`-password`. `-lang` memilih bahasa (`en`, `zh`, `hi`,
`es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

Tombol: 1–8 halaman, ↑↓ pilih, Enter aksi untuk nilai terpilih, f tampilkan
hanya ini, x kecualikan, / cari, t rentang waktu, c hapus filter, w buka
tampilan yang sama di browser, q keluar.

![Antarmuka terminal: ringkasan](images/tui-overview.png)

![Antarmuka terminal: percakapan Top 66](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Capture lokal

Selain menerima ekspor flow, traffic66 bisa membuat flow sendiri dari paket di
interface jaringan mesin tempat ia berjalan. Apa yang terlihat bergantung pada
interface-nya:

| Interface | Yang dilihat traffic66 |
|---|---|
| Port jaringan cadangan yang terhubung ke port mirror (SPAN) sebuah switch | Semua trafik yang di-mirror switch: seluruh jaringan atau uplink |
| Ethernet atau Wi-Fi milik mesin itu sendiri | Hanya trafik mesin ini sendiri |

Adapter Wi-Fi tidak bisa melihat trafik perangkat lain. Untuk melihat seluruh
jaringan Wi-Fi, biarkan router atau access point mengekspor flow (bagian 4),
atau mirror port switch tempat access point terhubung.

<a id="windows-1"></a>

### Windows

1. Pasang [Npcap](https://npcap.com) dengan opsi default-nya. Jika Anda
   mencentang "Restrict Npcap driver's access to Administrators only",
   jalankan traffic66 sebagai Administrator.
2. Tampilkan daftar interface (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   Kolom Name adalah nama koneksi dari pengaturan jaringan Windows; interface
   yang sedang dipakai punya alamat.
3. Capture di Wi-Fi, berdasarkan nama atau nomor:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Beri tanda kutip pada nama yang mengandung spasi: `-capture "Ethernet 2"`.
   Ulangi `-capture` untuk capture di beberapa interface. Tambahkan `-listen=`
   jika Anda hanya ingin capture tanpa collector flow. Untuk startup task di
   bagian 2, tambahkan opsinya ke `-Argument`:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

Capture butuh root, atau capability `CAP_NET_RAW` dan `CAP_NET_ADMIN`: baris
`setcap` di atas, atau baris `AmbientCapabilities` di unit systemd di bagian 2.
Interface Wi-Fi biasanya bernama `wlan0` atau `wlp…`.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

Capture butuh root; tidak ada yang perlu dipasang. Di MacBook, `en0` adalah
Wi-Fi.

<a id="checking-that-it-works"></a>

### Memeriksa apakah capture berjalan

**Sumber** mencantumkan setiap interface yang di-capture beserta metode
capture dan jumlah paket yang terlihat. Flow-nya tampil seolah berasal dari
perangkat `127.0.0.1` (mesin ini), di setiap halaman, sama seperti flow
perangkat lain. Paket yang terlihat dua kali (misalnya di dua port mirror)
dihitung dua kali.

<a id="12-options"></a>

## 12. Opsi

`traffic66 -h` dan `traffic66 <command> -h` menampilkan semuanya.

Perintah:

| Perintah | |
|---|---|
| `traffic66` | mengumpulkan flow dan menyajikan antarmuka web |
| `traffic66 demo` | sama, dengan jaringan simulasi |
| `traffic66 tui` | antarmuka terminal untuk traffic66 yang sedang berjalan |
| `traffic66 passwd` | menambah, mengubah, menampilkan daftar, atau menghapus user (lihat [User dan kata sandi](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | mengirim ekspor simulasi ke collector |
| `traffic66 interfaces` | menampilkan interface untuk capture lokal |
| `traffic66 version` | mencetak versi |

Opsi untuk `traffic66` dan `traffic66 demo`:

| Opsi | Default | |
|---|---|---|
| `-addr` | `:8066` | alamat antarmuka web; `127.0.0.1:8066` agar hanya bisa diakses dari mesin ini |
| `-data` | `traffic66-data` di sebelah program | direktori data |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collector UDP dalam format `name=address`, dipisah koma; kosong berarti nonaktif |
| `-user` | `admin` | nama user yang dibuat pada start pertama, dan user yang dikenai `-password` |
| `-password` | tidak diatur | hanya menerima `-user` dengan kata sandi ini untuk run ini, mengabaikan file `password` (juga `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | berapa hari detail flow disimpan; ringkasan disimpan 400 hari |
| `-memory` | `0.10` | porsi memori fisik untuk cache database, dan sebesar itu lagi sebagai batas lunak untuk bagian program lainnya (masing-masing minimal 256 MB) |
| `-l2-overhead` | `18` | byte per paket yang ditambahkan ke hitungan byte NetFlow/IPFIX |
| `-sampling-wait` | `5m` | berapa lama record menunggu sampling rate |
| `-capture` | | capture di interface lokal (bisa diulang) |
| `-inventory` | `<data>/inventory.txt` | file nama |
| `-asn` | `<data>/asn.tsv.gz` | tabel IP-to-ASN (file `.mmdb`: unggah, atau `<data>/country.mmdb` dan `<data>/asn.mmdb`) |
| `-threat` | `<data>/threats/*.txt` | daftar ancaman tambahan dalam format `name=path` (bisa diulang) |
| `-dns-upstream` | resolver sistem | server DNS untuk menampilkan nama host |
| `-dns-rate` | `20` | maksimum reverse lookup per detik |
| `-dns-cache` | `2m` | berapa lama nama host di-cache |
| `-no-dns` | | tanpa reverse lookup |
| `-tui` | | sekaligus membuka antarmuka terminal |

Contoh: port collector kedua, detail selama setahun, dan antarmuka web
hanya di mesin lokal:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. Data, backup, upgrade, uninstall

Semua tersimpan di direktori data:

| | |
|---|---|
| `raw/` | detail flow, satu file terkompresi per jam |
| `traffic66.duckdb` | ringkasan, counter interface, dan jam berjalan |
| `password` | kata sandi login (dalam bentuk hash) |
| `inventory.txt` | nama (**Sumber → Nama**) |
| `logo.png` (atau `.svg`, `.jpg`, `.webp`, `.gif`) | logo Anda (**Sumber → Logo**), jika Anda mengunggahnya |
| `country.mmdb`, `asn.mmdb`, `asn.tsv.gz`, `threats/` | basis data negara dan jaringan serta daftar ancaman yang Anda tambahkan |

**Berapa lama data disimpan**: detail flow 30 hari, ringkasan (ikhtisar dan rentang waktu panjang) 400 hari. Data
yang lebih lama dihapus otomatis, diperiksa setiap 5 menit; tidak ada yang lain dihapus dan tidak ada batas lain.
Ubah masa simpan detail dengan `-retention-days`, berapa pun harinya, misalnya `-retention-days 365`. Pemakaian disk
ikut bertambah: **Tersedia** di menu samping menjadi merah bila hari yang disimpan tidak muat. Jika disk penuh, flow
baru tidak dapat disimpan sampai ada ruang kosong.

- **Backup**: hentikan traffic66 lalu salin direktorinya. Tanpa
  menghentikannya, salin `raw/`, `password`, dan `inventory.txt`; jam
  berjalan dan ringkasan tidak ikut tersalin.
- **Pindah**: hentikan traffic66, pindahkan direktori, lalu jalankan dengan
  `-data` yang menunjuk ke lokasi baru.
- **Upgrade**: hentikan traffic66, ganti file program, lalu jalankan lagi.
  Data tetap utuh. Contoh di Linux:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Uninstall**: hentikan dan hapus service atau startup task (lihat
  [Instalasi](#2-install)), lalu hapus folder program dan direktori data.

<a id="14-security"></a>

## 14. Keamanan

- Antarmuka web memakai HTTP biasa: kata sandi dan data melintasi jaringan
  tanpa enkripsi. Di jaringan yang tidak sepenuhnya Anda percayai, listen
  hanya di mesin ini (`-addr 127.0.0.1:8066`) dan pasang reverse proxy TLS
  di depannya, misalnya dengan [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  Atau akses lewat VPN atau tunnel SSH:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, lalu buka
  http://127.0.0.1:8066.
- Izinkan port collector UDP hanya dari alamat perangkat Anda.
- Community SNMP di `inventory.txt` disimpan sebagai teks biasa; gunakan
  community read-only.

<a id="15-sizing"></a>

## 15. Kebutuhan sumber daya

Diukur pada 5.000 flow per detik di mesin 2 core: detail memakai sekitar
12 GB disk per hari ditambah sekitar 1,5 GB untuk jam berjalan, programnya
seperenam satu core. Ringkasan untuk rentang
waktu panjang diambil dari data ringkasan dan selesai di bawah 0,2 detik.
Query pada detail memindai sekitar 22 juta baris per jam: satu host selama
1 jam butuh di bawah 1 detik, Top 66 semua percakapan selama 1 jam sekitar
9 detik; waktunya bertambah seiring rentang dan berkurang dengan lebih
banyak core.

Jadi, disk untuk 30 hari pada 5.000 flow/detik sekitar 360 GB;
sesuaikan dengan laju flow Anda (terlihat di **Sumber**) dan
`-retention-days`.

Memori: `-memory` (default 10% RAM, minimal 256 MB) membatasi cache
database, dan bagian program lainnya mendapat batas lunak sebesar yang
sama. Pada 5.000 flow per detik, data milik program sendiri (decoding,
deteksi duplikat, batch) memakai sekitar 90 MB; secara total perkirakan
0,6–0,8 GB, jadi mesin dengan RAM 2 GB sudah cukup. Diukur selama 10 menit
pengumpulan terus-menerus (puncak 0,58 GB di mesin 8 GB) dan saat memuat
flow satu jam dengan laju sebelas kali lipat dengan batas mesin 2 GB
(puncak 0,74 GB).

`-memory` adalah anggaran, bukan batas keras: batas Go bersifat lunak dan
database bisa sesaat melebihi bagiannya. Untuk batas keras gunakan milik
sistem operasi: `MemoryMax=` di unit systemd (bagian 2) atau batas memori
container. Sediakan sekitar 2,5 kali bagian `-memory` dan minimal 1 GB;
`MemoryMax=2G` cocok untuk mesin dengan RAM hingga 8 GB pada bagian
default. Dengan begitu traffic66 yang di-restart, bukan mesin yang
kehabisan memori.

<a id="16-troubleshooting"></a>

## 16. Pemecahan masalah

| Gejala | Penyebab dan solusi |
|---|---|
| Perangkat tidak muncul di **Sumber** | Paket tidak sampai: lihat [Pastikan flow masuk](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | Perangkat belum mengirim sampler options; kebanyakan mengirim ulang dalam beberapa menit. Jika tidak pernah, ekspor opsi itu (`option sampler-table` di Cisco) atau tandai perangkat sebagai `unsampled` di Nama jika memang 1:1 |
| Angka lebih rendah dari counter interface | Lihat **Pencocokan antarmuka**: loss di jalan, interface tidak di-sampling, atau flow masih di cache perangkat (active timeout lebih dari 60 detik) |
| Angka lebih tinggi dari counter interface | Trafik yang sama di-sampling di dua interface atau dua perangkat |
| Tidak ada negara atau jaringan ("Tidak diketahui") | Tidak ada basis data yang dimuat: unggah di **Sumber**, lihat [Negara](#8-countries-networks-and-threat-lists) |
| "Basis data mencapai batas memorinya dan tidak bisa menjawab" di sebuah halaman | Pilih rentang waktu yang lebih pendek, atau jalankan dengan `-memory` yang lebih besar; detailnya ada di log |
| Lupa kata sandi | `traffic66 passwd` di mesin traffic66 (tambahkan `-data` jika traffic66 berjalan dengannya) |
| `Conflicting lock is held` | traffic66 lain sudah memakai direktori data ini |
| `receive buffer is only … KB` | Linux membatasi buffer UDP: atur `net.core.rmem_max=16777216` (lihat [Linux](#linux)) |
| `cannot create the data directory` | Folder program tidak bisa ditulis oleh user ini: beri `-data` |
| macOS: "cannot be opened" atau "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protected your PC" (Windows melindungi PC Anda) | **More info** (info selengkapnya) → **Run anyway** (tetap jalankan); program ini belum ditandatangani |
| Capture di Windows: Npcap tidak ditemukan | Pasang [Npcap](https://npcap.com) |
| `address already in use` | Port dipakai program lain: pilih port lain dengan `-addr` atau `-listen` |

<a id="17-build-from-source"></a>

## 17. Build dari source

Go 1.24 dan compiler C (gcc atau clang; MinGW-w64 di Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
