#!/usr/bin/env bash
# Buka SSH tunnel ke Postgres di VM nvm-server. Biarkan terminal ini terbuka selama dev, tutup dengan Ctrl+C.
set -euo pipefail

ssh_target="${DB_TUNNEL_SSH:-nvmadmin@192.168.68.144}"
local_port="${DB_TUNNEL_LOCAL_PORT:-15432}"
remote_port="${DB_TUNNEL_REMOTE_PORT:-5432}"

echo "Tunnel 127.0.0.1:${local_port} ke ${ssh_target}, port ${remote_port} di VM"
echo "Tutup dengan Ctrl+C."

# ExitOnForwardFailure membuat ssh langsung berhenti kalau port lokal sudah dipakai,
# bukannya tersambung tanpa tunnel. Penyebab paling umum: tunnel NVM-Internal masih terbuka.
# Tunnel itu menuju Postgres yang sama, jadi boleh dipakai bersama.
exec ssh -N \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -L "127.0.0.1:${local_port}:127.0.0.1:${remote_port}" \
  "$ssh_target"