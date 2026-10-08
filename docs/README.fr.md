[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | **Français** | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66 — collecteur et analyseur de trafic NetFlow, sFlow et IPFIX

Analyse de flux sFlow, NetFlow et IPFIX en un seul programme : qui consomme
la bande passante, où va le trafic et si les chiffres concordent avec les
compteurs d'interface des équipements eux-mêmes, dans une interface web et
dans une interface terminal.

Une alternative auto-hébergée à ntopng, ElastiFlow, pmacct avec Grafana ou
aux modules de flux de PRTG et SolarWinds NTA, pour la supervision réseau
(network monitoring), la surveillance de la bande passante
(bandwidth monitoring), les plus gros consommateurs (top talkers), la
détection de DDoS et de scans et l'analyse de pcap, sans Elasticsearch,
Kafka ni base de données séparée.

- Un seul exécutable pour Windows, Linux et macOS ; pas de base de données à installer, fonctionne hors ligne.
- sFlow v5, NetFlow v5/v9 et IPFIX sur n'importe quel port UDP, ou capture locale sur une interface.
- Vérifie ses chiffres par rapport aux compteurs d'interface (sFlow ou SNMP) et explique pourquoi ils diffèrent.
- Repère les scans, les essais de mots de passe, les mouvements latéraux, les envois inhabituels, les inondations et le trafic des listes de menaces, y compris à travers l'échantillonnage.
- `traffic66 capture.pcap` analyse des captures de paquets sans rien configurer.
- 13 langues. Gratuit pour l'évaluation et pour les organisations de moins de 100 personnes ([licence](#licence)).

![Vue d'ensemble : détections ouvertes, bande passante par application comparée à la même heure hier, principaux clients et services](images/overview.png)

<sub>Toutes les captures d'écran proviennent de `traffic66 demo`, un réseau d'entreprise simulé.</sub>

<a id="contents"></a>

## Sommaire

1. [Essayer la démo](#1-try-the-demo)
2. [Installation](#2-install)
3. [Utilisateurs et mots de passe](#3-users-and-passwords)
4. [Envoyer les flux depuis vos équipements](#4-send-flows-from-your-devices)
5. [Vérifier que les flux arrivent](#5-check-that-flows-arrive)
6. [Interfaces et compteurs](#6-interfaces-and-counters)
7. [Noms, pays et listes de menaces](#7-names-countries-and-threat-lists)
8. [Utiliser l'interface web](#8-using-the-web-ui)
9. [Analyse hors ligne de pcap, interface terminal, capture locale](#9-offline-pcap-terminal-ui-local-capture)
10. [Options et données](#10-options-and-data)
11. [Sécurité, dimensionnement, dépannage](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. Essayer la démo

Téléchargez l'archive de votre système depuis la
[page des versions](https://github.com/githubflyideas/traffic66/releases)
(Windows x64, Linux x86-64/ARM64 avec noyau 3.2+, macOS 11+), décompressez-la et lancez :

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

Sous macOS, lancez d'abord `xattr -dr com.apple.quarantine <folder>`. Ouvrez
http://127.0.0.1:8066 en tant que `admin` / `try66` : une journée
d'historique et du trafic en direct venant de quatre équipements simulés,
dont une attaque montrée étape par étape dans **Détections**. Ctrl+C
l'arrête ; supprimez `traffic66-demo` pour repartir de zéro. Pour la lancer
à côté d'une vraie installation : `-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. Installation

traffic66 tient en un seul fichier. Ports : UDP 6343 (sFlow), 2055
(NetFlow), 4739 (IPFIX), TCP 8066 (interface web) ; chaque port UDP accepte
tous les protocoles.

**Linux** (systemd) :

```
sudo mkdir -p /opt/traffic66 && sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
sudo useradd --system --no-create-home --shell /usr/sbin/nologin traffic66
sudo install -d -o traffic66 -g traffic66 /var/lib/traffic66
sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66
```

`/etc/systemd/system/traffic66.service` :

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

**Windows** (PowerShell en tant qu'administrateur) : décompressez dans
`C:\traffic66`, lancez `C:\traffic66\traffic66.exe passwd`, ouvrez les ports
et faites-le démarrer au boot :

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

Double-cliquer sur `traffic66.exe` fonctionne aussi : il ouvre l'interface
web et affiche le premier mot de passe dans sa fenêtre.

**macOS** : décompressez dans `/usr/local/traffic66`, retirez l'attribut de
quarantaine, lancez `traffic66 passwd -data "/Library/Application Support/traffic66"`
et démarrez-le depuis un LaunchDaemon dont les `ProgramArguments` sont le
programme, `-data` et ce répertoire, avec `RunAtLoad` et `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. Utilisateurs et mots de passe

Au premier démarrage, traffic66 crée l'utilisateur `admin` avec un mot de
passe aléatoire et l'affiche une seule fois (dans la fenêtre, le terminal
ou `journalctl -u traffic66 | grep "first start"`). Les utilisateurs sont
enregistrés sous forme de hachages salés dans `password` dans le répertoire
de données et se gèrent avec une seule commande sur la machine traffic66
(ajoutez `-data …` si traffic66 tourne avec cette option) :

| Pour | Commande |
|---|---|
| Changer le mot de passe de `admin` | `traffic66 passwd` |
| Ajouter l'utilisateur `alice`, ou changer son mot de passe | `traffic66 passwd -user alice` |
| Supprimer l'utilisateur `alice` | `traffic66 passwd -user alice -delete` |
| Lister les utilisateurs | `traffic66 passwd -list` |

Les changements s'appliquent immédiatement. Tous les utilisateurs ont les
mêmes droits. Pour les scripts et les conteneurs, `TRAFFIC66_PASSWORD=…` (ou
`-password`) n'accepte que `-user` avec ce mot de passe pour cette
exécution. Cinq mots de passe erronés en une minute bloquent l'adresse
pendant une minute.

<a id="4-send-flows-from-your-devices"></a>

## 4. Envoyer les flux depuis vos équipements

`192.0.2.50` est traffic66, `192.0.2.1` l'équipement. Réglez le timeout
actif à 60 secondes, laissez les équipements NetFlow/IPFIX exporter leurs
options d'échantillonneur et échantillonnez **chaque interface en entrée**
(ou seulement les interfaces de bordure) : chaque paquet compte alors une
seule fois. Taux sFlow : environ 1:1000 pour 1 Gb/s, 1:4096 pour 10 Gb/s,
1:8192 pour 40/100 Gb/s.

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

## 5. Vérifier que les flux arrivent

**Paramètres** liste en quelques secondes chaque équipement qui envoie quoi
que ce soit : protocole, taux d'échantillonnage, pertes, les interfaces
qu'il échantillonne et ce qu'il faut corriger quand l'état n'est pas vert.
Les pertes sFlow se répartissent entre les pertes en chemin (augmentez
`net.core.rmem_max` si `netstat -su` montre des erreurs de tampon) et les
échantillons que l'équipement a lui-même abandonnés.

![Paramètres : chaque équipement avec son protocole, son échantillonnage, ses pertes et ce qu'il faut corriger](images/sources.png)

Un équipement manque ? Lancez `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739` : rien ne s'affiche, c'est le routage, un pare-feu ou la
configuration de l'équipement ; des paquets arrivent mais rien dans
**Paramètres**, c'est le pare-feu local ou `-listen`.
`traffic66 simulate -to 192.0.2.50` depuis une autre machine teste le
chemin avec des équipements simulés.

<a id="6-interfaces-and-counters"></a>

## 6. Interfaces et compteurs

Les chiffres de flux sont des estimations (échantillons × taux
d'échantillonnage). **Contrôle des interfaces** les compare aux compteurs
d'interface de l'équipement (compteurs sFlow, ou SNMP via une ligne `snmp`
dans Noms) et explique pourquoi ils diffèrent : interfaces non
échantillonnées, le même trafic échantillonné deux fois, pertes en chemin
ou taux d'échantillonnage inconnu. Chaque interface a un graphique en bits/s
et un en paquets/s, entrée en vert et sortie en bleu, compteurs en
pointillés.

Sur chaque ligne, **✎** définit un nom et une courte étiquette (comme
*uplink*) et **☆** en fait l'interface par défaut (★), sur laquelle les
pages s'ouvrent.

Un équipement qui n'échantillonne que certaines interfaces montre aussi
l'autre extrémité de ces flux. Ces **interfaces d’en face** sont listées en
dernier en petits caractères gris : elles ne contiennent que le trafic
passant par l'interface échantillonnée. L'interface échantillonnée est
connue grâce à la source de données sFlow ou au champ flowDirection
(IPFIX 61) ; à défaut, c'est l'interface qui porte 90 % du trafic de
l'équipement.

![Contrôle des interfaces : trafic de chaque interface, et l'estimation des flux à côté du compteur de l'équipement](images/interfaces.png)

traffic66 utilise déjà le taux appliqué par l'équipement, attend les taux
inconnus, compense les pertes d'export, répartit les longs flux sur leurs
minutes et ajoute 18 octets par paquet de surcoût Ethernet à NetFlow/IPFIX
(`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Noms, pays et listes de menaces

Cliquez sur n'importe quelle adresse et choisissez **Nommer…**, ou utilisez
**Paramètres → Noms**. Les noms sont enregistrés dans `inventory.txt` dans
le répertoire de données :

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

Les plages privées sont toujours les vôtres. Les changements s'appliquent
avec **Enregistrer**, sans redémarrage.

Les pays et les réseaux (AS) fonctionnent d'emblée avec les bases gratuites
Lite de DB-IP ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)) ; **Paramètres** les
met à jour, ou accepte à leur place des fichiers MaxMind GeoLite2, IPinfo
Lite ou IPtoASN. Contours de la carte : [Natural Earth](https://www.naturalearthdata.com).

![Géographie et réseaux : trafic distant par pays sur une carte du monde](images/geo.png)

Les listes de menaces sont des fichiers texte avec une adresse ou un réseau
par ligne dans `<data>/threats/<name>.txt` (par exemple Spamhaus DROP) ;
redémarrez après les avoir modifiées. Les correspondances apparaissent dans
**Menaces**.

![Menaces : un hôte interne qui envoie des données à une adresse figurant sur une liste de menaces](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Utiliser l'interface web

Chaque valeur de chaque page est cliquable : **Afficher seulement ceci** /
**Exclure ceci** (les filtres s'appliquent à toutes les pages), **Voir ses
enregistrements de flux**, **Voir les détails** (une page consacrée à un
hôte ou à un service), **Nommer…**, **Rechercher en ligne**, **Copier**.

| Page | Ce qu'elle montre |
|---|---|
| Vue d'ensemble | La bande passante de l'interface choisie ; le trafic par application (total, entrant ou sortant) par rapport à hier ou à la semaine dernière ; les détections ouvertes ; principaux clients et services |
| Top 66 | Les 66 premières conversations, triables par n'importe quelle colonne, ou regroupées par application, réseau, segment, équipement, encapsulation, VLAN ; les 30 interlocuteurs principaux |
| Détails du trafic | Graphiques en anneaux : serveurs et leurs clients (ou l'inverse), et services |
| Chemins du trafic | Hôte → application → pays, ou client → service → serveur, ou par réseau |
| Contrôle des interfaces | Chaque interface dans le temps face à ses compteurs ; noms, étiquettes, interface par défaut |
| Enregistrements de flux | Les flux individuels, en direct toutes les 5 secondes ou sur n'importe quelle plage de temps |
| Détections, Menaces | Ce qui demande votre attention ([ci-dessous](#findings)) ; trafic avec des adresses listées |
| Géographie et réseaux | Une carte du monde par pays, les réseaux (AS) dans le temps |
| Paramètres | Équipements, échantillonnage, pertes, SNMP, bases de données, logo, noms |
| Analyse hors ligne de pcap, Nettoyage des données | Fichiers de capture ([ci-dessous](#9-offline-pcap-terminal-ui-local-capture)) ; suppression des anciennes données |

Au-dessus des pages : **Interface** (toutes, ou une interface
échantillonnée ; les pages de trafic ne montrent alors que le trafic qui la
traverse), la plage de temps (de 15 minutes à 30 jours, ou personnalisée),
le rafraîchissement toutes les 30 s et **Copier le lien** pour la vue
exacte. La langue et cinq thèmes de couleurs se trouvent en bas du menu.
Les graphiques s'arrêtent là où les données sont complètes : avec
NetFlow/IPFIX, aussi tard que les équipements exportent (2 minutes au
plus). Les plages de plus de 6 heures commencent à une heure pile ; une
interface sur 7 ou 30 jours lit le détail des flux, c'est donc plus lent et
cela ne remonte que jusqu'où le détail est conservé.

![Top 66 : les 66 premières conversations, triables par n'importe quelle colonne](images/topn.png)

![Détails du trafic : les serveurs avec leurs clients, et les services avec leurs serveurs, en graphiques en anneaux](images/traffic.png)

![Détails d'un hôte : les détections qui le concernent, son trafic, avec qui il communique, services, pays et derniers flux](images/detail.png)

![Chemins du trafic : quel hôte utilise quelle application vers quel pays](images/paths.png)

![Vue d'ensemble en chinois](images/overview-zh.png)

<a id="findings"></a>

### Détections

Vérifiées toutes les 5 minutes sur les 10 dernières ; ce qui dure une heure
forme une seule détection qui grandit.

| Détection | Ce que cela signifie |
|---|---|
| Scan, scan de ports | Petites sondes vers de nombreux hôtes sur un port, ou vers de nombreux ports d'un hôte |
| Essais de mots de passe | De nombreuses connexions courtes vers un service de connexion |
| Mouvement latéral | Partage de fichiers ou administration à distance vers des hôtes internes qui ne l'avaient jamais proposé |
| Envoi inhabituel | 100 Mo en 10 minutes vers une nouvelle adresse, trois fois ce qui est revenu |
| Inondation | 20 000+ petits paquets/s vers une adresse, dix fois son débit habituel |
| Liste de menaces | Trafic avec une adresse listée |

Depuis votre réseau, leur gravité est élevée ; depuis internet, faible.
**Traité** ferme une détection, **Fausse alerte** la fait taire pour de
bon. Les mouvements latéraux et les envois demandent une journée
d'historique. À travers un échantillonnage 1:4096, l'attaque de la démo est
entièrement détectée ; de très petits scans peuvent se cacher derrière
l'échantillonnage.

![Détections : chaque étape d'une attaque, trouvée à travers un échantillonnage sFlow 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Analyse hors ligne de pcap, interface terminal, capture locale

**Analyse hors ligne de pcap** montre des captures de paquets (pcap,
pcapng) avec les mêmes pages, à part des données en direct :
`traffic66 a.pcap b.pcapng` démarre sur 127.0.0.1 et ouvre le navigateur
(jusqu'à 3 fichiers, 3 Go ; Ctrl+C supprime les données importées), ou
importez sur cette page jusqu'à 3 fichiers de 50 Mo. Un seul fichier est
analysé à la fois, chacun dans sa propre base de données : **Analyser** sur
la ligne d'un fichier l'affiche sur toutes les pages, et la barre du haut
passe à un autre. L'analyse porte sur les flux, pas sur le contenu des
paquets.

![Analyse hors ligne : fichiers de capture avec leurs paquets, flux et période](images/sandbox.png)

**Interface terminal** : `traffic66 tui` sur la machine traffic66, ou
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Touches : 1–8 pages, Entrée actions, f afficher seulement, x exclure,
t plage de temps, w ouvrir dans un navigateur, q quitter ; `-lang` choisit
la langue.

![Interface terminal : vue d'ensemble](images/tui-overview.png)

![Interface terminal : Top 66 des conversations](images/tui-topn.png)

**Capture locale** construit des flux à partir d'une interface locale,
idéalement un port relié au port miroir d'un switch : `traffic66 interfaces`
les liste, `-capture eth1` capture. Sous Windows :

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

Linux demande root ou `setcap cap_net_raw,cap_net_admin+ep`, macOS root,
Windows [Npcap](https://npcap.com). Les flux capturés proviennent de
l'équipement `127.0.0.1`. La capture locale n'a ni interfaces ni compteurs
d'équipement, le **Contrôle des interfaces** n'a donc rien à comparer pour
elle.

<a id="10-options-and-data"></a>

## 10. Options et données

`traffic66 -h` liste tout. Les plus utilisées :

| Option | Défaut | |
|---|---|---|
| `-data` | `traffic66-data` à côté du programme | répertoire de données |
| `-addr` | `:8066` | interface web ; `127.0.0.1:8066` pour cette machine uniquement |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | collecteurs UDP ; vide pour désactiver |
| `-retention-days` | `30` | jours de détail des flux ; les agrégats sont gardés 400 jours |
| `-memory` | `0.10` | part de la RAM pour le cache de la base de données |
| `-sampling-wait` | `5m` | durée pendant laquelle les enregistrements attendent un taux d'échantillonnage |
| `-capture` | | interface locale (répétable) |
| `-no-dns` | | pas de résolution inverse |

Le répertoire de données contient `raw/` (détail, un fichier par heure),
`traffic66.duckdb` (agrégats et compteurs), `password`, `inventory.txt`,
`license.json`, votre logo et les bases de données. Pour sauvegarder,
arrêtez traffic66 et copiez-le ; pour mettre à jour, remplacez le fichier
du programme. **Nettoyage des données** supprime les données de plus de
7 à 120 jours, ou toutes.

<a id="licence"></a>

### Licence

Source disponible sous la [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
et le [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (le texte
anglais fait foi) : gratuit pour l'évaluation et pour les organisations de
moins de 100 personnes ; les organisations plus grandes s'enregistrent après
30 jours d'utilisation en production ; la vente, l'hébergement pour des
tiers ou les produits concurrents nécessitent une licence commerciale. Rien
n'est jamais désactivé. Le bas de chaque page affiche le numéro
d'installation à 8 chiffres ; envoyez-le à l'auteur et placez le
`license.json` reçu en retour dans le répertoire de données. Contact :
<https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Sécurité, dimensionnement, dépannage

L'interface web est en HTTP simple : sur des réseaux non fiables, utilisez
`-addr 127.0.0.1:8066` derrière un proxy TLS (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) ou un tunnel SSH. N'autorisez les ports UDP que depuis vos
équipements. Les communautés SNMP sont stockées en clair ; utilisez des
communautés en lecture seule.

À 5 000 flux/s sur 2 cœurs : environ 12 Go de disque par jour de détail
(360 Go pour 30 jours), un sixième de cœur, 0,6 à 0,8 Go de mémoire. Les
vues d'ensemble sur de longues plages prennent moins de 0,2 s ; un Top 66
d'une heure de toutes les conversations, environ 9 s.

| Symptôme | Solution |
|---|---|
| "en attente du taux d'échantillonnage" | Exportez les options de l'échantillonneur, ou `sampling=N` / `unsampled` sur la ligne de l'équipement |
| Inférieur aux compteurs | Interfaces non échantillonnées, pertes, ou timeout actif supérieur à 60 s |
| Supérieur aux compteurs | Le même trafic échantillonné sur deux interfaces ou équipements |
| Mot de passe oublié | `traffic66 passwd` sur la machine traffic66 |
| `Conflicting lock is held` | Un autre traffic66 utilise ce répertoire de données |
| `address already in use` | Choisissez d'autres ports avec `-addr` ou `-listen` |
| Windows : "Windows a protégé votre ordinateur" | **Informations complémentaires** → **Exécuter quand même** |

Compiler depuis les sources : Go 1.24 et un compilateur C, puis `scripts/build.sh 0.1.0 traffic66`.
