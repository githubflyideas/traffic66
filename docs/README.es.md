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
- Detecta en los flujos escaneos, adivinación de contraseñas, movimiento
  lateral, subidas inusuales, inundaciones y tráfico de listas de amenazas,
  también a través del muestreo, y los muestra como hallazgos que atender.
- Listas Top 66, quién habla con quién en gráficos de anillos (servidores
  y sus clientes, servicios y sus servidores), tráfico en el tiempo por
  interfaz y red (AS), rutas de tráfico, países en un mapa del mundo,
  coincidencias con listas de amenazas, registros de flujo, encapsulación
  (GRE, IPIP, VXLAN, GENEVE, MPLS).
- `traffic66 captura.pcap` abre hasta 3 capturas de paquetes (3 GB en total) en la interfaz web: flujos, hallazgos, países y registros de toda la captura, sin configurar nada.
- 13 idiomas en la interfaz web y en la de terminal.
- Código fuente disponible: gratis para evaluación y para organizaciones
  de menos de 100 personas; las organizaciones más grandes se registran
  tras 30 días de uso en producción. Nunca se desactiva nada (vea
  [Prueba y licencia](#trial-and-licence)).

![Resumen: hallazgos abiertos, ancho de banda por aplicación frente a la misma hora de ayer, principales clientes y servicios](images/overview.png)

<sub>Todas las capturas de pantalla proceden de `traffic66 demo`, una red de empresa simulada que puede ejecutar usted mismo (vea [Probar la demo](#1-try-the-demo)).</sub>

<a id="contents"></a>

## Contenido

1. [Probar la demo](#1-try-the-demo)
2. [Instalación](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Usuarios y contraseñas](#3-users-and-passwords)
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
vivo de cuatro equipos simulados, e incluye un ataque: **Hallazgos**
muestra cada uno de sus pasos (un escaneo, un escaneo de puertos,
adivinación de contraseñas, movimiento lateral, una subida a un servidor de
control) y una inundación contra el sitio web público. Haga clic en
**Detalles** en un hallazgo, o empiece en **Resumen**, haga clic en un host
de **Principales clientes**, elija **Ver detalles** y siga haciendo clic a
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
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
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

<a id="3-users-and-passwords"></a>

## 3. Usuarios y contraseñas

**En resumen:** los usuarios y las contraseñas están en un solo archivo,
`password`, en el directorio de datos. Nunca se edita a mano: el comando
`traffic66 passwd` añade, cambia, lista y elimina usuarios. Abra
`http://<traffic66 machine>:8066` e inicie sesión con uno de ellos.

<a id="the-first-sign-in"></a>

### El primer inicio de sesión

En su primer arranque, traffic66 crea el usuario `admin` con una contraseña
aleatoria y la muestra una sola vez:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Iniciado con doble clic en Windows: en la ventana negra.
- En una terminal: en la terminal.
- Servicio de Linux: `journalctl -u traffic66 | grep "first start"`
- Servicio de macOS: `grep "first start" /Library/Logs/traffic66.log`

¿No la vio? Fije una nueva con `traffic66 passwd` (más abajo). Si fijó una
contraseña con `traffic66 passwd` antes del primer arranque, como hacen los
pasos de instalación anteriores, no se genera ninguna.

<a id="where-the-users-are-stored"></a>

### Dónde se guardan los usuarios

En el archivo `password` del directorio de datos:

| Cómo se ejecuta traffic66 | Archivo |
|---|---|
| Descomprimido y arrancado desde su carpeta (predeterminado) | `traffic66-data/password` junto al programa |
| Servicio de Linux (sección 2) | `/var/lib/traffic66/password` |
| Tarea de inicio de Windows (sección 2) | `C:\traffic66\traffic66-data\password` |
| Servicio de macOS (sección 2) | `/Library/Application Support/traffic66/password` |
| Demo | `traffic66-demo/password` junto al programa |

Una línea por usuario. Las contraseñas se guardan como hashes con sal, así
que nadie puede recuperarlas del archivo, ni siquiera usted; si se olvida
una contraseña, fije una nueva. Solo el propietario puede leer el archivo.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### Gestión de usuarios

Ejecute esto en la máquina de traffic66:

| Para | Comando |
|---|---|
| Cambiar la contraseña de `admin` | `traffic66 passwd` |
| Añadir el usuario `alice`, o cambiar su contraseña | `traffic66 passwd -user alice` |
| Eliminar el usuario `alice` | `traffic66 passwd -user alice -delete` |
| Listar los usuarios | `traffic66 passwd -list` |
| Fijar una contraseña aleatoria y mostrarla | `traffic66 passwd -generate` (con `-user` para otros usuarios) |

- El comando pide la nueva contraseña dos veces y no muestra lo que escribe.
  Use al menos 8 caracteres.
- Si traffic66 se ejecuta con `-data`, añada el mismo `-data` al comando.
  Para el servicio de Linux de la sección 2:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  En Windows (PowerShell como administrador):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- Los cambios se aplican de inmediato, sin reiniciar: una contraseña nueva
  sirve en el siguiente inicio de sesión, y un usuario eliminado pierde la
  sesión en los navegadores abiertos.
- No se puede eliminar el último usuario que queda; añada otro antes.
- Todos los usuarios ven y pueden cambiar lo mismo; no hay roles.

<a id="passwords-for-scripts-and-containers"></a>

### Contraseñas para scripts y contenedores

`TRAFFIC66_PASSWORD=…` en el entorno, o `-password …` en la línea de
comandos, hace que traffic66 acepte exactamente un usuario en esa ejecución:
el indicado con `-user` (por defecto `admin`) con esa contraseña. El archivo
`password` se ignora entonces y no se modifica. Es preferible la variable de
entorno: las líneas de comandos son visibles para los demás usuarios de la
máquina.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

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

Abra **Configuración**. Cada equipo que envía algo aparece en segundos, con su
protocolo, tasa de muestreo, pérdidas, último paquete y un estado. Cuando el
estado no está en verde, el texto de al lado indica qué falla y qué hay que
cambiar.

**Perdidos** cuenta las muestras o registros que nunca llegaron. Para sFlow
el texto indica dónde se perdieron: por el camino (saltos en los números
de secuencia: la red, o el búfer de recepción UDP de esta máquina; si
`netstat -su` muestra errores de búfer de recepción en aumento, suba
`net.core.rmem_max`), o en el propio equipo (sFlow informa de las muestras
que descartó el equipo: su exportación sFlow tiene un límite de ritmo, así
que muestree con menos frecuencia o suba el límite del equipo). Los totales
se compensan en ambos casos; el detalle por host no.

![Configuración: cada equipo con su protocolo, muestreo, pérdidas y qué corregir](images/sources.png)

Si un equipo no aparece:

1. Observe si llegan paquetes a la máquina de traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Si no se ve nada, los paquetes no llegan a la máquina: revise la
   configuración del equipo, el enrutamiento y los firewalls del camino.
2. Llegan paquetes pero **Configuración** sigue vacía: el firewall local los
   descarta (vea [Instalación](#2-install)) o traffic66 escucha en otros
   puertos (`-listen`).
3. Para probar el camino desde otra máquina sin tocar ningún equipo, ejecute
   allí `traffic66 simulate -to 192.0.2.50` durante unos segundos. Envía
   sFlow, NetFlow e IPFIX desde equipos simulados, que luego aparecen en
   **Configuración** y en los datos, así que mejor hágalo en una instalación de
   pruebas.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Hacer que las cifras cuadren con los contadores de interfaz

Las cifras de flujo son estimaciones: paquetes muestreados por la tasa de
muestreo. traffic66 las compara con los contadores de interfaz del propio
equipo y muestra la diferencia en **Verificación de interfaces**, con la
causa probable cuando es mayor de lo que explica el propio muestreo. Cada
interfaz tiene un gráfico en bits/s a todo el ancho y, debajo, otro en
paquetes/s, con la entrada (verde) y la salida (azul); los contadores del
propio equipo aparecen como líneas discontinuas en el gráfico de bits/s. Al
elegir una interfaz en la lista se muestran sus gráficos.

Cada fila de la lista tiene dos botones. **✎** da a la interfaz un nombre y
una etiqueta corta (como *uplink*), que aparece junto a su nombre en todas
partes. **☆** la convierte en la interfaz predeterminada (**★**); solo hay
una. Las páginas se abren entonces en ella (vea la opción **Interfaz** en
[Uso de la interfaz web](#9-using-the-web-ui)), y el resumen muestra su
ancho de banda. Ambos se guardan al momento en la línea `iface` de Nombres.

![Verificación de interfaces: tráfico de cada interfaz, y la estimación de flujos junto al contador del equipo](images/interfaces.png)

Para disponer de contadores con los que comparar:

- Los equipos sFlow los envían por su cuenta cuando hay un intervalo de
  contadores configurado (`sflow counter interval 30` o similar).
- Para equipos NetFlow e IPFIX, añada una línea `snmp` en
  **Configuración → Nombres** (vea [Nombres](#7-names-snmp-and-your-own-networks)).
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

La forma más rápida de ponerle nombre a un host o a un equipo: haga clic en
su dirección en cualquier página y elija **Ponerle nombre…**. Escriba el
nombre y pulse Enter; se guarda al momento y se muestra en todas partes en
lugar de la dirección sin más.

Para redes, interfaces y SNMP, use **Configuración → Nombres**: elija el tipo
(host, red, dispositivo, interfaz, SNMP), rellene la dirección y el nombre,
y haga clic en **Añadir**. La tabla de abajo lista todos los nombres con
**Editar** y **Borrar**; añadir de nuevo la misma dirección sustituye la
entrada anterior. Las direcciones y redes se comprueban antes de guardar.

Los nombres se guardan como `inventory.txt` en el directorio de datos, una
entrada por línea. **Editar como texto (avanzado)** muestra ese archivo, y
también puede editarlo directamente (vea `inventory.txt.example`). Todas
las líneas son opcionales.

```
# your networks: traffic between them is "internal"
net    10.10.0.0/16  Office LAN
net    203.0.113.0/24  Public servers country=JP

# device names; "unsampled" if it exports every packet (1:1),
# sampling=N if it samples 1:N but does not say so in its export
device 192.0.2.1     Core router
device 192.0.2.9     Branch firewall unsampled
device 192.0.2.20    Edge router sampling=1000

# interface names, by device address and ifIndex; speed in bits per second,
# tag= a short tag, default = the interface the pages open on (one only)
iface  192.0.2.1 3   ISP uplink speed=1000000000 tag=uplink default

# host names shown instead of addresses
host   10.10.3.27    Finance PC

# read interface counters over SNMPv2c (IF-MIB 64-bit counters)
snmp   192.0.2.1     public
snmp   192.0.2.9     s3cret  10.99.0.9:161
```

- `net`: los rangos privados (10/8, 172.16/12, 192.168/16, 100.64/10) se
  consideran siempre propios. Añada sus rangos públicos para que el tráfico
  hacia y desde ellos también cuente como propio; el nombre aparece en
  **Top 66** agrupado por segmento y en las rutas de tráfico por segmento.
  `country=JP` (un código de país de dos letras) indica dónde está la red;
  el mapa del mundo traza entonces líneas desde ella hasta los países con
  los que se comunica.
- `snmp <device> <community> [<management address>[:port]]`: el equipo es
  la dirección de la que vienen los flujos. Añada la dirección de gestión
  cuando el equipo responda a SNMP en otra dirección. Las descripciones de
  interfaz leídas por SNMP se usan como nombres, salvo que nombre la
  interfaz con `iface`. Permita la máquina de traffic66 en la lista de
  acceso SNMP del equipo.
- Los cambios se aplican al pulsar **Guardar**; no hace falta reiniciar.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Países, redes y listas de amenazas

Los países y las redes (AS) funcionan desde el primer momento: traffic66 incluye las bases de datos gratuitas **IP to Country Lite** e **IP to ASN Lite** de DB-IP (licencia [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/); "IP Geolocation by DB-IP", [db-ip.com](https://db-ip.com)). Las páginas que muestran países y redes indican de dónde vienen los datos.

La copia incluida es la de la versión que usa. DB-IP publica una nueva cada mes; **Configuración → Base de datos de países y redes → Actualizar DB-IP Lite ahora** descarga la última de db-ip.com (el servidor donde corre traffic66 necesita acceso a internet; si falla, la interfaz lo indica).

También puede usar otra base de datos gratuita. Descárguela y súbala en la misma página con **Subir un archivo de base de datos…**. Se comprueba, se guarda en el directorio de datos y se usa para el tráfico nuevo al instante, sin reiniciar. El tráfico ya guardado conserva el país con el que se guardó.

| Base de datos | Da | Licencia | Dónde obtenerla |
|---|---|---|---|
| DB-IP Lite (incluida) | países; redes | CC BY 4.0, sin cuenta | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country y ASN, `.mmdb` | países; redes | GeoLite2 EULA, cuenta gratuita | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite, `ipinfo_lite.mmdb` | países y redes en un solo archivo | CC BY-SA 4.0, cuenta gratuita | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN, `ip2asn-combined.tsv.gz` | redes con su país | PDDL 1.0, sin cuenta | [iptoasn.com](https://iptoasn.com) |

Primero se usan sus archivos; lo que no cubren lo responde la DB-IP Lite incluida. **Quitar** junto a un archivo vuelve al resto. La página muestra lo que está en uso y la fecha de cada base de datos.

Sin la interfaz web, copie el archivo al directorio de datos como `country.mmdb`, `asn.mmdb`, `both.mmdb` (un archivo con países y redes, como IPinfo Lite) o `asn.tsv.gz` y reinicie traffic66.

**Geografía y redes** muestra en un mapa del mundo el tráfico con otros países: cuanto más oscuro, más tráfico. Al pasar el ratón sobre un país se ve su tráfico; al hacer clic se puede filtrar o abrir sus registros de flujo. Cuando sus redes tienen un país (`country=` en una línea `net`, vea [Nombres](#7-names-snmp-and-your-own-networks)), unas líneas van de ese país a los países con los que intercambian tráfico, más gruesas cuanto más tráfico. Los contornos de los países son de [Natural Earth](https://www.naturalearthdata.com) (dominio público).

![Geografía y redes: tráfico remoto por país en un mapa del mundo](images/geo.png)

Las listas de amenazas son archivos de texto plano con una dirección o red
por línea (se ignora lo que va tras `#` o `;`), guardados como
`<data directory>/threats/<name>.txt`, por ejemplo:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Reinicie traffic66 después de añadir o modificar listas. Las coincidencias
aparecen en **Inteligencia de amenazas**, por nombre de lista.

![Inteligencia de amenazas: un host interno enviando datos a una dirección de una lista de amenazas](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Uso de la interfaz web

Casi nunca hace falta escribir. Cualquier valor de cualquier página (una
dirección, un puerto, una aplicación, un país, un equipo) admite clic:

- **Mostrar solo esto** / **Excluir esto** añade un filtro. Los filtros
  aparecen bajo la barra superior y se aplican a todas las páginas hasta
  que los quite.
- **Ver sus registros de flujo** abre los flujos individuales
  correspondientes.
- **Ver detalles** (hosts, equipos y servicios) abre una página sobre ese
  host o servicio: su tráfico en el tiempo por aplicación, con quién habla,
  qué servicios o clientes, países y sus últimos flujos. Ahí también se
  puede hacer clic en cualquier valor, así que puede seguir profundizando;
  el botón Atrás del navegador vuelve.
- **Ponerle nombre…** (hosts y equipos) le da un nombre a la dirección, que
  se muestra en todas partes a partir de entonces.
- **Buscarlo en línea** abre la dirección o el AS en un sitio público de
  consulta.
- **Copiar** copia el valor.

Páginas:

| Página | Qué responde |
|---|---|
| Resumen | El ancho de banda de la interfaz elegida (o de la predeterminada, o si no de la de más tráfico) en bits/s, entrada y salida vistas desde la interfaz; cuánto tráfico hay ahora, por aplicación (**Total**, o solo el tráfico **Entrante** o **Saliente** de sus redes), frente a la misma hora de ayer (periodos de hasta un día), la semana pasada (hasta una semana) o los días anteriores (periodos más largos), cuando hay datos de entonces; hallazgos abiertos; dirección y protocolo; principales clientes y servicios |
| Top 66 | Se abre en **Tabla**, una sola tabla de los 66 primeros: por defecto, conversaciones (cliente, servidor, servicio, país). Cualquier encabezado ordena; las columnas numéricas (tráfico, paquetes, paquete medio, flujos) clasifican todo el tráfico del periodo, así el menor paquete medio revela escaneos e inundaciones. **Agrupar por** cambia a aplicaciones, redes, segmentos, equipos, encapsulación y VLAN. **Interlocutores principales** muestra los 30 primeros clientes y servidores lado a lado con tráfico, paquetes y registros de flujo, sobre una fila para todo el tráfico |
| Detalles del tráfico | Dos gráficos de anillos. **Servidores y clientes**: el anillo interior son los 8 servidores con más tráfico, el exterior los clientes de cada uno; **Clientes dentro** le da la vuelta (clientes dentro, fuera los servidores que usa cada uno), ya que a menudo un lado explica más que el otro. **Servicios**: un anillo con los servicios con más tráfico. Al pasar el ratón sobre un segmento se ve su tráfico; se hace clic en él como en cualquier valor |
| Rutas de tráfico | Qué host usa qué aplicación hacia qué país: los 8 hosts con más tráfico, el resto como Otros. **Cliente → servidor** muestra cliente → servicio → servidor; **Por segmento** muestra segmentos en lugar de hosts. Los nombres largos se acortan a 22 caracteres; al pasar el ratón se ve el nombre completo |
| Hallazgos | Qué requiere atención: escaneos, adivinación de contraseñas, movimiento lateral, subidas inusuales, inundaciones y tráfico de listas de amenazas ([más](#findings)) |
| Inteligencia de amenazas | Hosts que hablaron con direcciones de sus listas de amenazas y cuánto enviaron |
| Geografía y redes | Un mapa del mundo del tráfico por país, con líneas desde sus redes; las redes (AS) de las que vino y a las que fue el tráfico, en el tiempo en bits/s y paquetes/s; tráfico por país y por red |
| Configuración | Equipos, muestreo, pérdidas, colectores, SNMP, la base de datos de países y redes, el logotipo y **Nombres** |
| Verificación de interfaces | Tráfico de cada interfaz en el tiempo en bits/s y, debajo, paquetes/s, entrada (verde) y salida (azul), con los contadores del equipo como líneas discontinuas; cuánto se alejan las cifras de flujo de los contadores, de peor a mejor, con motivos; un nombre, una etiqueta y la predeterminada para cada interfaz |
| Registros de flujo | Cuántos registros de flujo hubo y cuándo (una barra por intervalo), y los registros mismos, del más reciente al más antiguo, página a página, con columnas seleccionables. Se abre en los últimos 15 minutos, actualizados cada 5 segundos; abierta desde un valor de otra página (**Ver sus registros de flujo**) conserva el rango de tiempo de esa página, y **Volver a tiempo real** regresa |
| Limpieza de datos | Borra los datos de más de 120, 90, 60, 30 o 7 días, o todos, indicando cuánto libera cada opción ([más](#13-data-backup-upgrade-uninstall)) |
| Análisis offline de pcap | Capturas de paquetes (pcap, pcapng) analizadas aparte de los datos en vivo ([más](#análisis-offline-de-pcap)) |

El menú lateral agrupa las páginas en cuatro bloques: tráfico (Resumen,
Top 66, Detalles del tráfico, Rutas de tráfico, Verificación de
interfaces), seguridad (Hallazgos, Inteligencia de amenazas, Geografía y
redes), configuración y datos (Configuración, Registros de flujo, Limpieza
de datos) y Análisis offline de pcap. Bajo el logotipo aparecen la versión y
la fecha y hora del servidor.

Encima de las páginas de tráfico (Resumen, Top 66, Detalles del tráfico,
Rutas de tráfico, Geografía y redes, Registros de flujo y el detalle de un
valor) está **Interfaz**: **Todas las interfaces**, o una interfaz, para que
estas páginas muestren solo el tráfico que pasa por ella (de entrada o de
salida). Empieza en la interfaz predeterminada (★, fijada en
**Verificación de interfaces**) y la elección forma parte del enlace.
Hallazgos, Inteligencia de amenazas, Verificación de interfaces y
Configuración siempre abarcan todo el tráfico. Para una interfaz en 7 o 30
días las páginas leen los registros de flujo en lugar de los resúmenes por
hora y por día, así que tardan más y llegan hasta donde se conservan los
registros de flujo (30 días por defecto).

Encima de las páginas: rango de tiempo (de 15 minutos a 30 días, o
**Personalizado…** para cualquier inicio y fin, también más allá de 30
días), refresco automático cada 30 segundos y **Copiar enlace**, que copia
un enlace exactamente a la vista actual (página, rango de tiempo y
filtros) para enviárselo a un compañero. En **Top 66** y **Detalles del
tráfico**, **Dispositivo**, **Cliente**, **Servidor** y **Servicio**
listan los valores con más tráfico del rango:
elija uno, o escríbalo, para filtrar; el filtro se aplica entonces a todas
las páginas hasta que vacíe el cuadro. El idioma sigue al del navegador; se
cambia al final del menú, encima de **Cerrar sesión**. Configuración,
Registros de flujo (en tiempo real), Limpieza de datos y Análisis offline
de pcap no tienen rango de tiempo.

Junto al idioma está el tema de colores, inspirado en los colores del
sistema de iOS: **Claro** (el predeterminado), **Gris**, **Negro** (para
pantallas de pared), **Turquesa** y **Naranja**. Cada clic pasa al
siguiente; la elección se guarda en el navegador.

Los gráficos en el tiempo muestran los 8 valores mayores en colores fijos y
el resto como Otros; la leyenda da el total de cada valor y se puede hacer
clic en ella como en cualquier otro valor. Los gráficos de clientes y
servidores dejan el resto fuera del dibujo, ya que con miles de hosts
aplanaría a los 8 primeros; la leyenda sigue dando su total.

Los gráficos terminan donde los datos están completos: con sFlow en el minuto
actual, con NetFlow e IPFIX un poco antes, tanto como tarden los dispositivos
en exportar sus flujos (traffic66 lo mide; como máximo 2 minutos).

Los rangos de más de 6 horas empiezan en una hora en punto, de modo que
todas las cifras de la página cuentan exactamente el mismo tiempo: "24 horas"
abarca las últimas 24 horas completas más la actual. En estos rangos, Top 66
sale de resúmenes horarios; ahí no hay filtros, y la página lo indica. Elija
un rango más corto para filtrar. Las conversaciones siempre leen el detalle de flujos,
así que en rangos largos con muchos flujos pueden tardar; una hora es lo más rápido.

El menú lateral muestra cuánto disco usan los datos y cuánto queda libre;
pase el ratón por el espacio libre para ver cuánto necesitan, al ritmo
actual, los días de detalle que se conservan (se estima en cuanto hay un
día de datos).

Para mostrar su propio logotipo en la página de inicio de sesión y arriba
del menú, use **Configuración → Logotipo → Subir un logotipo…**: PNG, SVG, JPEG,
WebP o GIF, hasta 1 MB, idealmente de 272 × 92 píxeles (otros tamaños se
ajustan). **Usar el logotipo integrado** vuelve al de traffic66.

<a id="findings"></a>

### Hallazgos

**Hallazgos** enumera lo que traffic66 encontró en los flujos, empezando por
lo más grave. Revisa los últimos 10 minutos cada 5 minutos; algo que dura
una hora es un solo hallazgo que crece, no uno nuevo en cada revisión.

| Hallazgo | Qué significa | Gravedad |
|---|---|---|
| Escaneo | Una dirección envió pequeñas sondas a muchas direcciones en un mismo puerto (TCP o ping) | Alta desde dentro de su red, baja desde internet |
| Escaneo de puertos | Una dirección envió pequeñas sondas a muchos puertos de un mismo host | Alta desde dentro, baja desde internet |
| Adivinación de contraseñas | Muchas conexiones cortas a un servicio de inicio de sesión (SSH, RDP, SMB, bases de datos y otros) | Alta desde dentro, baja desde internet |
| Movimiento lateral | Dentro de su red, sesiones de compartición de archivos o de administración remota (SMB, RDP, SSH, WinRM, VNC) hacia hosts que nunca antes ofrecieron ese servicio | Alta |
| Subida inusual | Un host interno envió mucho más de lo que recibió (100 MB en 10 minutos, el triple de lo recibido) a una dirección con la que no había intercambiado datos antes | Alta |
| Inundación | 20.000 o más paquetes pequeños por segundo hacia una dirección, diez veces su ritmo habitual | Media |
| Lista de amenazas | Tráfico con una dirección de una de sus listas de amenazas | Alta cuando su host se conectó a ella, baja cuando la dirección listada llamó desde fuera |

Cada hallazgo dice quién hizo qué a quién, cuándo y durante cuánto tiempo,
con las cifras que lo respaldan y cómo se muestrearon los datos.
**Detalles** abre la página del host, que también enumera los hallazgos
sobre él. **Resuelto** cierra un hallazgo; si vuelve a ocurrir, se abre uno
nuevo. **No es un problema** lo cierra para siempre: no se vuelve a
notificar. El número rojo junto a **Hallazgos** en el menú lateral cuenta
los hallazgos abiertos de gravedad alta y media de las últimas 24 horas.

El movimiento lateral y las subidas inusuales necesitan saber qué es
normal, así que se notifican en cuanto hay un día de historial. En el
primer arranque, traffic66 aprende del historial que ya tiene.

Con datos muestreados (sFlow, NetFlow muestreado), las reglas cuentan lo que
muestran las muestras y piden menos, pero entonces cada una debe parecer una
sonda corta, de modo que los hosts normales con mucha actividad no las
disparan. Lo que el muestreo oculta no se puede encontrar: tras un muestreo
1:4096, un escaneo de unas decenas de hosts envía demasiado pocos paquetes
para verse. El ataque de la demo pasa por un switch que muestrea a 1:4096 y
se encuentra entero; un día de tráfico normal de la demo no produce ningún
hallazgo salvo el escáner de internet que llama a la puerta del sitio web.

![Hallazgos: cada paso de un ataque, encontrado a través de un muestreo sFlow 1:4096](images/findings.png)

![Top 66: las 66 primeras conversaciones, ordenables por cualquier columna](images/topn.png)

![Detalles del tráfico: servidores con sus clientes, y servicios con sus servidores, en gráficos de anillos](images/traffic.png)

![Detalles de un host: los hallazgos sobre él, su tráfico, con quién habla, servicios, países y últimos flujos](images/detail.png)

![Rutas de tráfico: qué host usa qué aplicación hacia qué país](images/paths.png)

El mismo resumen en chino; todas las páginas están disponibles en 13 idiomas:

![Resumen en chino](images/overview-zh.png)

<a id="10-terminal-ui"></a>

### Análisis offline de pcap

**Análisis offline de pcap** muestra capturas de Wireshark o tcpdump con las mismas páginas que los datos en vivo, sin mezclarlas.

Resume todos los paquetes en flujos: quién habló con quién, cuánto, cuándo y qué parece un ataque. No decodifica protocolos ni muestra el contenido de los paquetes; para un paquete o un flujo TCP, use Wireshark.

Desde la línea de comandos, sin configurar nada:

```
traffic66 office.pcap
traffic66 a.pcap b.pcapng c.pcap
```

traffic66 arranca solo en este equipo (127.0.0.1, un puerto libre), muestra la dirección, la contraseña y un enlace de acceso de un solo uso, y abre el navegador en la captura. Hasta 3 archivos, 3 GB en total; se leen donde están y nunca se modifican. No se recoge ni se envía nada, y no se resuelven nombres de host (`-dns` lo activa). Ctrl+C detiene y borra los datos importados. En una máquina de 2 núcleos, una captura de 1 GB está lista en unos 5 segundos (1,2 millones de paquetes grandes) a 30 segundos (14 millones de paquetes pequeños).

```
$ traffic66 office.pcap

traffic66 0.3.1: analysing 1 capture file(s); nothing is collected or sent
  Web UI    http://127.0.0.1:38217  (port 38217, this computer only)
  Sign in   user admin, password gfhfhbuutz2e
  Open      http://127.0.0.1:38217/auto?t=b9388f…  (signs in once)
  Stop      Ctrl+C; the imported data is deleted, your files are kept
```

En la interfaz web de un traffic66 en marcha:

1. **Subir archivos de captura…**: `.pcap` o `.pcapng`, sin comprimir. Hasta 3 archivos, cada uno de 50 MB como máximo. Los archivos se convierten en flujos en una base de datos propia (`<data>/sandbox/`); los datos en vivo, sus cifras y hallazgos no se tocan.
2. **Analizar**: todas las páginas (resumen, Top 66, detalles del tráfico, hallazgos, rutas, mapa, registros de flujo) muestran los archivos durante todo su tiempo. Una barra naranja nombra los archivos; **Volver a los datos en vivo** vuelve. Cada archivo aparece como un dispositivo, así que el cuadro **Dispositivo** muestra un archivo cada vez.
3. Las reglas de detección se aplican a la captura: barridos, escaneos de puertos y adivinación de contraseñas aparecen en **Hallazgos**. Las reglas que necesitan un día de historial (movimiento lateral, subidas inusuales) no se aplican a una captura.
4. **Eliminar** borra un archivo y sus datos; **Eliminar todo** lo borra todo.

La demo incluye una captura de ejemplo con un ataque.

![Análisis offline: archivos de captura con sus paquetes, flujos y tiempo](images/sandbox.png)

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

![Interfaz de terminal: resumen](images/tui-overview.png)

![Interfaz de terminal: Top 66 de conversaciones](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Captura local

Además de recibir exportaciones de flujos, traffic66 puede generar flujos por
sí mismo a partir de los paquetes de una interfaz de red de la máquina en la
que se ejecuta. Lo que ve depende de la interfaz:

| Interfaz | Qué ve traffic66 |
|---|---|
| Un puerto de red libre conectado al puerto espejo (SPAN) de un switch | Todo el tráfico que replica el switch: una red entera o un enlace de subida |
| El Ethernet o la Wi-Fi de la propia máquina | Solo el tráfico de esta máquina |

Los adaptadores Wi-Fi no ven el tráfico de otros equipos. Para ver toda una
red Wi-Fi, haga que el router o el punto de acceso exporte flujos (sección 4),
o ponga en espejo el puerto del switch al que está conectado el punto de
acceso.

<a id="windows-1"></a>

### Windows

1. Instale [Npcap](https://npcap.com) con las opciones predeterminadas. Si
   marca "Restrict Npcap driver's access to Administrators only", ejecute
   traffic66 como administrador.
2. Liste las interfaces (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   La columna Name es el nombre de la conexión en la configuración de red de
   Windows; la interfaz en uso tiene una dirección.
3. Capture en la Wi-Fi, por nombre o por número:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Ponga entre comillas los nombres con espacios: `-capture "Ethernet 2"`.
   Repita `-capture` para capturar en varias interfaces. Añada `-listen=` si
   solo quiere captura y ningún colector de flujos. Para la tarea de inicio de
   la sección 2, añada la opción a `-Argument`:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

La captura requiere root o las capacidades `CAP_NET_RAW` y `CAP_NET_ADMIN`:
la línea `setcap` de arriba, o la línea `AmbientCapabilities` de la unidad
systemd de la sección 2. Las interfaces Wi-Fi suelen llamarse `wlan0` o
`wlp…`.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

La captura requiere root; no hay nada que instalar. En los MacBook, `en0` es
la Wi-Fi.

<a id="checking-that-it-works"></a>

### Comprobar que funciona

**Configuración** muestra cada interfaz capturada con el método de captura y el
número de paquetes vistos. Los flujos aparecen como procedentes del equipo
`127.0.0.1` (esta máquina), en todas las páginas, igual que los de cualquier
otro equipo. Los paquetes vistos dos veces (por ejemplo, en dos puertos
espejo) se cuentan dos veces.

<a id="12-options"></a>

## 12. Opciones

`traffic66 -h` y `traffic66 <command> -h` lo listan todo.

Comandos:

| Comando | |
|---|---|
| `traffic66` | recoge flujos y sirve la interfaz web |
| `traffic66 demo` | lo mismo, con una red simulada |
| `traffic66 tui` | interfaz de terminal para un traffic66 en marcha |
| `traffic66 passwd` | añade, cambia, lista o elimina usuarios (vea [Usuarios y contraseñas](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | envía exportaciones simuladas a un colector |
| `traffic66 interfaces` | lista las interfaces para captura local |
| `traffic66 version` | muestra la versión |

Opciones de `traffic66` y `traffic66 demo`:

| Opción | Por defecto | |
|---|---|---|
| `-addr` | `:8066` | dirección de la interfaz web; `127.0.0.1:8066` solo para esta máquina |
| `-data` | `traffic66-data` junto al programa | directorio de datos |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | colectores UDP como `name=address`, separados por comas; vacío los desactiva |
| `-user` | `admin` | nombre del usuario creado en el primer arranque, y del usuario al que se aplica `-password` |
| `-password` | sin definir | en esta ejecución acepta solo `-user` con esta contraseña, sin usar el archivo `password` (también `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | días de detalle de flujos que se conservan; los resúmenes se guardan 400 días |
| `-memory` | `0.10` | fracción de la memoria física para la caché de la base de datos, y otro tanto como límite blando para el resto del programa (cada uno al menos 256 MB) |
| `-l2-overhead` | `18` | bytes por paquete sumados a los bytes de NetFlow/IPFIX |
| `-sampling-wait` | `5m` | cuánto esperan los registros a una tasa de muestreo |
| `-capture` | | captura en una interfaz local (repetible) |
| `-inventory` | `<data>/inventory.txt` | archivo de nombres |
| `-asn` | `<data>/asn.tsv.gz` | tabla IP-a-ASN (archivos `.mmdb`: súbalos, o `<data>/country.mmdb` y `<data>/asn.mmdb`, `<data>/both.mmdb`) |
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
| `inventory.txt` | nombres (**Configuración → Nombres**) |
| `license.json` | número de instalación y licencia (vea [Prueba y licencia](#trial-and-licence)) |
| `logo.png` (o `.svg`, `.jpg`, `.webp`, `.gif`) | su logotipo (**Configuración → Logotipo**), si subió uno |
| `country.mmdb`, `asn.mmdb`, `both.mmdb`, `asn.tsv.gz`, `dbip-country.mmdb`, `dbip-asn.mmdb`, `threats/`, `sandbox/` | bases de datos de países y redes y listas de amenazas que haya añadido |

**Cuánto tiempo se guardan los datos**: el detalle de flujos 30 días; los resúmenes (vista general y periodos largos)
400 días. Lo más antiguo se borra automáticamente, con una comprobación cada 5 minutos; no se borra nada más ni hay
otro límite. Cambie el periodo del detalle con `-retention-days`, cualquier número de días, por ejemplo
`-retention-days 365`. El uso de disco crece con él: **Libre** en el menú lateral se pone en rojo cuando los días
guardados no caben. Si el disco se llena, los nuevos flujos no se pueden guardar hasta liberar espacio.

**Limpieza de datos** en el menú lateral borra datos antes de que haga
falta: los de más de 120, 90, 60, 30 o 7 días, o todos. Para cada opción
indica cuántos registros de flujo se van y cuánto disco libera
aproximadamente, y pide confirmación antes de borrar. Se borran los
registros de flujo, los resúmenes por hora y por día, los contadores de
interfaz y los hallazgos; borrar todos los datos también reinicia lo que
han aprendido las reglas de detección. No se puede deshacer.

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

<a id="trial-and-licence"></a>

### Prueba y licencia

traffic66 es de código fuente disponible bajo la
[PolyForm Noncommercial License 1.0.0](../LICENSE.md) y la
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md); el texto en
inglés de ambas es el vinculante. En resumen:

- **Evaluación**, pruebas, desarrollo y demostraciones: gratis para
  cualquiera, sin límite de tiempo.
- **Uso en producción** (tráfico real, para las operaciones de una
  organización) por una organización con menos de 100 empleados y
  contratistas: gratis.
- Uso en producción por **organizaciones más grandes**: gratis durante 30
  días; después se necesita una licencia de registro del autor.
- Contratistas y proveedores de servicios pueden ejecutarlo para un
  cliente, en un despliegue propio de ese cliente; decide el tamaño del
  cliente.
- No se permite sin una licencia comercial: venderlo o integrarlo en un
  producto, ofrecerlo a terceros como servicio alojado o multiinquilino, o
  un producto competidor.

Las tarifas, el alcance y la duración de una licencia de registro se
fijan caso por caso, y puede ser gratuita. Contacto:
<https://github.com/githubflyideas/traffic66>.

Toda instalación muestra la prueba, también donde no se necesita
licencia. En el primer arranque traffic66 escribe `license.json` en el
directorio de datos con un número de instalación de 8 cifras. El pie de cada página muestra cuántos días de prueba quedan y,
después, que la prueba ha terminado. En ningún caso se desactiva nada:
todas las funciones siguen funcionando.

Para registrarse, envíe al autor el número de instalación (que también
aparece al pie de cada página). La licencia llega como un nuevo
`license.json`; póngalo en el directorio de datos en lugar del anterior. Se
comprueba al arrancar traffic66 y cada 4 horas, así que no hace falta
reiniciar; el pie de la página muestra entonces a quién está licenciado y
cuántos días quedan.

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
curso, y el programa una sexta parte de un núcleo.
Las vistas generales de rangos largos salen de resúmenes y tardan menos de
0,2 s. Las consultas sobre el detalle recorren unos 22 millones de filas por
hora: un host durante 1 hora tarda menos de 1 s, un Top 66 de 1 hora de
todas las conversaciones unos 9 s; el tiempo crece con el rango y baja con
más núcleos.

Por tanto, 30 días a 5.000 flujos/s ocupan unos 360 GB de disco; escálelo
según su tasa de flujos (visible en **Configuración**) y `-retention-days`.

Memoria: `-memory` (por defecto el 10 % de la RAM, al menos 256 MB) limita
la caché de la base de datos, y el resto del programa recibe un límite
blando del mismo tamaño. A 5.000 flujos por segundo, los datos propios del
programa (decodificación, detección de duplicados, lotes) ocupan unos
90 MB; en total cuente con 0,6–0,8 GB, así que basta una máquina con 2 GB
de RAM. Medido durante 10 minutos de captura continua (pico de 0,58 GB en
una máquina de 8 GB) y al cargar una hora de flujos a once veces esa tasa
con los límites de una máquina de 2 GB (pico de 0,74 GB).

`-memory` es un presupuesto, no un tope estricto: el límite de Go es blando y
la base de datos puede superar brevemente su parte. Para un tope estricto use
el del sistema operativo: `MemoryMax=` en la unidad systemd (sección 2) o el
límite de memoria de un contenedor. Reserve unas 2,5 veces la parte de
`-memory` y al menos 1 GB; `MemoryMax=2G` sirve para máquinas de hasta 8 GB
con la parte por defecto. Así se reinicia traffic66 en lugar de quedarse la
máquina sin memoria.

<a id="16-troubleshooting"></a>

## 16. Resolución de problemas

| Síntoma | Causa y solución |
|---|---|
| El equipo no aparece en **Configuración** | Los paquetes no llegan: vea [Comprobar que llegan los flujos](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | El equipo aún no ha enviado sus opciones de sampler; la mayoría las reenvía en pocos minutos. Si no lo hace nunca, expórtelas (`option sampler-table` en Cisco) o márquelo como `unsampled` en Nombres si de verdad es 1:1, o indique su tasa con `sampling=N` en su línea `device`. **Configuración** muestra entonces las plantillas que envió el equipo, para ver qué declara |
| Cifras por debajo de los contadores de interfaz | Vea **Verificación de interfaces**: pérdidas por el camino, interfaces sin muestrear o flujos aún en la caché del equipo (timeout activo superior a 60 s) |
| Cifras por encima de los contadores de interfaz | El mismo tráfico se muestrea en dos interfaces o en dos equipos |
| No hay países ni redes ("Desconocido") | No hay ninguna base de datos cargada: suba una en **Configuración**; vea [Países](#8-countries-networks-and-threat-lists) |
| "La base de datos llegó a su límite de memoria y no pudo responder" en una página | Elija un intervalo más corto o inicie con un `-memory` mayor; los detalles están en el registro |
| Contraseña olvidada | `traffic66 passwd` en la máquina de traffic66 (añada `-data` si traffic66 se ejecuta con él) |
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
