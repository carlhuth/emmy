/*
 * Copyright 2017 XLAB d.o.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package commitments

import (
	"errors"
	"github.com/xlab-si/emmy/crypto/common"
	"github.com/xlab-si/emmy/crypto/dlog"
	"github.com/xlab-si/emmy/types"
	"math/big"
)

// Committer first needs to know H (it gets it from the receiver).
// Then committer can commit to some value x - it sends to receiver c = g^x * h^r.
// When decommitting, committer sends to receiver r, x; receiver checks whether c = g^x * h^r.
type PedersenECCommitter struct {
	dLog           *dlog.ECDLog
	h              *types.ECGroupElement
	committedValue *big.Int
	r              *big.Int
}

func NewPedersenECCommitter(curveType dlog.Curve) *PedersenECCommitter {
	dLog := dlog.NewECDLog(curveType)
	committer := PedersenECCommitter{
		dLog: dLog,
	}
	return &committer
}

// Value h needs to be obtained from a receiver and then set in a committer.
func (committer *PedersenECCommitter) SetH(h *types.ECGroupElement) {
	committer.h = h
}

// It receives a value x (to this value a commitment is made), chooses a random x, outputs c = g^x * g^r.
func (committer *PedersenECCommitter) GetCommitMsg(val *big.Int) (*types.ECGroupElement, error) {
	if val.Cmp(committer.dLog.OrderOfSubgroup) == 1 || val.Cmp(big.NewInt(0)) == -1 {
		err := errors.New("the committed value needs to be in Z_q (order of a base point)")
		return nil, err
	}

	// c = g^x * h^r
	r := common.GetRandomInt(committer.dLog.OrderOfSubgroup)

	committer.r = r
	committer.committedValue = val
	x1, y1 := committer.dLog.ExponentiateBaseG(val)
	x2, y2 := committer.dLog.Exponentiate(committer.h.X, committer.h.Y, r)
	c1, c2 := committer.dLog.Multiply(x1, y1, x2, y2)

	return &types.ECGroupElement{X: c1, Y: c2}, nil
}

// It returns values x and r (commitment was c = g^x * g^r).
func (committer *PedersenECCommitter) GetDecommitMsg() (*big.Int, *big.Int) {
	val := committer.committedValue
	r := committer.r

	return val, r
}

func (committer *PedersenECCommitter) VerifyTrapdoor(trapdoor *big.Int) bool {
	hx, hy := committer.dLog.ExponentiateBaseG(trapdoor)
	if hx.Cmp(committer.h.X) == 0 && hy.Cmp(committer.h.Y) == 0 {
		return true
	} else {
		return false
	}
}

type PedersenECReceiver struct {
	dLog       *dlog.ECDLog
	a          *big.Int
	h          *types.ECGroupElement
	commitment *types.ECGroupElement
}

func NewPedersenECReceiver(curve dlog.Curve) *PedersenECReceiver {
	dLog := dlog.NewECDLog(curve)

	a := common.GetRandomInt(dLog.OrderOfSubgroup)
	x, y := dLog.ExponentiateBaseG(a)

	receiver := new(PedersenECReceiver)
	receiver.dLog = dLog
	receiver.a = a
	receiver.h = &types.ECGroupElement{X: x, Y: y}

	return receiver
}

func (s *PedersenECReceiver) GetH() *types.ECGroupElement {
	return s.h
}

func (s *PedersenECReceiver) GetTrapdoor() *big.Int {
	return s.a
}

// When receiver receives a commitment, it stores the value using SetCommitment method.
func (s *PedersenECReceiver) SetCommitment(el *types.ECGroupElement) {
	s.commitment = el
}

// When receiver receives a decommitment, CheckDecommitment verifies it against the stored value
// (stored by SetCommitment).
func (s *PedersenECReceiver) CheckDecommitment(r, val *big.Int) bool {
	x1, y1 := s.dLog.ExponentiateBaseG(val)        // g^x
	x2, y2 := s.dLog.Exponentiate(s.h.X, s.h.Y, r) // h^r
	c1, c2 := s.dLog.Multiply(x1, y1, x2, y2)      // g^x * h^r

	var success bool
	if c1.Cmp(s.commitment.X) == 0 && c2.Cmp(s.commitment.Y) == 0 {
		success = true
	} else {
		success = false
	}

	return success
}
