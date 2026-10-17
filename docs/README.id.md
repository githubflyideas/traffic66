[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | **Bahasa Indonesia** | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — kolektor dan penganalisis trafik NetFlow, sFlow, dan IPFIX

Analitik flow untuk sFlow, NetFlow, dan IPFIX dalam satu program: siapa yang
memakai bandwidth, ke mana trafik mengalir, dan apakah angkanya cocok dengan
counter interface di perangkat itu sendiri, lewat antarmuka web maupun
antarmuka terminal.

## Mulai cepat

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

## Perintah umum

- `./traffic66`  
  Mulai dengan pengaturan bawaan di atas.

- `./traffic66 -addr :9000`  
  Antarmuka web di port 9000, bukan 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Ganti port kolektor (di sini NetFlow di 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Hanya mendengarkan di satu alamat IP, bukan semua antarmuka.

- `./traffic66 -data /var/lib/traffic66`  
  Simpan data di direktori lain (bawaan: `traffic66-data` di samping program).

- `./traffic66 passwd`  
  Atur atau reset kata sandi admin (tambahkan `-data DIR` bila dipakai).

- `traffic66.exe interfaces`  
  Windows: daftar kartu jaringan (nama dan nomor).

- `traffic66.exe -capture Wi-Fi`  
  Windows: analisis lalu lintas PC ini pada kartu itu (nama atau nomor). Perlu Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: daftar kartu jaringan.

- `sudo ./traffic66 -capture eth0`  
  Linux: analisis lalu lintas mesin ini di eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Antarmuka terminal di mesin traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Antarmuka terminal ke traffic66 jarak jauh.

- `./traffic66 demo`  
  Coba dengan perangkat simulasi.

## Tangkapan layar

<img src="images/overview.png" alt="Ringkasan" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Geografi & jaringan" width="49%"> <img src="images/interfaces.png" alt="Pencocokan antarmuka" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Manual lengkap

Di antarmuka web: **Bantuan** di menu. Di GitHub: [MANUAL.id.md](MANUAL.id.md)

15 bahasa. Gratis untuk evaluasi dan untuk organisasi dengan kurang dari 100 orang ([lisensi](../LICENSE.md)).
