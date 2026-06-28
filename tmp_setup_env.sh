#!/bin/bash
JWT=$(openssl rand -base64 64)
cd /root/chorus
cp .env.prod.example .env.prod
sed -i "s|JWT_SECRET=.*|JWT_SECRET=$JWT|" .env.prod
sed -i "s|DB_PASSWORD=.*|DB_PASSWORD=ChorusProd2026!|" .env.prod
echo "=== Environment configured ==="
grep -E "JWT_SECRET|DB_PASSWORD" .env.prod
