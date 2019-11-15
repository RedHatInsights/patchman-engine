#!/bin/env python3

import os
import json
import csv
import string
from datetime import datetime, date
from dateutil import parser

TIME_FORMAT = '%H:%M:%S'

def load_env(file):
    with open(file, 'r') as file:
        for l in file.readlines():
            if l.strip():
                [name, val] = l.split('=')
                print(name, val)
                os.environ[name] = val

load_env('./conf/common.env')


MSG_COUNT = int(os.getenv("BENCHMARK_MESSAGES", None))
LANGS = ["go", "rust", "python"]



def get_lang_time(file):
    obj = json.load(open(file))
    return obj['updated']


LOG_ROWS = []
with open('out/usages.csv') as csv_file:
    reader = csv.reader(csv_file, delimiter=',')
    for row in reader:
        for i, v in enumerate(row):
            row[i] = v.strip()
        LOG_ROWS.append(row)
        print(row)

TIMES = {}
CPUS = {}
RAMS = {}

for lang in LANGS:
    print(lang)
    end_time_str = get_lang_time(f"out/{lang}-{MSG_COUNT}.json")
    end_time = parser.isoparse(end_time_str).time()

    end_time_str = end_time.strftime(TIME_FORMAT)
    print(end_time_str)

    start_time_str = get_lang_time(f"out/{lang}-1.json")
    start_time = parser.isoparse(start_time_str).time()

    start_time_str = start_time.strftime(TIME_FORMAT)
    print(start_time_str)

    TIMES[lang] = {'start': start_time, 'end': end_time}
    rows = []
    for log in LOG_ROWS:
        time = datetime.strptime(log[0], TIME_FORMAT).time()
        if log[1] == lang and end_time >= time >= start_time:
            rows.append(log)
            CPUS.setdefault(lang, []).append(float(log[2].strip(" %")))
            ram = log[3].split('/')[0].split(' ')

            if ram[1].lower().split() == "gb":
                ram[0] = float(ram[0])  * 1000
            else:
                ram[0] = float(ram[0])


            RAMS.setdefault(lang, []).append(ram[0])

    print("CPU: ", CPUS[lang])
    print("RAM: ", RAMS[lang])


with open('report.csv', 'w') as out_file:
    writer = csv.writer(out_file)
    writer.writerow(['lang', 'time', 'min_cpu', 'max_cpu', 'min_ram', 'max_ram'])
    for lang in LANGS:
        start = datetime.combine(datetime.today(), TIMES[lang]['start'])
        end = datetime.combine(datetime.today(), TIMES[lang]['end'])
        writer.writerow([lang, end - start,
                         min(CPUS[lang]),
                         max(CPUS[lang]),
                         min(RAMS[lang]),
                         max(RAMS[lang])
                         ])

# Filter out rows within the run period, can be used for tracking resource usage
# Time is end-start
