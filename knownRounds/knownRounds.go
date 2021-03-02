///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Tracks which rounds have been checked and which are unchecked using a bit
// stream.
package knownRounds

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"math"
)

type RoundCheckFunc func(id id.Round) bool

// KnownRounds structure tracks which rounds are known and which are unknown.
// Each bit in bitStream corresponds to a round ID and if it is set, it means
// the round has been checked. All rounds before firstUnchecked are known to be
// checked. All rounds after lastChecked are unknown.
type KnownRounds struct {
	bitStream      uint64Buff // Buffer of check/unchecked rounds
	firstUnchecked id.Round   // ID of the first round that us unchecked
	lastChecked    id.Round   // ID of the last round that is checked
	fuPos          int        // Bit position of firstUnchecked in bitStream
}

// DiskKnownRounds structure is used to as an intermediary to marshal and
// unmarshal KnownRounds.
type DiskKnownRounds struct {
	BitStream                   []byte
	FirstUnchecked, LastChecked uint64
}

// NewKnownRound creates a new empty KnownRounds in the default state with a
// bit stream that can hold the given number of rounds.
func NewKnownRound(roundCapacity int) *KnownRounds {
	return &KnownRounds{
		bitStream:      make(uint64Buff, (roundCapacity+63)/64),
		firstUnchecked: 0,
		lastChecked:    0,
		fuPos:          0,
	}
}

// Marshal returns the JSON encoding of DiskKnownRounds, which contains the
// compressed information from KnownRounds. The bit stream is compressed such
// that the firstUnchecked occurs in the first block of the bit stream.
func (kr *KnownRounds) Marshal() ([]byte, error) {
	// Calculate length of compressed bit stream.
	startPos := kr.getBitStreamPos(kr.firstUnchecked)
	endPos := kr.getBitStreamPos(kr.lastChecked)
	length := kr.bitStream.delta(startPos, endPos)

	// Generate DiskKnownRounds with bit stream of the correct size
	dkr := DiskKnownRounds{
		BitStream:      []byte{},
		FirstUnchecked: uint64(kr.firstUnchecked),
		LastChecked:    uint64(kr.lastChecked),
	}

	// Copy only the blocks between firstUnchecked and lastChecked to the stream
	startBlock, _ := kr.bitStream.convertLoc(startPos)
	bitStream := make(uint64Buff, length)
	for i := 0; i < length; i++ {
		bitStream[i] = kr.bitStream[(i+startBlock)%len(kr.bitStream)]
	}

	dkr.BitStream = bitStream.marshal()

	return json.Marshal(dkr)
}

// Unmarshal parses the JSON-encoded data and stores it in the KnownRounds. An
// error is returned if the bit stream data is larger than the KnownRounds bit
// stream.
func (kr *KnownRounds) Unmarshal(data []byte) error {
	// Unmarshal JSON data
	dkr := &DiskKnownRounds{}
	err := json.Unmarshal(data, dkr)
	if err != nil {
		return err
	}

	bitStream := unmarshal(dkr.BitStream)

	// Handle the copying in of the bit stream
	if len(kr.bitStream) == 0 {
		// If there is no bitstream, like in the wire representations, then make
		// the size equal to what is coming in
		kr.bitStream = bitStream
	} else if len(kr.bitStream) >= len(bitStream) {
		// If a size already exists and the data fits within it, then copy it
		// into the beginning of the buffer
		copy(kr.bitStream, bitStream)
	} else {
		// If the passed in data is larger then the internal buffer, then return
		// an error
		return errors.Errorf("KnownRounds bitStream size of %d is too small "+
			"for passed in bit stream of size %d.",
			len(kr.bitStream), len(bitStream))
	}

	// Copy values over
	copy(kr.bitStream, bitStream)
	kr.firstUnchecked = id.Round(dkr.FirstUnchecked)
	kr.lastChecked = id.Round(dkr.LastChecked)
	kr.fuPos = int(dkr.FirstUnchecked % 64)

	return nil
}

// Checked determines if the round has been checked.
func (kr *KnownRounds) Checked(rid id.Round) bool {
	if rid < kr.firstUnchecked {
		return true
	} else if rid > kr.lastChecked {
		return false
	}

	pos := kr.getBitStreamPos(rid)

	return kr.bitStream.get(pos)
}

