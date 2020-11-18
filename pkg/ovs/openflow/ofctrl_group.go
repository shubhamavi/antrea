// Copyright 2020 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openflow

import (
	"fmt"

	"github.com/contiv/libOpenflow/openflow13"
	"github.com/contiv/ofnet/ofctrl"
)

type ofGroup struct {
	ofctrl *ofctrl.Group
	bridge *OFBridge
}

func (g *ofGroup) Reset() {
	g.ofctrl.Switch = g.bridge.ofSwitch
}

func (g *ofGroup) Add() error {
	return g.ofctrl.Install()
}

func (g *ofGroup) Modify() error {
	return g.ofctrl.Install()
}

func (g *ofGroup) Delete() error {
	return g.ofctrl.Delete()
}

func (g *ofGroup) Type() EntryType {
	return GroupEntry
}

func (g *ofGroup) KeyString() string {
	return fmt.Sprintf("group_id:%d", g.ofctrl.ID)
}

func (g *ofGroup) Bucket() BucketBuilder {
	return &bucketBuilder{
		group:  g,
		bucket: openflow13.NewBucket(),
	}
}

func (f *ofGroup) GetBundleMessage(entryOper OFOperation) (ofctrl.OpenFlowModMessage, error) {
	var operation int
	switch entryOper {
	case AddMessage:
		operation = openflow13.OFPGC_ADD
	case ModifyMessage:
		operation = openflow13.OFPGC_MODIFY
	case DeleteMessage:
		operation = openflow13.OFPGC_DELETE
	}
	message := f.ofctrl.GetBundleMessage(operation)
	return message, nil
}

func (g *ofGroup) ResetBuckets() Group {
	g.ofctrl.Buckets = nil
	return g
}

type bucketBuilder struct {
	group  *ofGroup
	bucket *openflow13.Bucket
}

// LoadReg makes the learned flow to load data to reg[regID] with specific range.
func (b *bucketBuilder) LoadReg(regID int, data uint32) BucketBuilder {
	return b.LoadRegRange(regID, data, Range{0, 31})
}

// LoadXXReg makes the learned flow to load data to xxreg[regID] with specific range.
func (b *bucketBuilder) LoadXXReg(regID int, data []byte) BucketBuilder {
	regAction := &ofctrl.NXLoadXXRegAction{FieldNumber: uint8(regID), Value: data, Mask: nil}
	b.bucket.AddAction(regAction.GetActionMessage())
	return b
}

// LoadRegRange is an action to Load data to the target register at specified range.
func (b *bucketBuilder) LoadRegRange(regID int, data uint32, rng Range) BucketBuilder {
	reg := fmt.Sprintf("%s%d", NxmFieldReg, regID)
	regField, _ := openflow13.FindFieldHeaderByName(reg, true)
	b.bucket.AddAction(openflow13.NewNXActionRegLoad(rng.ToNXRange().ToOfsBits(), regField, uint64(data)))
	return b
}

// ResubmitToTable is an action to resubmit packet to the specified table when the bucket is selected.
func (b *bucketBuilder) ResubmitToTable(tableID TableIDType) BucketBuilder {
	b.bucket.AddAction(openflow13.NewNXActionResubmitTableAction(openflow13.OFPP_IN_PORT, uint8(tableID)))
	return b
}

// Weight sets the weight of a bucket.
func (b *bucketBuilder) Weight(val uint16) BucketBuilder {
	b.bucket.Weight = val
	return b
}

func (b *bucketBuilder) Done() Group {
	b.group.ofctrl.AddBuckets(b.bucket)
	return b.group
}
