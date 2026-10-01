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

<a id="contents"></a>

## Sommaire

1. [Essayer la démo](#1-try-the-demo)
2. [Installation](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Connexion et mots de passe](#3-sign-in-and-passwords)
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
**Vue d'ensemble**, regardez **Qui a augmenté**, puis cliquez de proche en
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

<a id="3-sign-in-and-passwords"></a>

## 3. Connexion et mots de passe

Ouvrez `http://<traffic66 machine>:8066` et connectez-vous. L'utilisateur
est `admin`, sauf si vous en avez choisi un autre.

- Si vous n'avez pas défini de mot de passe avant le premier démarrage,
  traffic66 en génère un et l'affiche une seule fois dans son journal :
  `first start: sign in as user "admin" with password "…"`.
  Sous Linux, retrouvez-le avec `journalctl -u traffic66 | grep "first start"`.
- Le mot de passe est conservé, haché, dans le fichier `password` du
  répertoire de données. Il reste le même d'un redémarrage à l'autre.
- Pour le changer, ou en définir un nouveau après l'avoir oublié, sur la
  machine traffic66 :

  ```
  traffic66 passwd -data <data directory>
  ```

  `traffic66 passwd -generate` en génère un aléatoire et l'affiche. Un
  traffic66 en cours d'exécution accepte le nouveau mot de passe à la
  connexion suivante ; inutile de redémarrer.
- Plus d'utilisateurs : `traffic66 passwd -data <data directory> -user alice`.
  Tous les utilisateurs voient la même chose.
- Pour les scripts et les conteneurs, `TRAFFIC66_PASSWORD=…` dans
  l'environnement ou `-password …` sur la ligne de commande définit le mot
  de passe pour cette exécution, à la place de celui enregistré. Préférez
  l'environnement : les lignes de commande sont visibles par les autres
  utilisateurs de la machine.

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

**Sources → Noms** dans l'interface web accepte une entrée par ligne. Le
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

Les pays et les noms de réseau (AS) nécessitent une table IP-vers-ASN.
Téléchargez la table gratuite de [iptoasn.com](https://iptoasn.com) :

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

Tout fichier au même format convient (séparé par des tabulations : première
adresse, dernière adresse, numéro d'AS, code pays, nom de l'AS ; brut ou
gzip). Redémarrez traffic66 après l'avoir remplacé ; téléchargez-en un
nouveau environ tous les mois.

Les listes de menaces sont des fichiers texte avec une adresse ou un réseau
par ligne (le texte après `#` ou `;` est ignoré), enregistrés sous
`<data directory>/threats/<name>.txt`, par exemple :

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Redémarrez traffic66 après avoir ajouté ou modifié des listes. Les
correspondances apparaissent dans **Menaces**, par nom de liste.

<a id="9-using-the-web-ui"></a>

## 9. Utiliser l'interface web

Vous aurez rarement besoin de taper quoi que ce soit. Chaque valeur de
chaque page (adresse, port, application, pays, équipement) est cliquable :

- **Afficher seulement ceci** / **Exclure ceci** ajoute un filtre. Les
  filtres s'affichent sous la barre du haut et s'appliquent à toutes les
  pages jusqu'à ce que vous les retiriez.
- **Voir ses enregistrements de flux** ouvre les flux individuels
  correspondants.
- **Rechercher en ligne** ouvre l'adresse ou l'AS sur un site public de
  recherche.
- **Copier** copie la valeur.

Pages :

| Page | À quoi elle répond |
|---|---|
| Vue d'ensemble | Combien de trafic maintenant et par rapport à la semaine dernière, par application ; ce qui a augmenté ; principaux clients et services |
| Top-N | Le top 66 des clients, serveurs, conversations, applications, ports, pays, réseaux, segments, équipements, encapsulations ou VLAN |
| Chemins du trafic | Quel segment parle à quelle application dans quel pays |
| Géographie et réseaux | Trafic par pays et par réseau (AS) |
| Menaces | Hôtes ayant communiqué avec des adresses de vos listes de menaces, et volume envoyé |
| Enregistrements de flux | Flux individuels, du plus récent au plus ancien, avec colonnes au choix |
| Contrôle des interfaces | Chiffres de flux à côté des compteurs d'interface, les pires en premier, avec les causes |
| Sources | Équipements, échantillonnage, pertes, collecteurs, SNMP et **Noms** |

Au-dessus des pages : la plage de temps (de 15 minutes à 30 jours), un
champ de recherche facultatif, le rafraîchissement automatique toutes les
30 secondes et **Copier le lien**, qui copie un lien vers la vue exacte
(page, plage de temps et filtres) à envoyer à un collègue. La langue suit
celle du navigateur ; on la change en bas du menu.

Sur de longues plages de temps, Top-N s'appuie sur des agrégats horaires ;
les filtres n'y sont pas disponibles, et la page le signale. Choisissez une
plage plus courte pour filtrer.

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

<a id="11-local-capture"></a>

## 11. Capture locale

En plus des exports de flux, traffic66 peut construire lui-même des flux à
partir des paquets d'une interface réseau locale, par exemple un port
miroir (SPAN) :

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux : nécessite root, ou les capabilities `CAP_NET_RAW` et
  `CAP_NET_ADMIN` (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`,
  ou la ligne `AmbientCapabilities` de l'unité systemd ci-dessus).
- macOS : nécessite root (périphériques BPF) ; rien à installer.
- Windows : installez d'abord [Npcap](https://npcap.com).

Les interfaces capturées sont listées dans **Sources**. Les paquets vus
deux fois (par exemple sur deux ports miroir) sont comptés deux fois.

<a id="12-options"></a>

## 12. Options

`traffic66 -h` et `traffic66 <command> -h` listent tout.

Commandes :

| Commande | |
|---|---|
| `traffic66` | collecte les flux et sert l'interface web |
| `traffic66 demo` | idem, avec un réseau simulé |
| `traffic66 tui` | interface terminal pour un traffic66 en cours d'exécution |
| `traffic66 passwd` | définit un mot de passe de connexion |
| `traffic66 simulate -to HOST` | envoie des exports simulés à un collecteur |
| `traffic66 interfaces` | liste les interfaces pour la capture locale |
| `traffic66 version` | affiche la version |

Options de `traffic66` et `traffic66 demo` :

| Option | Défaut | |
|---|---|---|
| `-addr` | `:8066` | adresse de l'interface web ; `127.0.0.1:8066` pour cette machine uniquement |
| `-data` | `traffic66-data` à côté du programme | répertoire de données |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collecteurs UDP sous la forme `name=address`, séparés par des virgules ; vide pour désactiver |
| `-user` | `admin` | utilisateur pour le premier mot de passe généré et pour `-password` |
| `-password` | mot de passe enregistré | mot de passe pour cette exécution uniquement (aussi `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | jours de détail des flux conservés ; les agrégats sont gardés 400 jours |
| `-memory` | `0.10` | part de la mémoire physique utilisable par la base de données |
| `-l2-overhead` | `18` | octets par paquet ajoutés aux octets NetFlow/IPFIX |
| `-sampling-wait` | `5m` | durée pendant laquelle les enregistrements attendent un taux d'échantillonnage |
| `-capture` | | capture sur une interface locale (répétable) |
| `-inventory` | `<data>/inventory.txt` | fichier de noms |
| `-asn` | `<data>/asn.tsv.gz` | table IP-vers-ASN |
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
| `asn.tsv.gz`, `threats/` | tables de correspondance que vous avez ajoutées |

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
cours ; le programme utilise environ 0,5 Go de mémoire et un sixième d'un
cœur. Les vues d'ensemble sur de longues plages proviennent des agrégats et
prennent moins de 0,2 s. Les requêtes sur le détail parcourent environ
22 millions de lignes par heure : un hôte sur 1 heure prend moins de 1 s,
un Top 66 sur 1 heure de toutes les conversations environ 9 s ; le temps
augmente avec la plage et diminue avec le nombre de cœurs.

Il faut donc environ 360 Go de disque pour 30 jours à 5 000 flux/s ;
ajustez selon votre débit de flux (affiché dans **Sources**) et
`-retention-days`.

<a id="16-troubleshooting"></a>

## 16. Dépannage

| Symptôme | Cause et solution |
|---|---|
| Équipement absent de **Sources** | Les paquets n'arrivent pas : voir [Vérifier que les flux arrivent](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | L'équipement n'a pas encore envoyé ses options de sampler ; la plupart les renvoient en quelques minutes. S'il ne le fait jamais, exportez-les (`option sampler-table` sur Cisco) ou marquez-le `unsampled` dans Noms s'il est réellement en 1:1 |
| Chiffres inférieurs aux compteurs d'interface | Voir **Contrôle des interfaces** : pertes en route, interfaces non échantillonnées, ou flux encore dans le cache de l'équipement (timeout actif supérieur à 60 s) |
| Chiffres supérieurs aux compteurs d'interface | Le même trafic est échantillonné sur deux interfaces ou deux équipements |
| Pas de pays ni de réseaux | Pas de table IP-vers-ASN : voir [Pays](#8-countries-networks-and-threat-lists) |
| Mot de passe oublié | `traffic66 passwd -data <data directory>` sur la machine traffic66 |
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
