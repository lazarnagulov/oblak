#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

ok()   { echo -e "  ${GREEN}[OK]${RESET}    $1"; }
warn() { echo -e "  ${RED}[WARNING]${RESET} $1"; }
info() { echo -e "  ${YELLOW}[INFO]${RESET}  $1"; }
section() { echo -e "\n${CYAN}${BOLD}==============================${RESET}"; \
            echo -e "${CYAN}${BOLD}  $1${RESET}"; \
            echo -e "${CYAN}${BOLD}==============================${RESET}"; }

# Student 1
check_os_and_kernel() {
    section "OS & KERNEL INFORMATION"

    if [ -f /etc/os-release ]; then
        OS_NAME=$(grep -E '^PRETTY_NAME=' /etc/os-release | cut -d '"' -f 2)
        info "Operating System: $OS_NAME"
    elif [ -f /etc/debian_version ]; then
        info "Debian Version: $(cat /etc/debian_version)"
    else
        warn "Could not determine OS version."
    fi

    KERNEL_VER=$(uname -r)
    info "Kernel Version: $KERNEL_VER"

    UPTIME_RAW=$(uptime -p 2>/dev/null || uptime)
    info "System Uptime: $UPTIME_RAW"
    
    if echo "$UPTIME_RAW" | grep -qi "day"; then
        DAYS=$(uptime | awk -F'( |,|:)+' '{for(i=1;i<=NF;i++) if($i=="days" || $i=="day") print $(i-1)}')
        if [ -n "$DAYS" ] && [ "$DAYS" -gt 30 ]; then
            warn "System has been up for $DAYS days. Kernel updates usually require a reboot."
        else
            ok "Uptime is reasonable ($DAYS days)."
        fi
    fi
}

check_ntp() {
    # Network time protocol
    section "NTP & TIME SYNCHRONIZATION"

    if [ -f /etc/timezone ]; then
        TZ=$(cat /etc/timezone)
        info "Timezone configured: $TZ"
        if [ "$TZ" = "Etc/UTC" ] || [ "$TZ" = "UTC" ]; then
            ok "Timezone is set to UTC."
        else
            warn "Timezone is not UTC. Daylight savings time jumps can affect log correlation."
        fi
    fi

    if ps -edf | grep -E 'ntpd|chronyd|systemd-timesyncd' | grep -v grep > /dev/null; then
        ok "An NTP daemon (ntpd, chronyd, or timesyncd) is running."
    else
        warn "No NTP daemon appears to be running. Time synchronization is crucial for logs and SSL."
    fi
}

check_users_and_groups() {
    section "USERS & PRIVILEGES (/etc/passwd)"

    info "Checking for users with UID 0 (root privileges):"
    UID_ZERO=$(awk -F: '($3 == "0") {print $1}' /etc/passwd)
    for user in $UID_ZERO; do
        if [ "$user" != "root" ] && [ "$user" != "toor" ]; then
            warn "  User '$user' has UID 0. This is a major security risk/backdoor."
        else
            ok "  User '$user' has UID 0 (Standard)."
        fi
    done

    info "Users with an active shell access:"
    awk -F: '$7 !~ /(nologin|false|sync)$/ {print "    " $1 " -> " $7}' /etc/passwd
}

check_shadow_passwords() {
    section "PASSWORD HASHING ALGORITHMS (/etc/shadow)"

    if [ ! -r /etc/shadow ]; then
        warn "Cannot read /etc/shadow. You must run this script as root to check password hashes."
        return
    fi

    EMPTY_PASS=$(awk -F: '($2 == "") {print $1}' /etc/shadow)
    if [ -n "$EMPTY_PASS" ]; then
        warn "Users with NO PASSWORD set: $EMPTY_PASS"
    else
        ok "No accounts have empty passwords."
    fi

    if grep -E '^[^:]+:\$1\$' /etc/shadow >/dev/null; then
        warn "MD5 hashing (\$1\$) is used by some accounts. It is easily brute-forced."
    else
        ok "No MD5 hashes found."
    fi

    if grep -E '^[^:]+:[^\$*!]' /etc/shadow | grep -v ':\*:' >/dev/null; then
        warn "Some accounts might be using outdated DES encryption."
    fi

    if grep -E '^[^:]+:\$6\$' /etc/shadow >/dev/null; then
        ok "SHA-512 (\$6\$) hashing is in use."
    elif grep -E '^[^:]+:\$y\$' /etc/shadow >/dev/null; then
        ok "yescrypt (\$y\$) hashing is in use (modern default)."
    fi
}

