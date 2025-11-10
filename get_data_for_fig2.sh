#!/bin/bash

mkdir result &>/dev/null

for i in {1..5}; do
    ./consym.sh run
    cp test/result/put_log.txt result/put_log_${i}.txt
    cp test/result/put_report.txt result/put_report_${i}.txt
    cp test/result/conflict_log.txt result/conflict_log_${i}.txt
    cp test/result/conflict_report.txt result/conflict_report_${i}.txt
    ./consym.sh clean
    sleep 20
done