// Check denotes a round has been checked. If the passed in round occurred after
// the last checked round, then every round between them is set as unchecked and
// the passed in round becomes the last checked round. Will panic if the buffer
// is not large enough to hold the current data and the new data
func (kr *KnownRounds) Check(rid id.Round) {
	if abs(int(kr.lastChecked-rid))/(len(kr.bitStream)*64) > 0 {
		jww.FATAL.Panicf("Cannot check a round outside the current scope. " +
			"Scope is KnownRounds size more rounds than last checked. A call " +
			"to Forward() can be used to fix the scope.")
	}
	kr.check(rid)
}

// Check denotes a round has been checked. If the passed in round occurred after
// the last checked round, then every round between them is set as unchecked and
// the passed in round becomes the last checked round. Will shift the buffer
// forward, erasing old data, if the buffer is not large enough to hold the new
// checked input
func (kr *KnownRounds) ForceCheck(rid id.Round) {
	if abs(int(kr.lastChecked-rid))/(len(kr.bitStream)*64) > 0 {
		kr.Forward(rid - id.Round(len(kr.bitStream)*64))
	}
	kr.check(rid)
}

func (kr *KnownRounds) check(rid id.Round) {
	if rid < kr.firstUnchecked {
		return
	}
	pos := kr.getBitStreamPos(rid)

	// Set round as checked
	kr.bitStream.set(pos)

	// If the round ID is newer, then set it as the last checked ID and uncheck
	// all the newly added rounds in the buffer
	if rid > kr.lastChecked {
		kr.bitStream.clearRange(kr.getBitStreamPos(kr.lastChecked+1), pos)
		kr.lastChecked = rid
	}

	if kr.getBitStreamPos(kr.firstUnchecked) == pos {
		if kr.getBitStreamPos(kr.lastChecked) == pos {
			kr.fuPos = kr.getBitStreamPos(rid + 1)
			kr.firstUnchecked = rid + 1
			kr.lastChecked = rid + 1
			kr.bitStream.clear(kr.fuPos)
		} else {
			kr.migrateFirstUnchecked(rid)
		}
	}

	// Handle cases where rid lapse firstUnchecked one or more times.
	if rid > kr.firstUnchecked && (rid-kr.firstUnchecked) >= id.Round(kr.Len()) {
		newFu := rid + 1 - id.Round(kr.Len())
		kr.fuPos = kr.getBitStreamPos(newFu)
		kr.firstUnchecked = rid + 1 - id.Round(kr.Len())
		kr.migrateFirstUnchecked(rid)
	}

	// Set round as checked
	kr.bitStream.set(pos)
}

// abs returns the absolute value of the passed in integer.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// migrateFirstUnchecked moves firstUnchecked to the next unchecked round or
// sets it to lastUnchecked if all rounds are checked.
func (kr *KnownRounds) migrateFirstUnchecked(rid id.Round) {
	for ; kr.bitStream.get(kr.getBitStreamPos(rid)) &&
		rid < kr.lastChecked; rid++ {
	}
	kr.fuPos = kr.getBitStreamPos(rid)
	kr.firstUnchecked = rid
}

// Forward sets all rounds before the given round ID as checked.
func (kr *KnownRounds) Forward(rid id.Round) {
	if rid > kr.lastChecked {
		kr.firstUnchecked = rid
		kr.lastChecked = rid
		kr.fuPos = int(rid % 64)
	} else if rid >= kr.firstUnchecked {
		kr.migrateFirstUnchecked(rid)
	}
}

// RangeUnchecked runs the passed function over all rounds starting with oldest
// unknown and ending with
func (kr *KnownRounds) RangeUnchecked(oldestUnknown id.Round, maxChecked uint,
	roundCheck func(id id.Round) bool) id.Round {

	numChecked := uint(0)
	earliestChecked := id.Round(math.MaxUint64)

	// If the newest round is in the range of known rounds, then skip checking
	if oldestUnknown > kr.lastChecked {
		return oldestUnknown
	}

	// Check all the rounds after the last checked round
	newestRound := kr.firstUnchecked
	if oldestUnknown > kr.firstUnchecked {
		newestRound = oldestUnknown
	}

	// Check the unknown region before buffer
	if oldestUnknown < kr.firstUnchecked {
		for i := oldestUnknown; i < kr.firstUnchecked; i++ {
			if numChecked >= maxChecked {
				if i < earliestChecked {
					earliestChecked = i
				}
				return earliestChecked
			}
			if !roundCheck(i) && i < earliestChecked {
				earliestChecked = i
			}
			numChecked++
		}
	}

	if newestRound >= kr.firstUnchecked {
		for i := newestRound; i <= kr.lastChecked; i++ {
			if numChecked >= maxChecked {
				if i < earliestChecked {
					earliestChecked = i
				}
				return earliestChecked
			}

			if !kr.Checked(i) {
				if i < earliestChecked {
					earliestChecked = i
				}
				continue
			}

			if !roundCheck(i) && i < earliestChecked {
				earliestChecked = i
			}
			numChecked++
		}
	}

	if kr.lastChecked+1 < earliestChecked {
		earliestChecked = kr.lastChecked
	}

	//return the next round
	return earliestChecked + 1
}

