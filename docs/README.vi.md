[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | **Tiếng Việt** | [한국어](README.ko.md)

# traffic66 — Bộ thu thập và phân tích lưu lượng NetFlow, sFlow và IPFIX

Phân tích luồng sFlow, NetFlow và IPFIX trong một chương trình duy nhất: ai
đang dùng băng thông, lưu lượng đi đâu, và số liệu có khớp với bộ đếm giao
diện của chính thiết bị hay không, trên giao diện web và giao diện dòng lệnh
(terminal).

## Bắt đầu nhanh

```
# Linux, macOS: tải về ./traffic66 và bắt đầu thu thập
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: tương tự
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

Terminal in ra địa chỉ giao diện web cùng IP và cổng UDP để trỏ thiết bị tới ([cấu hình thiết bị](MANUAL.vi.md#device-configuration)). Đăng nhập bằng `admin` / `traffic66` rồi đặt mật khẩu mới. Chỉ muốn xem thử? Bản demo với thiết bị mô phỏng:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Cổng: TCP 8066 giao diện web, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (mỗi cổng UDP nhận mọi giao thức). Terminal in ra địa chỉ để trỏ thiết bị tới; lần đăng nhập đầu sẽ yêu cầu mật khẩu mới.

macOS thủ công: chạy trước `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows "đã bảo vệ PC": **Thông tin thêm** → **Vẫn chạy**.

## Bảng lệnh nhanh

| Việc | Lệnh |
|---|---|
| Thu sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (mặc định) | `./traffic66` |
| Chỉ NetFlow v5/v9 cổng 2055 và IPFIX cổng 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Chỉ sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Giao diện web trên IP và cổng chỉ định | `./traffic66 -addr 10.0.0.5:8080` |
| Bắt gói card mạng máy này (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Đặt hoặc đặt lại mật khẩu admin (thêm -data DIR nếu dùng) | `./traffic66 passwd` |
| Giao diện terminal tới máy chủ từ xa | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Tổng quan: các phát hiện đang mở, băng thông theo ứng dụng so với cùng thời điểm hôm qua, các client và dịch vụ hàng đầu](images/overview.png)

## Hướng dẫn đầy đủ

trong giao diện web: **Trợ giúp** trên menu; trên GitHub: [MANUAL.vi.md](MANUAL.vi.md)

15 ngôn ngữ. Miễn phí để dùng thử và cho tổ chức dưới 100 người ([giấy phép](../LICENSE.md)).
