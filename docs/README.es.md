[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | **Español** | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — colector y analizador de tráfico NetFlow, sFlow e IPFIX

Análisis de flujos sFlow, NetFlow e IPFIX en un solo programa: quién consume
el ancho de banda, adónde va el tráfico y si las cifras cuadran con los
contadores de interfaz de los propios equipos, en una interfaz web y en una
interfaz de terminal.

## Inicio rápido

```
# Linux, macOS: descargar en ./traffic66 y empezar a recoger
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: lo mismo
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

El terminal muestra la dirección de la interfaz web y la IP y los puertos UDP a los que apuntar sus equipos ([configuración de equipos](MANUAL.es.md#device-configuration)). Inicie sesión como `admin` / `traffic66` y elija una contraseña nueva. ¿Solo mirar? La demo, con equipos simulados:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Puertos: TCP 8066 interfaz web, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (cada puerto UDP acepta todos los protocolos). El terminal muestra las direcciones a las que apuntar los equipos; el primer inicio de sesión pide una contraseña nueva.

macOS a mano: primero `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows «protegió su PC»: **Más información** → **Ejecutar de todas formas**.

## Chuleta de comandos

| Tarea | Comando |
|---|---|
| Recoger sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (por defecto) | `./traffic66` |
| Solo NetFlow v5/v9 en 2055 e IPFIX en 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Solo sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Interfaz web en una IP y puerto | `./traffic66 -addr 10.0.0.5:8080` |
| Capturar la tarjeta de red local (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Fijar o restablecer la contraseña de admin (añada -data DIR si lo usa) | `./traffic66 passwd` |
| Interfaz de terminal a un servidor remoto | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Resumen: hallazgos abiertos, ancho de banda por aplicación frente a la misma hora de ayer, principales clientes y servicios](images/overview.png)

## Manual completo

en la interfaz web: **Ayuda** en el menú; en GitHub: [MANUAL.es.md](MANUAL.es.md)

15 idiomas. Gratis para evaluación y para organizaciones de menos de 100 personas ([licencia](../LICENSE.md)).
