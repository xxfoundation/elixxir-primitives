////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package format

import jww "github.com/spf13/jwalterweatherman"

const (
	// Length of the entire message serial
	TotalLen = 512 // 4096 bits

	// Length, start index, and end index of the payloads
	PayloadLen    = 256 // 2048 bits
	payloadAStart = 0
	payloadAEnd   = payloadAStart + PayloadLen
	payloadBStart = payloadAEnd
	payloadBEnd   = payloadBStart + PayloadLen

	// Length, start index, and end index of grpByte
	GrpByteLen   = 1 // 8 bits
	grpByteStart = associatedDataEnd
	grpByteEnd   = grpByteStart + GrpByteLen
)

/*                               Message Structure (not to scale)
+----------------------------------------------------------------------------------------+
|                                         Message                                        |
|                                        4096 bits                                       |
+----------------------------------------------------------------------------------------+
|                  payloadA                  |                 payloadB                  |
|                 2048 bits                  |                2048 bits                  |
+------------------------------------+-------+---------------------------------+---------+
|              Contents              |             AssociatedData              | grpByte |
|              3192 bits             |                896 bits                 | 8 bits  |
+------------------------------------+-----------------------------------------+         |
|     padding     |       data       | recipientID | keyFP | timestamp |  mac  |         |
|   88–3192 bits  |    0–3104 bits   |   256 bits  | 256 b |  128 bits | 256 b |         |
+-----------------+------------------+-------------+-------+-----------+-------+---------+
*/

// Message structure stores all the data serially. Subsequent fields point to
// subsections of the serialised data so that the message is always serialized,
// it is ready to go, and no copies are required.
type Message struct {
	master         [TotalLen]byte // serialised message data
	Contents                      // points to the contents of the message
	AssociatedData                // points to the associate data of the message
	payloadA       []byte         // points to the first half of the message
	payloadB       []byte         // points to the second half of the message
	grpByte        []byte         // zero value byte ensures payloadB is in the group
}

// NewMessage creates a new empty message. It points the contents, associated
// data, payload A, and payload B, to their respective parts of master.
func NewMessage() *Message {
	newMsg := &Message{master: [TotalLen]byte{}}

	newMsg.Contents = *NewContents(newMsg.master[contentsStart:contentsEnd])
	newMsg.AssociatedData = *NewAssociatedData(
		newMsg.master[associatedDataStart:associatedDataEnd])

	newMsg.payloadA = newMsg.master[payloadAStart:payloadAEnd]
	newMsg.payloadB = newMsg.master[payloadBStart:payloadBEnd]

	newMsg.grpByte = newMsg.master[grpByteStart:grpByteEnd]
	copy(newMsg.grpByte, []byte{0})

	return newMsg
}

// GetMaster returns the entire serialised message.
func (m *Message) GetMaster() []byte {
	return m.master[:]
}

// GetPayloadA returns payload A, which is the first half of the message.
func (m *Message) GetPayloadA() []byte {
	return m.payloadA
}

// SetPayloadA copies the passed byte slice into payload A. If the specified
// byte slice is not exactly the same size as payload A, then it panics.
func (m *Message) SetPayloadA(payload []byte) {
	if len(payload) != PayloadLen {
		jww.ERROR.Panicf("new payload not the same size as PayloadA;"+
			"Expected: %v, Recieved: %v",
			PayloadLen, len(payload))
	}

	copy(m.payloadA, payload)
}

// GetPayloadB returns payload B, which is the last half of the message.
func (m *Message) GetPayloadB() []byte {
	return m.payloadB
}

// SetPayloadB copies the passed byte slice into payload B. If the specified
// byte slice is not exactly the same size as payload B, then it panics.
func (m *Message) SetPayloadB(payload []byte) {
	if len(payload) != PayloadLen {
		jww.ERROR.Panicf("new payload not the same size as PayloadB;"+
			"Expected: %v, Recieved: %v",
			PayloadLen, len(payload))
	}

	copy(m.payloadB, payload)
}

// GetPayloadBForEncryption ensures payload B is in the group for encrypting. It
// returns payload B with the first byte swapped to the end and the first byte
// to zero.
func (m *Message) GetPayloadBForEncryption() []byte {
	payloadCopy := make([]byte, PayloadLen)
	copy(payloadCopy, m.payloadB)
	payloadCopy[PayloadLen-1] = payloadCopy[0]
	payloadCopy[0] = 0

	return payloadCopy
}

// SetDecryptedPayloadB is used when receiving a decrypted payload B to ensure
// all data is put back in the right order. If the specified byte array is not
// exactly the same size as payload B, then it panics. Specifically, it moves
// the last byte to the front and sets the last byte to zero. Assumes the
// newPayload is in the group and that its first byte is zero.
func (m *Message) SetDecryptedPayloadB(newPayload []byte) {
	if len(newPayload) != PayloadLen {
		jww.ERROR.Panicf("new payload not the same size as PayloadA;"+
			"Expected: %v, Recieved: %v",
			PayloadLen, len(newPayload))
	}

	copy(m.payloadB, newPayload)
	m.payloadB[0] = m.payloadB[PayloadLen-1]
	m.payloadB[PayloadLen-1] = 0
}
