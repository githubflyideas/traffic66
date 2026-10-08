[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | **Português** | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66 — coletor e analisador de tráfego NetFlow, sFlow e IPFIX

Análise de fluxos sFlow, NetFlow e IPFIX em um único programa: quem usa a
banda, para onde vai o tráfego e se os números batem com os contadores de
interface dos próprios equipamentos, em uma interface web e em uma
interface de terminal.

Uma alternativa auto-hospedada ao ntopng, ElastiFlow, pmacct com Grafana ou
aos módulos de fluxo do PRTG e do SolarWinds NTA, para monitoramento de rede
(network monitoring), monitoramento de banda (bandwidth monitoring), maiores
consumidores (top talkers), detecção de DDoS e varreduras e análise de pcap,
sem Elasticsearch, Kafka nem um banco de dados separado.

- Um único executável para Windows, Linux e macOS; sem banco de dados para instalar, funciona offline.
- sFlow v5, NetFlow v5/v9 e IPFIX em qualquer porta UDP, ou captura local a partir de uma interface.
- Confere os próprios números com os contadores de interface (sFlow ou SNMP) e explica por que diferem.
- Encontra varreduras, tentativas de senhas, movimento lateral, envios incomuns, inundações e tráfego de listas de ameaças, inclusive através da amostragem.
- `traffic66 capture.pcap` analisa capturas de pacotes sem configurar nada.
- 13 idiomas. Gratuito para avaliação e para organizações com menos de 100 pessoas ([licença](#licence)).

![Visão geral: detecções abertas, banda por aplicação em comparação com o mesmo horário de ontem, principais clientes e serviços](images/overview.png)

<sub>Todas as capturas de tela vêm do `traffic66 demo`, uma rede corporativa simulada.</sub>

<a id="contents"></a>

## Conteúdo

1. [Testar a demo](#1-try-the-demo)
2. [Instalação](#2-install)
3. [Usuários e senhas](#3-users-and-passwords)
4. [Enviar fluxos dos seus equipamentos](#4-send-flows-from-your-devices)
5. [Verificar se os fluxos estão chegando](#5-check-that-flows-arrive)
6. [Interfaces e contadores](#6-interfaces-and-counters)
7. [Nomes, países e listas de ameaças](#7-names-countries-and-threat-lists)
8. [Usando a interface web](#8-using-the-web-ui)
9. [Pcap offline, interface de terminal, captura local](#9-offline-pcap-terminal-ui-local-capture)
10. [Opções e dados](#10-options-and-data)
11. [Segurança, dimensionamento, solução de problemas](#11-security-sizing-troubleshooting)

<a id="1-try-the-demo"></a>

## 1. Testar a demo

Baixe o arquivo do seu sistema na
[página de releases](https://github.com/githubflyideas/traffic66/releases)
(Windows x64, Linux x86-64/ARM64 com kernel 3.2+, macOS 11+), descompacte e execute:

```
./traffic66 demo -password try66          # Linux, macOS
.\traffic66.exe demo -password try66      # Windows
```

No macOS, execute antes `xattr -dr com.apple.quarantine <folder>`. Abra
http://127.0.0.1:8066 como `admin` / `try66`: um dia de histórico e tráfego
ao vivo de quatro equipamentos simulados, incluindo um ataque mostrado etapa
por etapa em **Detecções**. Ctrl+C para; apague `traffic66-demo` para
recomeçar do zero. Para rodá-la ao lado de uma instalação real:
`-addr :8067 -listen ""`.

<a id="2-install"></a>

## 2. Instalação

O traffic66 é um único arquivo. Portas: UDP 6343 (sFlow), 2055 (NetFlow),
4739 (IPFIX), TCP 8066 (interface web); toda porta UDP aceita todos os
protocolos.

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

**Windows** (PowerShell como administrador): descompacte em `C:\traffic66`,
execute `C:\traffic66\traffic66.exe passwd`, libere as portas e configure-o
para iniciar no boot:

```
New-NetFirewallRule -DisplayName traffic66 -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName traffic66-web -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
$a = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$s = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName traffic66 -Action $a -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $s -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

Um clique duplo em `traffic66.exe` também funciona: ele abre a interface
web e mostra a primeira senha na janela.

**macOS**: descompacte em `/usr/local/traffic66`, remova a marca de
quarentena, execute `traffic66 passwd -data "/Library/Application Support/traffic66"`
e inicie-o a partir de um LaunchDaemon cujos `ProgramArguments` sejam o
programa, `-data` e esse diretório, com `RunAtLoad` e `KeepAlive`.

<a id="3-users-and-passwords"></a>

## 3. Usuários e senhas

Na primeira execução, o traffic66 cria o usuário `admin` com uma senha
aleatória e a mostra uma única vez (na janela, no terminal ou em
`journalctl -u traffic66 | grep "first start"`). Os usuários ficam guardados
como hashes com salt em `password`, no diretório de dados, e são gerenciados
com um único comando na máquina do traffic66 (adicione `-data …` quando o
traffic66 roda com ele):

| Para | Comando |
|---|---|
| Trocar a senha de `admin` | `traffic66 passwd` |
| Adicionar `alice` ou trocar a senha dela | `traffic66 passwd -user alice` |
| Remover `alice` | `traffic66 passwd -user alice -delete` |
| Listar os usuários | `traffic66 passwd -list` |

As mudanças valem na hora. Todos os usuários têm os mesmos direitos. Para
scripts e contêineres, `TRAFFIC66_PASSWORD=…` (ou `-password`) aceita só
`-user` com essa senha naquela execução. Cinco senhas erradas em um minuto
bloqueiam o endereço por um minuto.

<a id="4-send-flows-from-your-devices"></a>

## 4. Enviar fluxos dos seus equipamentos

`192.0.2.50` é o traffic66, `192.0.2.1` o equipamento. Configure o timeout
ativo em 60 segundos, faça os equipamentos NetFlow/IPFIX exportarem as
opções do sampler e amostre **todas as interfaces na entrada** (ou só as de
borda): assim cada pacote conta uma vez. Taxas sFlow: cerca de 1:1000 para
1 Gb/s, 1:4096 para 10 Gb/s, 1:8192 para 40/100 Gb/s.

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

## 5. Verificar se os fluxos estão chegando

**Configurações** lista em segundos cada equipamento que envia algo:
protocolo, taxa de amostragem, perdas, as interfaces que ele amostra e o que
corrigir quando o status não está verde. As perdas sFlow são separadas em
perdas no caminho (aumente `net.core.rmem_max` se `netstat -su` mostrar
erros de buffer) e amostras que o próprio equipamento descartou.

![Configurações: cada equipamento com protocolo, amostragem, perdas e o que corrigir](images/sources.png)

Falta um equipamento? Execute `sudo tcpdump -ni any udp port 6343 or udp port 2055
or udp port 4739`: se nada aparecer, é o roteamento, um firewall ou a
configuração do equipamento; se chegam pacotes mas nada aparece em
**Configurações**, é o firewall local ou `-listen`. `traffic66 simulate -to 192.0.2.50`
em outra máquina testa o caminho com equipamentos simulados.

<a id="6-interfaces-and-counters"></a>

## 6. Interfaces e contadores

Os números de fluxo são estimativas (amostras × taxa de amostragem).
**Conferência de interfaces** os compara com os contadores de interface do
equipamento (contadores sFlow, ou SNMP por uma linha `snmp` em Nomes) e diz
por que diferem: interfaces não amostradas, o mesmo tráfego amostrado duas
vezes, perdas no caminho ou uma taxa de amostragem desconhecida. Cada
interface tem um gráfico em bits/s e outro em pacotes/s, entrada em verde e
saída em azul, contadores tracejados.

Em cada linha, **✎** define um nome e um rótulo curto (como *uplink*) e
**☆** a torna a interface padrão (★), na qual as páginas abrem.

Um equipamento que amostra só algumas interfaces mostra também as outras
pontas desses fluxos. Essas **interfaces do outro lado** aparecem por último,
em letra cinza menor: contêm só o tráfego que passou pela interface
amostrada. A interface amostrada é conhecida pela fonte de dados do sFlow ou
pelo campo flowDirection (IPFIX 61); sem ele, é uma interface presente em
90% do tráfego de um equipamento.

![Conferência de interfaces: tráfego de cada interface, e a estimativa de fluxo ao lado do contador do equipamento](images/interfaces.png)

O traffic66 já usa a taxa que o equipamento aplicou, espera por taxas
desconhecidas, compensa perdas de exportação, distribui fluxos longos pelos
seus minutos e soma 18 bytes de overhead Ethernet por pacote ao NetFlow/IPFIX
(`-l2-overhead`).

<a id="7-names-countries-and-threat-lists"></a>

## 7. Nomes, países e listas de ameaças

Clique em qualquer endereço e escolha **Dar um nome…**, ou use
**Configurações → Nomes**. Os nomes ficam em `inventory.txt` no diretório
de dados:

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

As faixas privadas são sempre suas. As alterações valem ao clicar em
**Salvar**, sem reiniciar.

Países e redes (AS) funcionam desde o início com as bases gratuitas Lite da
DB-IP ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), "IP
Geolocation by DB-IP", [db-ip.com](https://db-ip.com)); **Configurações** as
atualiza, ou aceita no lugar delas arquivos MaxMind GeoLite2, IPinfo Lite ou
IPtoASN. Contornos do mapa: [Natural Earth](https://www.naturalearthdata.com).

![Geografia e redes: tráfego remoto por país num mapa-múndi](images/geo.png)

Listas de ameaças são arquivos de texto com um endereço ou rede por linha em
`<data>/threats/<name>.txt` (por exemplo, Spamhaus DROP); reinicie depois de
alterá-las. As ocorrências aparecem em **Ameaças**.

![Ameaças: um host interno enviando dados para um endereço de uma lista de ameaças](images/threats.png)

<a id="8-using-the-web-ui"></a>

## 8. Usando a interface web

Todo valor de toda página é clicável: **Mostrar só isto** / **Excluir isto**
(os filtros valem para todas as páginas), **Ver os registros de fluxo**,
**Ver detalhes** (uma página sobre um host ou serviço), **Dar um nome…**,
**Consultar on-line**, **Copiar**.

| Página | O que mostra |
|---|---|
| Visão geral | A banda da interface escolhida; tráfego por aplicação (total, entrada ou saída) em comparação com ontem ou a semana passada; detecções abertas; principais clientes e serviços |
| Top 66 | As 66 maiores conversas, ordenáveis por qualquer coluna, ou agrupadas por aplicação, rede, segmento, equipamento, encapsulamento, VLAN; os 30 maiores interlocutores |
| Detalhes do tráfego | Gráficos de anéis: servidores e seus clientes (ou o inverso), e serviços |
| Caminhos do tráfego | Host → aplicação → país, ou cliente → serviço → servidor, ou por rede |
| Conferência de interfaces | Cada interface ao longo do tempo em comparação com seus contadores; nomes, rótulos, a padrão |
| Registros de fluxo | Os fluxos individuais, ao vivo a cada 5 segundos ou para qualquer intervalo de tempo |
| Detecções, Ameaças | O que precisa de atenção ([abaixo](#findings)); tráfego com endereços listados |
| Geografia e redes | Um mapa-múndi por país, redes (AS) ao longo do tempo |
| Configurações | Equipamentos, amostragem, perdas, SNMP, bancos de dados, logotipo, nomes |
| Análise offline de pcap, Limpeza de dados | Arquivos de captura ([abaixo](#9-offline-pcap-terminal-ui-local-capture)); apagar dados antigos |

Acima das páginas: **Interface** (todas, ou uma interface amostrada; as
páginas de tráfego então mostram só o tráfego que passa por ela), o
intervalo de tempo (de 15 minutos a 30 dias, ou personalizado), atualização
a cada 30 s e **Copiar link** para exatamente a visão atual. O idioma e
cinco temas de cores ficam no fim do menu. Os gráficos terminam onde os
dados estão completos: com NetFlow/IPFIX, tão tarde quanto os equipamentos
exportam (no máximo 2 minutos). Intervalos maiores que 6 horas começam em
uma hora cheia; uma interface em 7 ou 30 dias lê o detalhe dos fluxos, então
demora mais e alcança até onde o detalhe é mantido.

![Top 66: as 66 maiores conversas, ordenáveis por qualquer coluna](images/topn.png)

![Detalhes do tráfego: servidores com seus clientes, e serviços com seus servidores, em gráficos de anéis](images/traffic.png)

![Detalhes de um host: as detecções sobre ele, o tráfego, com quem fala, serviços, países e fluxos mais recentes](images/detail.png)

![Caminhos do tráfego: qual host usa qual aplicação para qual país](images/paths.png)

![Visão geral em chinês](images/overview-zh.png)

<a id="findings"></a>

### Detecções

Verificadas a cada 5 minutos sobre os últimos 10; algo que dura uma hora é
uma única detecção que cresce.

| Detecção | Significado |
|---|---|
| Varredura, varredura de portas | Pequenas sondas a muitos hosts em uma porta, ou a muitas portas de um host |
| Tentativa de senhas | Muitas conexões curtas a um serviço de login |
| Movimento lateral | Compartilhamento de arquivos ou administração remota para hosts internos que nunca ofereceram isso antes |
| Envio incomum | 100 MB em 10 minutos para um endereço novo, três vezes o que voltou |
| Inundação | 20.000+ pacotes pequenos/s para um endereço, dez vezes a taxa habitual dele |
| Lista de ameaças | Tráfego com um endereço listado |

De dentro da sua rede são de gravidade alta, da internet baixa.
**Resolvido** fecha uma detecção, **Não é problema** a silencia de vez.
Movimento lateral e envios incomuns precisam de um dia de histórico. Através
de amostragem 1:4096, o ataque da demo é encontrado por completo; varreduras
muito pequenas podem se esconder atrás da amostragem.

![Detecções: cada etapa de um ataque, encontrada através de amostragem sFlow 1:4096](images/findings.png)

<a id="9-offline-pcap-terminal-ui-local-capture"></a>

## 9. Pcap offline, interface de terminal, captura local

**Análise offline de pcap** mostra capturas de pacotes (pcap, pcapng) com as
mesmas páginas, separadas dos dados ao vivo: `traffic66 a.pcap b.pcapng`
inicia em 127.0.0.1 e abre o navegador (até 3 arquivos, 3 GB; Ctrl+C apaga
os dados importados), ou envie até 3 arquivos de 50 MB nessa página. Um
arquivo é analisado por vez, cada um em seu próprio banco de dados:
**Analisar** na linha de um arquivo o mostra em todas as páginas, e a barra
no topo troca para outro. Ela trabalha com fluxos, não com o conteúdo dos
pacotes.

![Análise offline: arquivos de captura com pacotes, fluxos e período](images/sandbox.png)

**Interface de terminal**: `traffic66 tui` na máquina do traffic66, ou
`traffic66 tui -server http://192.0.2.50:8066 -user admin -password …`.
Teclas: 1–8 páginas, Enter ações, f mostrar só, x excluir, t intervalo de
tempo, w abrir no navegador, q sair; `-lang` escolhe o idioma.

![Interface de terminal: visão geral](images/tui-overview.png)

![Interface de terminal: Top 66 de conversas](images/tui-topn.png)

**Captura local** gera fluxos a partir de uma interface local, de
preferência uma porta ligada à porta espelho de um switch:
`traffic66 interfaces` as lista, `-capture eth1` captura. No Windows:

```
traffic66.exe interfaces          # list the network cards: name, number, address
traffic66.exe -capture Wi-Fi      # capture on the wireless card (or by number: -capture 2)
```

O Linux precisa de root ou `setcap cap_net_raw,cap_net_admin+ep`, o macOS de
root, o Windows do [Npcap](https://npcap.com). Os fluxos capturados vêm do
equipamento `127.0.0.1`. A captura local não tem interfaces nem contadores
do equipamento, então a **Conferência de interfaces** não tem nada a
comparar para ela.

<a id="10-options-and-data"></a>

## 10. Opções e dados

`traffic66 -h` lista tudo. As mais usadas:

| Opção | Padrão | |
|---|---|---|
| `-data` | `traffic66-data` ao lado do programa | diretório de dados |
| `-addr` | `:8066` | interface web; `127.0.0.1:8066` só para esta máquina |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | coletores UDP; vazio desativa |
| `-retention-days` | `30` | dias de detalhe de fluxo; os resumos são mantidos por 400 dias |
| `-memory` | `0.10` | fração da RAM para o cache do banco de dados |
| `-sampling-wait` | `5m` | quanto tempo os registros esperam por uma taxa de amostragem |
| `-capture` | | interface local (repetível) |
| `-no-dns` | | sem consultas reversas |

O diretório de dados guarda `raw/` (detalhe, um arquivo por hora),
`traffic66.duckdb` (resumos e contadores), `password`, `inventory.txt`,
`license.json`, seu logotipo e bancos de dados. Para fazer backup, pare o
traffic66 e copie-o; para atualizar, substitua o arquivo do programa.
**Limpeza de dados** apaga dados com mais de 7–120 dias, ou todos.

<a id="licence"></a>

### Licença

Código-fonte disponível sob a [PolyForm Noncommercial License 1.0.0](../LICENSE.md)
e a [Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md) (o texto em
inglês é o que vale): gratuito para avaliação e para organizações com menos
de 100 pessoas; organizações maiores se registram após 30 dias de uso em
produção; vender, hospedar para terceiros ou produtos concorrentes exigem
uma licença comercial. Nada é desligado, nunca. O rodapé de cada página
mostra o número de instalação de 8 dígitos; envie-o ao autor e coloque o
`license.json` que receber de volta no diretório de dados. Contato:
<https://github.com/githubflyideas/traffic66>.

<a id="11-security-sizing-troubleshooting"></a>

## 11. Segurança, dimensionamento, solução de problemas

A interface web usa HTTP puro: em redes não confiáveis, use
`-addr 127.0.0.1:8066` atrás de um proxy TLS (`caddy reverse-proxy --from traffic66.example.com --to
127.0.0.1:8066`) ou de um túnel SSH. Libere as portas UDP apenas para os
seus equipamentos. As communities SNMP ficam gravadas em texto puro; use
communities somente leitura.

A 5.000 fluxos/s em 2 núcleos: cerca de 12 GB de disco por dia de detalhe
(360 GB para 30 dias), um sexto de um núcleo, 0,6–0,8 GB de memória. Visões
gerais de intervalos longos levam menos de 0,2 s; um Top 66 de 1 hora de
todas as conversas, cerca de 9 s.

| Sintoma | Solução |
|---|---|
| "waiting for the sampling rate" | Exporte as opções do sampler, ou `sampling=N` / `unsampled` na linha do equipamento |
| Abaixo dos contadores | Interfaces não amostradas, perdas ou um timeout ativo acima de 60 s |
| Acima dos contadores | O mesmo tráfego amostrado em duas interfaces ou dois equipamentos |
| Esqueci a senha | `traffic66 passwd` na máquina do traffic66 |
| `Conflicting lock is held` | Outro traffic66 usa este diretório de dados |
| `address already in use` | Escolha outras portas com `-addr` ou `-listen` |
| Windows: "O Windows protegeu o computador" | **Mais informações** → **Executar assim mesmo** |

Compilar a partir do código-fonte: Go 1.24 e um compilador C, depois `scripts/build.sh 0.1.0 traffic66`.
