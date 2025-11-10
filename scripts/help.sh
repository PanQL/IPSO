#!/bin/bash

function print_help() {
  #TODO
  CMD="$1"
  if [[ -z $CMD ]]; then
    infoln "Usage:"
    infoln "  consym.sh <COMMAND> [<FLAG>...]"
    infoln
    infoln "    COMMAND:"
    infoln "      build   - Build HLF and/or tape images"
    infoln "      load    - Load HLF and/or tape images from tar file"
    infoln "      run     - Benchmark the performance of the specified images"
    infoln "      analyze - Analyze the raw data and generate reports"
    infoln "      clean   - Clean the results and/or images"
  fi
}
