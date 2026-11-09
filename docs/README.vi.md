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
- Ngay sau khi khởi động, chưa có dữ liệu một lúc là bình thường: thiết bị xuất hiện trong **Cài đặt** vài giây sau gói đầu tiên, biểu đồ chậm khoảng một phút. NetFlow v9/IPFIX cần template của thiết bị trước; bản ghi từ thiết bị chưa gửi tỷ lệ lấy mẫu được giữ tối đa 5 phút (`-sampling-wait`), sau đó tính 1:1.
- Yêu cầu: chương trình khoảng 73 MB, không cần cài thêm gì; bộ nhớ khoảng 0,1 GB khi rảnh và 0,6–0,8 GB ở 5.000 flow/giây, nên dành 1 GB trở lên. Đừng để giao diện web lộ trực tiếp ra Internet: dùng trong mạng nội bộ hoặc sau reverse proxy có HTTPS.

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
  Dùng thử với thiết bị mô phỏng. Lần chạy đầu tiên sẽ tạo trước một ngày lịch sử mô phỏng (vài giây, lâu hơn trên ổ đĩa chậm), xong mới mở giao diện web.

## Ảnh chụp màn hình

<img src="images/overview.png" alt="Tổng quan" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Địa lý & mạng" width="49%"> <img src="images/interfaces.png" alt="Kiểm tra giao diện" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Hướng dẫn đầy đủ

Trong giao diện web: **Trợ giúp** trên menu. Trên GitHub: [MANUAL.vi.md](MANUAL.vi.md)

15 ngôn ngữ. Miễn phí để dùng thử và cho tổ chức dưới 100 người ([giấy phép](../LICENSE.md)).

- **Miễn phí**: mọi tổ chức được đánh giá, thử nghiệm và trình diễn không giới hạn thời gian; dùng thực tế miễn phí cho tổ chức dưới 100 người (nhân viên cộng nhà thầu) và cho mục đích phi thương mại. Tổ chức lớn hơn: dùng thực tế 30 ngày, sau đó cần đăng ký. Chi tiết: [LICENSE.md](../LICENSE.md), [Additional Use Grant](../ADDITIONAL-USE-GRANT.md).
- **Chưa đăng ký**: báo cáo và biểu đồ có dấu "DÙNG THỬ"; tối đa 3 tài khoản web.
- **Đã đăng ký**: không còn dấu. Liên hệ: route66traffic66@gmail.com.
