#!/usr/bin/env python

## routesmash.py - 29/07/2014
## ben.dale@gmail.com
## Spam a list of generated /24 prefixes
## Use with ExaBGP for load testing

import sys
import time

for third in range(0, 255):
    sys.stdout.write('announce route 172.16.%d.0/24 next-hop 10.10.10.10\n' % third)
    sys.stdout.flush()
    ## Back off timer if router is too slow:
    ## time.sleep(0.001)

while True:
    time.sleep(1)