check_packages() {
    section "INSTALLED PACKAGES"
    
    if command -v dpkg &>/dev/null; then
        PKG_COUNT=$(dpkg -l | grep '^ii' | wc -l)
        info "Total Debian/Ubuntu packages installed: $PKG_COUNT"
    elif command -v rpm &>/dev/null; then
        PKG_COUNT=$(rpm -qa | wc -l)
        info "Total RHEL/CentOS packages installed: $PKG_COUNT"
    else
        warn "Package manager not found (not dpkg or rpm)."
    fi
    info "Recommendation: Keep package count to a minimum. Remove X11, games, and unused tools."
}

# Student 2
check_open_ports() {
    section "OPEN PORTS AND SERVICES"
 
    info "Listening TCP ports:"
    ss -tlnp 2>/dev/null | tail -n +2 | while read -r line; do
        ADDR=$(echo "$line" | awk '{print $4}')
        if echo "$ADDR" | grep -qE '0\.0\.0\.0|::'; then
            warn "  $line  <- accessible from all interfaces!"
        else
            ok "  $line  <- restricted to local interface"
        fi
    done
 
    info "Listening UDP ports:"
    ss -ulnp 2>/dev/null | tail -n +2 | awk '{print "    " $0}'
}

check_firewall() {
    section "FIREWALL RULES"
 
    if ! command -v iptables &>/dev/null; then
        warn "iptables is not available on this system."
        return
    fi
 
    info "IPv4 INPUT chain policy:"
    IPV4_POLICY=$(iptables -L INPUT 2>/dev/null | head -1)
    echo "    $IPV4_POLICY"
    if echo "$IPV4_POLICY" | grep -q "policy ACCEPT"; then
        warn "INPUT chain has ACCEPT policy - all connections are allowed by default!"
    else
        ok "INPUT chain has a restrictive policy (DROP/REJECT)."
    fi
 
    if iptables -L INPUT -n 2>/dev/null | grep -q "dpt:22"; then
        warn "SSH (port 22) is open without an IP whitelist - consider restricting access."
    fi
 
    info "IPv6 firewall check:"
    IPV6_INPUT=$(ip6tables -L INPUT 2>/dev/null | head -1)
    if echo "$IPV6_INPUT" | grep -q "policy ACCEPT"; then
        warn "IPv6 INPUT has no restrictive rules - risk of unfiltered IPv6 traffic!"
    else
        ok "IPv6 INPUT chain has a restrictive policy."
    fi
 
    if [ -f /etc/iptables.up.rules ] || [ -f /etc/iptables/rules.v4 ]; then
        ok "Persistent iptables rules file found."
    else
        warn "Persistent iptables rules file not found - rules may be lost after reboot!"
    fi
}

check_network_interfaces() {
    section "NETWORK INTERFACES"
 
    if command -v ip &>/dev/null; then
        info "List of network interfaces (ip addr):"
        ip addr 2>/dev/null | awk '{print "    " $0}'
 
        info "Routing table:"
        ip route 2>/dev/null | awk '{print "    " $0}'
    elif command -v ifconfig &>/dev/null; then
        info "List of network interfaces (ifconfig):"
        ifconfig -a 2>/dev/null | awk '{print "    " $0}'
    else
        warn "Neither ip nor ifconfig is available."
    fi
 
    UNEXPECTED=$(ip link 2>/dev/null | grep -E 'tun|tap|vpn|wg' | awk -F: '{print $2}')
    if [ -n "$UNEXPECTED" ]; then
        warn "Potentially unexpected interfaces found (tunnel/VPN):"
        echo "$UNEXPECTED" | while read -r iface; do
            warn "  -> $iface"
        done
    else
        ok "No tunnel/VPN interfaces found."
    fi
}

