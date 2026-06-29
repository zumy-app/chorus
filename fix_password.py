#!/usr/bin/env python3
import os
os.chdir('/root/chorus')
with open('.env', 'r') as f:
    c = f.read()
c = c.replace('ChorusProd2026!', 'ChorusProd2026x')
with open('.env', 'w') as f:
    f.write(c)
print('Password updated to ChorusProd2026x')
