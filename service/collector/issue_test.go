package collector

import (
	"strconv"
	"testing"

	"github.com/giantswarm/to"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/github"
)

func Test_Collector_Issue_hasLabels(t *testing.T) {
	testCases := []struct {
		name           string
		issue          *github.Issue
		selector       string
		expectedResult bool
	}{
		{
			name: "case 0 issue labels do not match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
				},
			},
			selector:       "two",
			expectedResult: false,
		},
		{
			name: "case 1 issue labels do not match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "four",
			expectedResult: false,
		},
		{
			name: "case 2 issue labels do match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "one",
			expectedResult: true,
		},
		{
			name: "case 3 issue labels do match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "two",
			expectedResult: true,
		},
		{
			name: "case 4 issue labels do match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "one,two",
			expectedResult: true,
		},
		{
			name: "case 5 issue labels do match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "two,one",
			expectedResult: true,
		},
		{
			name: "case 6 issue labels do not match selector",
			issue: &github.Issue{
				Labels: []github.Label{
					{Name: to.StringP("one")},
					{Name: to.StringP("two")},
					{Name: to.StringP("three")},
				},
			},
			selector:       "one,four",
			expectedResult: false,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := hasLabels(tc.issue, tc.selector)

			if result != tc.expectedResult {
				t.Fatalf("\n\n%s\n", cmp.Diff(result, tc.expectedResult))
			}
		})
	}
}
