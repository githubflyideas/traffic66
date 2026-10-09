[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | **Español** | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — colector y analizador de tráfico NetFlow, sFlow e IPFIX

Análisis de flujos sFlow, NetFlow e IPFIX en un solo programa: quién consume
el ancho de banda, adónde va el tráfico y si las cifras cuadran con los
contadores de interfaz de los propios equipos, en una interfaz web y en una
interfaz de terminal.

Una alternativa autoalojada a ntopng, ElastiFlow, pmacct con Grafana o los
módulos de flujos de PRTG y SolarWinds NTA, para monitorización de red
(network monitoring), monitorización del ancho de banda
(bandwidth monitoring), principales consumidores (top talkers), detección de
DDoS y escaneos y análisis de pcap, sin Elasticsearch, Kafka ni una base de
datos aparte.

- Un solo ejecutable para Windows, Linux y macOS; sin base de datos que instalar, funciona sin conexión.
- sFlow v5, NetFlow v5/v9 e IPFIX en cualquier puerto UDP, o captura local desde una interfaz.
- Contrasta sus cifras con los contadores de interfaz (sFlow o SNMP) y explica por qué difieren.
- Detecta escaneos, adivinación de contraseñas, movimiento lateral, subidas inusuales, inundaciones y tráfico de listas de amenazas, también a través del muestreo.
- `traffic66 capture.pcap` analiza capturas de paquetes sin configurar nada.
- 15 idiomas. Gratis para evaluación y para organizaciones de menos de 100 personas ([licencia](#licence)).

![Resumen: hallazgos abiertos, ancho de banda por aplicación frente a la misma hora de ayer, principales clientes y servicios](images/overview.png)

<sub>Todas las capturas de pantalla proceden de `traffic66 demo`, una red de empresa simulada.</sub>

<a id="contents"></a>

## Contenido

1. [Probar la demo](#1-try-the-demo)
2. [Instalación](#2-install)
3. [Usuarios y contraseñas](#3-users-and-passwords)
4. [Enviar flujos desde los equipos](#4-send-flows-from-your-devices)
5. [Comprobar que llegan los flujos](#5-check-that-flows-arrive)
6. [Interfaces y contadores](#6-interfaces-and-counters)
7. [Nombres, países y listas de amenazas](#7-names-countries-and-threat-lists)
8. [Uso de la interfaz web](#8-using-the-web-ui)
9. [Análisis offline de pcap, interfaz de terminal, captura local](#9-offline-pcap-terminal-ui-local-capture)
10. [Opciones y datos](#10-options-and-data)
11. [Seguridad, dimensionamiento, solución de problemas](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. Probar la demo

Descargue el archivo para su sistema desde la
[página de versiones](https://github.com/githubflyideas/traffic66/releases)
(Windows x64, Linux x86-64/ARM64 con kernel 3.2+, macOS 11+), descomprímalo y ejecute:

```
./traffic66 demo          # Linux, macOS
.\traffic66.exe demo      # Windows
```

En macOS ejecute antes `xattr -dr com.apple.quarantine <folder>`. Abra
http://127.0.0.1:8066 como `admin` / `traffic66`: un día de historial y tráfico
en vivo de cuatro equipos simulados, incluido un ataque mostrado paso a paso
en **Hallazgos**. Ctrl+C la detiene; borre `traffic66-demo` para empezar de
cero. Para ejecutarla junto a una instalación real: `-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. Instalación

traffic66 es un solo archivo. Puertos: UDP 6343 (sFlow), 2055 (NetFlow), 4739
(IPFIX), TCP 8066 (interfaz web); cada puerto UDP acepta cualquier protocolo.

**Linux** (systemd):

```
sudo mkdir -p /opt/traffic66 && sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
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
sudo firewall-cmd --permanent --add-port={6343,2055,4739}/udp --add-port=8066/tcp && sudo firewall-cmd --reload
```

**Windows** (PowerShell como administrador): descomprima en `C:\traffic66`,
ejecute `C:\traffic66\traffic66.exe passwd`, abra los puertos y haga que
arranque con el sistema:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

También funciona hacer doble clic en `traffic66.exe`: abre la interfaz web.

**macOS**: descomprima en `/usr/local/traffic66`, quite la marca de
cuarentena, ejecute `traffic66 passwd -data "/Library/Application Support/traffic66"`
y arránquelo desde un LaunchDaemon cuyos `ProgramArguments` sean el
programa, `-data` y ese directorio, con `RunAtLoad` y `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. Usuarios y contraseñas

Una instalación nueva se entra como `admin` / `traffic66`, y el primer
inicio de sesión pide una contraseña nueva antes de mostrar nada más (la
demo mantiene `traffic66`). Después, **Cuenta**, al pie del menú, cambia su
contraseña; el administrador (`admin`) también añade y elimina usuarios y
restablece sus contraseñas allí. El inicio de sesión con LDAP / Active
Directory está en desarrollo.

Los usuarios se guardan como hashes con sal en `password` dentro del
directorio de datos. Lo mismo puede hacerse en la máquina de traffic66 con un
solo comando (añada `-data …` si traffic66 se ejecuta con esa opción):

| Para | Comando |
|---|---|
| Cambiar la contraseña de `admin` | `traffic66 passwd` |
| Añadir el usuario `alice`, o cambiar su contraseña | `traffic66 passwd -user alice` |
| Eliminar el usuario `alice` | `traffic66 passwd -user alice -delete` |
| Listar los usuarios | `traffic66 passwd -list` |

Los cambios se aplican al instante. Salvo la gestión de usuarios, todos los
usuarios tienen los mismos permisos. Para scripts y contenedores, `TRAFFIC66_PASSWORD=…` (o `-password`)
acepta en esa ejecución solo `-user` con esa contraseña. Cinco contraseñas
erróneas en un minuto bloquean la dirección durante un minuto.

<a id="4-send-flows-from-your-devices"></a>

## 4. Enviar flujos desde los equipos

`192.0.2.50` es traffic66 y `192.0.2.1` el equipo. Fije el timeout activo en
60 segundos, deje que los equipos NetFlow/IPFIX exporten sus opciones de
muestreador y muestree **cada interfaz en entrada** (o solo las interfaces
de borde): así cada paquete cuenta una sola vez. Tasas sFlow: alrededor de
1:1000 para 1 Gb/s, 1:4096 para 10 Gb/s, 1:8192 para 40/100 Gb/s.

```
! Cisco IOS-XE, NetFlow v9
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
# Huawei CloudEngine, sFlow (H3C Comware is similar)
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
# Juniper EX/QFX, sFlow
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow interfaces ge-0/0/0

# Arista EOS, sFlow
sflow sample 4096
sflow destination 192.0.2.50
sflow run

# MikroTik RouterOS 7
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9

# Linux host, softflowd
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Comprobar que llegan los flujos

**Configuración** muestra en segundos cada equipo que envía algo: protocolo,
tasa de muestreo, pérdidas, las interfaces que muestrea y qué corregir
cuando el estado no está en verde. Las pérdidas de sFlow se dividen en
pérdidas por el camino (suba `net.core.rmem_max` si `netstat -su` muestra
errores de búfer) y muestras que descartó el propio equipo.

![Configuración: cada equipo con su protocolo, muestreo, pérdidas y qué corregir](images/sources.png)

¿Falta un equipo? Ejecute `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739`: si no aparece nada, el problema es el enrutamiento, un
firewall o la configuración del equipo; si llegan paquetes pero no hay nada
en **Configuración**, es el firewall local o `-listen`.
`traffic66 simulate -to 192.0.2.50` desde otra máquina prueba el camino con
equipos simulados.

<a id="6-interfaces-and-counters"></a>

## 6. Interfaces y contadores

Las cifras de flujo son estimaciones (muestras × tasa de muestreo).
**Verificación de interfaces** las compara con los contadores de interfaz
del equipo (contadores sFlow, o SNMP mediante una línea `snmp` en Nombres) y
explica por qué difieren: interfaces sin muestrear, el mismo tráfico
muestreado dos veces, pérdidas por el camino o una tasa de muestreo
desconocida. Cada interfaz tiene un gráfico en bits/s y otro en paquetes/s,
la entrada en verde y la salida en azul, los contadores discontinuos.

En cada fila, **✎** asigna un nombre y una etiqueta corta (como *uplink*) y
**☆** la convierte en la interfaz predeterminada (★), en la que se abren las
páginas.

Un equipo que muestrea solo algunas interfaces muestra también los otros
extremos de esos flujos. Estas **interfaces del otro extremo** aparecen al
final en letra pequeña y gris: solo contienen el tráfico que pasa por la
interfaz muestreada. La interfaz muestreada se conoce por la fuente de datos
de sFlow o por el campo flowDirection (IPFIX 61); sin él, es la interfaz que
lleva el 90% del tráfico del equipo.

![Verificación de interfaces: tráfico de cada interfaz, y la estimación de flujos junto al contador del equipo](images/interfaces.png)

traffic66 ya usa la tasa que aplicó el equipo, espera a las tasas
desconocidas, compensa las pérdidas de exportación, reparte los flujos
largos entre sus minutos y añade 18 bytes por paquete de sobrecarga
Ethernet a NetFlow/IPFIX (`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Nombres, países y listas de amenazas

Haga clic en cualquier dirección y elija **Ponerle nombre…**, o use
**Configuración → Nombres**. Los nombres se guardan en `inventory.txt`
dentro del directorio de datos:

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

Los rangos privados siempre son suyos. Los cambios se aplican con
**Guardar**, sin reiniciar.

Los países y las redes (AS) funcionan de entrada con las bases gratuitas
Lite de DB-IP ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **Configuración**
las actualiza, o acepta en su lugar archivos de MaxMind GeoLite2, IPinfo Lite
o IPtoASN. Contornos del mapa: [Natural Earth](https://www.naturalearthdata.com).

![Geografía y redes: tráfico remoto por país en un mapa del mundo](images/geo.png)

Las listas de amenazas son archivos de texto con una dirección o red por
línea en `<data>/threats/<name>.txt` (por ejemplo Spamhaus DROP); reinicie
tras modificarlas. Las coincidencias aparecen en **Inteligencia de
amenazas**.

![Inteligencia de amenazas: un host interno enviando datos a una dirección de una lista de amenazas](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Uso de la interfaz web

Se puede hacer clic en cualquier valor de cualquier página: **Mostrar solo
esto** / **Excluir esto** (los filtros se aplican a todas las páginas),
**Ver sus registros de flujo**, **Ver detalles** (una página sobre un host
o servicio), **Ponerle nombre…**, **Buscarlo en línea**, **Copiar**.

| Página | Qué muestra |
|---|---|
| Resumen | El ancho de banda de la interfaz elegida; tráfico por aplicación (total, entrante o saliente) frente a ayer o la semana pasada; hallazgos abiertos; principales clientes y servicios |
| Top 66 | Las 66 primeras conversaciones, ordenables por cualquier columna, o agrupadas por aplicación, red, segmento, equipo, encapsulación, VLAN; los 30 interlocutores principales |
| Detalles del tráfico | Gráficos de anillos: servidores y sus clientes (o al revés), y servicios |
| Rutas de tráfico | Host → aplicación → país, o cliente → servicio → servidor, o por red |
| Verificación de interfaces | Cada interfaz en el tiempo frente a sus contadores; nombres, etiquetas, la predeterminada |
| Registros de flujo | Los flujos individuales, en vivo cada 5 segundos o para cualquier rango de tiempo |
| Hallazgos, Inteligencia de amenazas | Qué requiere atención ([más abajo](#findings)); tráfico con direcciones listadas |
| Geografía y redes | Un mapa del mundo por país, redes (AS) en el tiempo |
| Configuración | Equipos, muestreo, pérdidas, SNMP, bases de datos, logotipo, nombres |
| Análisis offline de pcap, Limpieza de datos | Archivos de captura ([más abajo](#9-offline-pcap-terminal-ui-local-capture)); borrado de datos antiguos |

Encima de las páginas: **Interfaz** (todas, o una interfaz muestreada; las
páginas de tráfico muestran entonces solo el tráfico que pasa por ella), el
rango de tiempo (de 15 minutos a 30 días, o personalizado), actualización
cada 30 s y **Copiar enlace** para la vista exacta. El idioma y cinco temas
de color están al pie del menú. Los gráficos terminan donde los datos están
completos: con NetFlow/IPFIX, tan tarde como exporten los equipos (como
mucho 2 minutos). Los rangos de más de 6 horas empiezan en una hora en
punto; una interfaz durante 7 o 30 días lee el detalle de flujos, así que es
más lento y llega solo hasta donde se conserva el detalle.

![Top 66: las 66 primeras conversaciones, ordenables por cualquier columna](images/topn.png)

![Detalles del tráfico: servidores con sus clientes, y servicios con sus servidores, en gráficos de anillos](images/traffic.png)

![Detalles de un host: los hallazgos sobre él, su tráfico, con quién habla, servicios, países y últimos flujos](images/detail.png)

![Rutas de tráfico: qué host usa qué aplicación hacia qué país](images/paths.png)

![Resumen en chino](images/overview-zh.png)

<a id="findings"></a>

### Hallazgos

Se comprueban cada 5 minutos sobre los últimos 10; algo que dura una hora es
un único hallazgo que va creciendo.

| Hallazgo | Qué significa |
|---|---|
| Escaneo, escaneo de puertos | Pequeñas sondas a muchos hosts en un puerto, o a muchos puertos de un host |
| Adivinación de contraseñas | Muchas conexiones cortas a un servicio de inicio de sesión |
| Movimiento lateral | Compartición de archivos o administración remota hacia hosts internos que nunca antes la ofrecieron |
| Subida inusual | 100 MB en 10 minutos a una dirección nueva, el triple de lo que volvió |
| Inundación | 20.000+ paquetes pequeños/s hacia una dirección, diez veces su ritmo habitual |
| Lista de amenazas | Tráfico con una dirección listada |

Desde dentro de su red son de gravedad alta, desde internet baja.
**Resuelto** cierra un hallazgo, **No es un problema** lo silencia para
siempre. El movimiento lateral y las subidas necesitan un día de historial.
A través de un muestreo 1:4096 el ataque de la demo se detecta por completo;
los escaneos muy pequeños pueden ocultarse tras el muestreo.

![Hallazgos: cada paso de un ataque, encontrado a través de un muestreo sFlow 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Análisis offline de pcap, interfaz de terminal, captura local

**Análisis offline de pcap** muestra capturas de paquetes (pcap, pcapng) con
las mismas páginas, aparte de los datos en vivo: `traffic66 a.pcap b.pcapng`
arranca en 127.0.0.1 y abre el navegador (hasta 3 archivos, 3 GB; Ctrl+C
borra los datos importados), o suba en esa página hasta 3 archivos de 50 MB.
Se analiza un archivo a la vez, cada uno en su propia base de datos:
**Analizar** en la fila de un archivo lo muestra en todas las páginas, y la
barra superior cambia a otro. Trabaja con flujos, no con el contenido de los
paquetes.

![Análisis offline: archivos de captura con sus paquetes, flujos y tiempo](images/sandbox.png)

**Interfaz de terminal**: `traffic66 tui` en la máquina de traffic66, o
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Teclas: 1–8 páginas, Enter acciones, f mostrar solo, x excluir, t rango de
tiempo, w abrir en un navegador, q salir; `-lang` elige el idioma.

![Interfaz de terminal: resumen](images/tui-overview.png)

![Interfaz de terminal: Top 66 de conversaciones](images/tui-topn.png)

**Captura local** construye flujos desde una interfaz local, idealmente un
puerto conectado al puerto espejo de un switch: `traffic66 interfaces` las
lista y `-capture eth1` captura. En Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux necesita root o `setcap cap_net_raw,cap_net_admin+ep`, macOS root,
Windows [Npcap](https://npcap.com). Los flujos capturados provienen del
equipo `127.0.0.1`. La captura local no tiene interfaces ni contadores del
equipo, así que **Verificación de interfaces** no tiene nada que comparar
para ella.

<a id="10-options-and-data"></a>

## 10. Opciones y datos

`traffic66 -h` lo lista todo. Las más usadas:

| Opción | Por defecto | |
|---|---|---|
| `-data` | `traffic66-data` junto al programa | directorio de datos |
| `-addr` | `:8066` | interfaz web; `127.0.0.1:8066` solo para esta máquina |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | colectores UDP; vacío los desactiva |
| `-retention-days` | `30` | días de detalle de flujos; los resúmenes se guardan 400 días |
| `-memory` | `0.10` | fracción de la RAM para la caché de la base de datos |
| `-sampling-wait` | `5m` | cuánto esperan los registros a una tasa de muestreo |
| `-capture` | | interfaz local (repetible) |
| `-no-dns` | | sin resoluciones inversas |

El directorio de datos contiene `raw/` (detalle, un archivo por hora),
`traffic66.duckdb` (resúmenes y contadores), `password`, `inventory.txt`,
`license.json`, su logotipo y las bases de datos. Para hacer una copia de
seguridad, detenga traffic66 y cópielo; para actualizar, reemplace el
archivo del programa. **Limpieza de datos** borra los datos de más de 7–120
días, o todos.

<a id="licence"></a>

### Licencia

Código fuente disponible bajo la [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
y la [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (el texto
en inglés es el vinculante): gratis para evaluación y para organizaciones de
menos de 100 personas; las organizaciones más grandes se registran tras 30
días de uso en producción; vender, alojar para terceros o productos
competidores requieren una licencia comercial. Nunca se desactiva nada. El
pie de cada página muestra el número de instalación de 8 dígitos; envíelo
al autor y ponga el `license.json` que reciba en el directorio de datos.
Contacto: <https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Seguridad, dimensionamiento, solución de problemas

La interfaz web es HTTP simple: en redes no confiables use
`-addr 127.0.0.1:8066` detrás de un proxy TLS (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) o un túnel SSH. Permita los puertos UDP solo desde sus
equipos. Las comunidades SNMP se guardan en texto claro; use comunidades de
solo lectura.

A 5.000 flujos/s en 2 núcleos: unos 12 GB de disco por día de detalle
(360 GB para 30 días), una sexta parte de un núcleo, 0,6–0,8 GB de memoria.
Los resúmenes de rangos largos tardan menos de 0,2 s; un Top 66 de una hora
de todas las conversaciones, unos 9 s.

| Síntoma | Solución |
|---|---|
| "esperando la tasa de muestreo" | Exporte las opciones del muestreador, o `sampling=N` / `unsampled` en la línea del equipo |
| Por debajo de los contadores | Interfaces sin muestrear, pérdidas o un timeout activo de más de 60 s |
| Por encima de los contadores | El mismo tráfico muestreado en dos interfaces o equipos |
| Olvidó la contraseña | `traffic66 passwd` en la máquina de traffic66 |
| `Conflicting lock is held` | Otro traffic66 usa este directorio de datos |
| `address already in use` | Elija otros puertos con `-addr` o `-listen` |
| Windows: "Windows protegió su PC" | **Más información** → **Ejecutar de todas formas** |

Compilar desde el código fuente: Go 1.24 y un compilador de C, luego `scripts/build.sh 0.1.0 traffic66`.
