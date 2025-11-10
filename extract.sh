#!/bin/bash

# 检查是否传入了conflict rate和results目录路径
if [ -z "$1" ] || [ -z "$2" ]; then
    echo "请指定conflict rate和results目录路径 (例如 cr0.1 / /path/to/results)。"
    exit 1
fi

# 获取指定的 conflict rate 和 results 目录路径
results_dir="$1"
hlf_type="$2"
specified_cr="$3"

# 定义结果数组
effective_tps_list=()
avg_commit_latency_list=()

# 遍历指定的 results 目录下的所有目录，并匹配指定的 conflict rate
# for dir in "$results_dir"/*; do
# for rate in 100 500 1000 2000 3000 4000 5000 6000 7000 8000 9000 10000; do
#     dir="$results_dir/vanilla_conn16_${rate}_cr${specified_cr}"
#     # echo "Processing directory: $dir"
#     # 确保是目录，并且目录名称包含指定的 conflict rate
#     if [ -d "$dir" ] && [[ "$dir" =~ cr$specified_cr$ ]]; then
#         echo "Processing directory: $dir"
#         # 检查 conflict_report.txt 文件是否存在
#         report_file="$dir/conflict_report.txt"
#         if [ -f "$report_file" ]; then
#             # 提取 Effective TPS 和 Average Commit Latency
#             effective_tps=$(grep "^Effective TPS:" "$report_file" | awk '{print $3}')
#             avg_commit_latency=$(grep '^Average Commit Latency:' "$report_file" | awk '{print $4}' | sed 's/s$//')
            
#             # 将结果添加到数组中
#             effective_tps_list+=("$effective_tps")
#             avg_commit_latency_list+=("$avg_commit_latency")
#         fi
#     fi
# done

for rate in 200 400 600 800 1000 1500 2000 2500 3000 4000 5000; do
# for rate in 200 400 600 800 1000 1500 2000 2500 3000; do
# for rate in 500 1000 2000 2500 3000 3200 3400 3600 4000 5000; do
    dir="$results_dir/${hlf_type}_conn16_${rate}_cr${specified_cr}"
    # echo "Processing directory: $dir"
    # 确保是目录，并且目录名称包含指定的 conflict rate
    if [ -d "$dir" ] && [[ "$dir" =~ cr$specified_cr$ ]]; then
        echo "Processing directory: $dir"
        # 检查 conflict_report.txt 文件是否存在
        report_file="$dir/conflict_report.txt"
        if [ -f "$report_file" ]; then
            # 提取 Effective TPS 和 Average Commit Latency
            tps=$(grep "^TPS:" "$report_file" | awk '{print $2}')
            # txn_num=$(grep "^ALL Transactions:" "$report_file" | awk '{print $3}')
            # aborted_txn_num=$(grep "0\.00$" "$report_file" | wc -l)
            avg_commit_latency=$(grep '^Average Commit Latency:' "$report_file" | awk '{print $4}' | sed 's/s$//')
            # effective_tps=$(echo "scale=2; $tps * ($txn_num - $aborted_txn_num) / $txn_num" | bc -l)
            effective_tps=$(grep "^Effective TPS:" "$report_file" | awk '{print $3}')
            abort_rate=$(grep "^Abort Rate:" "$report_file" | awk '{print $3}' | sed 's/%$//')

            # 将结果添加到数组中
            effective_tps_list+=("$effective_tps")
            avg_commit_latency_list+=("$avg_commit_latency")
            abort_rate_list+=("$abort_rate")
            tps_list+=("$tps")
        fi
    fi
done

# 输出两个列表
IFS=,
echo "Effective TPS: ${effective_tps_list[*]}"
echo "Average Commit Latency (单位: 秒): ${avg_commit_latency_list[*]}"
echo "Aborted Rate: ${abort_rate_list[*]}"
echo "TPS: ${tps_list[*]}"
