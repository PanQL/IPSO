/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"

	"github.com/Yunpeng-J/fabric-chaincode-go/shim"
	"github.com/Yunpeng-J/HLF-2.2/integration/chaincode/kvexecutor"
)

func main() {
	err := shim.Start(&kvexecutor.KVExcutor{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exiting Simple chaincode: %s", err)
		os.Exit(2)
	}
}
