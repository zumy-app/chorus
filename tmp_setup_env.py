#!/usr/bin/env python3
import secrets, os

os.chdir('/root/chorus')
jwt = secrets.token_urlsafe(64)

with open('.env.prod', 'r') as f:
    content = f.read()

content = content.replace(
    'JWT_SECRET=your-very-long-random-secret-at-least-64-chars',
    f'JWT_SECRET={jwt}'
)

with open('.env.prod', 'w') as f:
    f.write(content)

print(f'JWT_SECRET updated successfully')
print(f'JWT_SECRET={jwt[:32]}...')
