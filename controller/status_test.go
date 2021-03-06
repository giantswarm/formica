package controller

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/giantswarm/inago/fleet"
	"github.com/giantswarm/inago/logging"
)

func givenSingleUnitStatus(name, sliceID string) fleet.UnitStatus {
	return fleet.UnitStatus{
		Name:    "unit-" + name + "@" + sliceID,
		Current: "loaded",
		Desired: "loaded",
		Machine: []fleet.MachineStatus{
			{
				ID:            "machine1",
				IP:            net.ParseIP("10.0.0.101"),
				SystemdActive: "dead",
				UnitHash:      "1234",
			},
		},
	}
}

func givenGroupedStatus(sliceID string) fleet.UnitStatus {
	e := givenSingleUnitStatus("*", sliceID)
	e.Name = "*"
	return e
}

func TestUnitStatusList_Group_NoDiff(t *testing.T) {
	RegisterTestingT(t)

	input1 := givenSingleUnitStatus("main", "1")
	input2 := givenSingleUnitStatus("sidekick", "1")
	input3 := givenSingleUnitStatus("main", "2")
	input4 := givenSingleUnitStatus("sidekick", "2")

	output, err := UnitStatusList([]fleet.UnitStatus{input1, input2, input3, input4}).Group()

	Expect(err).To(Not(HaveOccurred()))
	Expect(output).To(ContainElement(givenGroupedStatus("1")))
	Expect(output).To(ContainElement(givenGroupedStatus("2")))
	Expect(len(output)).To(Equal(2))
}

func TestUnitStatusList_Group_UnitHashDiffs(t *testing.T) {
	RegisterTestingT(t)

	input1 := givenSingleUnitStatus("main", "1")
	input2 := givenSingleUnitStatus("sidekick", "1")
	input3 := givenSingleUnitStatus("main", "2")
	input4 := givenSingleUnitStatus("sidekick", "2")
	input3.Machine[0].UnitHash = "something-else"

	output, err := UnitStatusList([]fleet.UnitStatus{input1, input2, input3, input4}).Group()

	Expect(err).To(Not(HaveOccurred()))
	Expect(output).To(ContainElement(input1))
	Expect(output).To(ContainElement(input2))
	Expect(output).To(ContainElement(input3))
	Expect(output).To(ContainElement(input4))
	Expect(len(output)).To(Equal(4))
}

func inputUnitStatusList(configs ...map[string][]string) UnitStatusList {
	unitStatusList := UnitStatusList{}

	i := 0
	for _, c := range configs {
		for j, sliceID := range c["sliceIDs"] {
			state := c["states"][j]
			i++

			unitStatus := fleet.UnitStatus{
				Current: fmt.Sprintf("current-state-%s", state),
				Desired: fmt.Sprintf("desired-state-%s", state),
				Machine: []fleet.MachineStatus{
					{
						ID:            fmt.Sprintf("machine-ID-%s", sliceID),
						IP:            net.ParseIP(fmt.Sprintf("10.0.0.%s", sliceID)),
						SystemdActive: fmt.Sprintf("systemd-active-state-%s", state),
						UnitHash:      "1234",
					},
				},
				Name: fmt.Sprintf("name-%d@%s.service", i, sliceID),
			}

			unitStatusList = append(unitStatusList, unitStatus)
		}
	}

	return unitStatusList
}

func expectedUnitStatusList(configs ...map[string][]string) UnitStatusList {
	unitStatusList := UnitStatusList{}

	i := 0
	for _, c := range configs {
		for j, sliceID := range c["sliceIDs"] {
			state := c["states"][j]
			name := c["names"][j]
			i++

			unitStatus := fleet.UnitStatus{
				Current: fmt.Sprintf("current-state-%s", state),
				Desired: fmt.Sprintf("desired-state-%s", state),
				Machine: []fleet.MachineStatus{
					{
						ID:            fmt.Sprintf("machine-ID-%s", sliceID),
						IP:            net.ParseIP(fmt.Sprintf("10.0.0.%s", sliceID)),
						SystemdActive: fmt.Sprintf("systemd-active-state-%s", state),
						UnitHash:      "1234",
					},
				},
				Name: name,
			}
			unitStatusList = append(unitStatusList, unitStatus)
		}
	}

	return unitStatusList
}

