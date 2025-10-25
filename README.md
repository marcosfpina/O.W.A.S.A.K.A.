# Wazuh SIEM - Network Security Monitoring

## 🔐 Arquitetura

```
Containers → Wazuh Agent → Wazuh Manager → Wazuh Indexer → Wazuh Dashboard
```

## 🚀 Quick Start

### 1. Iniciar o Wazuh Stack

```bash
cd "/home/kernelcore/Documents/nx/docker/sql & nvidia/wazuh-siem"
docker compose up -d
```

### 2. Verificar Status

```bash
docker compose ps
```

### 3. Acessar Dashboard

- URL: http://localhost:5601
- Username: `admin`
- Password: `SecretPassword`

## 📊 Componentes

### Wazuh Manager (Port 1514, 1515, 55000)
- Processa logs e eventos
- Executa rules engine
- Detecta vulnerabilidades
- File Integrity Monitoring (FIM)

### Wazuh Indexer (Port 9200)
- OpenSearch para armazenamento
- Indexação de logs
- Query engine

### Wazuh Dashboard (Port 5601)
- Interface Web
- Visualizações e dashboards
- Alertas e relatórios
- Compliance reports

## 🔧 Configuração de Agentes

### Instalar Agente em Container Existente

```bash
# Entre no container
docker exec -it <container_name> bash

# Instale o agente (Debian/Ubuntu)
curl -s https://packages.wazuh.com/key/GPG-KEY-WAZUH | apt-key add -
echo "deb https://packages.wazuh.com/4.x/apt/ stable main" | tee /etc/apt/sources.list.d/wazuh.list
apt-get update
apt-get install wazuh-agent

# Configure o manager
echo "WAZUH_MANAGER='172.25.0.2'" > /var/ossec/etc/ossec.conf.d/manager.conf

# Inicie o agente
systemctl enable wazuh-agent
systemctl start wazuh-agent
```

### Monitorar Docker Socket (Já configurado)

O Wazuh Manager já está configurado para monitorar:
- Docker events via socket
- Container lifecycle
- Resource usage
- Network activity

## 📝 Custom Rules

As regras customizadas estão em: `./custom-rules/docker-rules.xml`

Incluem detecção de:
- ✅ Container crashes e restarts
- ✅ Alto uso de CPU/Memory
- ✅ Tentativas de autenticação falhadas
- ✅ Brute force attacks
- ✅ SQL injection attempts
- ✅ Network anomalies
- ✅ Privilege escalation
- ✅ File integrity violations
- ✅ Web server errors (4xx/5xx)
- ✅ GPU alerts

## 🎯 Monitoramento Ativo

### Verificar Agentes Conectados

```bash
docker exec -it wazuh-manager /var/ossec/bin/agent_control -l
```

### Ver Alertas em Tempo Real

```bash
docker exec -it wazuh-manager tail -f /var/ossec/logs/alerts/alerts.log
```

### API do Wazuh

```bash
# Listar agentes
curl -k -X GET "http://localhost:55000/agents" \
  -H "Authorization: Bearer $(curl -u wazuh-wui:MyS3cr37P450r.*- -k -X POST 'http://localhost:55000/security/user/authenticate' | jq -r .data.token)"
```

## 🔍 Casos de Uso

### 1. Detectar Brute Force
- Monitora logs de autenticação
- Alerta após 5 tentativas em 2 minutos

### 2. Monitorar Containers Unhealthy
- Detecta containers em restart loop
- Alerta sobre crashes

### 3. Análise de Vulnerabilidades
- Scanneia packages instalados
- Alerta sobre CVEs conhecidas

### 4. Compliance
- PCI-DSS
- GDPR
- HIPAA
- CIS benchmarks

## 🛠️ Comandos Úteis

```bash
# Parar stack
docker compose down

# Ver logs
docker compose logs -f wazuh-manager

# Reiniciar componente
docker compose restart wazuh-dashboard

# Backup de configuração
docker cp wazuh-manager:/var/ossec/etc ./backup-config

# Ver regras ativas
docker exec wazuh-manager /var/ossec/bin/ossec-logtest
```

## 🔐 Credenciais Padrão

**Wazuh Dashboard:**
- User: `admin`
- Pass: `SecretPassword`

**Wazuh API:**
- User: `wazuh-wui`
- Pass: `MyS3cr37P450r.*-`

⚠️ **IMPORTANTE:** Mude as senhas em produção!

## 📈 Próximos Passos

1. ✅ Deploy do stack Wazuh
2. 🔄 Instalar agentes nos containers
3. 📊 Configurar dashboards customizados
4. 🔔 Configurar alertas (Discord/Slack/Email)
5. 📝 Criar playbooks de resposta
6. 🔍 Integrar com threat intelligence feeds

## 🐛 Troubleshooting

### Dashboard não carrega
```bash
docker compose logs wazuh-dashboard
# Verifique se indexer está UP
curl http://localhost:9200
```

### Agente não conecta
```bash
# No container do agente
tail -f /var/ossec/logs/ossec.log
# Verifique firewall e conectividade
```

### Performance lenta
```bash
# Aumente memória do indexer no docker-compose.yml
OPENSEARCH_JAVA_OPTS=-Xms1g -Xmx1g
```
