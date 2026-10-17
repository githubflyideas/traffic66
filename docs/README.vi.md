[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | **Tiếng Việt** | [한국어](README.ko.md)

# traffic66 — Bộ thu thập và phân tích lưu lượng NetFlow, sFlow và IPFIX

Phân tích luồng sFlow, NetFlow và IPFIX trong một chương trình duy nhất: ai
đang dùng băng thông, lưu lượng đi đâu, và số liệu có khớp với bộ đếm giao
diện của chính thiết bị hay không, trên giao diện web và giao diện dòng lệnh
(terminal).

## Bắt đầu nhanh

1. Tải gói cho hệ điều hành của bạn: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Giải nén rồi chạy chương trình:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: giải nén `traffic66-windows-amd64.zip` rồi chạy `traffic66.exe` (nhấp đúp cũng được).

macOS: trước tiên chạy `xattr -dr com.apple.quarantine .` trong thư mục đó.

Sau đó:

- Giao diện web: `http://<IP máy chủ>:8066`. Lắng nghe trên mọi card mạng.
- Đăng nhập: người dùng `admin`, mật khẩu `traffic66`. Lần đăng nhập đầu bắt buộc đặt mật khẩu mới.
- Cho thiết bị gửi về IP máy chủ: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (mọi card mạng; mỗi cổng nhận mọi giao thức).

## Lệnh thường dùng

- `./traffic66`  
  Chạy với các mặc định ở trên.

- `./traffic66 -addr :9000`  
  Giao diện web ở cổng 9000 thay vì 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Đổi cổng collector (ở đây NetFlow sang 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Chỉ lắng nghe trên một địa chỉ IP thay vì mọi card mạng.

- `./traffic66 -data /var/lib/traffic66`  
  Lưu dữ liệu ở thư mục khác (mặc định: `traffic66-data` cạnh chương trình).

- `./traffic66 passwd`  
  Đặt hoặc đặt lại mật khẩu admin (thêm `-data DIR` nếu có dùng).

- `traffic66.exe interfaces`  
  Windows: liệt kê card mạng (tên và số).

- `traffic66.exe -capture Wi-Fi`  
  Windows: phân tích lưu lượng của chính PC này trên card đó (tên hoặc số). Cần Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: liệt kê card mạng.

- `sudo ./traffic66 -capture eth0`  
  Linux: phân tích lưu lượng của máy này trên eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Giao diện terminal trên máy traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Giao diện terminal kết nối tới traffic66 từ xa.

- `./traffic66 demo`  
  Dùng thử với thiết bị mô phỏng.

## Ảnh chụp màn hình

<img src="images/overview.png" alt="Tổng quan" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Địa lý & mạng" width="49%"> <img src="images/interfaces.png" alt="Kiểm tra giao diện" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Hướng dẫn đầy đủ

Trong giao diện web: **Trợ giúp** trên menu. Trên GitHub: [MANUAL.vi.md](MANUAL.vi.md)

15 ngôn ngữ. Miễn phí để dùng thử và cho tổ chức dưới 100 người ([giấy phép](../LICENSE.md)).
