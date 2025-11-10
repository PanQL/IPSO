/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package etcdraft_test

import (
	"testing"

	"github.com/Yunpeng-J/HLF-2.2/common/channelconfig"
	"github.com/Yunpeng-J/HLF-2.2/msp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testingInstance *testing.T

func TestEtcdraft(t *testing.T) {
	testingInstance = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "Etcdraft Suite")
}

//go:generate counterfeiter -o mocks/orderer_org.go --fake-name OrdererOrg . channelConfigOrdererOrg
type channelConfigOrdererOrg interface {
	channelconfig.OrdererOrg
}

//go:generate counterfeiter -o mocks/msp.go --fake-name MSP . mspInterface
type mspInterface interface {
	msp.MSP
}
