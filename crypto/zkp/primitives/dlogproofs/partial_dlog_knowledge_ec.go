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

package dlogproofs

import (
	"github.com/xlab-si/emmy/crypto/common"
	"github.com/xlab-si/emmy/crypto/dlog"
	"github.com/xlab-si/emmy/types"
	"math/big"
)

// ProvePartialECDLogKnowledge demonstrates how prover can prove that he knows dlog_a2(b2) and
// the verifier does not know whether knowledge of dlog_a1(b1) or knowledge of dlog_a2(b2) was proved.
func ProvePartialECDLogKnowledge(dlog *dlog.ECDLog, secret1 *big.Int,
	a1, a2, b2 *types.ECGroupElement) bool {
	prover := NewPartialECDLogProver(dlog)
	verifier := NewPartialECDLogVerifier(dlog)

	b1X, b1Y := prover.DLog.Exponentiate(a1.X, a1.Y, secret1)
	b1 := types.NewECGroupElement(b1X, b1Y)
	triple1, triple2 := prover.GetProofRandomData(secret1, a1, b1, a2, b2)

	verifier.SetProofRandomData(triple1, triple2)
	challenge := verifier.GetChallenge()

	c1, z1, c2, z2 := prover.GetProofData(challenge)
	verified := verifier.Verify(c1, z1, c2, z2)

	return verified
}

// Proving that it knows either secret1 such that a1^secret1 = b1 or
//  secret2 such that a2^secret2 = b2.
type PartialECDLogProver struct {
	DLog    *dlog.ECDLog
	secret1 *big.Int
	a1      *types.ECGroupElement
	a2      *types.ECGroupElement
	r1      *big.Int
	c2      *big.Int
	z2      *big.Int
	ord     int
}

func NewPartialECDLogProver(dlog *dlog.ECDLog) *PartialECDLogProver {
	return &PartialECDLogProver{
		DLog: dlog,
	}
}

func (prover *PartialECDLogProver) GetProofRandomData(secret1 *big.Int, a1, b1, a2,
	b2 *types.ECGroupElement) (*types.ECTriple, *types.ECTriple) {
	prover.a1 = a1
	prover.a2 = a2
	prover.secret1 = secret1
	r1 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	c2 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	z2 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	prover.r1 = r1
	prover.c2 = c2
	prover.z2 = z2
	x1X, x1Y := prover.DLog.Exponentiate(a1.X, a1.Y, r1)
	x2X, x2Y := prover.DLog.Exponentiate(a2.X, a2.Y, z2)
	b2ToC2X, b2ToC2Y := prover.DLog.Exponentiate(b2.X, b2.Y, c2)
	b2ToC2InvX, b2ToC2InvY := prover.DLog.Inverse(b2ToC2X, b2ToC2Y)
	x2X, x2Y = prover.DLog.Multiply(x2X, x2Y, b2ToC2InvX, b2ToC2InvY)

	x1 := types.NewECGroupElement(x1X, x1Y)
	x2 := types.NewECGroupElement(x2X, x2Y)

	// we need to make sure that the order does not reveal which secret we do know:
	ord := common.GetRandomInt(big.NewInt(2))
	triple1 := types.NewECTriple(x1, a1, b1)
	triple2 := types.NewECTriple(x2, a2, b2)

	if ord.Cmp(big.NewInt(0)) == 0 {
		prover.ord = 0
		return triple1, triple2
	} else {
		prover.ord = 1
		return triple2, triple1
	}
}

func (prover *PartialECDLogProver) GetProofData(challenge *big.Int) (*big.Int, *big.Int,
	*big.Int, *big.Int) {
	c1 := new(big.Int).Xor(prover.c2, challenge)

	z1 := new(big.Int)
	z1.Mul(c1, prover.secret1)
	z1.Add(z1, prover.r1)
	z1.Mod(z1, prover.DLog.GetOrderOfSubgroup())

	if prover.ord == 0 {
		return c1, z1, prover.c2, prover.z2
	} else {
		return prover.c2, prover.z2, c1, z1
	}
}

type PartialECDLogVerifier struct {
	DLog      *dlog.ECDLog
	triple1   *types.ECTriple // contains x1, a1, b1
	triple2   *types.ECTriple // contains x2, a2, b2
	challenge *big.Int
}

func NewPartialECDLogVerifier(dlog *dlog.ECDLog) *PartialECDLogVerifier {
	return &PartialECDLogVerifier{
		DLog: dlog,
	}
}

func (verifier *PartialECDLogVerifier) SetProofRandomData(triple1, triple2 *types.ECTriple) {
	verifier.triple1 = triple1
	verifier.triple2 = triple2
}

func (verifier *PartialECDLogVerifier) GetChallenge() *big.Int {
	challenge := common.GetRandomInt(verifier.DLog.GetOrderOfSubgroup())
	verifier.challenge = challenge
	return challenge
}

func (verifier *PartialECDLogVerifier) verifyTriple(triple *types.ECTriple,
	challenge, z *big.Int) bool {
	left1, left2 := verifier.DLog.Exponentiate(triple.B.X, triple.B.Y, z)    // a.X, a.Y, z
	r1, r2 := verifier.DLog.Exponentiate(triple.C.X, triple.C.Y, challenge)  // b.X, b.Y, challenge
	right1, right2 := verifier.DLog.Multiply(r1, r2, triple.A.X, triple.A.Y) // r1, r2, x.X, x.Y

	return left1.Cmp(right1) == 0 && left2.Cmp(right2) == 0
}

func (verifier *PartialECDLogVerifier) Verify(c1, z1, c2, z2 *big.Int) bool {
	c := new(big.Int).Xor(c1, c2)
	if c.Cmp(verifier.challenge) != 0 {
		return false
	}

	verified1 := verifier.verifyTriple(verifier.triple1, c1, z1)
	verified2 := verifier.verifyTriple(verifier.triple2, c2, z2)
	return verified1 && verified2
}
