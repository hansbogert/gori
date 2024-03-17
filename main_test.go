package main

import (
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	upToDate, upToDateErr     = git.PlainOpen("test/up-to-date")
	noUpstream, noUpstreamErr = git.PlainOpen("test/no-upstream")
)

func Test_isBranchUpstreamed(t *testing.T) {
	type args struct {
		repo       *git.Repository
		branchName string
	}
	tests := []struct {
		name string
		args args
		want bool
		err  error
	}{
		{
			name: "no-upstream",
			args: args{
				noUpstream,
				"main",
			},
			want: false,
			err:  plumbing.ErrReferenceNotFound,
		},
		{
			name: "up-to-date",
			args: args{
				upToDate,
				"main",
			},
			want: true,
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isBranchUpstreamed(tt.args.repo, tt.args.branchName)
			if err != tt.err {
				t.Errorf("isBranchUpstreamed() error = %v, expected err = %v", err, tt.err)
				return
			}
			if got != tt.want {
				t.Errorf("isBranchUpstreamed() = %v, expected =  %v", got, tt.want)
			}
		})
	}
}
