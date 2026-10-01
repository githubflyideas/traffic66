[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | **Español** | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

Análisis de flujos sFlow, NetFlow e IPFIX en un solo programa. Recibe las
exportaciones de flujos de switches, routers y firewalls, las guarda en una
base de datos embebida y muestra quién consume el ancho de banda, adónde va
el tráfico y si las cifras cuadran con los contadores de interfaz de los
propios equipos, tanto en una interfaz web como en una interfaz de terminal.

- Un solo ejecutable para Windows, Linux y macOS. Sin base de datos que
  instalar, sin runtime, funciona sin conexión.
- sFlow v5, NetFlow v5, NetFlow v9 e IPFIX en cualquier puerto UDP; captura
  local opcional desde una interfaz de red o un puerto espejo.
- Contrasta sus cifras con los contadores de interfaz (contadores sFlow o
  SNMP) y explica por qué difieren cuando no coinciden.
- Listas Top 66, rutas de tráfico, países y redes, coincidencias con listas
  de amenazas, registros de flujo, encapsulación (GRE, IPIP, VXLAN, GENEVE,
  MPLS).
- 13 idiomas en la interfaz web y en la de terminal.

<a id="contents"></a>

## Contenido

1. [Probar la demo](#1-try-the-demo)
2. [Instalación](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Inicio de sesión y contraseñas](#3-sign-in-and-passwords)
4. [Enviar flujos desde los equipos](#4-send-flows-from-your-devices)
5. [Comprobar que llegan los flujos](#5-check-that-flows-arrive)
6. [Hacer que las cifras cuadren con los contadores de interfaz](#6-make-the-numbers-match-the-interface-counters)
7. [Nombres, SNMP y redes propias](#7-names-snmp-and-your-own-networks)
8. [Países, redes y listas de amenazas](#8-countries-networks-and-threat-lists)
9. [Uso de la interfaz web](#9-using-the-web-ui)
10. [Interfaz de terminal](#10-terminal-ui)
11. [Captura local](#11-local-capture)
12. [Opciones](#12-options)
13. [Datos, copia de seguridad, actualización, desinstalación](#13-data-backup-upgrade-uninstall)
14. [Seguridad](#14-security)
15. [Dimensionamiento](#15-sizing)
16. [Resolución de problemas](#16-troubleshooting)
17. [Compilar desde el código fuente](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. Probar la demo

Descargue el archivo para su sistema desde la
[página de versiones](https://github.com/githubflyideas/traffic66/releases):

| Sistema | Archivo |
|---|---|
| Windows 10/11, Server 2016 o posterior (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: cualquier distribución con kernel 3.2 o posterior, incluidos CentOS 7 y Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: las mismas distribuciones | `traffic66-linux-arm64.tar.gz` |
| macOS 11 o posterior, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 o posterior, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (la segunda línea permite que macOS ejecute un programa descargado de
internet que no procede de la App Store):

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

Abra http://127.0.0.1:8066 e inicie sesión como `admin` / `try66`. La demo
monta la red de una pequeña empresa con un día de historial y tráfico en
vivo de cuatro equipos simulados, e incluye dos incidentes por descubrir:
empiece en **Resumen**, mire **Quién ha crecido** y siga haciendo clic a
partir de ahí. Deténgala con Ctrl+C. Los datos de la demo se guardan en
`traffic66-demo`, junto al programa; borre esa carpeta para empezar la demo
desde cero.

La demo usa los mismos puertos que una instalación real (8066 y UDP 6343,
2055, 4739). Para ejecutarla junto a una real, asígnele otros puertos:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

En Windows también puede simplemente hacer doble clic en `traffic66.exe`.
Eso inicia traffic66 de verdad (no la demo) y abre la interfaz web en su
navegador; la contraseña del primer arranque aparece en la ventana negra, y
al cerrar la ventana se detiene traffic66. Si Windows muestra
"Windows protegió su PC", haga clic en **Más información** →
**Ejecutar de todas formas**.

<a id="2-install"></a>

## 2. Instalación

traffic66 es un único archivo. Instalarlo consiste en colocarlo en algún
sitio, elegir un directorio de datos, fijar una contraseña, abrir el
firewall y hacer que arranque con el sistema. En los ejemplos, `192.0.2.50`
es la máquina de traffic66 y `192.0.2.1` un router; sustitúyalas por sus
direcciones.

Puertos:

| Puerto | Uso |
|---|---|
| UDP 6343 | sFlow (por defecto) |
| UDP 2055 | NetFlow (por defecto) |
| UDP 4739 | IPFIX (por defecto) |
| TCP 8066 | interfaz web y API |

Todos los puertos UDP aceptan todos los protocolos, así que un equipo puede
enviar NetFlow al 6343 si resulta más cómodo. Cambie o añada puertos con
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

El último comando pide la contraseña del usuario `admin`.

Cree `/etc/systemd/system/traffic66.service`:

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

Arránquelo y permita búferes UDP más grandes para no perder ráfagas:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, con firewalld (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

o con ufw (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

Descomprima en `C:\traffic66` y fije la contraseña (PowerShell como
administrador):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Los datos van a `C:\traffic66\traffic66-data`, junto al programa.

Abra el firewall:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

Para probarlo en primer plano, ejecute `C:\traffic66\traffic66.exe` y
deténgalo con Ctrl+C. Para que corra en segundo plano desde el arranque, sin
que nadie haya iniciado sesión, regístrelo como tarea de inicio:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` es importante: sin él, Windows
detiene la tarea a los tres días. Deténgala con `Stop-ScheduledTask -TaskName
traffic66` y elimínela con `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Cree `/Library/LaunchDaemons/traffic66.plist`:

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

Arrancarlo y volver a detenerlo:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

Si el firewall de macOS está activo, permita las conexiones entrantes para
traffic66 en Ajustes del Sistema → Red → Firewall → Opciones.

<a id="3-sign-in-and-passwords"></a>

## 3. Inicio de sesión y contraseñas

Abra `http://<traffic66 machine>:8066` e inicie sesión. El usuario es
`admin`, salvo que haya elegido otro.

- Si no fijó una contraseña antes del primer arranque, traffic66 genera una
  y la muestra una sola vez en su log:
  `first start: sign in as user "admin" with password "…"`.
  En Linux se encuentra con `journalctl -u traffic66 | grep "first start"`.
- La contraseña se guarda, en forma de hash, en el archivo `password` del
  directorio de datos. Se mantiene entre reinicios.
- Para cambiarla, o para poner una nueva si la ha olvidado, en la máquina de
  traffic66:

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` genera una aleatoria y la muestra. Un
  traffic66 en marcha acepta la nueva contraseña en el siguiente inicio de
  sesión; no hace falta reiniciar.
- Más usuarios: `traffic66 passwd -data <data directory> -user alice`. Todos
  los usuarios ven lo mismo.
- Para scripts y contenedores, `TRAFFIC66_PASSWORD=…` en el entorno o
  `-password …` en la línea de comandos fija la contraseña para esa
  ejecución en lugar de la guardada. Es preferible el entorno: las líneas de
  comandos son visibles para los demás usuarios de la máquina.

Tras cinco contraseñas incorrectas en un minuto, la dirección queda
bloqueada durante un minuto.

<a id="4-send-flows-from-your-devices"></a>

## 4. Enviar flujos desde los equipos

Apunte cada equipo a la máquina de traffic66. Los comandos varían según el
modelo y la versión de software; consulte el manual de su equipo. En todos
los ejemplos, `192.0.2.50` es traffic66 y `192.0.2.1` la dirección del
propio equipo.

Recomendaciones generales:

- Ponga el timeout de flujo activo en 60 segundos. Con timeouts más largos
  el tráfico llega tarde y en grandes bloques.
- Si el equipo muestrea NetFlow/IPFIX, haga que exporte sus opciones de
  sampler para que se conozca la tasa. traffic66 retiene los registros hasta
  que llega la tasa en lugar de contarlos 1:1.
- Muestree todas las interfaces o solo las de borde, en un solo sentido.
  Muestrear el mismo tráfico a la entrada y a la salida lo cuenta dos veces;
  **Verificación de interfaces** lo señala.
- Tasa de muestreo sFlow: en torno a 1:1000 para enlaces de 1 Gb/s, 1:4096
  para 10 Gb/s y 1:8192 para 40/100 Gb/s.

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

FortiGate FortiOS 7.4.2 o posterior (NetFlow v9):

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

Servidores y hosts Linux, con softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Comprobar que llegan los flujos

Abra **Fuentes**. Cada equipo que envía algo aparece en segundos, con su
protocolo, tasa de muestreo, pérdidas, último paquete y un estado. Cuando el
estado no está en verde, el texto de al lado indica qué falla y qué hay que
cambiar.

Si un equipo no aparece:

1. Observe si llegan paquetes a la máquina de traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Si no se ve nada, los paquetes no llegan a la máquina: revise la
   configuración del equipo, el enrutamiento y los firewalls del camino.
2. Llegan paquetes pero **Fuentes** sigue vacía: el firewall local los
   descarta (vea [Instalación](#2-install)) o traffic66 escucha en otros
   puertos (`-listen`).
3. Para probar el camino desde otra máquina sin tocar ningún equipo, ejecute
   allí `traffic66 simulate -to 192.0.2.50` durante unos segundos. Envía
   sFlow, NetFlow e IPFIX desde equipos simulados, que luego aparecen en
   **Fuentes** y en los datos, así que mejor hágalo en una instalación de
   pruebas.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Hacer que las cifras cuadren con los contadores de interfaz

Las cifras de flujo son estimaciones: paquetes muestreados por la tasa de
muestreo. traffic66 las compara con los contadores de interfaz del propio
equipo y muestra la diferencia en **Verificación de interfaces**, con la
causa probable cuando es mayor de lo que explica el propio muestreo.

Para disponer de contadores con los que comparar:

- Los equipos sFlow los envían por su cuenta cuando hay un intervalo de
  contadores configurado (`sflow counter interval 30` o similar).
- Para equipos NetFlow e IPFIX, añada una línea `snmp` en
  **Fuentes → Nombres** (vea [Nombres](#7-names-snmp-and-your-own-networks)).
  traffic66 lee entonces los contadores de interfaz cada minuto.

Lo que traffic66 ya hace para que las cifras cuadren: usa la tasa de
muestreo que el equipo aplicó realmente, retiene los registros NetFlow/IPFIX
hasta conocer la tasa de muestreo, compensa los paquetes de exportación
perdidos por el camino, reparte los flujos largos entre los minutos que
duraron y suma 18 bytes por paquete de overhead Ethernet a los bytes de
NetFlow/IPFIX (los contadores de interfaz lo incluyen, los recuentos de
flujo a nivel IP no; se cambia con `-l2-overhead`).

Motivos habituales de una diferencia residual, todos indicados en
**Verificación de interfaces**: hay interfaces sin muestrear, el mismo
tráfico se muestrea en dos interfaces, se pierden paquetes de exportación
antes de llegar a traffic66 o aún no se conoce la tasa de muestreo.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. Nombres, SNMP y redes propias

**Fuentes → Nombres** en la interfaz web admite una entrada por línea. Se
guarda como `inventory.txt` en el directorio de datos, así que también
puede editar ese archivo (vea `inventory.txt.example`). Todas las líneas
son opcionales.

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

- `net`: los rangos privados (10/8, 172.16/12, 192.168/16, 100.64/10) se
  consideran siempre propios. Añada sus rangos públicos para que el tráfico
  hacia y desde ellos también cuente como propio; el nombre aparece en
  **Top-N → Segmentos** y en las rutas de tráfico.
- `snmp <device> <community> [<management address>[:port]]`: el equipo es
  la dirección de la que vienen los flujos. Añada la dirección de gestión
  cuando el equipo responda a SNMP en otra dirección. Las descripciones de
  interfaz leídas por SNMP se usan como nombres, salvo que nombre la
  interfaz con `iface`. Permita la máquina de traffic66 en la lista de
  acceso SNMP del equipo.
- Los cambios se aplican al pulsar **Guardar**; no hace falta reiniciar.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Países, redes y listas de amenazas

Los países y los nombres de red (AS) necesitan una tabla IP-a-ASN. Descargue
la gratuita de [iptoasn.com](https://iptoasn.com):

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

Sirve cualquier archivo con el mismo formato (separado por tabuladores:
primera dirección, última dirección, número de AS, código de país, nombre
del AS; en texto plano o gzip). Reinicie traffic66 después de sustituirlo;
descargue uno nuevo más o menos cada mes.

Las listas de amenazas son archivos de texto plano con una dirección o red
por línea (se ignora lo que va tras `#` o `;`), guardados como
`<data directory>/threats/<name>.txt`, por ejemplo:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Reinicie traffic66 después de añadir o modificar listas. Las coincidencias
aparecen en **Inteligencia de amenazas**, por nombre de lista.

<a id="9-using-the-web-ui"></a>

## 9. Uso de la interfaz web

Casi nunca hace falta escribir. Cualquier valor de cualquier página (una
dirección, un puerto, una aplicación, un país, un equipo) admite clic:

- **Mostrar solo esto** / **Excluir esto** añade un filtro. Los filtros
  aparecen bajo la barra superior y se aplican a todas las páginas hasta
  que los quite.
- **Ver sus registros de flujo** abre los flujos individuales
  correspondientes.
- **Buscarlo en línea** abre la dirección o el AS en un sitio público de
  consulta.
- **Copiar** copia el valor.

Páginas:

| Página | Qué responde |
|---|---|
| Resumen | Cuánto tráfico hay ahora y frente a la semana pasada, por aplicación; qué ha crecido; principales clientes y servicios |
| Top-N | Los 66 primeros clientes, servidores, conversaciones, aplicaciones, puertos, países, redes, segmentos, equipos, encapsulaciones o VLAN |
| Rutas de tráfico | Qué segmento habla con qué aplicación en qué país |
| Geografía y redes | Tráfico por país y por red (AS) |
| Inteligencia de amenazas | Hosts que hablaron con direcciones de sus listas de amenazas y cuánto enviaron |
| Registros de flujo | Flujos individuales, del más reciente al más antiguo, con columnas seleccionables |
| Verificación de interfaces | Cifras de flujo junto a los contadores de interfaz, de peor a mejor, con motivos |
| Fuentes | Equipos, muestreo, pérdidas, colectores, SNMP y **Nombres** |

Encima de las páginas: rango de tiempo (de 15 minutos a 30 días), un cuadro
de búsqueda opcional, refresco automático cada 30 segundos y **Copiar enlace**, que copia un enlace exactamente a la vista actual
(página, rango de tiempo y filtros) para enviárselo a un compañero. El idioma sigue al del
navegador; se cambia al final del menú.

En rangos de tiempo largos, Top-N sale de resúmenes horarios; ahí no hay
filtros, y la página lo indica. Elija un rango más corto para filtrar.

<a id="10-terminal-ui"></a>

## 10. Interfaz de terminal

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

En la máquina de traffic66, `traffic66 tui` inicia sesión por sí solo si
puede leer el directorio de datos (indique `-data` si no es el de por
defecto). Si traffic66 se ejecuta con otro usuario, como ocurre con un
servicio, use `-user` y `-password`. `-lang` elige el idioma (`en`, `zh`,
`hi`, `es`, `ar`, `fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

Teclas: 1–8 páginas, ↑↓ seleccionar, Enter acciones sobre el valor
seleccionado, f mostrar solo, x excluir, / buscar, t rango de tiempo,
c quitar filtros, w abrir la misma vista en un navegador, q salir.

<a id="11-local-capture"></a>

## 11. Captura local

Además de las exportaciones de flujos, traffic66 puede generar flujos por sí
mismo a partir de los paquetes de una interfaz de red local, por ejemplo un
puerto espejo (SPAN):

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: requiere root o las capacidades `CAP_NET_RAW` y `CAP_NET_ADMIN`
  (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`, o
  la línea `AmbientCapabilities` de la unidad systemd anterior).
- macOS: requiere root (dispositivos BPF); no hay nada que instalar.
- Windows: instale antes [Npcap](https://npcap.com).

Las interfaces capturadas aparecen en **Fuentes**. Los paquetes vistos dos
veces (por ejemplo, en dos puertos espejo) se cuentan dos veces.

<a id="12-options"></a>

## 12. Opciones

`traffic66 -h` y `traffic66 <command> -h` lo listan todo.

Comandos:

| Comando | |
|---|---|
| `traffic66` | recoge flujos y sirve la interfaz web |
| `traffic66 demo` | lo mismo, con una red simulada |
| `traffic66 tui` | interfaz de terminal para un traffic66 en marcha |
| `traffic66 passwd` | fija una contraseña de acceso |
| `traffic66 simulate -to HOST` | envía exportaciones simuladas a un colector |
| `traffic66 interfaces` | lista las interfaces para captura local |
| `traffic66 version` | muestra la versión |

Opciones de `traffic66` y `traffic66 demo`:

| Opción | Por defecto | |
|---|---|---|
| `-addr` | `:8066` | dirección de la interfaz web; `127.0.0.1:8066` solo para esta máquina |
| `-data` | `traffic66-data` junto al programa | directorio de datos |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | colectores UDP como `name=address`, separados por comas; vacío los desactiva |
| `-user` | `admin` | usuario para la primera contraseña generada y para `-password` |
| `-password` | contraseña guardada | contraseña solo para esta ejecución (también `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | días de detalle de flujos que se conservan; los resúmenes se guardan 400 días |
| `-memory` | `0.10` | fracción de la memoria física que puede usar la base de datos |
| `-l2-overhead` | `18` | bytes por paquete sumados a los bytes de NetFlow/IPFIX |
| `-sampling-wait` | `5m` | cuánto esperan los registros a una tasa de muestreo |
| `-capture` | | captura en una interfaz local (repetible) |
| `-inventory` | `<data>/inventory.txt` | archivo de nombres |
| `-asn` | `<data>/asn.tsv.gz` | tabla IP-a-ASN |
| `-threat` | `<data>/threats/*.txt` | lista de amenazas adicional como `name=path` (repetible) |
| `-dns-upstream` | resolver del sistema | servidor DNS para mostrar nombres de host |
| `-dns-rate` | `20` | máximo de resoluciones inversas por segundo |
| `-dns-cache` | `2m` | cuánto tiempo se guardan en caché los nombres de host |
| `-no-dns` | | sin resoluciones inversas |
| `-tui` | | abre también la interfaz de terminal |

Ejemplo: un segundo puerto de colector, un año de detalle y la interfaz web
solo en la máquina local:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. Datos, copia de seguridad, actualización, desinstalación

El directorio de datos lo contiene todo:

| | |
|---|---|
| `raw/` | detalle de flujos, un archivo comprimido por hora |
| `traffic66.duckdb` | resúmenes, contadores de interfaz y la hora en curso |
| `password` | contraseñas de acceso (con hash) |
| `inventory.txt` | nombres (**Fuentes → Nombres**) |
| `asn.tsv.gz`, `threats/` | tablas de consulta que haya añadido |

- **Copia de seguridad**: detenga traffic66 y copie el directorio. Sin
  detenerlo, copie `raw/`, `password` e `inventory.txt`; en ese caso faltan
  la hora en curso y los resúmenes.
- **Traslado**: detenga traffic66, mueva el directorio y arranque con
  `-data` apuntando a la nueva ubicación.
- **Actualización**: detenga traffic66, sustituya el archivo del programa y
  vuelva a arrancarlo. Los datos se conservan. En Linux, por ejemplo:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Desinstalación**: detenga y elimine el servicio o la tarea de inicio
  (vea [Instalación](#2-install)) y después borre la carpeta del programa y
  el directorio de datos.

<a id="14-security"></a>

## 14. Seguridad

- La interfaz web usa HTTP sin cifrar: contraseñas y datos viajan por la red
  en claro. En redes que no sean de plena confianza, escuche solo en esta
  máquina (`-addr 127.0.0.1:8066`) y ponga delante un proxy inverso con TLS,
  por ejemplo con [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  O acceda por VPN o por un túnel SSH:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50` y abra
  http://127.0.0.1:8066.
- Permita los puertos UDP de colector solo desde las direcciones de sus
  equipos.
- Las comunidades SNMP de `inventory.txt` se guardan en texto plano; use una
  comunidad de solo lectura.

<a id="15-sizing"></a>

## 15. Dimensionamiento

Medido a 5.000 flujos por segundo en una máquina de 2 núcleos: el detalle
ocupa unos 12 GB de disco al día más aproximadamente 1,5 GB para la hora en
curso, y el programa unos 0,5 GB de memoria y una sexta parte de un núcleo.
Las vistas generales de rangos largos salen de resúmenes y tardan menos de
0,2 s. Las consultas sobre el detalle recorren unos 22 millones de filas por
hora: un host durante 1 hora tarda menos de 1 s, un Top 66 de 1 hora de
todas las conversaciones unos 9 s; el tiempo crece con el rango y baja con
más núcleos.

Por tanto, 30 días a 5.000 flujos/s ocupan unos 360 GB de disco; escálelo
según su tasa de flujos (visible en **Fuentes**) y `-retention-days`.

<a id="16-troubleshooting"></a>

## 16. Resolución de problemas

| Síntoma | Causa y solución |
|---|---|
| El equipo no aparece en **Fuentes** | Los paquetes no llegan: vea [Comprobar que llegan los flujos](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | El equipo aún no ha enviado sus opciones de sampler; la mayoría las reenvía en pocos minutos. Si no lo hace nunca, expórtelas (`option sampler-table` en Cisco) o márquelo como `unsampled` en Nombres si de verdad es 1:1 |
| Cifras por debajo de los contadores de interfaz | Vea **Verificación de interfaces**: pérdidas por el camino, interfaces sin muestrear o flujos aún en la caché del equipo (timeout activo superior a 60 s) |
| Cifras por encima de los contadores de interfaz | El mismo tráfico se muestrea en dos interfaces o en dos equipos |
| No hay países ni redes | Falta la tabla IP-a-ASN: vea [Países](#8-countries-networks-and-threat-lists) |
| Contraseña olvidada | `traffic66 passwd -data <data directory>` en la máquina de traffic66 |
| `Conflicting lock is held` | Otro traffic66 ya usa este directorio de datos |
| `receive buffer is only … KB` | Linux limita los búferes UDP: fije `net.core.rmem_max=16777216` (vea [Linux](#linux)) |
| `cannot create the data directory` | Este usuario no puede escribir en la carpeta del programa: indique `-data` |
| macOS: "cannot be opened" o "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "Windows protegió su PC" | **Más información** → **Ejecutar de todas formas**; el programa aún no está firmado |
| Captura en Windows: no se encuentra Npcap | Instale [Npcap](https://npcap.com) |
| `address already in use` | Otro programa usa el puerto: elija otros con `-addr` o `-listen` |

<a id="17-build-from-source"></a>

## 17. Compilar desde el código fuente

Go 1.24 y un compilador de C (gcc o clang; MinGW-w64 en Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
