////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fact

import (
	"testing"
)

func TestFactType_String(t *testing.T) {
	// FactTypes and expected strings for them
	FTs := []FactType{Username, Email, Phone, FactType(200)}
	Strs := []string{"Username", "Email", "Phone", "Unknown Fact FactType: 200"}
	for i, ft := range FTs {
		if FactType.String(ft) != Strs[i] {
			t.Errorf("Unexpected FactType string.\nexpected: %s\nreceived: %s",
				Strs[i], FactType.String(ft))
		}
	}
}

func TestFactType_Stringify(t *testing.T) {
	// FactTypes and expected strings for them
	FTs := []FactType{Username, Email, Phone}
	Strs := []string{"U", "E", "P"}
	for i, ft := range FTs {
		if FactType.Stringify(ft) != Strs[i] {
			t.Errorf("Unexpected stringified FactType.\nexpected: %s\nreceived: %s",
				Strs[i], FactType.Stringify(ft))
		}
	}
}

func TestFactType_Unstringify(t *testing.T) {
	// FactTypes and expected strings for them
	FTs := []FactType{Username, Email, Phone}
	Strs := []string{"U", "E", "P"}
	for i, ft := range FTs {
		gotft, err := UnstringifyFactType(Strs[i])
		if err != nil {
			t.Error(err)
		}
		if gotft != ft {
			t.Errorf("Unexpected unstringified FactType."+
				"\nexpected: %s\nreceived: %s", ft, gotft)
		}
	}

	_, err := UnstringifyFactType("x")
	if err == nil {
		t.Errorf("UnstringifyFactType did not return an error on an invalid type")
	}
}

func TestFactType_IsValid(t *testing.T) {
	if !FactType.IsValid(Username) ||
		!FactType.IsValid(Email) ||
		!FactType.IsValid(Phone) {

		t.Errorf("FactType.IsValid did not report a FactType as valid")
	}

	if FactType.IsValid(FactType(200)) {
		t.Errorf("FactType.IsValid reported a non-valid FactType value as valid")
	}
}
