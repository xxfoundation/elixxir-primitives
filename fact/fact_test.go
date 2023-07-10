////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fact

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Tests that NewFact returns a correctly formatted Fact.
func TestNewFact(t *testing.T) {
	tests := []struct {
		ft       FactType
		fact     string
		expected Fact
	}{
		{Username, "muUsername", Fact{"muUsername", Username}},
		{Email, "email@example.com", Fact{"email@example.com", Email}},
		{Phone, "8005559486US", Fact{"8005559486US", Phone}},
		{Nickname, "myNickname", Fact{"myNickname", Nickname}},
	}

	for i, tt := range tests {
		fact, err := NewFact(tt.ft, tt.fact)
		if err != nil {
			t.Errorf("Failed to make new fact (%d): %+v", i, err)
		} else if !reflect.DeepEqual(tt.expected, fact) {
			t.Errorf("Unexpected new Fact (%d).\nexpected: %s\nreceived: %s",
				i, tt.expected, fact)
		}
	}
}

// Error path: Tests that NewFact returns error when a fact exceeds the
// maxFactCharacterLimit.
func TestNewFact_ExceedMaxFactError(t *testing.T) {
	_, err := NewFact(Email,
		"devinputvalidation_devinputvalidation_devinputvalidation@elixxir.io")
	if err == nil {
		t.Fatal("Expected error when the fact is longer than the maximum " +
			"character length.")
	}

}

// Tests that a Fact marshalled by Fact.Stringify and unmarshalled by
// UnstringifyFact matches the original.
func TestFact_Stringify_UnstringifyFact(t *testing.T) {
	facts := []Fact{
		{"muUsername", Username},
		{"email@example.com", Email},
		{"8005559486US", Phone},
		{"myNickname", Nickname},
	}

	for i, expected := range facts {
		factString := expected.Stringify()
		fact, err := UnstringifyFact(factString)
		if err != nil {
			t.Errorf(
				"Failed to unstringify fact %s (%d): %+v", expected, i, err)
		} else if !reflect.DeepEqual(expected, fact) {
			t.Errorf("Unexpected unstringified Fact %s (%d)."+
				"\nexpected: %s\nreceived: %s",
				factString, i, expected, fact)
		}
	}
}

// Consistency test of Fact.Stringify.
func TestFact_Stringify(t *testing.T) {
	tests := []struct {
		fact     Fact
		expected string
	}{
		{Fact{"muUsername", Username}, "UmuUsername"},
		{Fact{"email@example.com", Email}, "Eemail@example.com"},
		{Fact{"8005559486US", Phone}, "P8005559486US"},
		{Fact{"myNickname", Nickname}, "NmyNickname"},
	}

	for i, tt := range tests {
		factString := tt.fact.Stringify()

		if factString != tt.expected {
			t.Errorf("Unexpected strified Fact %s (%d)."+
				"\nexpected: %s\nreceived: %s",
				tt.fact, i, tt.expected, factString)
		}
	}
}

// Consistency test of UnstringifyFact
func TestUnstringifyFact(t *testing.T) {
	tests := []struct {
		factString string
		expected   Fact
	}{
		{"UmuUsername", Fact{"muUsername", Username}},
		{"Eemail@example.com", Fact{"email@example.com", Email}},
		{"P8005559486US", Fact{"8005559486US", Phone}},
		{"NmyNickname", Fact{"myNickname", Nickname}},
	}

	for i, tt := range tests {
		fact, err := UnstringifyFact(tt.factString)
		if err != nil {
			t.Errorf(
				"Failed to unstringify fact %s (%d): %+v", tt.factString, i, err)
		} else if !reflect.DeepEqual(tt.expected, fact) {
			t.Errorf("Unexpected unstringified Fact %s (%d)."+
				"\nexpected: %s\nreceived: %s",
				tt.factString, i, tt.expected, fact)
		}
	}
}

// Tests that ValidateFact correctly validates various facts.
func TestValidateFact(t *testing.T) {
	facts := []Fact{
		{"muUsername", Username},
		{"email@example.com", Email},
		{"8005559486US", Phone},
		{"myNickname", Nickname},
	}

	for i, fact := range facts {
		err := ValidateFact(fact)
		if err != nil {
			t.Errorf(
				"Failed to validate fact %s (%d): %+v", fact, i, err)
		}
	}
}

// Error path: Tests that ValidateFact does not validate invalid facts
func TestValidateFact_InvalidFactsError(t *testing.T) {
	facts := []Fact{
		{"test@gmail@gmail.com", Email},
		{"US8005559486", Phone},
		{"020 8743 8000135UK", Phone},
		{"me", Nickname},
	}

	for i, fact := range facts {
		err := ValidateFact(fact)
		if err == nil {
			t.Errorf("Did not error on invalid fact %s (%d)", fact, i)
		}
	}
}

// Tests that a Fact JSON marshalled and unmarshalled matches the original.
func TestFact_JsonMarshalUnmarshal(t *testing.T) {
	facts := []Fact{
		{"muUsername", Username},
		{"email@example.com", Email},
		{"8005559486US", Phone},
		{"myNickname", Nickname},
	}

	for i, expected := range facts {
		data, err := json.Marshal(expected)
		if err != nil {
			t.Errorf("Failed to JSON marshal %s (%d): %+v", expected, i, err)
		}

		var fact Fact
		if err = json.Unmarshal(data, &fact); err != nil {
			t.Errorf("Failed to JSON unmarshal %s (%d): %+v", expected, i, err)
		}

		if !reflect.DeepEqual(expected, fact) {
			t.Errorf("Unexpected unmarshalled fact (%d)."+
				"\nexpected: %+v\nreceived: %+v", i, expected, fact)
		}
	}
}
