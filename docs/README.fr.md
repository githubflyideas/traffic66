[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | **Français** | [বাংলা](README.bn.md) | [Português](README.pt.md) | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — collecteur et analyseur de trafic NetFlow, sFlow et IPFIX

Analyse de flux sFlow, NetFlow et IPFIX en un seul programme : qui consomme
la bande passante, où va le trafic et si les chiffres concordent avec les
compteurs d'interface des équipements eux-mêmes, dans une interface web et
dans une interface terminal.

## Démarrage rapide

```
# Linux, macOS : télécharger dans ./traffic66 et commencer la collecte
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell : idem
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

Le terminal affiche l'adresse de l'interface web, ainsi que l'IP et les ports UDP à donner à vos équipements ([configuration des équipements](MANUAL.fr.md#device-configuration)). Connectez-vous en `admin` / `traffic66` et choisissez un nouveau mot de passe. Juste pour voir ? La démo, avec des équipements simulés :

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Ports : TCP 8066 interface web, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (chaque port UDP accepte tous les protocoles). Le terminal affiche les adresses à donner aux équipements ; la première connexion demande un nouveau mot de passe.

macOS à la main : d'abord `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows « a protégé votre ordinateur » : **Informations complémentaires** → **Exécuter quand même**.

## Aide-mémoire des commandes

| Tâche | Commande |
|---|---|
| Collecter sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (par défaut) | `./traffic66` |
| NetFlow v5/v9 sur 2055 et IPFIX sur 4739 seulement | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| sFlow seul, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Interface web sur une IP et un port | `./traffic66 -addr 10.0.0.5:8080` |
| Capturer la carte réseau locale (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Définir ou réinitialiser le mot de passe admin (ajouter -data DIR si utilisé) | `./traffic66 passwd` |
| Interface terminal vers un serveur distant | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Vue d'ensemble : détections ouvertes, bande passante par application comparée à la même heure hier, principaux clients et services](images/overview.png)

## Manuel complet

dans l'interface web : **Aide** dans le menu ; sur GitHub : [MANUAL.fr.md](MANUAL.fr.md)

15 langues. Gratuit pour l'évaluation et pour les organisations de moins de 100 personnes ([licence](../LICENSE.md)).
