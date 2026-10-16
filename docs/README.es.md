[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | **Español** | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — colector y analizador de tráfico NetFlow, sFlow e IPFIX

Análisis de flujos sFlow, NetFlow e IPFIX en un solo programa: quién consume
el ancho de banda, adónde va el tráfico y si las cifras cuadran con los
contadores de interfaz de los propios equipos, en una interfaz web y en una
interfaz de terminal.

## Inicio rápido

1. Descargue el archivo para su sistema: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Descomprímalo y ejecute el programa:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: descomprima `traffic66-windows-amd64.zip` y ejecute `traffic66.exe` (basta con doble clic).

macOS: primero ejecute `xattr -dr com.apple.quarantine .` en la carpeta.

Después:

- Interfaz web: `http://<IP del servidor>:8066`. Escucha en todas las interfaces de red.
- Inicio de sesión: usuario `admin`, contraseña `traffic66`. El primer inicio de sesión obliga a poner una contraseña nueva.
- Haga que sus equipos envíen a la IP del servidor: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (todas las interfaces; cada puerto acepta todos los protocolos).

## Comandos habituales

- `./traffic66`  
  Arrancar con los valores por defecto de arriba.

- `./traffic66 -addr :9000`  
  Interfaz web en el puerto 9000 en lugar de 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Cambiar los puertos de los colectores (aquí NetFlow en 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Escuchar solo en una dirección IP en lugar de en todas las interfaces.

- `./traffic66 -data /var/lib/traffic66`  
  Guardar los datos en otro directorio (por defecto: `traffic66-data` junto al programa).

- `./traffic66 passwd`  
  Fijar o restablecer la contraseña de admin (añada `-data DIR` si usa uno).

- `traffic66.exe interfaces`  
  Windows: listar las tarjetas de red (nombre y número).

- `traffic66.exe -capture Wi-Fi`  
  Windows: analizar el tráfico propio de este PC en esa tarjeta (nombre o número). Requiere Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: listar las tarjetas de red.

- `sudo ./traffic66 -capture eth0`  
  Linux: analizar el tráfico propio de esta máquina en eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Interfaz de terminal en la máquina de traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Interfaz de terminal conectada a un traffic66 remoto.

- `./traffic66 demo`  
  Probarlo con equipos simulados.

## Capturas de pantalla

<img src="images/overview.png" alt="Resumen" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Geografía y redes" width="49%"> <img src="images/interfaces.png" alt="Verificación de interfaces" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Manual completo

En la interfaz web: **Ayuda** en el menú. En GitHub: [MANUAL.es.md](MANUAL.es.md)

15 idiomas. Gratis para evaluación y para organizaciones de menos de 100 personas ([licencia](../LICENSE.md)).
