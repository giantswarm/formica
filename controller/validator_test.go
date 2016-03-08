package controller

import (
	"testing"
)

// TestValidateRequest tests the ValidateRequest function.
func TestValidateRequest(t *testing.T) {
	var tests = []struct {
		request      Request
		valid        bool
		errAssertion func(error) bool
	}{
		// Test a group with no units in it is not valid.
		{
			request: Request{
				Group: "empty",
			},
			valid:        false,
			errAssertion: IsNoUnitsInGroup,
		},
		// Test a group with one well-named unit is valid.
		{
			request: Request{
				Group: "single",
				Units: []Unit{
					Unit{
						Name: "single-unit.service",
					},
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test a group with two well-named units is valid.
		{
			request: Request{
				Group: "single",
				Units: []Unit{
					Unit{
						Name: "single-unit.service",
					},
					Unit{
						Name: "single-unit2.timer",
					},
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test a group with a scalable unit is valid.
		{
			request: Request{
				Group: "scalable",
				Units: []Unit{
					Unit{
						Name: "scalable-unit@.service",
					},
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test a group with two scalable units is valid.
		{
			request: Request{
				Group: "scalable",
				Units: []Unit{
					Unit{
						Name: "scalable-unit@.service",
					},
					Unit{
						Name: "scalable-unit2@.timer",
					},
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test that a group mixing scalable and unscalable units is not valid.
		{
			request: Request{
				Group: "mix",
				Units: []Unit{
					Unit{
						Name: "mix-unit1.service",
					},
					Unit{
						Name: "mix-unit2@.service",
					},
				},
			},
			valid:        false,
			errAssertion: IsMixedSliceInstance,
		},
		// Test that units must be prefixed with their group name.
		{
			request: Request{
				Group: "single",
				Units: []Unit{
					Unit{
						Name: "bad-prefix.service",
					},
				},
			},
			valid:        false,
			errAssertion: IsBadUnitPrefix,
		},
		// Test that group names cannot contain @ symbols.
		{
			request: Request{
				Group: "bad@groupname@",
				Units: []Unit{
					Unit{
						Name: "bad@groupname@.service",
					},
				},
			},
			valid:        false,
			errAssertion: IsAtInGroupNameError,
		},
		// Test that unit names cannot contain multiple @ symbols.
		{
			request: Request{
				Group: "group",
				Units: []Unit{
					Unit{
						Name: "group-un@it@.service",
					},
				},
			},
			valid:        false,
			errAssertion: IsMultipleAtInUnitName,
		},
		// Test that a group cannot have multiple units with the same name.
		{
			request: Request{
				Group: "group",
				Units: []Unit{
					Unit{
						Name: "group-unit1@.service",
					},
					Unit{
						Name: "group-unit@.service",
					},
					Unit{
						Name: "group-unit2@.service",
					},
					Unit{
						Name: "group-unit@.service",
					},
				},
			},
			valid:        false,
			errAssertion: IsUnitsSameName,
		},
	}

	for index, test := range tests {
		valid, err := ValidateRequest(test.request)
		if test.valid != valid {
			t.Errorf("%v: Request validity should be: '%v', was '%v'", index, test.valid, valid)
		}
		if test.valid && err != nil {
			t.Errorf("%v: Request should be valid, but returned err: '%v'", index, err)
		}
		if !test.valid && !test.errAssertion(err) {
			t.Errorf("%v: Request should be invalid, but returned incorrect err '%v'", index, err)
		}
	}
}

// TestValidateMultipleRequest tests the ValidateMultipleRequest function.
func TestValidateMultipleRequest(t *testing.T) {
	var tests = []struct {
		requests     []Request
		valid        bool
		errAssertion func(error) bool
	}{
		// Test that two differently named groups are valid.
		{
			requests: []Request{
				Request{
					Group: "a",
				},
				Request{
					Group: "b",
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test that groups which are prefixes of another are invalid.
		{
			requests: []Request{
				Request{
					Group: "bat",
				},
				Request{
					Group: "batman",
				},
			},
			valid:        false,
			errAssertion: IsGroupsArePrefix,
		},
		// Test that the group prefix rule applies to the entire group name.
		{
			requests: []Request{
				Request{
					Group: "batwoman",
				},
				Request{
					Group: "batman",
				},
			},
			valid:        true,
			errAssertion: nil,
		},
		// Test that group names must be unique.
		{
			requests: []Request{
				Request{
					Group: "joker",
				},
				Request{
					Group: "joker",
				},
			},
			valid:        false,
			errAssertion: IsGroupsSameName,
		},
	}

	for index, test := range tests {
		valid, err := ValidateMultipleRequest(test.requests)
		if test.valid != valid {
			t.Errorf("%v: Requests validity should be: '%v', was '%v'", index, test.valid, valid)
		}
		if test.valid && err != nil {
			t.Errorf("%v: Requests should be valid, but returned err: '%v'", index, err)
		}
		if !test.valid && !test.errAssertion(err) {
			t.Errorf("%v: Requests should be invalid, but returned incorrect err '%v'", index, err)
		}
	}
}