// RangeUncheckedMasked masks the bit stream with the provided mask.
func (kr *KnownRounds) RangeUncheckedMasked(mask *KnownRounds,
	roundCheck RoundCheckFunc, maxChecked int) {

	kr.RangeUncheckedMaskedRange(mask, roundCheck, 0, math.MaxUint64, maxChecked)
}

// RangeUncheckedMaskedRange masks the bit stream with the provided mask.
func (kr *KnownRounds) RangeUncheckedMaskedRange(mask *KnownRounds,
	roundCheck RoundCheckFunc, start, end id.Round, maxChecked int) {

	numChecked := 0

	if mask.firstUnchecked != mask.lastChecked {
		jww.TRACE.Printf("mask (before Forward()) {\n\tbitStream:      %064b\n\tfirstUnchecked: %d\n\tlastChecked:    %d\n\tfuPos:          %d\n}", mask.bitStream, mask.firstUnchecked, mask.lastChecked, mask.fuPos)
		mask.Forward(kr.firstUnchecked)
		subSample, delta := kr.subSample(mask.firstUnchecked, mask.lastChecked)
		// FIXME: it is inefficient to make a copy of the mask here.

		jww.TRACE.Printf("mask (after Forward()) {\n\tbitStream:      %064b\n\tfirstUnchecked: %d\n\tlastChecked:    %d\n\tfuPos:          %d\n}", mask.bitStream, mask.firstUnchecked, mask.lastChecked, mask.fuPos)
		jww.TRACE.Printf("kr {\n\tbitStream:      %064b\n\tfirstUnchecked: %d\n\tlastChecked:    %d\n\tfuPos:          %d\n}", kr.bitStream, kr.firstUnchecked, kr.lastChecked, kr.fuPos)
		jww.TRACE.Printf("delta: %d", delta)
		jww.TRACE.Printf("subSample:     %064b", subSample)
		// jww.TRACE.Printf("maskSubSample: %064b", maskSubSample)
		result := subSample.implies(mask.bitStream)

		for i := mask.firstUnchecked + id.Round(delta) - 1; i >= mask.firstUnchecked && numChecked < maxChecked; i, numChecked = i-1, numChecked+1 {
			if !result.get(int(i-mask.firstUnchecked)) && roundCheck(i) {
				kr.Check(i)
			}
		}
	}

	if start < kr.firstUnchecked {
		start = kr.firstUnchecked
	}

	if end > mask.firstUnchecked {
		end = mask.firstUnchecked
	}

	for i := start; i < end && numChecked < maxChecked; i, numChecked = i+1, numChecked+1 {
		if !kr.Checked(i) && roundCheck(i) {
			kr.Check(i)
		}
	}
}

// subSample returns a subsample of the KnownRounds buffer from the start to end
// round and its length.
func (kr *KnownRounds) subSample(start, end id.Round) (uint64Buff, int) {
	// Get the number of blocks spanned by the range
	numBlocks := kr.bitStream.delta(kr.getBitStreamPos(start),
		kr.getBitStreamPos(end))

	if start > kr.lastChecked {
		return make(uint64Buff, numBlocks), numBlocks
	}

	copyEnd := end
	if kr.lastChecked < end {
		copyEnd = kr.lastChecked
	}

	// Create subsample of the buffer
	buff := kr.bitStream.copy(kr.getBitStreamPos(start),
		kr.getBitStreamPos(copyEnd+1))

	// Return a buffer of the correct size and its length
	return buff.extend(numBlocks), abs(int(end - start))
}

// Get the position of the bit in the bit stream for the given round ID.
func (kr *KnownRounds) getBitStreamPos(rid id.Round) int {
	var delta int
	if rid < kr.firstUnchecked {
		delta = -int(kr.firstUnchecked - rid)
	} else {
		delta = int(rid - kr.firstUnchecked)
	}

	return (kr.fuPos + delta) % kr.Len()
}

// Len returns the max number of round IDs the buffer can hold.
func (kr *KnownRounds) Len() int {
	return len(kr.bitStream) * 64
}
