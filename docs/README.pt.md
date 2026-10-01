[English](../README.md) | [中文](README.zh.md) | [हिन्दी](README.hi.md) | [Español](README.es.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [বাংলা](README.bn.md) | **Português** | [Русский](README.ru.md) | [Bahasa Indonesia](README.id.md) | [اردو](README.ur.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

# traffic66

Análise de fluxos sFlow, NetFlow e IPFIX em um único programa. Ele coleta
as exportações de fluxo de switches, roteadores e firewalls, armazena tudo
em um banco de dados embutido e mostra quem usa a banda, para onde vai o
tráfego e se os números batem com os contadores de interface dos próprios
equipamentos, em uma interface web e em uma interface de terminal.

- Um único executável para Windows, Linux e macOS. Sem banco de dados para
  instalar, sem runtime, funciona offline.
- sFlow v5, NetFlow v5, NetFlow v9 e IPFIX em qualquer porta UDP; captura
  local opcional a partir de uma interface de rede ou porta espelho.
- Confere os próprios números com os contadores de interface (contadores
  sFlow ou SNMP) e explica por que diferem quando há diferença.
- Listas Top 66, caminhos do tráfego, países e redes, ocorrências em listas
  de ameaças, registros de fluxo, encapsulamento (GRE, IPIP, VXLAN, GENEVE,
  MPLS).
- 13 idiomas na interface web e na interface de terminal.

![Visão geral: banda por aplicação em comparação com a semana passada, o que cresceu, principais clientes e serviços](images/overview.png)

<sub>Todas as capturas de tela vêm do `traffic66 demo`, uma rede corporativa simulada que você mesmo pode executar (veja [Testar a demo](#1-try-the-demo)).</sub>

<a id="contents"></a>

## Conteúdo

1. [Testar a demo](#1-try-the-demo)
2. [Instalação](#2-install) — [Linux](#linux) · [Windows](#windows) · [macOS](#macos)
3. [Usuários e senhas](#3-users-and-passwords)
4. [Enviar fluxos dos seus equipamentos](#4-send-flows-from-your-devices)
5. [Verificar se os fluxos estão chegando](#5-check-that-flows-arrive)
6. [Fazer os números baterem com os contadores de interface](#6-make-the-numbers-match-the-interface-counters)
7. [Nomes, SNMP e suas próprias redes](#7-names-snmp-and-your-own-networks)
8. [Países, redes e listas de ameaças](#8-countries-networks-and-threat-lists)
9. [Usando a interface web](#9-using-the-web-ui)
10. [Interface de terminal](#10-terminal-ui)
11. [Captura local](#11-local-capture)
12. [Opções](#12-options)
13. [Dados, backup, atualização, desinstalação](#13-data-backup-upgrade-uninstall)
14. [Segurança](#14-security)
15. [Dimensionamento](#15-sizing)
16. [Solução de problemas](#16-troubleshooting)
17. [Compilar a partir do código-fonte](#17-build-from-source)

<a id="1-try-the-demo"></a>

## 1. Testar a demo

Baixe o arquivo do seu sistema na
[página de releases](https://github.com/githubflyideas/traffic66/releases):

| Sistema | Arquivo |
|---|---|
| Windows 10/11, Server 2016 ou posterior (x64) | `traffic66-windows-amd64.zip` |
| Linux x86-64: qualquer distribuição com kernel 3.2 ou posterior, incluindo CentOS 7 e Alpine | `traffic66-linux-amd64.tar.gz` |
| Linux ARM64: as mesmas distribuições | `traffic66-linux-arm64.tar.gz` |
| macOS 11 ou posterior, Apple silicon | `traffic66-darwin-arm64.tar.gz` |
| macOS 11 ou posterior, Intel | `traffic66-darwin-amd64.tar.gz` |

Linux:

```
tar xzf traffic66-linux-amd64.tar.gz
cd traffic66-linux-amd64
./traffic66 demo -password try66
```

macOS (a segunda linha permite que o macOS execute um programa baixado da
internet que não veio da App Store):

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

Abra http://127.0.0.1:8066 e entre como `admin` / `try66`. A demo monta a
rede de uma pequena empresa com um dia de histórico e tráfego ao vivo de
quatro equipamentos simulados, incluindo dois incidentes para encontrar:
comece em **Visão geral**, veja **Quem cresceu** e vá clicando a partir
daí. Pare com Ctrl+C. Os dados da demo ficam em `traffic66-demo`, ao lado
do programa; apague essa pasta para recomeçar a demo do zero.

A demo usa as mesmas portas de uma instalação real (8066 e UDP 6343, 2055,
4739). Para rodá-la ao lado de uma instalação real, use outras portas:
`traffic66 demo -password try66 -addr :8067 -listen ""`.

No Windows, você também pode simplesmente dar um clique duplo em
`traffic66.exe`. Isso inicia o traffic66 de verdade (não a demo) e abre a
interface web no seu navegador; a senha do primeiro início aparece na janela
preta, e fechar a janela encerra o traffic66. Se o Windows exibir
"O Windows protegeu o computador", clique em **Mais informações** →
**Executar assim mesmo**.

<a id="2-install"></a>

## 2. Instalação

O traffic66 é um único arquivo. Instalar significa colocá-lo em algum
lugar, escolher um diretório de dados, definir uma senha, liberar o
firewall e configurá-lo para iniciar no boot. Os exemplos usam `192.0.2.50`
para a máquina do traffic66 e `192.0.2.1` para um roteador; troque pelos
seus endereços.

Portas:

| Porta | Uso |
|---|---|
| UDP 6343 | sFlow (padrão) |
| UDP 2055 | NetFlow (padrão) |
| UDP 4739 | IPFIX (padrão) |
| TCP 8066 | interface web e API |

Toda porta UDP aceita todos os protocolos, então um equipamento pode mandar
NetFlow para a 6343 se for mais prático. Altere ou adicione portas com
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

O último comando pede a senha do usuário `admin`.

Crie `/etc/systemd/system/traffic66.service`:

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

Inicie o serviço e permita buffers UDP maiores para não perder rajadas:

```
echo 'net.core.rmem_max=16777216' | sudo tee /etc/sysctl.d/90-traffic66.conf
sudo sysctl --system
sudo systemctl daemon-reload
sudo systemctl enable --now traffic66
sudo systemctl status traffic66
journalctl -u traffic66 -f
```

Firewall, com firewalld (RHEL, Rocky, Alma, Fedora):

```
sudo firewall-cmd --permanent --add-port=6343/udp --add-port=2055/udp --add-port=4739/udp --add-port=8066/tcp
sudo firewall-cmd --reload
```

ou com ufw (Ubuntu, Debian):

```
sudo ufw allow 6343,2055,4739/udp
sudo ufw allow 8066/tcp
```

<a id="windows"></a>

### Windows

Descompacte em `C:\traffic66` e defina a senha (PowerShell como
administrador):

```
Expand-Archive traffic66-windows-amd64.zip C:\
Rename-Item C:\traffic66-windows-amd64 C:\traffic66
C:\traffic66\traffic66.exe passwd
```

Os dados ficam em `C:\traffic66\traffic66-data`, ao lado do programa.

Libere o firewall:

```
New-NetFirewallRule -DisplayName "traffic66 flows" -Direction Inbound -Protocol UDP -LocalPort 6343,2055,4739 -Action Allow
New-NetFirewallRule -DisplayName "traffic66 web" -Direction Inbound -Protocol TCP -LocalPort 8066 -Action Allow
```

Para testar em primeiro plano, execute `C:\traffic66\traffic66.exe` e pare
com Ctrl+C. Para rodar em segundo plano desde o boot, sem ninguém logado,
registre-o como tarefa de inicialização:

```
$action   = New-ScheduledTaskAction -Execute 'C:\traffic66\traffic66.exe' -Argument '-data C:\traffic66\traffic66-data'
$trigger  = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName traffic66 -Action $action -Trigger $trigger -Settings $settings -User 'NT AUTHORITY\SYSTEM' -RunLevel Highest
Start-ScheduledTask -TaskName traffic66
```

`-ExecutionTimeLimit ([TimeSpan]::Zero)` é importante: sem ele, o Windows
encerra a tarefa depois de três dias. Pare-a com `Stop-ScheduledTask -TaskName
traffic66` e remova-a com `Unregister-ScheduledTask -TaskName traffic66`.

<a id="macos"></a>

### macOS

```
sudo mkdir -p /usr/local/traffic66
sudo tar xzf traffic66-darwin-arm64.tar.gz -C /usr/local/traffic66 --strip-components=1
sudo xattr -dr com.apple.quarantine /usr/local/traffic66
sudo /usr/local/traffic66/traffic66 passwd -data "/Library/Application Support/traffic66"
```

Crie `/Library/LaunchDaemons/traffic66.plist`:

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

Para iniciar e parar de novo:

```
sudo launchctl bootstrap system /Library/LaunchDaemons/traffic66.plist
tail -f /Library/Logs/traffic66.log
sudo launchctl bootout system/traffic66
```

Se o firewall do macOS estiver ativo, permita conexões de entrada para o
traffic66 em Ajustes do Sistema → Rede → Firewall → Opções.

<a id="3-users-and-passwords"></a>

## 3. Usuários e senhas

**Em resumo:** usuários e senhas ficam em um único arquivo, `password`, no
diretório de dados. Você nunca o edita à mão: o comando `traffic66 passwd`
adiciona, altera, lista e remove usuários. Abra
`http://<traffic66 machine>:8066` e faça login com um deles.

<a id="the-first-sign-in"></a>

### O primeiro login

Na primeira execução, o traffic66 cria o usuário `admin` com uma senha
aleatória e a mostra uma única vez:

```
first start: sign in as user "admin" with password "3f9c2a7e5b1d8046"
```

- Iniciado com clique duplo no Windows: na janela preta.
- Em um terminal: no terminal.
- Serviço do Linux: `journalctl -u traffic66 | grep "first start"`
- Serviço do macOS: `grep "first start" /Library/Logs/traffic66.log`

Perdeu? Defina uma nova com `traffic66 passwd` (abaixo). Se você definiu uma
senha com `traffic66 passwd` antes da primeira execução, como fazem os
passos de instalação acima, nenhuma é gerada.

<a id="where-the-users-are-stored"></a>

### Onde os usuários ficam guardados

No arquivo `password` do diretório de dados:

| Como o traffic66 é executado | Arquivo |
|---|---|
| Descompactado e iniciado da própria pasta (padrão) | `traffic66-data/password` ao lado do programa |
| Serviço do Linux (seção 2) | `/var/lib/traffic66/password` |
| Tarefa de inicialização do Windows (seção 2) | `C:\traffic66\traffic66-data\password` |
| Serviço do macOS (seção 2) | `/Library/Application Support/traffic66/password` |
| Demo | `traffic66-demo/password` ao lado do programa |

Uma linha por usuário. As senhas são guardadas como hashes com salt, então
ninguém consegue recuperá-las do arquivo, nem mesmo você; se uma senha for
esquecida, defina uma nova. Só o dono do arquivo pode lê-lo.

```
# traffic66 login, one user per line; change with: traffic66 passwd
admin:pbkdf2-sha256$210000$…
alice:pbkdf2-sha256$210000$…
```

<a id="managing-users"></a>

### Gerenciar usuários

Execute estes comandos na máquina do traffic66:

| Para | Comando |
|---|---|
| Trocar a senha de `admin` | `traffic66 passwd` |
| Adicionar o usuário `alice`, ou trocar a senha dela | `traffic66 passwd -user alice` |
| Remover o usuário `alice` | `traffic66 passwd -user alice -delete` |
| Listar os usuários | `traffic66 passwd -list` |
| Definir uma senha aleatória e exibi-la | `traffic66 passwd -generate` (com `-user` para outros usuários) |

- O comando pede a nova senha duas vezes e não mostra o que você digita.
  Use pelo menos 8 caracteres.
- Quando o traffic66 roda com `-data`, adicione o mesmo `-data` ao comando.
  Para o serviço do Linux da seção 2:

  ```
  sudo -u traffic66 /opt/traffic66/traffic66 passwd -data /var/lib/traffic66 -user alice
  ```

  No Windows (PowerShell como administrador):

  ```
  C:\traffic66\traffic66.exe passwd -user alice
  ```

- As mudanças valem na hora, sem reiniciar: uma senha nova funciona no
  próximo login, e um usuário removido é desconectado dos navegadores
  abertos.
- O último usuário restante não pode ser removido; adicione outro antes.
- Todos os usuários veem e podem alterar as mesmas coisas; não há papéis.

<a id="passwords-for-scripts-and-containers"></a>

### Senhas para scripts e contêineres

`TRAFFIC66_PASSWORD=…` no ambiente, ou `-password …` na linha de comando,
faz o traffic66 aceitar exatamente um usuário naquela execução: o indicado
por `-user` (padrão `admin`) com essa senha. O arquivo `password` é então
ignorado e não é alterado. Prefira a variável de ambiente: linhas de comando
ficam visíveis para outros usuários da máquina.

```
TRAFFIC66_PASSWORD='s3cret-pass' traffic66 -user ops
```

Depois de cinco senhas erradas em um minuto, o endereço fica bloqueado por
um minuto.

<a id="4-send-flows-from-your-devices"></a>

## 4. Enviar fluxos dos seus equipamentos

Aponte cada equipamento para a máquina do traffic66. Os comandos variam
conforme o modelo e a versão de software; consulte o manual do seu
equipamento. Em todos os exemplos, `192.0.2.50` é o traffic66 e
`192.0.2.1` é o endereço do próprio equipamento.

Recomendações gerais:

- Configure o timeout de fluxo ativo em 60 segundos. Timeouts maiores fazem
  o tráfego chegar atrasado e em grandes blocos.
- Se o equipamento amostra NetFlow/IPFIX, faça-o exportar as opções do
  sampler para que a taxa seja conhecida. O traffic66 segura os registros
  até a taxa chegar, em vez de contá-los como 1:1.
- Amostre todas as interfaces ou só as de borda, em um único sentido.
  Amostrar o mesmo tráfego na entrada e na saída conta tudo em dobro;
  **Conferência de interfaces** aponta isso.
- Taxa de amostragem sFlow: cerca de 1:1000 para links de 1 Gb/s, 1:4096
  para 10 Gb/s e 1:8192 para 40/100 Gb/s.

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

FortiGate FortiOS 7.4.2 ou posterior (NetFlow v9):

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

Servidores e hosts Linux, com softflowd (NetFlow v9):

```
softflowd -i eth0 -n 192.0.2.50:2055 -v 9 -t maxlife=60
```

<a id="5-check-that-flows-arrive"></a>

## 5. Verificar se os fluxos estão chegando

Abra **Fontes**. Cada equipamento que envia algo aparece em segundos, com
protocolo, taxa de amostragem, perdas, último pacote e um status. Quando o
status não está verde, o texto ao lado diz o que está errado e o que
mudar.

![Fontes: cada equipamento com protocolo, amostragem, perdas e o que corrigir](images/sources.png)

Se um equipamento não aparecer:

1. Verifique se chegam pacotes na máquina do traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Se nada aparecer, os pacotes não chegam à máquina: confira a
   configuração do equipamento, o roteamento e os firewalls no caminho.
2. Os pacotes chegam, mas **Fontes** continua vazia: o firewall local está
   descartando (veja [Instalação](#2-install)) ou o traffic66 escuta em
   outras portas (`-listen`).
3. Para testar o caminho a partir de outra máquina sem mexer em nenhum
   equipamento, rode `traffic66 simulate -to 192.0.2.50` nela por alguns
   segundos. Ele envia sFlow, NetFlow e IPFIX de equipamentos simulados,
   que depois aparecem em **Fontes** e nos dados; por isso, prefira fazer
   isso em uma instalação de teste.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Fazer os números baterem com os contadores de interface

Os números de fluxo são estimativas: pacotes amostrados vezes a taxa de
amostragem. O traffic66 os compara com os contadores de interface do
próprio equipamento e mostra a diferença em **Conferência de interfaces**,
com a causa provável quando ela é maior do que a amostragem sozinha
explica.

![Conferência de interfaces: estimativa de fluxo ao lado do contador do equipamento para cada interface](images/interfaces.png)

Para ter contadores com que comparar:

- Equipamentos sFlow enviam os contadores por conta própria quando há um
  intervalo de contadores configurado (`sflow counter interval 30` e
  similares).
- Para equipamentos NetFlow e IPFIX, adicione uma linha `snmp` em
  **Fontes → Nomes** (veja [Nomes](#7-names-snmp-and-your-own-networks)).
  O traffic66 passa a ler os contadores de interface a cada minuto.

O que o traffic66 já faz para os números baterem: usa a taxa de amostragem
que o equipamento realmente aplicou, segura os registros NetFlow/IPFIX até
a taxa de amostragem ser conhecida, compensa pacotes de exportação perdidos
no caminho, distribui fluxos longos pelos minutos em que duraram e soma 18
bytes de overhead Ethernet por pacote às contagens de bytes de
NetFlow/IPFIX (os contadores de interface incluem esse overhead, as
contagens de fluxo na camada IP não; altere com `-l2-overhead`).

Motivos comuns para a diferença que sobra, todos apontados em
**Conferência de interfaces**: algumas interfaces não são amostradas, o
mesmo tráfego é amostrado em duas interfaces, pacotes de exportação se
perdem antes de chegar ao traffic66 ou a taxa de amostragem ainda não é
conhecida.

<a id="7-names-snmp-and-your-own-networks"></a>

## 7. Nomes, SNMP e suas próprias redes

**Fontes → Nomes** na interface web aceita uma entrada por linha. O
conteúdo é salvo como `inventory.txt` no diretório de dados, então você
também pode editar esse arquivo (veja `inventory.txt.example`). Todas as
linhas são opcionais.

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

- `net`: as faixas privadas (10/8, 172.16/12, 192.168/16, 100.64/10) são
  sempre consideradas suas. Adicione suas faixas públicas para que o
  tráfego de e para elas também conte como seu; o nome aparece em
  **Top-N → Segmentos** e nos caminhos do tráfego.
- `snmp <device> <community> [<management address>[:port]]`: o equipamento
  é o endereço de onde vêm os fluxos. Adicione o endereço de gerência
  quando o equipamento responder SNMP em outro endereço. As descrições de
  interface lidas via SNMP são usadas como nomes, a menos que você nomeie a
  interface com `iface`. Libere a máquina do traffic66 na lista de acesso
  SNMP do equipamento.
- As alterações valem assim que você clica em **Salvar**; não é preciso
  reiniciar.

<a id="8-countries-networks-and-threat-lists"></a>

## 8. Países, redes e listas de ameaças

Países e nomes de rede (AS) exigem uma tabela IP-para-ASN. Baixe a
gratuita em [iptoasn.com](https://iptoasn.com):

```
curl -LO https://iptoasn.com/data/ip2asn-combined.tsv.gz
mv ip2asn-combined.tsv.gz <data directory>/asn.tsv.gz
```

Qualquer arquivo no mesmo formato serve (separado por tabulação: primeiro
endereço, último endereço, número do AS, código do país, nome do AS; texto
puro ou gzip). Reinicie o traffic66 depois de trocá-lo; baixe um novo mais
ou menos todo mês.

Listas de ameaças são arquivos de texto puro com um endereço ou rede por
linha (o texto depois de `#` ou `;` é ignorado), salvos como
`<data directory>/threats/<name>.txt`, por exemplo:

```
mkdir -p <data directory>/threats
curl -L https://www.spamhaus.org/drop/drop.txt -o <data directory>/threats/spamhaus-drop.txt
```

Reinicie o traffic66 depois de adicionar ou alterar listas. As ocorrências
aparecem em **Ameaças**, pelo nome da lista.

![Ameaças: um host interno enviando dados para um endereço de uma lista de ameaças](images/threats.png)

<a id="9-using-the-web-ui"></a>

## 9. Usando a interface web

Raramente é preciso digitar. Todo valor de toda página (um endereço, uma
porta, uma aplicação, um país, um equipamento) é clicável:

- **Mostrar só isto** / **Excluir isto** adiciona um filtro. Os filtros
  aparecem abaixo da barra superior e valem para todas as páginas até você
  removê-los.
- **Ver os registros de fluxo** abre os fluxos individuais correspondentes.
- **Consultar on-line** abre o endereço ou o AS em um site público de
  consulta.
- **Copiar** copia o valor.

Páginas:

| Página | O que responde |
|---|---|
| Visão geral | Quanto tráfego há agora e em comparação com a semana passada, por aplicação; o que cresceu; principais clientes e serviços |
| Top-N | Os 66 maiores clientes, servidores, conversas, aplicações, portas, países, redes, segmentos, equipamentos, encapsulamentos ou VLANs |
| Caminhos do tráfego | Qual segmento fala com qual aplicação em qual país |
| Geografia e redes | Tráfego por país e por rede (AS) |
| Ameaças | Hosts que se comunicaram com endereços das suas listas de ameaças, e quanto enviaram |
| Registros de fluxo | Fluxos individuais, dos mais recentes para os mais antigos, com colunas selecionáveis |
| Conferência de interfaces | Números de fluxo ao lado dos contadores de interface, piores primeiro, com os motivos |
| Fontes | Equipamentos, amostragem, perdas, coletores, SNMP e **Nomes** |

Acima das páginas: intervalo de tempo (de 15 minutos a 30 dias), uma caixa
de busca opcional, atualização automática a cada 30 segundos e
**Copiar link**, que copia um link para exatamente a visão atual (página,
intervalo de tempo e filtros) para mandar a um colega. O idioma segue o do
navegador; dá para trocar no fim do menu.

Intervalos maiores que 6 horas começam em uma hora cheia, para que todos os
números da página contem exatamente o mesmo tempo: "24 horas" cobre as
últimas 24 horas cheias mais a atual. Nesses intervalos, o Top-N vem de
resumos por hora; lá não há filtros, e a página avisa. Escolha um intervalo
menor para filtrar.

![Top-N: as 66 maiores conversas da última hora](images/topn.png)

![Caminhos do tráfego: qual segmento usa qual aplicação para qual país](images/paths.png)

A mesma visão geral em chinês; todas as páginas estão disponíveis em 13 idiomas:

![Visão geral em chinês](images/overview-zh.png)

<a id="10-terminal-ui"></a>

## 10. Interface de terminal

```
traffic66 tui                                     # traffic66 on this machine
traffic66 tui -server http://192.0.2.50:8066 -user admin -password …
traffic66 -tui                                    # collect and show the terminal UI in one process
```

Na máquina do traffic66, `traffic66 tui` faz login sozinho quando consegue
ler o diretório de dados (informe `-data` quando não for o padrão). Se o
traffic66 roda com outro usuário, como acontece com um serviço, use `-user`
e `-password`. `-lang` escolhe o idioma (`en`, `zh`, `hi`, `es`, `ar`,
`fr`, `bn`, `pt`, `ru`, `id`, `ur`, `ja`, `ko`).

Teclas: 1–8 páginas, ↑↓ selecionar, Enter ações sobre o valor selecionado,
f mostrar só, x excluir, / buscar, t intervalo de tempo, c limpar filtros,
w abrir a mesma visão no navegador, q sair.

![Interface de terminal: visão geral](images/tui-overview.png)

![Interface de terminal: Top-N de clientes](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Captura local

Além das exportações de fluxo, o traffic66 pode gerar fluxos por conta
própria a partir dos pacotes de uma interface de rede local, por exemplo
uma porta espelho (SPAN):

```
traffic66 interfaces                  # list interfaces
traffic66 -capture eth1               # repeat -capture for more interfaces
```

- Linux: precisa de root ou das capabilities `CAP_NET_RAW` e
  `CAP_NET_ADMIN` (`sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66`,
  ou a linha `AmbientCapabilities` da unit systemd acima).
- macOS: precisa de root (dispositivos BPF); nada a instalar.
- Windows: instale antes o [Npcap](https://npcap.com).

As interfaces capturadas aparecem em **Fontes**. Pacotes vistos duas vezes
(por exemplo, em duas portas espelho) são contados duas vezes.

<a id="12-options"></a>

## 12. Opções

`traffic66 -h` e `traffic66 <command> -h` listam tudo.

Comandos:

| Comando | |
|---|---|
| `traffic66` | coleta fluxos e serve a interface web |
| `traffic66 demo` | o mesmo, com uma rede simulada |
| `traffic66 tui` | interface de terminal para um traffic66 em execução |
| `traffic66 passwd` | adiciona, altera, lista ou remove usuários (veja [Usuários e senhas](#3-users-and-passwords)) |
| `traffic66 simulate -to HOST` | envia exportações simuladas para um coletor |
| `traffic66 interfaces` | lista as interfaces para captura local |
| `traffic66 version` | mostra a versão |

Opções de `traffic66` e `traffic66 demo`:

| Opção | Padrão | |
|---|---|---|
| `-addr` | `:8066` | endereço da interface web; `127.0.0.1:8066` só para esta máquina |
| `-data` | `traffic66-data` ao lado do programa | diretório de dados |
| `-listen` | `sflow=:6343,netflow=:2055,ipfix=:4739` | coletores UDP no formato `name=address`, separados por vírgula; vazio desativa |
| `-user` | `admin` | nome do usuário criado na primeira execução e do usuário ao qual `-password` se aplica |
| `-password` | não definido | aceita só `-user` com esta senha nesta execução, ignorando o arquivo `password` (também `TRAFFIC66_PASSWORD`) |
| `-retention-days` | `30` | dias de detalhe de fluxo mantidos; os resumos são mantidos por 400 dias |
| `-memory` | `0.10` | fração da memória física para o cache do banco de dados, e o mesmo valor como limite flexível para o resto do programa (cada um com pelo menos 256 MB) |
| `-l2-overhead` | `18` | bytes por pacote somados às contagens de bytes de NetFlow/IPFIX |
| `-sampling-wait` | `5m` | quanto tempo os registros esperam por uma taxa de amostragem |
| `-capture` | | captura em uma interface local (repetível) |
| `-inventory` | `<data>/inventory.txt` | arquivo de nomes |
| `-asn` | `<data>/asn.tsv.gz` | tabela IP-para-ASN |
| `-threat` | `<data>/threats/*.txt` | lista de ameaças extra no formato `name=path` (repetível) |
| `-dns-upstream` | resolvedor do sistema | servidor DNS para exibir nomes de host |
| `-dns-rate` | `20` | máximo de consultas reversas por segundo |
| `-dns-cache` | `2m` | por quanto tempo os nomes de host ficam em cache |
| `-no-dns` | | sem consultas reversas |
| `-tui` | | abre também a interface de terminal |

Exemplo: uma segunda porta de coleta, um ano de detalhe e a interface web
só na máquina local:

```
traffic66 -data /var/lib/traffic66 -listen "sflow=:6343,netflow=:2055,ipfix=:4739,netflow=:9995" -retention-days 365 -addr 127.0.0.1:8066
```

<a id="13-data-backup-upgrade-uninstall"></a>

## 13. Dados, backup, atualização, desinstalação

O diretório de dados guarda tudo:

| | |
|---|---|
| `raw/` | detalhe dos fluxos, um arquivo compactado por hora |
| `traffic66.duckdb` | resumos, contadores de interface e a hora atual |
| `password` | senhas de login (em hash) |
| `inventory.txt` | nomes (**Fontes → Nomes**) |
| `asn.tsv.gz`, `threats/` | tabelas de consulta que você adicionou |

- **Backup**: pare o traffic66 e copie o diretório. Sem parar, copie
  `raw/`, `password` e `inventory.txt`; nesse caso ficam faltando a hora
  atual e os resumos.
- **Mudança de local**: pare o traffic66, mova o diretório e inicie com
  `-data` apontando para o novo local.
- **Atualização**: pare o traffic66, substitua o arquivo do programa e
  inicie de novo. Os dados são mantidos. No Linux, por exemplo:

  ```
  sudo systemctl stop traffic66
  sudo tar xzf traffic66-linux-amd64.tar.gz -C /opt/traffic66 --strip-components=1
  sudo systemctl start traffic66
  ```

- **Desinstalação**: pare e remova o serviço ou a tarefa de inicialização
  (veja [Instalação](#2-install)) e depois apague a pasta do programa e o
  diretório de dados.

<a id="14-security"></a>

## 14. Segurança

- A interface web usa HTTP puro: senhas e dados trafegam pela rede sem
  criptografia. Em redes em que você não confia totalmente, escute só nesta
  máquina (`-addr 127.0.0.1:8066`) e coloque um proxy reverso com TLS na
  frente, por exemplo com o [Caddy](https://caddyserver.com):
  `caddy reverse-proxy --from traffic66.example.com --to 127.0.0.1:8066`.
  Ou acesse por VPN ou túnel SSH:
  `ssh -L 8066:127.0.0.1:8066 user@192.0.2.50` e abra
  http://127.0.0.1:8066.
- Libere as portas UDP de coleta apenas para os endereços dos seus
  equipamentos.
- As communities SNMP em `inventory.txt` ficam gravadas em texto puro; use
  uma community somente leitura.

<a id="15-sizing"></a>

## 15. Dimensionamento

Medido a 5.000 fluxos por segundo em uma máquina com 2 núcleos: o detalhe
ocupa cerca de 12 GB de disco por dia, mais cerca de 1,5 GB para a hora
atual; o programa usa um sexto de um núcleo.
Visões gerais de intervalos longos vêm dos resumos e levam menos de 0,2 s.
Consultas sobre o detalhe varrem cerca de 22 milhões de linhas por hora: um
host em 1 hora leva menos de 1 s, um Top 66 de 1 hora de todas as conversas
cerca de 9 s; o tempo cresce com o intervalo e cai com mais núcleos.

Portanto, 30 dias a 5.000 fluxos/s ocupam cerca de 360 GB de disco; ajuste
conforme a sua taxa de fluxos (mostrada em **Fontes**) e `-retention-days`.

Memória: `-memory` (padrão 10% da RAM, pelo menos 256 MB) limita o cache do
banco de dados, e o resto do programa recebe um limite flexível do mesmo
tamanho. A 5.000 fluxos por segundo, os dados do próprio programa
(decodificação, detecção de duplicatas, lotes) ocupam cerca de 90 MB; no
total, conte com 0,6–0,8 GB, então uma máquina com 2 GB de RAM basta.
Medido em 10 minutos de coleta contínua (pico de 0,58 GB em uma máquina de
8 GB) e ao carregar uma hora de fluxos a onze vezes essa taxa com os limites
de uma máquina de 2 GB (pico de 0,74 GB).

<a id="16-troubleshooting"></a>

## 16. Solução de problemas

| Sintoma | Causa e solução |
|---|---|
| Equipamento não aparece em **Fontes** | Os pacotes não chegam: veja [Verificar se os fluxos estão chegando](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | O equipamento ainda não enviou as opções do sampler; a maioria reenvia em poucos minutos. Se nunca enviar, exporte-as (`option sampler-table` no Cisco) ou marque-o como `unsampled` em Nomes se ele for de fato 1:1 |
| Números abaixo dos contadores de interface | Veja **Conferência de interfaces**: perdas no caminho, interfaces não amostradas ou fluxos ainda no cache do equipamento (timeout ativo acima de 60 s) |
| Números acima dos contadores de interface | O mesmo tráfego amostrado em duas interfaces ou dois equipamentos |
| Sem países nem redes | Falta a tabela IP-para-ASN: veja [Países](#8-countries-networks-and-threat-lists) |
| Esqueci a senha | `traffic66 passwd` na máquina do traffic66 (adicione `-data` se o traffic66 roda com ele) |
| `Conflicting lock is held` | Outro traffic66 já usa este diretório de dados |
| `receive buffer is only … KB` | O Linux limita os buffers UDP: defina `net.core.rmem_max=16777216` (veja [Linux](#linux)) |
| `cannot create the data directory` | Este usuário não tem permissão de escrita na pasta do programa: informe `-data` |
| macOS: "cannot be opened" ou "developer cannot be verified" | `xattr -dr com.apple.quarantine <folder>` |
| Windows: "O Windows protegeu o computador" | **Mais informações** → **Executar assim mesmo**; o programa ainda não é assinado |
| Captura no Windows: Npcap não encontrado | Instale o [Npcap](https://npcap.com) |
| `address already in use` | Outro programa usa a porta: escolha outras com `-addr` ou `-listen` |

<a id="17-build-from-source"></a>

## 17. Compilar a partir do código-fonte

Go 1.24 e um compilador C (gcc ou clang; MinGW-w64 no Windows):

```
git clone https://github.com/githubflyideas/traffic66
cd traffic66
scripts/build.sh 0.1.0 traffic66
```
