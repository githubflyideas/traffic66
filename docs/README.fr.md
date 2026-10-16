[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | **Français** | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — collecteur et analyseur de trafic NetFlow, sFlow et IPFIX

Analyse de flux sFlow, NetFlow et IPFIX en un seul programme : qui consomme
la bande passante, où va le trafic et si les chiffres concordent avec les
compteurs d'interface des équipements eux-mêmes, dans une interface web et
dans une interface terminal.

## Démarrage rapide

1. Téléchargez l'archive de votre système : [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Décompressez-la et lancez le programme :

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows : décompressez `traffic66-windows-amd64.zip` et lancez `traffic66.exe` (un double-clic suffit).

macOS : lancez d'abord `xattr -dr com.apple.quarantine .` dans le dossier.

Ensuite :

- Interface web : `http://<IP du serveur>:8066`. Elle écoute sur toutes les interfaces réseau.
- Connexion : utilisateur `admin`, mot de passe `traffic66`. La première connexion impose un nouveau mot de passe.
- Faites envoyer vos équipements vers l'IP du serveur : sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (toutes les interfaces ; chaque port accepte tous les protocoles).

## Commandes courantes

- `./traffic66`  
  Démarrer avec les valeurs par défaut ci-dessus.

- `./traffic66 -addr :9000`  
  Interface web sur le port 9000 au lieu de 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Changer les ports des collecteurs (ici NetFlow sur 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Écouter sur une seule adresse IP au lieu de toutes les interfaces.

- `./traffic66 -data /var/lib/traffic66`  
  Garder les données dans un autre répertoire (par défaut : `traffic66-data` à côté du programme).

- `./traffic66 passwd`  
  Définir ou réinitialiser le mot de passe admin (ajoutez `-data DIR` si vous en utilisez un).

- `traffic66.exe interfaces`  
  Windows : lister les cartes réseau (nom et numéro).

- `traffic66.exe -capture Wi-Fi`  
  Windows : analyser le trafic de ce PC sur cette carte (nom ou numéro). Nécessite Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS : lister les cartes réseau.

- `sudo ./traffic66 -capture eth0`  
  Linux : analyser le trafic de cette machine sur eth0 (macOS : `-capture en0`).

- `./traffic66 tui`  
  Interface terminal sur la machine traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Interface terminal connectée à un traffic66 distant.

- `./traffic66 demo`  
  Essayer avec des équipements simulés.

## Captures d'écran

<img src="images/overview.png" alt="Vue d'ensemble" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Géographie et réseaux" width="49%"> <img src="images/interfaces.png" alt="Contrôle des interfaces" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Manuel complet

Dans l'interface web : **Aide** dans le menu. Sur GitHub : [MANUAL.fr.md](MANUAL.fr.md)

15 langues. Gratuit pour l'évaluation et pour les organisations de moins de 100 personnes ([licence](../LICENSE.md)).
