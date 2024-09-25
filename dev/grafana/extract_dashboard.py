#!/usr/bin/env python3
import sys
import yaml


with sys.stdin as f:
    configmap = yaml.safe_load(f)
    dashboard_json = configmap["data"]["grafana.json"]
    replaced = dashboard_json.replace("$datasource", "Prometheus")
    print(replaced)
