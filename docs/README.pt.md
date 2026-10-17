[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | **Português** | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [Deutsch](README.de.md) | [日本語](README.ja.md) | [Tiếng Việt](README.vi.md) | [한국어](README.ko.md)

# traffic66 — coletor e analisador de tráfego NetFlow, sFlow e IPFIX

Análise de fluxos sFlow, NetFlow e IPFIX em um único programa: quem usa a
banda, para onde vai o tráfego e se os números batem com os contadores de
interface dos próprios equipamentos, em uma interface web e em uma
interface de terminal.

## Início rápido

1. Baixe o arquivo do seu sistema: [Releases](https://github.com/githubflyideas/traffic66/releases/latest): `traffic66-linux-amd64.tar.gz`, `traffic66-linux-arm64.tar.gz`, `traffic66-darwin-arm64.tar.gz`, `traffic66-darwin-amd64.tar.gz`, `traffic66-windows-amd64.zip`.
2. Descompacte e execute o programa:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66
```

Windows: descompacte `traffic66-windows-amd64.zip` e execute `traffic66.exe` (clique duplo funciona).

macOS: antes, execute `xattr -dr com.apple.quarantine .` na pasta.

Depois:

- Interface web: `http://<IP do servidor>:8066`. Ela escuta em todas as interfaces de rede.
- Login: usuário `admin`, senha `traffic66`. O primeiro login obriga a definir uma senha nova.
- Aponte seus equipamentos para o IP do servidor: sFlow UDP 6343, NetFlow v5/v9 UDP 2055, IPFIX UDP 4739 (todas as interfaces; toda porta aceita todos os protocolos).

## Comandos comuns

- `./traffic66`  
  Iniciar com os padrões acima.

- `./traffic66 -addr :9000`  
  Interface web na porta 9000 em vez de 8066.

- `./traffic66 -listen sflow=:6343,netflow=:9995,ipfix=:4739`  
  Mudar as portas dos coletores (aqui NetFlow na 9995).

- `./traffic66 -addr 192.168.1.10:8066 -listen sflow=192.168.1.10:6343,netflow=192.168.1.10:2055,ipfix=192.168.1.10:4739`  
  Escutar só em um endereço IP em vez de todas as interfaces.

- `./traffic66 -data /var/lib/traffic66`  
  Guardar os dados em outro diretório (padrão: `traffic66-data` ao lado do programa).

- `./traffic66 passwd`  
  Definir ou redefinir a senha do admin (com `-data DIR` se usar um).

- `traffic66.exe interfaces`  
  Windows: listar as placas de rede (nome e número).

- `traffic66.exe -capture Wi-Fi`  
  Windows: analisar o tráfego deste PC nessa placa (nome ou número). Requer Npcap (npcap.com).

- `./traffic66 interfaces`  
  Linux, macOS: listar as placas de rede.

- `sudo ./traffic66 -capture eth0`  
  Linux: analisar o tráfego desta máquina na eth0 (macOS: `-capture en0`).

- `./traffic66 tui`  
  Interface de terminal na máquina do traffic66.

- `./traffic66 tui -server http://192.168.1.10:8066 -user admin -password 'your-password'`  
  Interface de terminal conectada a um traffic66 remoto.

- `./traffic66 demo`  
  Experimentar com equipamentos simulados.

## Capturas de tela

<img src="images/overview.png" alt="Visão geral" width="49%"> <img src="images/topn.png" alt="Top 66" width="49%">

<img src="images/geo.png" alt="Geografia e redes" width="49%"> <img src="images/interfaces.png" alt="Conferência de interfaces" width="49%">

<img src="images/tui-top.png" alt="TUI" width="49%">

## Manual completo

Na interface web: **Ajuda** no menu. No GitHub: [MANUAL.pt.md](MANUAL.pt.md)

15 idiomas. Gratuito para avaliação e para organizações com menos de 100 pessoas ([licença](../LICENSE.md)).
