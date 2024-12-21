package collector

import (
	"math"
	"testing"
)

func TestSplitStringToFloats(t *testing.T) {
	testCases := []struct {
		input    string
		expected struct {
			f1 float64
			f2 float64
		}
		isNaN    bool
		hasError bool
	}{
		{
			"1.2,2.1",
			struct {
				f1 float64
				f2 float64
			}{
				1.2,
				2.1,
			},
			false,
			false,
		},
		{
			input:    "1.2,",
			isNaN:    true,
			hasError: true,
		},
		{
			input:    ",2.1",
			isNaN:    true,
			hasError: true,
		},
		{
			"1.2,2.1,3.2",
			struct {
				f1 float64
				f2 float64
			}{
				1.2,
				2.1,
			},
			false,
			false,
		},
		{
			input:    "",
			isNaN:    true,
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			f1, f2, err := splitStringToFloats(testCase.input)

			if testCase.hasError && err == nil {
				t.Fatalf("expected an error but got nil")
			} else if !testCase.hasError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if testCase.isNaN {
				if !math.IsNaN(f1) {
					t.Errorf("expected f1 to be NaN")
				}
				if !math.IsNaN(f2) {
					t.Errorf("expected f2 to be NaN")
				}
			} else {
				if testCase.expected.f1 != f1 {
					t.Errorf("expected value %f, got %f", testCase.expected.f1, f1)
				}
				if testCase.expected.f2 != f2 {
					t.Errorf("expected value %f, got %f", testCase.expected.f2, f2)
				}
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	testCases := []struct {
		input    string
		output   float64
		hasError bool
	}{
		{
			"3d3h42m53s",
			272573,
			false,
		},
		{
			"15w3d3h42m53s",
			9344573,
			false,
		},
		{
			"42m53s",
			2573,
			false,
		},
		{
			"7w6d9h34m",
			4786440,
			false,
		},
		{
			"59",
			0,
			true,
		},
		{
			"s",
			0,
			false,
		},
		{
			"",
			0,
			false,
		},
	}

	for _, testCase := range testCases {
		f, err := parseDuration(testCase.input)

		if testCase.hasError && err == nil {
			t.Fatalf("expected an error but got nil")
		} else if !testCase.hasError && err != nil {
			t.Fatalf("expected no error but got: %v", err)
		}

		if testCase.output != f {
			t.Errorf("expected %f, got %f", testCase.output, f)
		}
	}
}
