package main

import (
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	dirty, dirtyErr                               = git.PlainOpen("test/local-dirty")
	upToDate, upToDateErr                         = git.PlainOpen("test/up-to-date")
	noUpstream, noUpstreamErr                     = git.PlainOpen("test/no-upstream")
	notUpstreamed, notUpstreamedErr               = git.PlainOpen("test/not-upstreamed")
	behindUpstream, behindUpstreamErr             = git.PlainOpen("test/behind-upstream")
	behindUpstreamMaster, behindUpstreamMasterErr = git.PlainOpen("test/behind-upstream-master")
	featBehindUpstream, featBehindUpstreamErr     = git.PlainOpen("test/feat-behind-upstream")
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
		{
			name: "not-upstreamed",
			args: args{
				notUpstreamed,
				"main",
			},
			want: false,
			err:  nil,
		},
		{
			name: "behind upstream",
			args: args{
				behindUpstream,
				"main",
			},
			want: true,
			err:  nil,
		},
		{
			name: "behind upstream with feature branch",
			args: args{
				featBehindUpstream,
				"feat",
			},
			want: false,
			err:  plumbing.ErrReferenceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isBranchUpstreamed(tt.args.repo, tt.args.branchName, tt.args.branchName)
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

func Test_isUpstreamed(t *testing.T) {
	type args struct {
		repo     *git.Repository
		repoPath string
		ref      *plumbing.Reference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "feat is behind upstream main",
			args: args{
				repo:     featBehindUpstream,
				repoPath: "doesnt even matter",
			},
			want: true,
		},
		{
			name: "feat is upstreamed in main",
			args: args{
				repo:     behindUpstream,
				repoPath: "doesnt even matter",
			},
			want: true,
		},
		{
			name: "feat is upstreamed in master",
			args: args{
				repo:     behindUpstreamMaster,
				repoPath: "doesnt even matter",
			},
			want: true,
		},
		{
			name: "not upstreamed in main",
			args: args{
				repo:     notUpstreamed,
				repoPath: "doesnt even matter",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpstreamed(tt.args.repo, tt.args.repoPath); got != tt.want {
				t.Errorf("isUpstreamed() = %v, want %v", got, tt.want)
			}
		})
	}
}
