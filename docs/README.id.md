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
- Daftar Top 66, jalur trafik, negara dan jaringan, kecocokan dengan daftar
  ancaman, catatan flow, enkapsulasi (GRE, IPIP, VXLAN, GENEVE, MPLS).
- 13 bahasa di antarmuka web dan antarmuka terminal.

<a id="contents"></a>

## Daftar isi

1. [Coba demo](#1-try-the-demo)
2. [Instalasi](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Login dan kata sandi](#3-sign-in-and-passwords)
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
| Linux x86-64 | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64 | `traffic66-linux-arm64.tar.gz` |
| macOS Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS Intel | `traffic66-darwin-amd64.tar.gz` |

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
membangun jaringan kantor kecil dengan riwayat satu hari dan trafik live
dari empat perangkat simulasi, lengkap dengan dua insiden yang bisa Anda
cari: mulai dari **Ringkasan**, lihat **Siapa yang naik**, lalu telusuri
dengan mengeklik. Hentikan dengan Ctrl+C. Data demo disimpan di
`traffic66-demo` di sebelah program; hapus folder itu untuk memulai demo
dari awal.

Demo memakai port yang sama dengan instalasi sungguhan (8066, serta UDP
6343, 2055, 4739). Untuk menjalankannya berdampingan dengan instalasi
sungguhan, beri port lain:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

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

<a id="3-sign-in-and-passwords"></a>

## 3. Login dan kata sandi

Buka `http://<traffic66 machine>:8066` dan login. Usernya `admin`, kecuali
Anda memilih yang lain.

- Jika Anda belum mengatur kata sandi sebelum start pertama, traffic66
  membuatnya sendiri dan mencetaknya sekali di log:
  `first start: sign in as user "admin" with password "…"`.
  Di Linux, cari dengan `journalctl -u traffic66 | grep "first start"`.
- Kata sandi disimpan dalam bentuk hash di file `password` di direktori
  data. Kata sandi tetap sama setelah restart.
- Untuk menggantinya, atau mengatur yang baru karena lupa, jalankan di
  mesin traffic66:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` membuat kata sandi acak dan mencetaknya.
  traffic66 yang sedang berjalan langsung menerima kata sandi baru pada
  login berikutnya; tidak perlu restart.
- User tambahan: `traffic66 passwd -data <data directory> -user alice`.
  Semua user melihat hal yang sama.
- Untuk skrip dan container, `TRAFFIC66_PASSWORD=…` di environment atau
  `-password …` di command line mengatur kata sandi untuk run itu saja,
  menggantikan yang tersimpan. Utamakan environment: command line bisa
  dilihat user lain di mesin yang sama.

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

**Sumber → Nama** di antarmuka web menerima satu entri per baris. Isinya
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
  **Top-N → Segmen** dan di jalur trafik.
- `snmp <device> <community> [<management address>[:port]]`: device adalah
  alamat asal flow. Tambahkan alamat manajemen bila perangkat menjawab SNMP
  di alamat lain. Deskripsi interface yang dibaca lewat SNMP dipakai
  sebagai nama, kecuali Anda menamai interface itu dengan `iface`.
  Izinkan mesin traffic66 di access list SNMP perangkat.
- Perubahan berlaku saat Anda mengeklik **Simpan**; tidak perlu restart.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Negara, jaringan, dan daftar ancaman

Nama negara dan jaringan (AS) membutuhkan tabel IP-to-ASN. Unduh yang
gratis dari [iptoasn.com](https://iptoasn.com):

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

File apa pun dengan format yang sama bisa dipakai (dipisah tab: alamat
pertama, alamat terakhir, nomor AS, kode negara, nama AS; teks biasa atau
gzip). Restart traffic66 setelah menggantinya; unduh yang baru kira-kira
sebulan sekali.

Daftar ancaman adalah file teks biasa berisi satu alamat atau jaringan per
baris (teks setelah `#` atau `;` diabaikan), disimpan sebagai
`<data directory>/threats/<name>.txt`, misalnya:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Restart traffic66 setelah menambah atau mengubah daftar. Kecocokan muncul
di **Intel ancaman**, dikelompokkan per nama daftar.

<a id="9-using-the-web-ui"></a>

## 9. Memakai antarmuka web

Anda jarang perlu mengetik. Setiap nilai di setiap halaman — alamat, port,
aplikasi, negara, perangkat — bisa diklik:

- **Tampilkan hanya ini** / **Kecualikan ini** menambahkan filter. Filter
  muncul di bawah bar atas dan berlaku di semua halaman sampai Anda
  menghapusnya.
- **Lihat catatan flow-nya** membuka flow-flow individual yang cocok.
- **Cari secara online** membuka alamat atau AS tersebut di situs lookup
  publik.
- **Salin** menyalin nilainya.

Halaman:

| Halaman | Pertanyaan yang dijawab |
|---|---|
| Ringkasan | Berapa trafik sekarang dan dibanding minggu lalu, per aplikasi; apa yang naik; klien dan layanan teratas |
| Top-N | 66 teratas untuk klien, server, percakapan, aplikasi, port, negara, jaringan, segmen, perangkat, enkapsulasi, atau VLAN |
| Jalur trafik | Segmen mana berbicara dengan aplikasi apa di negara mana |
| Geografi & jaringan | Trafik per negara dan per jaringan (AS) |
| Intel ancaman | Host yang berkomunikasi dengan alamat di daftar ancaman Anda, dan berapa banyak yang mereka kirim |
| Catatan flow | Flow individual, terbaru di atas, dengan kolom yang bisa dipilih |
| Pencocokan antarmuka | Angka flow di samping counter interface, yang paling buruk di atas, beserta alasannya |
| Sumber | Perangkat, sampling, loss, collector, SNMP, dan **Nama** |

Di atas halaman: rentang waktu (15 menit sampai 30 hari), kotak pencarian
opsional, refresh otomatis setiap 30 detik, dan **Salin tautan**, yang
menyalin tautan ke tampilan saat ini secara persis (halaman, rentang waktu,
dan filter) untuk dikirim ke rekan kerja. Bahasa mengikuti browser; ubah di
bagian bawah menu.

Top-N untuk rentang waktu panjang diambil dari ringkasan per jam; filter
tidak tersedia di sana, dan halaman memberi tahu hal itu. Pilih rentang
yang lebih pendek untuk memfilter.

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

<a id="11-local-capture"></a>

## 11. Capture lokal

Selain menerima ekspor flow, traffic66 bisa membuat flow sendiri dari paket
di interface jaringan lokal, misalnya port mirror (SPAN):

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: butuh root, atau capability `CAP_NET_RAW` dan `CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`,
  atau baris `AmbientCapabilities` di unit systemd di atas).
- macOS: butuh root (perangkat BPF); tidak ada yang perlu dipasang.
- Windows: pasang [Npcap](https://npcap.com) terlebih dahulu.

Interface yang di-capture tercantum di **Sumber**. Paket yang terlihat dua
kali (misalnya di dua port mirror) dihitung dua kali.

<a id="12-options"></a>

## 12. Opsi

`traffic66 -h` dan `traffic66 <command> -h` menampilkan semuanya.

Perintah:

| Perintah | |
|---|---|
| `traffic66` | mengumpulkan flow dan menyajikan antarmuka web |
| `traffic66 demo` | sama, dengan jaringan simulasi |
| `traffic66 tui` | antarmuka terminal untuk traffic66 yang sedang berjalan |
| `traffic66 passwd` | mengatur kata sandi login |
| `traffic66 simulate -to HOST` | mengirim ekspor simulasi ke collector |
| `traffic66 interfaces` | menampilkan interface untuk capture lokal |
| `traffic66 version` | mencetak versi |

Opsi untuk `traffic66` dan `traffic66 demo`:

| Opsi | Default | |
|---|---|---|
| `-addr` | `:8066` | alamat antarmuka web; `127.0.0.1:8066` agar hanya bisa diakses dari mesin ini |
| `-data` | `traffic66-data` di sebelah program | direktori data |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collector UDP dalam format `name=address`, dipisah koma; kosong berarti nonaktif |
| `-user` | `admin` | user untuk kata sandi pertama yang dibuat otomatis dan untuk `-password` |
| `-password` | kata sandi tersimpan | kata sandi hanya untuk run ini (juga `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | berapa hari detail flow disimpan; ringkasan disimpan 400 hari |
| `-memory` | `0.10` | porsi memori fisik yang boleh dipakai database |
| `-l2-overhead` | `18` | byte per paket yang ditambahkan ke hitungan byte NetFlow/IPFIX |
| `-sampling-wait` | `5m` | berapa lama record menunggu sampling rate |
| `-capture` | | capture di interface lokal (bisa diulang) |
| `-inventory` | `<data>/inventory.txt` | file nama |
| `-asn` | `<data>/asn.tsv.gz` | tabel IP-to-ASN |
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
| `asn.tsv.gz`, `threats/` | tabel lookup yang Anda tambahkan |

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
sekitar 0,5 GB memori dan seperenam satu core. Ringkasan untuk rentang
waktu panjang diambil dari data ringkasan dan selesai di bawah 0,2 detik.
Query pada detail memindai sekitar 22 juta baris per jam: satu host selama
1 jam butuh di bawah 1 detik, Top 66 semua percakapan selama 1 jam sekitar
9 detik; waktunya bertambah seiring rentang dan berkurang dengan lebih
banyak core.

Jadi, disk untuk 30 hari pada 5.000 flow/detik sekitar 360 GB;
sesuaikan dengan laju flow Anda (terlihat di **Sumber**) dan
`-retention-days`.

<a id="16-troubleshooting"></a>

## 16. Pemecahan masalah

| Gejala | Penyebab dan solusi |
|---|---|
| Perangkat tidak muncul di **Sumber** | Paket tidak sampai: lihat [Pastikan flow masuk](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | Perangkat belum mengirim sampler options; kebanyakan mengirim ulang dalam beberapa menit. Jika tidak pernah, ekspor opsi itu (`option sampler-table` di Cisco) atau tandai perangkat sebagai `unsampled` di Nama jika memang 1:1 |
| Angka lebih rendah dari counter interface | Lihat **Pencocokan antarmuka**: loss di jalan, interface tidak di-sampling, atau flow masih di cache perangkat (active timeout lebih dari 60 detik) |
| Angka lebih tinggi dari counter interface | Trafik yang sama di-sampling di dua interface atau dua perangkat |
| Tidak ada negara atau jaringan | Tidak ada tabel IP-to-ASN: lihat [Negara](#8-countries-networks-and-threat-lists) |
| Lupa kata sandi | `traffic66 passwd -data <data directory>` di mesin traffic66 |
| `Conflicting lock is held` | traffic66 lain sudah memakai direktori data ini |
| `receive buffer is only … KB` | Linux membatasi buffer UDP: atur `net.core.rmem_max=16777216` (lihat [Linux](#linux)) |
| `cannot create the data directory` | Folder program tidak bisa ditulis oleh user ini: beri `-data` |
| macOS: "cannot be opened" atau "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
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
