#!/usr/bin/env python3

import sys, yaml

config = {}
with open("refcard/data/config.yaml", 'r') as stream:
    try:
        config = yaml.safe_load(stream)
    except yaml.YAMLError as exc:
        print(exc)

print("Config:{c}\n".format(c=config))