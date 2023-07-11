////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package notifications

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"strings"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

type Data struct {
	EphemeralID int64
	RoundID     uint64
	IdentityFP  []byte
	MessageHash []byte
}

func BuildNotificationCSV(ndList []*Data, maxSize int) ([]byte, []*Data) {
	var buf bytes.Buffer
	var numWritten int

	for _, nd := range ndList {
		var line bytes.Buffer
		w := csv.NewWriter(&line)
		output := []string{
			base64.StdEncoding.EncodeToString(nd.MessageHash),
			base64.StdEncoding.EncodeToString(nd.IdentityFP)}

		if err := w.Write(output); err != nil {
			jww.FATAL.Printf("Failed to write notificationsCSV line: %+v", err)
		}
		w.Flush()

		if buf.Len()+line.Len() > maxSize {
			break
		}

		if _, err := buf.Write(line.Bytes()); err != nil {
			jww.FATAL.Printf("Failed to write to notificationsCSV: %+v", err)
		}

		numWritten++
	}

	return buf.Bytes(), ndList[numWritten:]
}

func DecodeNotificationsCSV(data string) ([]*Data, error) {
	r := csv.NewReader(strings.NewReader(data))
	read, err := r.ReadAll()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to decode notifications CSV")
	}

	l := make([]*Data, len(read))
	for i, tuple := range read {
		messageHash, err := base64.StdEncoding.DecodeString(tuple[0])
		if err != nil {
			return nil, errors.WithMessage(err, "Failed decode an element")
		}
		identityFP, err := base64.StdEncoding.DecodeString(tuple[1])
		if err != nil {
			return nil, errors.WithMessage(err, "Failed decode an element")
		}
		l[i] = &Data{
			EphemeralID: 0,
			IdentityFP:  identityFP,
			MessageHash: messageHash,
		}
	}
	return l, nil
}
