[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | **Português** | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — coletor e analisador de tráfego NetFlow, sFlow e IPFIX

Análise de fluxos sFlow, NetFlow e IPFIX em um único programa: quem usa a
banda, para onde vai o tráfego e se os números batem com os contadores de
interface dos próprios equipamentos, em uma interface web e em uma
interface de terminal.

## Início rápido

```
# Linux, macOS: baixar em ./traffic66 e começar a coletar
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh
```

```
# Windows, PowerShell: o mesmo
irm https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.ps1 | iex
```

O terminal mostra o endereço da interface web e o IP e as portas UDP para onde apontar os equipamentos ([configuração dos equipamentos](MANUAL.pt.md#device-configuration)). Entre como `admin` / `traffic66` e escolha uma senha nova. Só para ver? A demo, com equipamentos simulados:

```
curl -fsSL https://raw.githubusercontent.com/githubflyideas/traffic66/main/scripts/install.sh | sh -s -- demo
```

Portas: TCP 8066 interface web, UDP 6343 sFlow, 2055 NetFlow, 4739 IPFIX (toda porta UDP aceita todos os protocolos). O terminal mostra os endereços para onde apontar os equipamentos; o primeiro login pede uma senha nova.

macOS manualmente: antes, `xattr -dr com.apple.quarantine traffic66-darwin-*`. Windows "protegeu seu PC": **Mais informações** → **Executar assim mesmo**.

## Resumo de comandos

| Tarefa | Comando |
|---|---|
| Coletar sFlow 6343, NetFlow v5/v9 2055, IPFIX 4739 (padrão) | `./traffic66` |
| Só NetFlow v5/v9 na 2055 e IPFIX na 4739 | `./traffic66 -listen netflow=:2055,ipfix=:4739` |
| Só sFlow, UDP 6343 | `./traffic66 -listen sflow=:6343` |
| Interface web em IP e porta definidos | `./traffic66 -addr 10.0.0.5:8080` |
| Capturar a placa de rede local (Linux) | `sudo ./traffic66 -capture eth0 -listen ""` |
| Definir ou redefinir a senha do admin (com -data DIR se usar) | `./traffic66 passwd` |
| Interface de terminal para um servidor remoto | `./traffic66 tui -server http://192.0.2.50:8066 -user admin -password 'S3cret-pass'` |

![Visão geral: detecções abertas, banda por aplicação em comparação com o mesmo horário de ontem, principais clientes e serviços](images/overview.png)

## Manual completo

na interface web: **Ajuda** no menu; no GitHub: [MANUAL.pt.md](MANUAL.pt.md)

15 idiomas. Gratuito para avaliação e para organizações com menos de 100 pessoas ([licença](../LICENSE.md)).