func Test_UnitStatusList_Group(t *testing.T) {
	testCases := []struct {
		Error    error
		Input    UnitStatusList
		Expected UnitStatusList
	}{
		// This test ensures that creating our own status structures works as
		// expected.
		{
			Error: nil,
			Input: inputUnitStatusList(
				map[string][]string{"sliceIDs": {"1", "1"}, "states": {"1", "1"}},
				map[string][]string{"sliceIDs": {"2", "2"}, "states": {"2", "2"}},
			),
			Expected: expectedUnitStatusList(
				map[string][]string{"sliceIDs": {"1"}, "states": {"1"}, "names": {"*"}},
				map[string][]string{"sliceIDs": {"2"}, "states": {"2"}, "names": {"*"}},
			),
		},

		// This test ensures that different states expand the status list.
		{
			Error: nil,
			Input: inputUnitStatusList(
				map[string][]string{"sliceIDs": {"1", "1"}, "states": {"1", "2"}}, // the last state differs
				map[string][]string{"sliceIDs": {"2", "2"}, "states": {"2", "2"}},
			),
			Expected: expectedUnitStatusList(
				map[string][]string{"sliceIDs": {"1", "1"}, "states": {"1", "2"}, "names": {"name-1@1.service", "name-2@1.service"}}, // the states expand
				map[string][]string{"sliceIDs": {"2"}, "states": {"2"}, "names": {"*"}},
			),
		},
	}

	for i, testCase := range testCases {
		output, err := testCase.Input.Group()
		if err != nil {
			t.Fatalf("UnitStatusList.Group returned error: %#v", err)
		}

		if !reflect.DeepEqual(output, testCase.Expected) {
			t.Fatalf("test case %d: grouped status list '%#v' is not equal to expected status list '%#v'", i+1, output, testCase.Expected)
		}
	}
}

func Test_Status_AggregateStatus(t *testing.T) {
	testCases := []struct {
		FC           string
		FD           string
		SA           string
		SS           string
		ErrorMatcher func(err error) bool
		Expected     Status
	}{
		{
			FC:           "inactive",
			FD:           "",
			SA:           "",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStopped,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "inactive",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStopped,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "inactive",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStopped,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "failed",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusFailed,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "failed",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusFailed,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "activating",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStarting,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "activating",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStarting,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "deactivating",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStopping,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "deactivating",
			SS:           "",
			ErrorMatcher: nil,
			Expected:     StatusStopping,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "active",
			SS:           "stop-sigterm",
			ErrorMatcher: nil,
			Expected:     StatusStopping,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "reloading",
			SS:           "stop-post",
			ErrorMatcher: nil,
			Expected:     StatusStopping,
		},
		{
			FC:           "loaded",
			FD:           "",
			SA:           "reloading",
			SS:           "launched",
			ErrorMatcher: nil,
			Expected:     StatusStarting,
		},
		{
			FC:           "launched",
			FD:           "",
			SA:           "active",
			SS:           "exited",
			ErrorMatcher: nil,
			Expected:     StatusRunning,
		},
		{
			FC:           "foo",
			FD:           "",
			SA:           "bar",
			SS:           "baz",
			ErrorMatcher: IsInvalidUnitStatus,
			Expected:     "",
		},
		{
			FC:           "inactive",
			FD:           "launched",
			SA:           "active",
			SS:           "running",
			ErrorMatcher: nil,
			Expected:     StatusRunning,
		},
	}

	for i, testCase := range testCases {
		aggregator := Aggregator{
			Logger: logging.NewLogger(logging.DefaultConfig()),
		}
		output, err := aggregator.AggregateStatus(testCase.FC, testCase.FD, testCase.SA, testCase.SS)
		if testCase.ErrorMatcher != nil {
			m := testCase.ErrorMatcher(err)
			if !m {
				t.Fatalf("test case %d: expected %t got %t", i+1, !m, m)
			}
		} else if err != nil {
			t.Fatalf("test case %d: expected %#v got %#v", i+1, nil, err)
		}

		if output != testCase.Expected {
			t.Fatalf("test case %d: expected %s got %s", i+1, testCase.Expected, output)
		}
	}
}
