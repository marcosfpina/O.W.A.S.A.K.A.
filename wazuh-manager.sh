#!/bin/bash
# ============================================================
# Wazuh SIEM Manager Script
# Quick management commands for Wazuh stack
# ============================================================

set -e

WAZUH_DIR="/home/kernelcore/Documents/nx/docker/sql & nvidia/wazuh-siem"
COMPOSE_FILE="$WAZUH_DIR/docker-compose.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================
# HELPER FUNCTIONS
# ============================================================

print_header() {
    echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       Wazuh SIEM Manager v1.0              ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# ============================================================
# MAIN COMMANDS
# ============================================================

start() {
    print_info "Starting Wazuh SIEM stack..."
    cd "$WAZUH_DIR"
    docker compose up -d

    echo ""
    print_success "Wazuh stack started!"
    print_info "Waiting for services to be ready (this may take 60-90 seconds)..."

    sleep 30
    status

    echo ""
    print_info "Dashboard: http://localhost:5601"
    print_info "Username: admin"
    print_info "Password: SecretPassword"
}

stop() {
    print_info "Stopping Wazuh SIEM stack..."
    cd "$WAZUH_DIR"
    docker compose down
    print_success "Wazuh stack stopped!"
}

restart() {
    print_info "Restarting Wazuh SIEM stack..."
    stop
    sleep 3
    start
}

status() {
    print_header
    echo ""
    print_info "Container Status:"
    docker compose -f "$COMPOSE_FILE" ps

    echo ""
    print_info "Health Checks:"

    # Check Indexer
    if curl -s http://localhost:9200 > /dev/null 2>&1; then
        print_success "Indexer (OpenSearch) is UP"
    else
        print_error "Indexer is DOWN"
    fi

    # Check Manager API
    if curl -s http://localhost:55000 > /dev/null 2>&1; then
        print_success "Manager API is UP"
    else
        print_error "Manager API is DOWN"
    fi

    # Check Dashboard
    if curl -s http://localhost:5601/status > /dev/null 2>&1; then
        print_success "Dashboard is UP"
    else
        print_error "Dashboard is DOWN or still starting"
    fi
}

logs() {
    local service=${1:-wazuh-manager}
    print_info "Following logs for $service (Ctrl+C to exit)..."
    cd "$WAZUH_DIR"
    docker compose logs -f "$service"
}

agents() {
    print_info "Connected Wazuh Agents:"
    docker exec wazuh-manager /var/ossec/bin/agent_control -l 2>/dev/null || print_error "Manager not running or no agents connected"
}

alerts() {
    print_info "Real-time alerts (Ctrl+C to exit)..."
    docker exec -it wazuh-manager tail -f /var/ossec/logs/alerts/alerts.log
}

stats() {
    print_info "Wazuh Stack Resource Usage:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" wazuh-manager wazuh-indexer wazuh-dashboard
}

shell() {
    local service=${1:-wazuh-manager}
    print_info "Opening shell in $service..."
    docker exec -it "$service" /bin/bash
}

install_agent() {
    local container_name=$1

    if [ -z "$container_name" ]; then
        print_error "Usage: $0 install-agent <container_name>"
        return 1
    fi

    print_info "Installing Wazuh agent in container: $container_name"

    # Detect OS in container
    if docker exec "$container_name" which apt-get > /dev/null 2>&1; then
        print_info "Detected Debian/Ubuntu..."
        docker exec "$container_name" bash -c "
            curl -s https://packages.wazuh.com/key/GPG-KEY-WAZUH | apt-key add -
            echo 'deb https://packages.wazuh.com/4.x/apt/ stable main' > /etc/apt/sources.list.d/wazuh.list
            apt-get update
            DEBIAN_FRONTEND=noninteractive apt-get install -y wazuh-agent
            echo 'WAZUH_MANAGER=\"172.25.0.2\"' > /var/ossec/etc/ossec.conf.d/manager.conf
            /var/ossec/bin/wazuh-control start
        "
    elif docker exec "$container_name" which yum > /dev/null 2>&1; then
        print_info "Detected RHEL/CentOS..."
        docker exec "$container_name" bash -c "
            rpm --import https://packages.wazuh.com/key/GPG-KEY-WAZUH
            cat > /etc/yum.repos.d/wazuh.repo << EOF
[wazuh]
gpgcheck=1
gpgkey=https://packages.wazuh.com/key/GPG-KEY-WAZUH
enabled=1
name=EL-\\\$releasever - Wazuh
baseurl=https://packages.wazuh.com/4.x/yum/
protect=1
EOF
            yum install -y wazuh-agent
            echo 'WAZUH_MANAGER=\"172.25.0.2\"' > /var/ossec/etc/ossec.conf.d/manager.conf
            /var/ossec/bin/wazuh-control start
        "
    else
        print_error "Unsupported OS in container"
        return 1
    fi

    print_success "Agent installed in $container_name"
    print_info "Check agent connection with: $0 agents"
}

backup() {
    local backup_dir="$HOME/wazuh-backups/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"

    print_info "Creating backup in $backup_dir..."

    docker cp wazuh-manager:/var/ossec/etc "$backup_dir/manager-config"
    docker cp wazuh-manager:/var/ossec/rules "$backup_dir/rules"

    print_success "Backup created: $backup_dir"
}

update() {
    print_info "Updating Wazuh images..."
    cd "$WAZUH_DIR"
    docker compose pull
    print_success "Images updated. Run '$0 restart' to apply changes"
}

clean() {
    print_warning "This will remove all Wazuh data and volumes!"
    read -p "Are you sure? (yes/no): " confirm

    if [ "$confirm" = "yes" ]; then
        cd "$WAZUH_DIR"
        docker compose down -v
        print_success "Wazuh stack and volumes removed"
    else
        print_info "Cancelled"
    fi
}

help_menu() {
    print_header
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start              - Start Wazuh stack"
    echo "  stop               - Stop Wazuh stack"
    echo "  restart            - Restart Wazuh stack"
    echo "  status             - Show status of all services"
    echo "  logs [service]     - Follow logs (default: wazuh-manager)"
    echo "  agents             - List connected agents"
    echo "  alerts             - Stream real-time alerts"
    echo "  stats              - Show resource usage"
    echo "  shell [service]    - Open shell in container"
    echo "  install-agent <container> - Install agent in container"
    echo "  backup             - Backup configurations"
    echo "  update             - Update Wazuh images"
    echo "  clean              - Remove stack and volumes (destructive)"
    echo "  help               - Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 start"
    echo "  $0 logs wazuh-dashboard"
    echo "  $0 install-agent nginx-proxy"
    echo "  $0 agents"
    echo ""
}

# ============================================================
# MAIN
# ============================================================

case "${1:-help}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs "$2"
        ;;
    agents)
        agents
        ;;
    alerts)
        alerts
        ;;
    stats)
        stats
        ;;
    shell)
        shell "$2"
        ;;
    install-agent)
        install_agent "$2"
        ;;
    backup)
        backup
        ;;
    update)
        update
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        help_menu
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        help_menu
        exit 1
        ;;
esac
