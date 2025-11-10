/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	cb "github.com/PanQL/fabric-protos-go/common"
	"github.com/Yunpeng-J/HLF-2.2/common/flogging"
	"github.com/Yunpeng-J/HLF-2.2/protoutil"
	"github.com/golang/protobuf/proto"
)

// blockCreator holds number and hash of latest block
// so that next block will be created based on it.
type blockCreator struct {
	hash   []byte
	number uint64

	logger *flogging.FabricLogger
}

func (bc *blockCreator) createNextBlock(envs []*cb.Envelope) *cb.Block {
	data := &cb.BlockData{
		// Data: make([][]byte, len(envs)),
	}

	for _, env := range envs {
		temp, err := proto.Marshal(env)
		if err != nil {
			// bc.logger.Panicf("Could not marshal envelope: %s", err)
			bc.logger.Infof("debug urgent: why nil")
		} else {
			data.Data = append(data.Data, temp)
		}
	}

	bc.number++

	block := protoutil.NewBlock(bc.number, bc.hash)
	block.Header.DataHash = protoutil.BlockDataHash(data)
	block.Data = data

	bc.hash = protoutil.BlockHeaderHash(block.Header)
	return block
}
