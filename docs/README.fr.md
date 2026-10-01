[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | **Français** | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

Analyse de flux sFlow, NetFlow et IPFIX en un seul programme. Il collecte
les exports de flux des switchs, routeurs et pare-feu, les stocke dans une
base de données embarquée et montre qui consomme la bande passante, où va
le trafic et si les chiffres concordent avec les compteurs d'interface des
équipements eux-mêmes, dans une interface web et dans une interface
terminal.

- Un seul exécutable pour Windows, Linux et macOS. Pas de base de données à
  installer, pas de runtime, fonctionne hors ligne.
- sFlow v5, NetFlow v5, NetFlow v9 et IPFIX sur n'importe quel port UDP ;
  capture locale facultative sur une interface réseau ou un port miroir.
- Vérifie ses propres chiffres par rapport aux compteurs d'interface
  (compteurs sFlow ou SNMP) et explique pourquoi ils diffèrent le cas
  échéant.
- Classements Top 66, chemins du trafic, pays et réseaux, correspondances
  avec des listes de menaces, enregistrements de flux, encapsulation (GRE,
  IPIP, VXLAN, GENEVE, MPLS).
- 13 langues dans l'interface web et dans l'interface terminal.

![Vue d'ensemble : bande passante par application comparée à la semaine dernière, principaux clients et services](images/overview.png)

<sub>Toutes les captures d'écran proviennent de `traffic66 demo`, un réseau d'entreprise simulé que vous pouvez lancer vous-même (voir [Essayer la démo](#1-try-the-demo)).</sub>

<a id="contents"></a>

## Sommaire

1. [Essayer la démo](#1-try-the-demo)
2. [Installation](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Utilisateurs et mots de passe](#3-users-and-passwords)
4. [Envoyer les flux depuis vos équipements](#4-send-flows-from-your-devices)
5. [Vérifier que les flux arrivent](#5-check-that-flows-arrive)
6. [Faire concorder les chiffres avec les compteurs d'interface](#6-make-the-numbers-match-the-interface-counters)
7. [Noms, SNMP et vos propres réseaux](#7-names-snmp-and-your-own-networks)
8. [Pays, réseaux et listes de menaces](#8-countries-networks-and-threat-lists)
9. [Utiliser l'interface web](#9-using-the-web-ui)
10. [Interface terminal](#10-terminal-ui)
11. [Capture locale](#11-local-capture)
12. [Options](#12-options)
13. [Données, sauvegarde, mise à jour, désinstallation](#13-data-backup-upgrade-uninstall)
14. [Sécurité](#14-security)
15. [Dimensionnement](#15-sizing)
16. [Dépannage](#16-troubleshooting)
17. [Compiler depuis les sources](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. Essayer la démo

Téléchargez l'archive correspondant à votre système sur la
[page des versions](https://github.com/githubflyideas/traffic66/releases) :

| Système | Archive |
|---|---|
| Windows 10/11, Server 2016 ou plus récent (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64 : toute distribution avec un noyau 3.2 ou ultérieur, y compris CentOS 7 et Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64 : mêmes distributions | `traffic66-linux-arm64.tar.gz` |
| macOS 11 ou plus récent, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 ou plus récent, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux :

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (la deuxième ligne autorise macOS à lancer un programme téléchargé
sur internet qui ne vient pas de l'App Store) :

```
tar xzf traffic66-darwin-arm64.tar.gz
xattr -dr com.apple.quarantine traffic66-darwin-arm64
cd traffic66-darwin-arm64
./traffic66 demo -password try66
```

Windows (PowerShell) :

```
Expand-Archive traffic66-windows-amd64.zip .
cd traffic66-windows-amd64
.\traffic66.exe demo -password try66
```

Ouvrez http://127.0.0.1:8066 et connectez-vous avec `admin` / `try66`. La
démo construit le réseau d'une petite entreprise avec une journée
d'historique et du trafic en direct provenant de quatre équipements
simulés, dont deux incidents à trouver : commencez par
**Vue d'ensemble**, cliquez sur un hôte dans **Principaux clients**,
choisissez **Voir les détails** et continuez à cliquer de proche en
proche. Arrêtez-la avec Ctrl+C. Les données de la démo sont conservées dans
`traffic66-demo` à côté du programme ; supprimez ce dossier pour repartir
de zéro.

La démo utilise les mêmes ports qu'une vraie installation (8066, et UDP
6343, 2055, 4739). Pour la lancer à côté d'une installation réelle,
donnez-lui d'autres ports :
`traffic66 demo -password try66 -addr :8067 -listen ""`.

Sous Windows, vous pouvez aussi simplement double-cliquer sur
`traffic66.exe`. traffic66 démarre alors pour de bon (pas la démo) et ouvre
l'interface web dans votre navigateur ; le mot de passe du premier démarrage
s'affiche dans la fenêtre noire, et fermer la fenêtre arrête traffic66. Si
Windows affiche "Windows a protégé votre ordinateur", cliquez sur
**Informations complémentaires** → **Exécuter quand même**.

<a id="2-install"></a>

## 2. Installation

traffic66 tient en un seul fichier. L'installer consiste à le déposer
quelque part, choisir un répertoire de données, définir un mot de passe,
ouvrir le pare-feu et le lancer au démarrage. Les exemples utilisent
`192.0.2.50` pour la machine traffic66 et `192.0.2.1` pour un routeur ;
remplacez-les par vos adresses.

Ports :

| Port | Usage |
|---|---|
| UDP 6343 | sFlow (par défaut) |
| UDP 2055 | NetFlow (par défaut) |
| UDP 4739 | IPFIX (par défaut) |
| TCP 8066 | interface web et API |

Chaque port UDP accepte tous les protocoles : un équipement peut donc
envoyer du NetFlow sur le 6343 si c'est plus simple. Modifiez ou ajoutez
des ports avec `-listen`.

<a id="linux"></a>

### Linux

```
sudo mkdir -p /opt/traffic66
sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

La dernière commande demande le mot de passe de l'utilisateur `admin`.

Créez `/etc/systemd/system/traffic66.service` :

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

Démarrez-le et autorisez des tampons UDP plus grands pour ne pas perdre les
rafales :

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Pare-feu, avec firewalld (RHEL, Rocky, Alma, Fedora) :

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

ou avec ufw (Ubuntu, Debian) :

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

Décompressez dans `C:\traffic66` et définissez le mot de passe (PowerShell
en tant qu'administrateur) :

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Les données vont dans `C:\traffic66\traffic66-data`, à côté du programme.

Ouvrez le pare-feu :

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

Pour l'essayer au premier plan, lancez `C:\traffic66\traffic66.exe` et
arrêtez-le avec Ctrl+C. Pour qu'il tourne en arrière-plan dès le
démarrage, sans session ouverte, enregistrez-le comme tâche de démarrage :

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` est important : sans lui, Windows
arrête la tâche au bout de trois jours. Arrêtez-la avec `Stop-ScheduledTask -TaskName
traffic66`, supprimez-la avec `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Créez `/Library/LaunchDaemons/traffic66.plist` :

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

Pour le démarrer, puis l'arrêter :

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

Si le pare-feu de macOS est actif, autorisez les connexions entrantes pour
traffic66 dans Réglages Système → Réseau → Coupe-feu → Options.

<a id="3-users-and-passwords"></a>

## 3. Utilisateurs et mots de passe

**En bref :** les utilisateurs et les mots de passe se trouvent dans un seul
fichier, `password`, dans le répertoire de données. Vous ne le modifiez
jamais à la main : la commande `traffic66 passwd` ajoute, modifie, liste et
supprime les utilisateurs. Ouvrez `http://<traffic66 machine>:8066` et
connectez-vous avec l'un d'eux.

<a id="the-first-sign-in"></a>

### La première connexion

Au premier démarrage, traffic66 crée l'utilisateur `admin` avec un mot de
passe aléatoire et l'affiche une seule fois :

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Lancé par double-clic sous Windows : dans la fenêtre noire.
- Dans un terminal : dans le terminal.
- Service Linux : `journalctl -u traffic66 | grep "first start"`
- Service macOS : `grep "first start" /Library/Logs/traffic66.log`

Vous l'avez manqué ? Définissez-en un nouveau avec `traffic66 passwd`
(ci-dessous). Si vous avez défini un mot de passe avec `traffic66 passwd`
avant le premier démarrage, comme le font les étapes d'installation
ci-dessus, aucun n'est généré.

<a id="where-the-users-are-stored"></a>

### Où sont enregistrés les utilisateurs

Dans le fichier `password` du répertoire de données :

| Mode d'exécution de traffic66 | Fichier |
|---|---|
| Décompressé et lancé depuis son dossier (par défaut) | `traffic66-data/password` à côté du programme |
| Service Linux (section 2) | `/var/lib/traffic66/password` |
| Tâche de démarrage Windows (section 2) | `C:\traffic66\traffic66-data\password` |
| Service macOS (section 2) | `/Library/Application Support/traffic66/password` |
| Démo | `traffic66-demo/password` à côté du programme |

Une ligne par utilisateur. Les mots de passe sont stockés sous forme de
hachages salés : personne ne peut les relire dans le fichier, pas même vous ;
si un mot de passe est oublié, définissez-en un nouveau. Seul le
propriétaire du fichier peut le lire.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### Gérer les utilisateurs

Exécutez ces commandes sur la machine traffic66 :

| Pour | Commande |
|---|---|
| Changer le mot de passe de `admin` | `traffic66 passwd` |
| Ajouter l'utilisateur `alice`, ou changer son mot de passe | `traffic66 passwd -user alice` |
| Supprimer l'utilisateur `alice` | `traffic66 passwd -user alice -delete` |
| Lister les utilisateurs | `traffic66 passwd -list` |
| Définir un mot de passe aléatoire et l'afficher | `traffic66 passwd -generate` (avec `-user` pour les autres utilisateurs) |

- La commande demande deux fois le nouveau mot de passe et n'affiche pas ce
  que vous tapez. Utilisez au moins 8 caractères.
- Si traffic66 tourne avec `-data`, ajoutez le même `-data` à la commande.
  Pour le service Linux de la section 2 :

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  Sous Windows (PowerShell en tant qu'administrateur) :

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- Les modifications s'appliquent immédiatement, sans redémarrage : un
  nouveau mot de passe fonctionne dès la connexion suivante, et un
  utilisateur supprimé est déconnecté des navigateurs ouverts.
- Le dernier utilisateur restant ne peut pas être supprimé ; ajoutez-en
  d'abord un autre.
- Tous les utilisateurs voient et peuvent modifier les mêmes choses ; il n'y
  a pas de rôles.

<a id="passwords-for-scripts-and-containers"></a>

### Mots de passe pour les scripts et les conteneurs

`TRAFFIC66_PASSWORD=…` dans l'environnement, ou `-password …` sur la ligne
de commande, fait accepter à traffic66 un seul utilisateur pour cette
exécution : celui désigné par `-user` (`admin` par défaut), avec ce mot de
passe. Le fichier `password` est alors ignoré et n'est pas modifié. Préférez
la variable d'environnement : les lignes de commande sont visibles par les
autres utilisateurs de la machine.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

Après cinq mots de passe erronés en une minute, l'adresse est bloquée
pendant une minute.

<a id="4-send-flows-from-your-devices"></a>

## 4. Envoyer les flux depuis vos équipements

Faites pointer chaque équipement vers la machine traffic66. Les commandes
varient selon les modèles et les versions logicielles ; reportez-vous au
manuel de votre équipement. Dans tous les exemples, `192.0.2.50` est
traffic66 et `192.0.2.1` l'adresse de l'équipement lui-même.

Conseils généraux :

- Réglez le timeout actif des flux à 60 secondes. Avec des timeouts plus
  longs, le trafic arrive en retard et par gros paquets.
- Si l'équipement échantillonne NetFlow/IPFIX, faites-lui exporter ses
  options de sampler pour que le taux soit connu. traffic66 retient les
  enregistrements jusqu'à réception du taux au lieu de les compter en 1:1.
- Échantillonnez soit toutes les interfaces, soit seulement les interfaces
  de bordure, dans un seul sens. Échantillonner le même trafic en entrée et
  en sortie le compte deux fois ; **Contrôle des interfaces** le signale.
- Taux d'échantillonnage sFlow : environ 1:1000 pour des liens à 1 Gb/s,
  1:4096 pour 10 Gb/s, 1:8192 pour 40/100 Gb/s.

Cisco IOS / IOS-XE (Flexible NetFlow) :

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

Cisco NX-OS (sFlow) :

```
feature sflow
sflow collector-ip 192.0.2.50 vrf default
sflow collector-port 6343
sflow agent-ip 192.0.2.1
sflow sampling-rate 4096
sflow counter-poll-interval 30
sflow data-source interface ethernet 1/1
```

Arista EOS (sFlow) :

```
sflow sample 4096
sflow destination 192.0.2.50
sflow source-interface Management1
sflow run
```

Juniper EX / QFX (sFlow) :

```
set protocols sflow collector 192.0.2.50 udp-port 6343
set protocols sflow sample-rate ingress 4096
set protocols sflow polling-interval 30
set protocols sflow interfaces ge-0/0/0
```

Huawei CloudEngine (sFlow) :

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

H3C Comware (sFlow) :

```
sflow agent ip 192.0.2.1
sflow collector 1 ip 192.0.2.50 port 6343
interface Ten-GigabitEthernet1/0/1
 sflow sampling-rate 4096
 sflow flow collector 1
 sflow counter interval 30
 sflow counter collector 1
```

MikroTik RouterOS 7 (NetFlow v9 / IPFIX) :

```
/ip traffic-flow set enabled=yes interfaces=all active-flow-timeout=1m
/ip traffic-flow target add dst-address=192.0.2.50 port=2055 version=9
```

FortiGate FortiOS 7.4.2 ou plus récent (NetFlow v9) :

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

Serveurs et hôtes Linux, avec softflowd (NetFlow v9) :

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Vérifier que les flux arrivent

Ouvrez **Sources**. Chaque équipement qui envoie quelque chose apparaît en
quelques secondes, avec son protocole, son taux d'échantillonnage, ses
pertes, son dernier paquet et un état. Quand l'état n'est pas vert, le
texte à côté indique ce qui ne va pas et ce qu'il faut changer.

![Sources : chaque équipement avec son protocole, son échantillonnage, ses pertes et ce qu'il faut corriger](images/sources.png)

Si un équipement n'apparaît pas :

1. Surveillez l'arrivée des paquets sur la machine traffic66 (Linux, macOS) :
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Si rien ne s'affiche, les paquets n'atteignent pas la machine : vérifiez
   la configuration de l'équipement, le routage et les pare-feu sur le
   chemin.
2. Les paquets arrivent mais **Sources** reste vide : le pare-feu local les
   rejette (voir [Installation](#2-install)), ou traffic66 écoute sur
   d'autres ports (`-listen`).
3. Pour tester le chemin depuis une autre machine sans toucher à un
   équipement, lancez-y `traffic66 simulate -to 192.0.2.50` pendant
   quelques secondes. La commande envoie du sFlow, du NetFlow et de l'IPFIX
   depuis des équipements simulés, qui apparaissent ensuite dans
   **Sources** et dans les données ; faites-le donc plutôt sur une
   installation de test.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Faire concorder les chiffres avec les compteurs d'interface

Les chiffres de flux sont des estimations : paquets échantillonnés
multipliés par le taux d'échantillonnage. traffic66 les compare aux
compteurs d'interface de l'équipement et affiche l'écart dans
**Contrôle des interfaces**, avec la cause probable quand il dépasse ce que
l'échantillonnage seul explique.

![Contrôle des interfaces : estimation des flux à côté du compteur de l'équipement pour chaque interface](images/interfaces.png)

Pour disposer de compteurs de comparaison :

- Les équipements sFlow les envoient d'eux-mêmes dès qu'un intervalle de
  compteurs est configuré (`sflow counter interval 30` ou équivalent).
- Pour les équipements NetFlow et IPFIX, ajoutez une ligne `snmp` dans
  **Sources → Noms** (voir [Noms](#7-names-snmp-and-your-own-networks)).
  traffic66 lit alors les compteurs d'interface toutes les minutes.

Ce que traffic66 fait déjà pour que les chiffres concordent : il utilise le
taux d'échantillonnage réellement appliqué par l'équipement, retient les
enregistrements NetFlow/IPFIX jusqu'à ce que ce taux soit connu, compense
les paquets d'export perdus en route, répartit les flux longs sur les
minutes qu'ils ont duré et ajoute 18 octets d'overhead Ethernet par paquet
aux octets NetFlow/IPFIX (les compteurs d'interface l'incluent, les
comptages de flux au niveau IP non ; réglable avec `-l2-overhead`).

Causes fréquentes d'un écart résiduel, toutes signalées dans
**Contrôle des interfaces** : certaines interfaces ne sont pas
échantillonnées, le même trafic est échantillonné sur deux interfaces, des
paquets d'export sont perdus avant d'atteindre traffic66, ou le taux
d'échantillonnage n'est pas encore connu.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. Noms, SNMP et vos propres réseaux

Le moyen le plus rapide de nommer un hôte ou un équipement : cliquez sur son
adresse sur n'importe quelle page et choisissez **Nommer…**. Tapez le nom
et appuyez sur Entrée ; il est enregistré aussitôt et affiché partout à la
place de l'adresse brute.

Pour les réseaux, les interfaces et SNMP, **Sources → Noms** dans
l'interface web accepte une entrée par ligne. Le
contenu est enregistré sous `inventory.txt` dans le répertoire de données ;
vous pouvez donc aussi éditer ce fichier (voir `inventory.txt.example`).
Toutes les lignes sont facultatives.

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

- `net` : les plages privées (10/8, 172.16/12, 192.168/16, 100.64/10) sont
  toujours considérées comme les vôtres. Ajoutez vos plages publiques pour
  que le trafic vers et depuis elles compte aussi comme le vôtre ; le nom
  apparaît dans **Top-N → Segments** et dans les chemins du trafic.
- `snmp <device> <community> [<management address>[:port]]` : l'équipement
  est l'adresse d'où proviennent les flux. Ajoutez l'adresse de gestion
  quand l'équipement répond en SNMP sur une autre adresse. Les descriptions
  d'interface lues en SNMP servent de noms, sauf si vous nommez l'interface
  avec `iface`. Autorisez la machine traffic66 dans la liste d'accès SNMP
  de l'équipement.
- Les modifications s'appliquent dès que vous cliquez sur **Enregistrer** ;
  inutile de redémarrer.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Pays, réseaux et listes de menaces

Les pays et les noms de réseau (AS) nécessitent une base de données qui
associe les adresses à ces informations. Importez-en une dans l'interface
web : **Sources → Base de données pays et réseaux → Importer un fichier de
base de données…**. Le fichier est vérifié, enregistré dans le répertoire
de données et utilisé aussitôt pour le nouveau trafic ; aucun redémarrage
n'est nécessaire. Le trafic déjà stocké garde le pays avec lequel il a été
enregistré.

Fichiers acceptés :

| Fichier | Fournit | Où le trouver |
|---|---|---|
| DB-IP Lite country ou ASN, `.mmdb` | pays, ou numéro et nom d'AS | gratuit, sans compte : [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country ou ASN, `.mmdb` | pays, ou numéro et nom d'AS | gratuit avec un compte MaxMind : [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| Table IP-vers-ASN, `.tsv` ou `.tsv.gz` | numéro d'AS, nom d'AS et pays | gratuit : [iptoasn.com](https://iptoasn.com) (`ip2asn-combined.tsv.gz`) |

Importez une base pays et une base ASN pour avoir les deux ; lorsque
plusieurs sont chargées, les fichiers `.mmdb` l'emportent pour ce qu'ils
contiennent. De nouvelles versions sortent chaque mois : importez le
nouveau fichier de la même manière pour remplacer l'ancien.

Sans l'interface web, copiez le fichier dans le répertoire de données sous
le nom `country.mmdb`, `asn.mmdb` ou `asn.tsv.gz` et redémarrez traffic66.

Les listes de menaces sont des fichiers texte avec une adresse ou un réseau
par ligne (le texte après `#` ou `;` est ignoré), enregistrés sous
`<data directory>/threats/<name>.txt`, par exemple :

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Redémarrez traffic66 après avoir ajouté ou modifié des listes. Les
correspondances apparaissent dans **Menaces**, par nom de liste.

![Menaces : un hôte interne qui envoie des données à une adresse figurant sur une liste de menaces](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Utiliser l'interface web

Vous aurez rarement besoin de taper quoi que ce soit. Chaque valeur de
chaque page (adresse, port, application, pays, équipement) est cliquable :

- **Afficher seulement ceci** / **Exclure ceci** ajoute un filtre. Les
  filtres s'affichent sous la barre du haut et s'appliquent à toutes les
  pages jusqu'à ce que vous les retiriez.
- **Voir ses enregistrements de flux** ouvre les flux individuels
  correspondants.
- **Voir les détails** (hôtes, équipements et services) ouvre une page
  consacrée à cet hôte ou à ce service : son trafic dans le temps par
  application, avec qui il communique, quels services ou clients, les pays
  et ses derniers flux. Chaque valeur y est à nouveau cliquable, ce qui
  permet de creuser toujours plus loin ; le bouton Retour du navigateur
  ramène en arrière.
- **Nommer…** (hôtes et équipements) donne un nom à l'adresse, affiché
  partout par la suite.
- **Rechercher en ligne** ouvre l'adresse ou l'AS sur un site public de
  recherche.
- **Copier** copie la valeur.

Pages :

| Page | À quoi elle répond |
|---|---|
| Vue d'ensemble | Combien de trafic maintenant et par rapport à la semaine dernière, par application ; principaux clients et services |
| Top-N | Un seul tableau du top 66 : par défaut les conversations (client, serveur, service, pays). Cliquez sur un en-tête de colonne bleu pour regrouper par cette colonne, sur un en-tête numérique pour trier ; **Regrouper par** propose applications, réseaux, segments, équipements, encapsulation et VLAN |
| Chemins du trafic | Quel segment parle à quelle application dans quel pays |
| Géographie et réseaux | Trafic par pays et par réseau (AS) |
| Menaces | Hôtes ayant communiqué avec des adresses de vos listes de menaces, et volume envoyé |
| Enregistrements de flux | Flux individuels, du plus récent au plus ancien, avec colonnes au choix |
| Contrôle des interfaces | Chiffres de flux à côté des compteurs d'interface, les pires en premier, avec les causes |
| Sources | Équipements, échantillonnage, pertes, collecteurs, SNMP, la base de données pays et réseaux, et **Noms** |

Au-dessus des pages : la plage de temps (de 15 minutes à 30 jours), un
champ de recherche facultatif, le rafraîchissement automatique toutes les
30 secondes et **Copier le lien**, qui copie un lien vers la vue exacte
(page, plage de temps et filtres) à envoyer à un collègue. La langue suit
celle du navigateur ; on la change en bas du menu.

Les plages de plus de 6 heures commencent à une heure pile, si bien que
chaque chiffre de la page porte exactement sur la même durée : "24 heures"
couvre les 24 dernières heures entières plus l'heure en cours. Sur ces
plages, Top-N s'appuie sur des agrégats horaires ; les filtres n'y sont pas
disponibles, et la page le signale. Choisissez une plage plus courte pour
filtrer. Les conversations lisent toujours le détail des flux : sur de longues plages
avec beaucoup de flux, cela peut prendre un moment ; une heure est le plus rapide.

Le menu latéral indique l'espace disque utilisé par les données et l'espace
libre ; survolez l'espace libre pour voir ce dont les jours de détail
conservés ont besoin au rythme actuel (estimation disponible dès qu'il y a
une journée de données).

![Top-N : le top 66 des conversations de la dernière heure](images/topn.png)

![Détails d'un hôte : son trafic, avec qui il communique, services, pays et derniers flux](images/detail.png)

![Chemins du trafic : quel segment utilise quelle application vers quel pays](images/paths.png)

La même vue d'ensemble en chinois ; toutes les pages sont disponibles en 13 langues :

![Vue d'ensemble en chinois](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. Interface terminal

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

Sur la machine traffic66, `traffic66 tui` se connecte tout seul s'il peut
lire le répertoire de données (indiquez `-data` s'il ne s'agit pas de
celui par défaut). Si traffic66 tourne sous un autre utilisateur, comme
c'est le cas d'un service, utilisez plutôt `-user` et `-password`. `-lang`
choisit la langue (`en`, `zh`, `hi`, `es`, `ar`, `fr`, `bn`, `pt`, `ru`,
`id`, `ur`, `ja`, `ko`).

Touches : 1–8 pages, ↑↓ sélection, Entrée actions sur la valeur
sélectionnée, f afficher seulement, x exclure, / rechercher, t plage de
temps, c effacer les filtres, w ouvrir la même vue dans un navigateur,
q quitter.

![Interface terminal : vue d'ensemble](images/tui-overview.png)

![Interface terminal : Top-N des conversations](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Capture locale

En plus de recevoir des exports de flux, traffic66 peut construire lui-même
des flux à partir des paquets d'une interface réseau de la machine sur
laquelle il tourne. Ce qu'il voit dépend de l'interface :

| Interface | Ce que voit traffic66 |
|---|---|
| Un port réseau libre relié au port miroir (SPAN) d'un switch | Tout le trafic que le switch recopie : un réseau entier ou un lien montant |
| L'Ethernet ou le Wi-Fi de la machine elle-même | Uniquement le trafic de cette machine |

Les cartes Wi-Fi ne voient pas le trafic des autres équipements. Pour voir
tout un réseau Wi-Fi, faites exporter les flux par le routeur ou le point
d'accès (section 4), ou mettez en miroir le port du switch auquel le point
d'accès est relié.

<a id="windows-1"></a>

### Windows

1. Installez [Npcap](https://npcap.com) avec ses options par défaut. Si vous
   cochez "Restrict Npcap driver's access to Administrators only", lancez
   traffic66 en tant qu'administrateur.
2. Listez les interfaces (PowerShell) :

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   La colonne Name est le nom de la connexion dans les paramètres réseau de
   Windows ; l'interface utilisée a une adresse.
3. Capturez sur le Wi-Fi, par nom ou par numéro :

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Mettez entre guillemets les noms qui contiennent des espaces :
   `-capture "Ethernet 2"`. Répétez `-capture` pour capturer sur plusieurs
   interfaces. Ajoutez `-listen=` si vous voulez seulement la capture, sans
   collecteurs de flux. Pour la tâche de démarrage de la section 2, ajoutez
   l'option à `-Argument` :
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

La capture nécessite root, ou les capabilities `CAP_NET_RAW` et
`CAP_NET_ADMIN` : la ligne `setcap` ci-dessus, ou la ligne
`AmbientCapabilities` de l'unité systemd de la section 2. Les interfaces
Wi-Fi s'appellent généralement `wlan0` ou `wlp…`.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

La capture nécessite root ; rien à installer. Sur les MacBook, `en0` est le
Wi-Fi.

<a id="checking-that-it-works"></a>

### Vérifier que ça fonctionne

**Sources** liste chaque interface capturée avec la méthode de capture et le
nombre de paquets vus. Les flux apparaissent comme provenant de l'équipement
`127.0.0.1` (cette machine), sur toutes les pages, comme ceux de n'importe
quel autre équipement. Les paquets vus deux fois (par exemple sur deux ports
miroir) sont comptés deux fois.

<a id="12-options"></a>

## 12. Options

`traffic66 -h` et `traffic66 <command> -h` listent tout.

Commandes :

| Commande | |
|---|---|
| `traffic66` | collecte les flux et sert l'interface web |
| `traffic66 demo` | idem, avec un réseau simulé |
| `traffic66 tui` | interface terminal pour un traffic66 en cours d'exécution |
| `traffic66 passwd` | ajoute, modifie, liste ou supprime des utilisateurs (voir [Utilisateurs et mots de passe](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | envoie des exports simulés à un collecteur |
| `traffic66 interfaces` | liste les interfaces pour la capture locale |
| `traffic66 version` | affiche la version |

Options de `traffic66` et `traffic66 demo` :

| Option | Défaut | |
|---|---|---|
| `-addr` | `:8066` | adresse de l'interface web ; `127.0.0.1:8066` pour cette machine uniquement |
| `-data` | `traffic66-data` à côté du programme | répertoire de données |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collecteurs UDP sous la forme `name=address`, séparés par des virgules ; vide pour désactiver |
| `-user` | `admin` | nom de l'utilisateur créé au premier démarrage, et de l'utilisateur auquel s'applique `-password` |
| `-password` | non défini | n'accepte que `-user` avec ce mot de passe pour cette exécution, en ignorant le fichier `password` (aussi `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | jours de détail des flux conservés ; les agrégats sont gardés 400 jours |
| `-memory` | `0.10` | part de la mémoire physique pour le cache de la base de données, et autant en limite souple pour le reste du programme (chacun au moins 256 Mo) |
| `-l2-overhead` | `18` | octets par paquet ajoutés aux octets NetFlow/IPFIX |
| `-sampling-wait` | `5m` | durée pendant laquelle les enregistrements attendent un taux d'échantillonnage |
| `-capture` | | capture sur une interface locale (répétable) |
| `-inventory` | `<data>/inventory.txt` | fichier de noms |
| `-asn` | `<data>/asn.tsv.gz` | table IP-vers-ASN (fichiers `.mmdb` : importez-les, ou `<data>/country.mmdb` et `<data>/asn.mmdb`) |
| `-threat` | `<data>/threats/*.txt` | liste de menaces supplémentaire sous la forme `name=path` (répétable) |
| `-dns-upstream` | résolveur du système | serveur DNS pour afficher les noms d'hôte |
| `-dns-rate` | `20` | nombre maximal de résolutions inverses par seconde |
| `-dns-cache` | `2m` | durée de mise en cache des noms d'hôte |
| `-no-dns` | | pas de résolution inverse |
| `-tui` | | ouvre aussi l'interface terminal |

Exemple : un second port de collecte, un an de détail et l'interface web
uniquement sur la machine locale :

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. Données, sauvegarde, mise à jour, désinstallation

Le répertoire de données contient tout :

| | |
|---|---|
| `raw/` | détail des flux, un fichier compressé par heure |
| `traffic66.duckdb` | agrégats, compteurs d'interface et heure en cours |
| `password` | mots de passe de connexion (hachés) |
| `inventory.txt` | noms (**Sources → Noms**) |
| `country.mmdb`, `asn.mmdb`, `asn.tsv.gz`, `threats/` | bases de données pays et réseaux et listes de menaces que vous avez ajoutées |

- **Sauvegarde** : arrêtez traffic66 et copiez le répertoire. Sans
  l'arrêter, copiez `raw/`, `password` et `inventory.txt` ; l'heure en
  cours et les agrégats manqueront alors.
- **Déplacement** : arrêtez traffic66, déplacez le répertoire, puis
  démarrez avec `-data` pointant vers le nouvel emplacement.
- **Mise à jour** : arrêtez traffic66, remplacez le fichier du programme,
  redémarrez-le. Les données sont conservées. Sous Linux, par exemple :

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Désinstallation** : arrêtez et supprimez le service ou la tâche de
  démarrage (voir [Installation](#2-install)), puis supprimez le dossier du
  programme et le répertoire de données.

<a id="14-security"></a>

## 14. Sécurité

- L'interface web utilise du HTTP simple : mots de passe et données
  circulent en clair sur le réseau. Sur les réseaux auxquels vous ne faites
  pas entièrement confiance, n'écoutez que sur cette machine
  (`-addr 127.0.0.1:8066`) et placez devant un reverse proxy TLS, par
  exemple avec [Caddy](https://caddyserver.com) :
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  Ou accédez-y par VPN ou tunnel SSH :
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50`, puis ouvrez
  http://127.0.0.1:8066.
- N'autorisez les ports UDP de collecte que depuis les adresses de vos
  équipements.
- Les communautés SNMP de `inventory.txt` sont stockées en clair ; utilisez
  une communauté en lecture seule.

<a id="15-sizing"></a>

## 15. Dimensionnement

Mesuré à 5 000 flux par seconde sur une machine à 2 cœurs : le détail
occupe environ 12 Go de disque par jour plus environ 1,5 Go pour l'heure en
cours ; le programme utilise un sixième d'un
cœur. Les vues d'ensemble sur de longues plages proviennent des agrégats et
prennent moins de 0,2 s. Les requêtes sur le détail parcourent environ
22 millions de lignes par heure : un hôte sur 1 heure prend moins de 1 s,
un Top 66 sur 1 heure de toutes les conversations environ 9 s ; le temps
augmente avec la plage et diminue avec le nombre de cœurs.

Il faut donc environ 360 Go de disque pour 30 jours à 5 000 flux/s ;
ajustez selon votre débit de flux (affiché dans **Sources**) et
`-retention-days`.

Mémoire : `-memory` (par défaut 10 % de la RAM, au moins 256 Mo) limite le
cache de la base de données, et le reste du programme reçoit une limite
souple de même taille. À 5 000 flux par seconde, les données propres du
programme (décodage, détection des doublons, lots) occupent environ 90 Mo ;
au total, comptez 0,6–0,8 Go, une machine avec 2 Go de RAM suffit donc.
Mesuré sur 10 minutes de collecte continue (pic de 0,58 Go sur une machine
de 8 Go) et pendant le chargement d'une heure de flux à onze fois ce débit
avec les limites d'une machine de 2 Go (pic de 0,74 Go).

`-memory` est un budget, pas un plafond strict : la limite de Go est souple
et la base de données peut dépasser brièvement sa part. Pour un plafond
strict, utilisez celui du système d'exploitation : `MemoryMax=` dans l'unité
systemd (section 2) ou la limite mémoire d'un conteneur. Prévoyez environ
2,5 fois la part `-memory` et au moins 1 Go ; `MemoryMax=2G` convient aux
machines jusqu'à 8 Go avec la part par défaut. traffic66 redémarre alors au
lieu que la machine manque de mémoire.

<a id="16-troubleshooting"></a>

## 16. Dépannage

| Symptôme | Cause et solution |
|---|---|
| Équipement absent de **Sources** | Les paquets n'arrivent pas : voir [Vérifier que les flux arrivent](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | L'équipement n'a pas encore envoyé ses options de sampler ; la plupart les renvoient en quelques minutes. S'il ne le fait jamais, exportez-les (`option sampler-table` sur Cisco) ou marquez-le `unsampled` dans Noms s'il est réellement en 1:1 |
| Chiffres inférieurs aux compteurs d'interface | Voir **Contrôle des interfaces** : pertes en route, interfaces non échantillonnées, ou flux encore dans le cache de l'équipement (timeout actif supérieur à 60 s) |
| Chiffres supérieurs aux compteurs d'interface | Le même trafic est échantillonné sur deux interfaces ou deux équipements |
| Pas de pays ni de réseaux ("Inconnu") | Aucune base de données chargée : importez-en une dans **Sources**, voir [Pays](#8-countries-networks-and-threat-lists) |
| "La base de données a atteint sa limite de mémoire et n'a pas pu répondre" sur une page | Choisissez une période plus courte, ou lancez avec un `-memory` plus grand ; les détails sont dans le journal |
| Mot de passe oublié | `traffic66 passwd` sur la machine traffic66 (ajoutez `-data` si traffic66 tourne avec) |
| `Conflicting lock is held` | Un autre traffic66 utilise déjà ce répertoire de données |
| `receive buffer is only … KB` | Linux limite les tampons UDP : définissez `net.core.rmem_max=16777216` (voir [Linux](#linux)) |
| `cannot create the data directory` | Le dossier du programme n'est pas accessible en écriture pour cet utilisateur : indiquez `-data` |
| macOS : "cannot be opened" ou "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows : "Windows a protégé votre ordinateur" | **Informations complémentaires** → **Exécuter quand même** ; le programme n'est pas encore signé |
| Capture sous Windows : Npcap introuvable | Installez [Npcap](https://npcap.com) |
| `address already in use` | Un autre programme utilise le port : choisissez-en d'autres avec `-addr` ou `-listen` |

<a id="17-build-from-source"></a>

## 17. Compiler depuis les sources

Go 1.24 et un compilateur C (gcc ou clang ; MinGW-w64 sous Windows) :

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
