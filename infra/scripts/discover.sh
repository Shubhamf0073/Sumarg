#!/usr/bin/env bash
# discover.sh — READ-ONLY audit of the VPS before installing Sumarg (docs/06 §1).
# Changes nothing. Run with sudo so it can see listening processes and firewall state:
#   sudo bash discover.sh | tee server-discovery.txt
set -u

hr() { printf '\n==== %s ====\n' "$1"; }
have() { command -v "$1" >/dev/null 2>&1; }
run() { "$@" 2>&1 || true; }

hr "Host"
run hostnamectl
# shellcheck source=/dev/null
[ -r /etc/os-release ] && . /etc/os-release && echo "OS: ${PRETTY_NAME:-unknown}"
echo "Kernel: $(uname -r)"
echo "Timezone: $(timedatectl show -p Timezone --value 2>/dev/null || cat /etc/timezone 2>/dev/null || echo unknown)"
run timedatectl show -p NTPSynchronized

hr "CPU / memory / swap / disk"
echo "vCPUs: $(nproc)"
run free -m
run swapon --show
run df -hT -x tmpfs -x devtmpfs -x overlay
echo "Largest dirs under /srv /var /home (depth 1):"
run du -xsh /srv/* /var/* /home/* 2>/dev/null | sort -rh | head -15

hr "Control panels"
for p in /usr/local/cpanel /usr/local/psa /usr/local/directadmin /usr/local/CyberCP /opt/webuzo; do
  [ -e "$p" ] && echo "FOUND: $p  → stop and use the panel's reverse-proxy feature (docs/06 §1)"
done
echo "(none listed above = no common panel detected)"

hr "Listening ports (80, 443, 5432, 8720 matter)"
if have ss; then run ss -ltnp; else run netstat -ltnp; fi
for port in 80 443 5432 8720; do
  if ss -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]${port}\$"; then echo "port ${port}: IN USE"; else echo "port ${port}: free"; fi
done

hr "Web servers"
have nginx && { run nginx -v; echo "nginx config test:"; run nginx -t; echo "conf.d:"; run ls -l /etc/nginx/conf.d; echo "sites-enabled:"; run ls -l /etc/nginx/sites-enabled; }
have apache2 && run apache2 -v
have httpd && run httpd -v
have caddy && run caddy version
echo "Web server user/group:"
run ps -eo user,group,comm | grep -E 'nginx|apache2|httpd|caddy' | sort -u
getent group www-data >/dev/null && echo "www-data gid: $(getent group www-data | cut -d: -f3)"
getent group nginx >/dev/null && echo "nginx gid: $(getent group nginx | cut -d: -f3)"

hr "Docker"
if have docker; then
  run docker version --format 'Docker server {{.Server.Version}}'
  run docker compose version
  echo "Running containers (name / image / mem):"
  run docker stats --no-stream --format '{{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}'
  run docker ps --format '{{.Names}}\t{{.Image}}\t{{.Ports}}'
  echo "Docker root dir: $(docker info --format '{{.DockerRootDir}}' 2>/dev/null)"
else
  echo "Docker NOT installed"
fi

hr "Host PostgreSQL"
have psql && run psql --version
run systemctl is-active postgresql

hr "Firewall"
have ufw && run ufw status verbose
have firewall-cmd && run firewall-cmd --list-all
have nft && run nft list ruleset | head -40
have iptables && run iptables -S | head -40

hr "SSH"
run sshd -T | grep -Ei '^(port|permitrootlogin|passwordauthentication|pubkeyauthentication) '

hr "Updates"
[ -e /etc/apt/apt.conf.d/20auto-upgrades ] && run cat /etc/apt/apt.conf.d/20auto-upgrades
run systemctl is-enabled dnf-automatic.timer

hr "Public IP and region"
if have curl; then
  run curl -s --max-time 5 https://ipinfo.io/json
else
  echo "curl not installed"
fi

hr "Existing Sumarg install"
run ls -la /srv/schoolfleet
echo
echo "Done. Paste this output into docs/server-notes.md and pick the path in docs/06 §1."
