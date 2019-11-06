#!/bin/env python3
import random
import json
import os

PKGLIST = open("./data/pkglist.txt")
ARCHS = ['i386', 'i586', 'i686', 'x86_64']
PKGS = PKGLIST.read().splitlines()
N_REQS = int(os.getenv('BENCHMARK_MESSAGES'))

os.makedirs('data/body', exist_ok=True)

for i in range(1, N_REQS+1):
    data = {
        'id': i,
        'arch': ARCHS[i % len(ARCHS)],
        'packages' : []
    }
    for p in range(0, 4000):
        data['packages'].append(random.choice(PKGS))
    out = open(f'data/body/{i}.json', 'w')
    out.write(json.dumps(data))
    print(f'JSON generated: {i}/{N_REQS}')