check_ssh_config() {
    section "SSH CONFIGURATION"
 
    SSH_CFG="/etc/ssh/sshd_config"
 
    if [ ! -f "$SSH_CFG" ]; then
        warn "SSH configuration file not found ($SSH_CFG)."
        return
    fi
 
    ROOT_LOGIN=$(grep -i "^PermitRootLogin" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$ROOT_LOGIN" ]; then
        warn "PermitRootLogin is not explicitly set - default is 'yes' on older systems!"
    elif echo "$ROOT_LOGIN" | grep -qi "^no$"; then
        ok "PermitRootLogin is disabled."
    else
        warn "PermitRootLogin = '$ROOT_LOGIN' - direct root login is allowed!"
    fi
 
    PROTOCOL=$(grep -i "^Protocol" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$PROTOCOL" ]; then
        info "Protocol is not explicitly set (modern OpenSSH uses v2 by default)."
    elif [ "$PROTOCOL" = "2" ]; then
        ok "SSH uses only Protocol 2."
    else
        warn "SSH Protocol = '$PROTOCOL' - SSH v1 is insecure and should be disabled!"
    fi
 
    TCP_FWD=$(grep -i "^AllowTcpForwarding" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$TCP_FWD" ] || echo "$TCP_FWD" | grep -qi "yes"; then
        warn "AllowTcpForwarding is enabled - users can tunnel traffic through the server!"
    else
        ok "AllowTcpForwarding is disabled."
    fi
 
    SSH_PORT=$(grep -i "^Port" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$SSH_PORT" ] || [ "$SSH_PORT" = "22" ]; then
        warn "SSH uses the standard port 22 - consider changing the port to reduce brute-force attempts."
    else
        ok "SSH uses a non-standard port: $SSH_PORT"
    fi
 
    PASS_AUTH=$(grep -i "^PasswordAuthentication" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$PASS_AUTH" ] || echo "$PASS_AUTH" | grep -qi "yes"; then
        warn "PasswordAuthentication is enabled - consider using SSH keys only."
    else
        ok "PasswordAuthentication is disabled (SSH keys are used)."
    fi
 
    MAX_TRIES=$(grep -i "^MaxAuthTries" "$SSH_CFG" | awk '{print $2}')
    if [ -z "$MAX_TRIES" ]; then
        warn "MaxAuthTries is not set - default is 6, consider a lower value (e.g. 3)."
    elif [ "$MAX_TRIES" -le 3 ]; then
        ok "MaxAuthTries = $MAX_TRIES (good value)."
    else
        warn "MaxAuthTries = $MAX_TRIES - consider reducing it to 3 or less."
    fi
}

check_dns_config() {
    section "DNS CONFIGURATION"
 
    if [ -f /etc/resolv.conf ]; then
        info "DNS servers (/etc/resolv.conf):"
        grep "^nameserver" /etc/resolv.conf | while read -r line; do
            DNS_IP=$(echo "$line" | awk '{print $2}')
            if echo "$DNS_IP" | grep -qE '^127\.|^192\.168\.|^10\.|^172\.(1[6-9]|2[0-9]|3[01])\.'; then
                ok "  $DNS_IP (local/private DNS)"
            else
                warn "  $DNS_IP - external DNS server, verify whether it is trustworthy!"
            fi
        done
    else
        warn "/etc/resolv.conf does not exist."
    fi
 
    if [ -f /etc/hosts ]; then
        info "Non-standard entries in /etc/hosts:"
        SUSPICIOUS=$(grep -v '^#\|^$\|^127\.\|^::1\|^0\.0\.0\.0' /etc/hosts)
        if [ -n "$SUSPICIOUS" ]; then
            warn "Non-standard entries found - manually review:"
            echo "$SUSPICIOUS" | while read -r line; do
                warn "  $line"
            done
        else
            ok "/etc/hosts appears standard."
        fi
    fi
 
    if [ -f /etc/nsswitch.conf ]; then
        info "nsswitch.conf configuration (passwd, group, hosts):"
        grep -E '^(passwd|group|hosts):' /etc/nsswitch.conf | awk '{print "    " $0}'
        if grep -qE 'ldap|nis|compat' /etc/nsswitch.conf; then
            warn "LDAP/NIS is in use - verify user and group configuration."
        fi
    fi
}

# Student 3
check_setuid_files() {
    section "SETUID FILES"
 
    info "Searching for files with the setuid bit set (find / -perm -4000):"
    SETUID_FILES=$(find / -perm -4000 -ls 2>/dev/null | grep -v '/proc')
 
    if [ -z "$SETUID_FILES" ]; then
        ok "No setuid files found."
        return
    fi
 
    SETUID_COUNT=$(echo "$SETUID_FILES" | wc -l)
    info "Found $SETUID_COUNT setuid file(s). Review each entry:"
    echo "$SETUID_FILES" | while read -r line; do
        OWNER=$(echo "$line" | awk '{print $5}')
        FILEPATH=$(echo "$line" | awk '{print $NF}')
        if [ "$OWNER" = "root" ]; then
            warn "  [root-owned setuid] $FILEPATH"
        else
            info "  $FILEPATH (owner: $OWNER)"
        fi
    done
    info "Verify that each setuid binary is legitimate and necessary."
}
 
check_world_writable_files() {
    section "WORLD-WRITABLE FILES"
 
    info "Searching for files writable by any user (find / -type f -perm -002):"
    WW_FILES=$(find / -type f -perm -002 2>/dev/null | grep -v '/proc' | grep -v '/sys')
 
    if [ -z "$WW_FILES" ]; then
        ok "No world-writable files found."
        return
    fi
 
    WW_COUNT=$(echo "$WW_FILES" | wc -l)
    warn "Found $WW_COUNT world-writable file(s) - any user can modify these:"
    echo "$WW_FILES" | while read -r filepath; do
        OWNER=$(stat -c '%U' "$filepath" 2>/dev/null)
        warn "  $filepath (owner: $OWNER)"
    done
    info "World-writable files can be modified by attackers to escalate privileges or inject code."
}
 
check_backup_permissions() {
    section "BACKUP FILE PERMISSIONS"
 
    info "Checking for backup files and directories with insecure permissions:"
 
    for BKPDIR in /backup /var/backup /var/backups /root/backup /home/backup /tmp/backup; do
        if [ -d "$BKPDIR" ]; then
            BKPDIR_PERM=$(stat -c '%a' "$BKPDIR" 2>/dev/null)
            if [ "$BKPDIR_PERM" -ge 5 ] && echo "$BKPDIR_PERM" | grep -qE '[57]$'; then
                warn "Backup directory $BKPDIR is world-readable (permissions: $BKPDIR_PERM)!"
            else
                ok "Backup directory $BKPDIR permissions: $BKPDIR_PERM"
            fi
            find "$BKPDIR" -maxdepth 2 -type f 2>/dev/null | while read -r f; do
                FPERM=$(stat -c '%a' "$f" 2>/dev/null)
                if echo "$FPERM" | grep -qE '[2367]$'; then
                    warn "  World-readable/writable backup file: $f (permissions: $FPERM)"
                fi
            done
        fi
    done
 
    info "Checking for sensitive backup files scattered on the filesystem:"
    find / -maxdepth 5 -type f \( -name '*.bak' -o -name '*.backup' -o -name '*.old' -o -name 'shadow.backup' \) \
        2>/dev/null | grep -v '/proc' | grep -v '/sys' | while read -r f; do
        FPERM=$(stat -c '%a' "$f" 2>/dev/null)
        OWNER=$(stat -c '%U' "$f" 2>/dev/null)
        warn "Sensitive backup file found: $f (permissions: $FPERM, owner: $OWNER)"
    done
}
 
check_crontab() {
    section "CRONTAB & SCHEDULED TASKS"
 
    info "System-wide cron jobs (/etc/crontab):"
    if [ -f /etc/crontab ]; then
        grep -v '^#\|^$' /etc/crontab | awk '{print "    " $0}'
    else
        info "  /etc/crontab not found."
    fi
 
    info "Jobs in /etc/cron.d/:"
    if [ -d /etc/cron.d ]; then
        for f in /etc/cron.d/*; do
            [ -f "$f" ] && grep -v '^#\|^$' "$f" | awk -v file="$f" '{print "    [" file "] " $0}'
        done
    fi
 
    info "Per-user crontabs (/var/spool/cron/crontabs/):"
    if [ -d /var/spool/cron/crontabs ]; then
        for f in /var/spool/cron/crontabs/*; do
            [ -f "$f" ] || continue
            USER=$(basename "$f")
            info "  Crontab for user: $USER"
            grep -v '^#\|^$' "$f" | while read -r job; do
                echo "      $job"
                SCRIPT=$(echo "$job" | awk '{print $NF}')
                if [ -f "$SCRIPT" ]; then
                    SPERM=$(stat -c '%a' "$SCRIPT" 2>/dev/null)
                    SOWNER=$(stat -c '%U' "$SCRIPT" 2>/dev/null)
                    if echo "$SPERM" | grep -qE '[2367]$'; then
                        warn "  Script $SCRIPT is world-writable (permissions: $SPERM, owner: $SOWNER)!"
                    else
                        ok "  Script $SCRIPT permissions look fine ($SPERM)."
                    fi
                fi
            done
        done
    else
        info "  No per-user crontabs found."
    fi
}
 
check_mounted_partitions() {
    section "MOUNTED PARTITIONS (/etc/fstab)"
 
    if [ ! -f /etc/fstab ]; then
        warn "/etc/fstab not found."
        return
    fi
 
    info "Reviewing mount options for each partition:"
    grep -v '^#\|^$' /etc/fstab | while read -r line; do
        MOUNTPOINT=$(echo "$line" | awk '{print $2}')
        OPTIONS=$(echo "$line" | awk '{print $4}')
        info "  $MOUNTPOINT ($OPTIONS)"
 
        if echo "$OPTIONS" | grep -q "noatime"; then
            warn "  $MOUNTPOINT uses 'noatime' - inode access times will not be recorded, hindering intrusion analysis."
        fi
 
        if echo "$MOUNTPOINT" | grep -qE '^/tmp$|^/home$|^/var$'; then
            if ! echo "$OPTIONS" | grep -q "noexec"; then
                warn "  $MOUNTPOINT is missing 'noexec' - users may execute binaries from this partition."
            else
                ok "  $MOUNTPOINT has 'noexec'."
            fi
            if ! echo "$OPTIONS" | grep -q "nosuid"; then
                warn "  $MOUNTPOINT is missing 'nosuid' - setuid binaries can be executed from this partition."
            else
                ok "  $MOUNTPOINT has 'nosuid'."
            fi
        fi
 
        if echo "$MOUNTPOINT" | grep -q '^/dev$'; then
            if ! echo "$OPTIONS" | grep -q "nosuid"; then
                warn "  /dev is missing 'nosuid'."
            else
                ok "  /dev has 'nosuid'."
            fi
        fi
    done
}

main() {
    echo -e "\n${BOLD}Linux Hardening Check Script${RESET}"
    echo -e "Started: $(date)"
    echo -e "User: $(whoami)\n"

    if [ "$(id -u)" -ne 0 ]; then
        info "The script is not running as root."
        info "Some checks (like /etc/shadow and /etc/sudoers) will be skipped or fail.\n"
    fi

    # Student 1
    check_os_and_kernel
    check_ntp
    check_users_and_groups
    check_shadow_passwords
    check_sudoers
    check_packages
    check_logging

    # Student 2
    check_open_ports
    check_firewall
    check_network_interfaces
    check_ssh_config
    check_dns_config

    # Student 3
    check_setuid_files
    check_world_writable_files
    check_backup_permissions
    check_crontab
    check_mounted_partitions
}

main