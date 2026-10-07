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
- Encontra nos fluxos varreduras, tentativas de senhas, movimento lateral,
  envios incomuns, inundações e tráfego de listas de ameaças, inclusive
  através da amostragem, e os lista como detecções a tratar.
- Listas Top 66, quem fala com quem em gráficos de anéis (servidores e
  seus clientes, serviços e seus servidores), tráfego ao longo do tempo por
  interface e rede (AS), caminhos do tráfego, países num mapa-múndi,
  ocorrências em listas de ameaças, registros de fluxo, encapsulamento
  (GRE, IPIP, VXLAN, GENEVE, MPLS).
- `traffic66 captura.pcap` abre até 3 capturas de pacotes (3 GB no total) na interface web: fluxos, detecções, países e registros de toda a captura, sem configurar nada.
- 13 idiomas na interface web e na interface de terminal.
- Código-fonte disponível: gratuito para avaliação e para organizações
  com menos de 100 pessoas; organizações maiores se registram após 30 dias
  de uso em produção. Nada é desligado, nunca (veja
  [Teste e licença](#trial-and-licence)).

![Visão geral: detecções abertas, banda por aplicação em comparação com o mesmo horário de ontem, principais clientes e serviços](images/overview.png)

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
quatro equipamentos simulados, incluindo um ataque: **Detecções** mostra
cada etapa dele (uma varredura, uma varredura de portas, tentativas de
senhas, movimento lateral, um envio para um servidor de controle) e uma
inundação contra o site público. Clique em **Detalhes** em uma detecção, ou
comece em **Visão geral**, clique em um host em **Principais clientes**,
escolha **Ver detalhes** e vá clicando a partir daí. Pare com Ctrl+C. Os dados da demo ficam em `traffic66-demo`, ao lado
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
# hard memory limit for the whole process (see Sizing)
MemoryMax=2G
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

Abra **Configurações**. Cada equipamento que envia algo aparece em segundos, com
protocolo, taxa de amostragem, perdas, último pacote e um status. Quando o
status não está verde, o texto ao lado diz o que está errado e o que
mudar.

**Perdidos** conta as amostras ou registros que nunca chegaram. Para sFlow
o texto diz onde eles se perderam: no caminho (saltos nos números de
sequência: a rede, ou o buffer de recepção UDP desta máquina; se
`netstat -su` mostrar erros de buffer de recepção aumentando, aumente
`net.core.rmem_max`), ou no próprio equipamento (o sFlow informa as
amostras que o equipamento descartou: a exportação sFlow dele tem limite
de taxa, então amostre com menos frequência ou aumente o limite do
equipamento). Os totais são compensados nos dois casos; o detalhe por host
não.

![Configurações: cada equipamento com protocolo, amostragem, perdas e o que corrigir](images/sources.png)

Se um equipamento não aparecer:

1. Verifique se chegam pacotes na máquina do traffic66 (Linux, macOS):
   `sudo tcpdump -ni any udp port 6343 or udp port 2055 or udp port 4739`.
   Se nada aparecer, os pacotes não chegam à máquina: confira a
   configuração do equipamento, o roteamento e os firewalls no caminho.
2. Os pacotes chegam, mas **Configurações** continua vazia: o firewall local está
   descartando (veja [Instalação](#2-install)) ou o traffic66 escuta em
   outras portas (`-listen`).
3. Para testar o caminho a partir de outra máquina sem mexer em nenhum
   equipamento, rode `traffic66 simulate -to 192.0.2.50` nela por alguns
   segundos. Ele envia sFlow, NetFlow e IPFIX de equipamentos simulados,
   que depois aparecem em **Configurações** e nos dados; por isso, prefira fazer
   isso em uma instalação de teste.

<a id="6-make-the-numbers-match-the-interface-counters"></a>

## 6. Fazer os números baterem com os contadores de interface

Os números de fluxo são estimativas: pacotes amostrados vezes a taxa de
amostragem. O traffic66 os compara com os contadores de interface do
próprio equipamento e mostra a diferença em **Conferência de interfaces**,
com a causa provável quando ela é maior do que a amostragem sozinha
explica. Cada interface tem um gráfico em bits/s na largura toda e, abaixo
dele, outro em pacotes/s, com a entrada (verde) e a saída (azul); os
contadores do próprio equipamento são linhas tracejadas no gráfico em
bits/s. Escolher uma interface na lista mostra os gráficos dela.

Cada linha da lista tem dois botões. **✎** dá à interface um nome e um
rótulo curto (como *uplink*), mostrado ao lado do nome dela em todo lugar.
**☆** a torna a interface padrão (**★**); só existe uma. As páginas então
abrem nela (veja a opção **Interface** em [Usando a interface
web](#9-using-the-web-ui)), e a visão geral mostra a banda dela. Os dois
são salvos na hora na linha `iface` de Nomes.

Os registros de fluxo nomeiam duas interfaces: aquela por onde um pacote
entrou e aquela por onde saiu. Por isso, um equipamento que amostra só
algumas interfaces mostra também as outras pontas dos fluxos delas. Essas
**Interfaces do outro lado** aparecem por último, em letra cinza menor, sob
um título próprio: os números delas contêm só o tráfego que passou por uma
interface amostrada, não todo o tráfego delas. Elas não são oferecidas em
**Interface** acima das páginas. O traffic66 sabe qual interface amostrou
um fluxo pela fonte de dados do sFlow, ou pelo campo flowDirection (IPFIX
61) do NetFlow v9 e do IPFIX (ingress: a interface de entrada, egress: a
interface de saída). Sem esse campo, uma interface presente em pelo menos
90% do tráfego de um equipamento é tomada como a amostrada; se não houver
nenhuma, nenhuma interface é marcada. **Configurações** mostra, para cada
equipamento, as interfaces amostradas (**Amostragem em:**) e se os modelos
dele trazem flowDirection (**com flowDirection (61)**). Para ver todo o
tráfego de um equipamento, amostre cada interface na entrada (veja [Enviar
fluxos dos seus equipamentos](#4-send-flows-from-your-devices)).

![Conferência de interfaces: tráfego de cada interface, e a estimativa de fluxo ao lado do contador do equipamento](images/interfaces.png)

Para ter contadores com que comparar:

- Equipamentos sFlow enviam os contadores por conta própria quando há um
  intervalo de contadores configurado (`sflow counter interval 30` e
  similares).
- Para equipamentos NetFlow e IPFIX, adicione uma linha `snmp` em
  **Configurações → Nomes** (veja [Nomes](#7-names-snmp-and-your-own-networks)).
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

O jeito mais rápido de dar nome a um host ou equipamento: clique no endereço
dele em qualquer página e escolha **Dar um nome…**. Digite o nome e tecle
Enter; ele é salvo na hora e mostrado em todo lugar no lugar do endereço
puro.

Para redes, interfaces e SNMP, use **Configurações → Nomes**: escolha o tipo
(host, rede, dispositivo, interface, SNMP), preencha o endereço e o nome e
clique em **Adicionar**. A tabela abaixo lista todos os nomes com
**Editar** e **Apagar**; adicionar o mesmo endereço de novo substitui a
entrada anterior. Endereços e redes são verificados antes de salvar.

Os nomes são salvos como `inventory.txt` no diretório de dados, uma
entrada por linha. **Editar como texto (avançado)** mostra esse arquivo, e
você também pode editá-lo diretamente (veja `inventory.txt.example`).
Todas as linhas são opcionais.

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

- `net`: as faixas privadas (10/8, 172.16/12, 192.168/16, 100.64/10) são
  sempre consideradas suas. Adicione suas faixas públicas para que o
  tráfego de e para elas também conte como seu; o nome aparece em
  **Top 66** agrupado por segmento e nos caminhos do tráfego por segmento.
  `country=JP` (um código de país de duas letras) diz onde a rede fica; o
  mapa-múndi então traça linhas dela até os países com que ela se
  comunica.
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

Países e redes (AS) funcionam desde o início: o traffic66 traz embutidas as bases gratuitas **IP to Country Lite** e **IP to ASN Lite** da DB-IP (licença [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/); "IP Geolocation by DB-IP", [db-ip.com](https://db-ip.com)). As páginas que mostram países e redes indicam a origem dos dados.

A cópia embutida é da versão que você usa. A DB-IP publica uma nova a cada mês; **Configurações → Banco de dados de países e redes → Atualizar DB-IP Lite agora** baixa a mais recente de db-ip.com (o servidor que roda o traffic66 precisa de acesso à internet; se falhar, a interface avisa).

Você também pode usar outro banco gratuito. Baixe-o e envie-o na mesma página com **Enviar arquivo de banco de dados…**. Ele é verificado, salvo no diretório de dados e usado para o tráfego novo na hora, sem reiniciar. O tráfego já guardado mantém o país com que foi salvo.

| Banco | Fornece | Licença | Onde obter |
|---|---|---|---|
| DB-IP Lite (embutida) | países; redes | CC BY 4.0, sem conta | [db-ip.com/db/lite.php](https://db-ip.com/db/lite.php) |
| MaxMind GeoLite2 Country e ASN, `.mmdb` | países; redes | GeoLite2 EULA, conta gratuita | [maxmind.com](https://www.maxmind.com/en/geolite2/signup) |
| IPinfo Lite, `ipinfo_lite.mmdb` | países e redes em um só arquivo | CC BY-SA 4.0, conta gratuita | [ipinfo.io/lite](https://ipinfo.io/lite) |
| IPtoASN, `ip2asn-combined.tsv.gz` | redes com seu país | PDDL 1.0, sem conta | [iptoasn.com](https://iptoasn.com) |

Seus arquivos são usados primeiro; o que eles não cobrem vem da DB-IP Lite embutida. **Remover** ao lado de um arquivo volta ao restante. A página lista o que está em uso e a data de cada banco.

Sem a interface web, copie o arquivo para o diretório de dados como `country.mmdb`, `asn.mmdb`, `both.mmdb` (um arquivo com países e redes, como o IPinfo Lite) ou `asn.tsv.gz` e reinicie o traffic66.

**Geografia e redes** mostra num mapa-múndi o tráfego com outros países: quanto mais escuro, mais tráfego. Passe o mouse sobre um país para ver o tráfego; clique para filtrar ou abrir os registros de fluxo. Quando suas redes têm um país (`country=` numa linha `net`, veja [Nomes](#7-names-snmp-and-your-own-networks)), linhas vão desse país até os países com que elas trocam tráfego, mais grossas quanto mais tráfego. Os contornos dos países vêm do [Natural Earth](https://www.naturalearthdata.com) (domínio público).

![Geografia e redes: tráfego remoto por país num mapa-múndi](images/geo.png)

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
- **Ver detalhes** (hosts, equipamentos e serviços) abre uma página sobre
  aquele host ou serviço: o tráfego ao longo do tempo por aplicação, com
  quem ele fala, quais serviços ou clientes, países e os fluxos mais
  recentes. Todo valor ali também é clicável, então dá para continuar
  aprofundando; o botão Voltar do navegador retorna.
- **Dar um nome…** (hosts e equipamentos) dá um nome ao endereço, mostrado
  em todo lugar a partir daí.
- **Consultar on-line** abre o endereço ou o AS em um site público de
  consulta.
- **Copiar** copia o valor.

Páginas:

| Página | O que responde |
|---|---|
| Visão geral | A banda da interface escolhida (ou da padrão, ou senão da com mais tráfego) em bits/s, entrada e saída vistas pela interface; quanto tráfego há agora, por aplicação (**Total**, ou só o tráfego de **Entrada** ou de **Saída** das suas redes), em comparação com o mesmo horário de ontem (intervalos de até um dia), a semana passada (até uma semana) ou os dias anteriores (intervalos mais longos), quando há dados daquela época; detecções abertas; direção e protocolo; principais clientes e serviços |
| Top 66 | Abre em **Tabela**, uma única tabela dos 66 maiores: por padrão, conversas (cliente, servidor, serviço, país). Qualquer cabeçalho ordena; as colunas numéricas (tráfego, pacotes, pacote médio, fluxos) classificam todo o tráfego do período, então o menor pacote médio revela varreduras e inundações. **Agrupar por** muda para aplicações, redes, segmentos, equipamentos, encapsulamento e VLAN. **Principais interlocutores** mostra os 30 maiores clientes e servidores lado a lado com tráfego, pacotes e registros de fluxo, acima de uma linha para todo o tráfego |
| Detalhes do tráfego | Dois gráficos de anéis. **Servidores e clientes**: o anel interno são os 8 servidores com mais tráfego, o externo os clientes de cada um; **Clientes dentro** inverte (clientes dentro, fora os servidores que cada um usa), já que muitas vezes um lado explica mais que o outro. **Serviços**: um anel com os serviços com mais tráfego. Passe o mouse sobre um segmento para ver o tráfego; clique nele como em qualquer valor |
| Caminhos do tráfego | Qual host usa qual aplicação para qual país: os 8 hosts com mais tráfego, o resto como Outros. **Cliente → servidor** mostra cliente → serviço → servidor; **Por segmento** mostra segmentos em vez de hosts. Nomes longos são encurtados para 22 caracteres; passe o mouse sobre um para ver o nome completo |
| Detecções | O que precisa de atenção: varreduras, tentativas de senhas, movimento lateral, envios incomuns, inundações e tráfego de listas de ameaças ([mais](#findings)) |
| Ameaças | Hosts que se comunicaram com endereços das suas listas de ameaças, e quanto enviaram |
| Geografia e redes | Um mapa-múndi do tráfego por país, com linhas a partir das suas redes; as redes (AS) de onde o tráfego veio e para onde foi, ao longo do tempo em bits/s e pacotes/s; tráfego por país e por rede |
| Configurações | Equipamentos, amostragem, as interfaces que cada equipamento amostra e se ele envia flowDirection, perdas, coletores, SNMP, o banco de dados de países e redes, o logotipo e **Nomes** |
| Conferência de interfaces | Tráfego de cada interface ao longo do tempo em bits/s e, abaixo, pacotes/s, entrada (verde) e saída (azul), com os contadores do equipamento como linhas tracejadas; quanto os números de fluxo se afastam dos contadores, piores primeiro, com os motivos; um nome, um rótulo e a padrão para cada interface |
| Registros de fluxo | Quantos registros de fluxo houve e quando (uma barra por intervalo), e os próprios registros, dos mais recentes para os mais antigos, página a página, com colunas selecionáveis. Abre nos últimos 15 minutos, atualizados a cada 5 segundos; aberta a partir de um valor em outra página (**Ver os registros de fluxo**), mantém o intervalo de tempo daquela página, e **Voltar ao tempo real** volta |
| Limpeza de dados | Apaga dados com mais de 120, 90, 60, 30 ou 7 dias, ou todos, mostrando quanto cada opção libera ([mais](#13-data-backup-upgrade-uninstall)) |
| Análise offline de pcap | Capturas de pacotes (pcap, pcapng) analisadas separadas dos dados ao vivo ([mais](#análise-offline-de-pcap)) |

O menu lateral organiza as páginas em quatro grupos: tráfego (Visão geral,
Top 66, Detalhes do tráfego, Caminhos do tráfego, Conferência de
interfaces), segurança (Detecções, Ameaças, Geografia e redes),
configuração e dados (Configurações, Registros de fluxo, Limpeza de dados)
e Análise offline de pcap. Abaixo do logotipo ficam a versão e a data e hora do servidor.

Acima das páginas de tráfego (Visão geral, Top 66, Detalhes do tráfego,
Caminhos do tráfego, Geografia e redes, Registros de fluxo e o detalhe de
um valor) fica **Interface**: **Todas as interfaces**, ou uma interface,
para que essas páginas mostrem só o tráfego que passa por ela (entrando ou
saindo). Lista as interfaces amostradas, agrupadas por equipamento, e
começa na interface padrão (★, definida em **Conferência de
interfaces**) e a escolha faz parte do link. Detecções, Ameaças,
Conferência de interfaces e Configurações sempre cobrem todo o tráfego.
Para uma interface em 7 ou 30 dias, as páginas leem os registros de fluxo
em vez dos resumos por hora e por dia, então demoram mais e alcançam até
onde os registros de fluxo são mantidos (30 dias por padrão).

Acima das páginas: intervalo de tempo (de 15 minutos a 30 dias, ou
**Personalizado…** para qualquer início e fim, inclusive antes dos últimos
30 dias), atualização automática a cada 30 segundos e **Copiar link**, que
copia um link para exatamente a visão atual (página, intervalo de tempo e
filtros) para mandar a um colega. Em **Top 66** e **Detalhes do tráfego**,
**Equipamento**, **Cliente**, **Servidor** e **Serviço** listam os valores com mais tráfego do intervalo: escolha um, ou
digite um, para filtrar; o filtro passa então a valer em todas as páginas
até você esvaziar a caixa. O idioma segue o do navegador; dá para trocar no
fim do menu, acima de **Sair**. Configurações, Registros de fluxo (em
tempo real), Limpeza de dados e Análise offline de pcap não têm intervalo de
tempo.

Ao lado do idioma fica o tema de cores, inspirado nas cores do sistema do
iOS: **Claro** (o padrão), **Cinza**, **Preto** (para telas de parede),
**Verde-azulado** e **Laranja**. Cada clique passa para o próximo; a
escolha fica guardada no navegador.

Os gráficos ao longo do tempo mostram os 8 maiores valores em cores fixas e
o resto como Outros; a legenda dá o total de cada valor e pode ser clicada
como qualquer outro valor. Os gráficos de clientes e servidores deixam o
resto fora do desenho, já que com milhares de hosts ele achataria os 8
maiores; a legenda continua dando o total.

Os gráficos terminam onde os dados estão completos: com sFlow no minuto
atual, com NetFlow e IPFIX um pouco antes, tanto quanto os dispositivos levam
para exportar seus fluxos (o traffic66 mede isso; no máximo 2 minutos).

Intervalos maiores que 6 horas começam em uma hora cheia, para que todos os
números da página contem exatamente o mesmo tempo: "24 horas" cobre as
últimas 24 horas cheias mais a atual. Nesses intervalos, o Top 66 vem de
resumos por hora; lá não há filtros, e a página avisa. Escolha um intervalo
menor para filtrar. As conversas sempre leem o detalhe dos fluxos, então em intervalos
longos com muitos fluxos podem demorar; uma hora é o mais rápido.

O menu lateral mostra quanto disco os dados usam e quanto está livre; passe
o mouse sobre o espaço livre para ver quanto os dias de detalhe mantidos
precisam no ritmo atual (estimado assim que houver um dia de dados).

Para mostrar o seu próprio logotipo na página de login e no topo do menu,
use **Configurações → Logotipo → Enviar um logotipo…**: PNG, SVG, JPEG, WebP ou
GIF, até 1 MB, de preferência com 272 × 92 pixels (outros tamanhos são
ajustados). **Usar o logotipo embutido** volta ao do traffic66.

<a id="findings"></a>

### Detecções

**Detecções** lista o que o traffic66 encontrou nos fluxos, as mais graves
primeiro. Ele verifica os últimos 10 minutos a cada 5 minutos; algo que
dura uma hora é uma única detecção que cresce, não uma nova a cada
verificação.

| Detecção | O que significa | Gravidade |
|---|---|---|
| Varredura | Um endereço enviou pequenas sondas a muitos endereços em uma mesma porta (TCP ou ping) | Alta de dentro da sua rede, baixa da internet |
| Varredura de portas | Um endereço enviou pequenas sondas a muitas portas de um mesmo host | Alta de dentro, baixa da internet |
| Tentativa de senhas | Muitas conexões curtas a um serviço de login (SSH, RDP, SMB, bancos de dados e outros) | Alta de dentro, baixa da internet |
| Movimento lateral | Dentro da sua rede, sessões de compartilhamento de arquivos ou de administração remota (SMB, RDP, SSH, WinRM, VNC) para hosts que nunca ofereceram esse serviço antes | Alta |
| Envio incomum | Um host interno enviou muito mais do que recebeu (100 MB em 10 minutos, três vezes o que recebeu) para um endereço com o qual não havia trocado dados antes | Alta |
| Inundação | 20.000 ou mais pacotes pequenos por segundo para um endereço, dez vezes a taxa habitual dele | Média |
| Lista de ameaças | Tráfego com um endereço de uma das suas listas de ameaças | Alta quando o seu host se conectou a ele, baixa quando o endereço listado bateu de fora |

Cada detecção diz quem fez o quê a quem, quando e por quanto tempo, com os
números por trás e como os dados foram amostrados. **Detalhes** abre a
página do host, que também lista as detecções sobre ele. **Resolvido**
fecha uma detecção; se acontecer de novo, uma nova é aberta.
**Não é problema** a fecha de vez: ela nunca mais é relatada. O número
vermelho ao lado de **Detecções** no menu lateral conta as detecções
abertas de gravidade alta e média das últimas 24 horas.

Movimento lateral e envios incomuns precisam saber o que é normal, então
são relatados quando já houver um dia de histórico. Na primeira
inicialização, o traffic66 aprende com o histórico que já tem.

Com dados amostrados (sFlow, NetFlow amostrado), as regras contam o que as
amostras mostram e pedem menos delas, mas então cada uma precisa parecer
uma sonda curta, de modo que hosts normais movimentados não as disparam. O
que a amostragem esconde não pode ser encontrado: com amostragem 1:4096,
uma varredura de algumas dezenas de hosts envia pacotes de menos para ser
vista. O ataque da demo passa por um switch que amostra a 1:4096 e é
encontrado por completo; um dia de tráfego normal da demo não produz
nenhuma detecção, exceto o scanner da internet batendo no site.

![Detecções: cada etapa de um ataque, encontrada através de amostragem sFlow 1:4096](images/findings.png)

![Top 66: as 66 maiores conversas, ordenáveis por qualquer coluna](images/topn.png)

![Detalhes do tráfego: servidores com seus clientes, e serviços com seus servidores, em gráficos de anéis](images/traffic.png)

![Detalhes de um host: as detecções sobre ele, o tráfego, com quem fala, serviços, países e fluxos mais recentes](images/detail.png)

![Caminhos do tráfego: qual host usa qual aplicação para qual país](images/paths.png)

A mesma visão geral em chinês; todas as páginas estão disponíveis em 13 idiomas:

![Visão geral em chinês](images/overview-zh.png)

<a id="10-terminal-ui"></a>

### Análise offline de pcap

**Análise offline de pcap** mostra capturas do Wireshark ou tcpdump com as mesmas páginas dos dados ao vivo, sem misturá-las.

Ele resume todos os pacotes em fluxos: quem falou com quem, quanto, quando e o que parece um ataque. Não decodifica protocolos nem mostra o conteúdo dos pacotes; para um pacote ou um fluxo TCP, use o Wireshark.

Pela linha de comando, sem configurar nada:

```
traffic66 office.pcap
traffic66 a.pcap b.pcapng c.pcap
```

O traffic66 inicia só neste computador (127.0.0.1, uma porta livre), mostra o endereço, a senha e um link de acesso de uso único, e abre o navegador na captura. Até 3 arquivos, 3 GB no total; eles são lidos onde estão e nunca alterados. Nada é coletado nem enviado, e nomes de host não são consultados (`-dns` liga isso). Ctrl+C para e apaga os dados importados. Numa máquina de 2 núcleos, uma captura de 1 GB fica pronta em cerca de 5 segundos (1,2 milhão de pacotes grandes) a 30 segundos (14 milhões de pacotes pequenos).

```
$ traffic66 office.pcap

traffic66 0.3.1: analysing 1 capture file(s); nothing is collected or sent
  Web UI    http://127.0.0.1:38217  (port 38217, this computer only)
  Sign in   user admin, password gfhfhbuutz2e
  Open      http://127.0.0.1:38217/auto?t=b9388f…  (signs in once)
  Stop      Ctrl+C; the imported data is deleted, your files are kept
```

Na interface web de um traffic66 em execução:

1. **Enviar arquivos de captura…**: `.pcap` ou `.pcapng`, sem compressão. Até 3 arquivos, cada um com no máximo 50 MB. Os arquivos viram fluxos num banco próprio (`<data>/sandbox/`); os dados ao vivo, seus números e detecções não são afetados.
2. **Analisar**: todas as páginas (visão geral, Top 66, detalhes do tráfego, detecções, caminhos, mapa, registros de fluxo) mostram os arquivos durante todo o seu período. Uma barra laranja nomeia os arquivos; **Voltar aos dados ao vivo** volta. Cada arquivo aparece como um dispositivo, então o campo **Equipamento** mostra um arquivo por vez.
3. As regras de detecção rodam sobre a captura: varreduras, varreduras de portas e tentativas de senha aparecem em **Detecções**. Regras que precisam de um dia de histórico (movimento lateral, envios incomuns) não se aplicam a uma captura.
4. **Excluir** apaga um arquivo e seus dados; **Excluir tudo** apaga tudo.

A demo inclui uma captura de exemplo com um ataque.

![Análise offline: arquivos de captura com pacotes, fluxos e período](images/sandbox.png)

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

![Interface de terminal: Top 66 de conversas](images/tui-topn.png)

<a id="11-local-capture"></a>

## 11. Captura local

Além de receber exportações de fluxo, o traffic66 pode gerar fluxos por conta
própria a partir dos pacotes de uma interface de rede da máquina em que roda.
O que ele vê depende da interface:

| Interface | O que o traffic66 vê |
|---|---|
| Uma porta de rede livre ligada à porta espelho (SPAN) de um switch | Todo o tráfego que o switch espelha: uma rede inteira ou um uplink |
| A Ethernet ou o Wi-Fi da própria máquina | Só o tráfego desta máquina |

Adaptadores Wi-Fi não veem o tráfego de outros equipamentos. Para ver uma
rede Wi-Fi inteira, faça o roteador ou o ponto de acesso exportar fluxos
(seção 4), ou espelhe a porta do switch à qual o ponto de acesso está ligado.

<a id="windows-1"></a>

### Windows

1. Instale o [Npcap](https://npcap.com) com as opções padrão. Se marcar
   "Restrict Npcap driver's access to Administrators only", execute o
   traffic66 como administrador.
2. Liste as interfaces (PowerShell):

   ```
   C:\traffic66\traffic66.exe interfaces
   ```

   ```
   #   Name      Address          Adapter / device
   1   Ethernet  -                Intel(R) Ethernet I219-V  \Device\NPF_{4B8A2C1E-…}
   2   Wi-Fi     192.168.1.23     Intel(R) Wi-Fi 6 AX201  \Device\NPF_{9F00AA11-…}
   3   Loopback  -                Adapter for loopback traffic capture  \Device\NPF_Loopback
   ```

   A coluna Name é o nome da conexão nas configurações de rede do Windows; a
   interface em uso tem um endereço.
3. Capture no Wi-Fi, pelo nome ou pelo número:

   ```
   C:\traffic66\traffic66.exe -capture Wi-Fi
   C:\traffic66\traffic66.exe -capture 2
   ```

   Coloque nomes com espaços entre aspas: `-capture "Ethernet 2"`. Repita
   `-capture` para capturar em várias interfaces. Adicione `-listen=` se
   quiser só a captura, sem coletores de fluxo. Para a tarefa de
   inicialização da seção 2, adicione a opção ao `-Argument`:
   `-Argument '-data C:\traffic66\traffic66-data -capture Wi-Fi'`.

<a id="linux-1"></a>

### Linux

```
traffic66 interfaces
sudo setcap cap_net_raw,cap_net_admin+ep /opt/traffic66/traffic66
traffic66 -capture eth1
```

A captura precisa de root ou das capabilities `CAP_NET_RAW` e
`CAP_NET_ADMIN`: a linha `setcap` acima, ou a linha `AmbientCapabilities` da
unit systemd da seção 2. Interfaces Wi-Fi costumam se chamar `wlan0` ou
`wlp…`.

<a id="macos-1"></a>

### macOS

```
traffic66 interfaces
sudo traffic66 -capture en0
```

A captura precisa de root; nada a instalar. Nos MacBooks, `en0` é o Wi-Fi.

<a id="checking-that-it-works"></a>

### Verificando se funciona

**Configurações** lista cada interface capturada com o método de captura e o número
de pacotes vistos. Os fluxos aparecem como vindos do equipamento `127.0.0.1`
(esta máquina), em todas as páginas, como os de qualquer outro equipamento.
Pacotes vistos duas vezes (por exemplo, em duas portas espelho) são contados
duas vezes.

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
| `-asn` | `<data>/asn.tsv.gz` | tabela IP-para-ASN (arquivos `.mmdb`: envie-os, ou `<data>/country.mmdb` e `<data>/asn.mmdb`, `<data>/both.mmdb`) |
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
| `inventory.txt` | nomes (**Configurações → Nomes**) |
| `license.json` | número de instalação e licença (veja [Teste e licença](#trial-and-licence)) |
| `logo.png` (ou `.svg`, `.jpg`, `.webp`, `.gif`) | seu logotipo (**Configurações → Logotipo**), se você enviou um |
| `country.mmdb`, `asn.mmdb`, `both.mmdb`, `asn.tsv.gz`, `dbip-country.mmdb`, `dbip-asn.mmdb`, `threats/`, `sandbox/` | bancos de dados de países e redes e listas de ameaças que você adicionou |

**Por quanto tempo os dados ficam**: o detalhe de fluxos 30 dias; os resumos (visão geral e períodos longos)
400 dias. O que for mais antigo é apagado automaticamente, com verificação a cada 5 minutos; nada mais é apagado e
não há outro limite. Altere o período do detalhe com `-retention-days`, qualquer número de dias, por exemplo
`-retention-days 365`. O uso de disco cresce junto: **Livre** no menu lateral fica vermelho quando os dias guardados
não cabem. Se o disco encher, novos fluxos não podem ser gravados até liberar espaço.

**Limpeza de dados** no menu lateral apaga dados antes que seja preciso:
os com mais de 120, 90, 60, 30 ou 7 dias, ou todos. Para cada opção mostra
quantos registros de fluxo saem e quanto disco, aproximadamente, é
liberado, e pede confirmação antes de apagar. São apagados os registros de
fluxo, os resumos por hora e por dia, os contadores de interface e as
detecções; apagar todos os dados também zera o que as regras de detecção
aprenderam. Não dá para desfazer.

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

<a id="trial-and-licence"></a>

### Teste e licença

O traffic66 tem código-fonte disponível sob a
[PolyForm Noncommercial License 1.0.0](../LICENSE.md) e a
[Traffic66 Additional Use Grant](../ADDITIONAL-USE-GRANT.md); o texto em
inglês de ambas é o que vale. Em resumo:

- **Avaliação**, testes, desenvolvimento e demonstrações: gratuito para
  qualquer pessoa, sem limite de tempo.
- **Uso em produção** (tráfego real, para as operações de uma organização)
  por uma organização com menos de 100 funcionários e terceirizados:
  gratuito.
- Uso em produção por **organizações maiores**: gratuito por 30 dias;
  depois é preciso uma licença de registro do autor.
- Terceirizados e prestadores de serviços podem executá-lo para um
  cliente, em uma implantação do próprio cliente; vale o tamanho do
  cliente.
- Não permitido sem licença comercial: vendê-lo ou incorporá-lo a um
  produto, oferecê-lo a terceiros como serviço hospedado ou multilocatário,
  ou um produto concorrente.

Preço, escopo e prazo de uma licença de registro são definidos caso a
caso, e ela pode ser gratuita. Contato:
<https://github.com/githubflyideas/traffic66>.

Toda instalação mostra o teste, também onde não é preciso licença. Na
primeira inicialização o traffic66 grava `license.json` no diretório de
dados com um número de instalação de 8 dígitos. O rodapé de cada página mostra quantos dias de teste faltam e,
depois, que o teste terminou. Nada é desligado em nenhum dos casos: todos
os recursos continuam funcionando.

Para registrar, envie ao autor o número de instalação (mostrado também no
rodapé de cada página). A licença volta como um novo `license.json`;
coloque-o no diretório de dados no lugar do antigo. Ela é verificada
quando o traffic66 inicia e a cada 4 horas, então não é preciso
reiniciar; o rodapé da página passa a mostrar para quem está licenciado e
quantos dias faltam.

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
conforme a sua taxa de fluxos (mostrada em **Configurações**) e `-retention-days`.

Memória: `-memory` (padrão 10% da RAM, pelo menos 256 MB) limita o cache do
banco de dados, e o resto do programa recebe um limite flexível do mesmo
tamanho. A 5.000 fluxos por segundo, os dados do próprio programa
(decodificação, detecção de duplicatas, lotes) ocupam cerca de 90 MB; no
total, conte com 0,6–0,8 GB, então uma máquina com 2 GB de RAM basta.
Medido em 10 minutos de coleta contínua (pico de 0,58 GB em uma máquina de
8 GB) e ao carregar uma hora de fluxos a onze vezes essa taxa com os limites
de uma máquina de 2 GB (pico de 0,74 GB).

`-memory` é um orçamento, não um teto rígido: o limite do Go é flexível e o
banco de dados pode ultrapassar brevemente sua parte. Para um teto rígido,
use o do sistema operacional: `MemoryMax=` na unit systemd (seção 2) ou o
limite de memória de um contêiner. Reserve cerca de 2,5 vezes a parte do
`-memory` e pelo menos 1 GB; `MemoryMax=2G` serve para máquinas com até
8 GB com a parte padrão. Assim o traffic66 reinicia em vez de a máquina
ficar sem memória.

<a id="16-troubleshooting"></a>

## 16. Solução de problemas

| Sintoma | Causa e solução |
|---|---|
| Equipamento não aparece em **Configurações** | Os pacotes não chegam: veja [Verificar se os fluxos estão chegando](#5-check-that-flows-arrive) |
| "waiting for the sampling rate" | O equipamento ainda não enviou as opções do sampler; a maioria reenvia em poucos minutos. Se nunca enviar, exporte-as (`option sampler-table` no Cisco) ou marque-o como `unsampled` em Nomes se ele for de fato 1:1, ou informe a taxa com `sampling=N` na linha `device` dele. **Configurações** lista então os templates que o equipamento enviou, para ver o que ele declara |
| Números abaixo dos contadores de interface | Veja **Conferência de interfaces**: perdas no caminho, interfaces não amostradas ou fluxos ainda no cache do equipamento (timeout ativo acima de 60 s) |
| Números acima dos contadores de interface | O mesmo tráfego amostrado em duas interfaces ou dois equipamentos |
| Sem países nem redes ("Desconhecido") | Nenhum banco de dados carregado: envie um em **Configurações**; veja [Países](#8-countries-networks-and-threat-lists) |
| "O banco de dados atingiu o limite de memória e não conseguiu responder" em uma página | Escolha um período menor ou inicie com um `-memory` maior; os detalhes estão no log |
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
