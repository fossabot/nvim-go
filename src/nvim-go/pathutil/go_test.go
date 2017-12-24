// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathutil_test

import (
	"go/build"
	"nvim-go/pathutil"
	"path/filepath"
	"testing"
)

func TestPackagePath(t *testing.T) {
	var gopath = filepath.Join("testdata", "go")

	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "package main (main.go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "testmain")},
			want:    filepath.Join(gopath, "src", "foo.org", "testmain"),
			wantErr: false,
		},
		{
			name:    "package foo (exists go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo")},
			want:    filepath.Join(gopath, "src", "foo.org", "foo"),
			wantErr: false,
		},
		{
			name:    "not exists go file(use parent dir)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar")},
			want:    filepath.Join(gopath, "src", "foo.org", "foo"),
			wantErr: false,
		},
		{
			name:    "package baz (parent dir is no go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz")},
			want:    filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz"),
			wantErr: false,
		},
		{
			name:    "package qux (parent dir is package)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz", "qux")},
			want:    filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz", "qux"),
			wantErr: false,
		},
		{
			name:    "no such file or directory",
			args:    args{dir: filepath.Join("nosuch", "src", "foo.org", "notexists")},
			want:    "",
			wantErr: true,
		},
		{
			name:    "GOPATH directory",
			args:    args{dir: gopath},
			want:    "",
			wantErr: true,
		},
		{
			name:    "GOROOT directory",
			args:    args{dir: build.Default.GOROOT},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		build.Default.GOPATH = gopath

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := pathutil.PackagePath(tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackagePath(%v) error = %v, wantErr %v", tt.args.dir, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PackagePath(%v) = got: %v, want %v", tt.args.dir, got, tt.want)
			}
		})
	}
}

func TestPackageID(t *testing.T) {
	var gopath = filepath.Join("testdata", "go")

	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "package main (main.go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "testmain")},
			want:    filepath.Join("foo.org", "testmain"),
			wantErr: false,
		},
		{
			name:    "package foo (exists go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo")},
			want:    filepath.Join("foo.org", "foo"),
			wantErr: false,
		},
		{
			name:    "not exists go file(use parent dir)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar")},
			want:    filepath.Join("foo.org", "foo"),
			wantErr: false,
		},
		{
			name:    "package baz (parent dir is no go file)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz")},
			want:    filepath.Join("foo.org", "foo", "bar", "baz"),
			wantErr: false,
		},
		{
			name:    "package qux (parent dir is package)",
			args:    args{dir: filepath.Join(gopath, "src", "foo.org", "foo", "bar", "baz", "qux")},
			want:    filepath.Join("foo.org", "foo", "bar", "baz", "qux"),
			wantErr: false,
		},
		{
			name:    "no such file or directory",
			args:    args{dir: filepath.Join("nosuch", "src", "foo.org", "notexists")},
			want:    "",
			wantErr: true,
		},
		{
			name:    "GOPATH directory",
			args:    args{dir: gopath},
			want:    "",
			wantErr: true,
		},
		{
			name:    "GOROOT directory",
			args:    args{dir: build.Default.GOROOT},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		build.Default.GOPATH = gopath

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := pathutil.PackageID(tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackageID(%v) error = %v, wantErr %v", tt.args.dir, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PackageID(%v) = %v, want %v", tt.args.dir, got, tt.want)
			}
		})
	}
}

func TestPackageRoot(t *testing.T) {
	t.Skip("for now")

	gopath := filepath.Join("testdata", "go", "src")
	build.Default.GOPATH = gopath

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *build.Package
		wantErr bool
	}{
		{
			name:    "default",
			args:    args{filepath.Join(gopath, "foo.org", "foo", "foo.go")},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := pathutil.PackageRoot(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackageRoot(%v) error = %#v, wantErr %#v", tt.args.path, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PackageRoot(%v) got: %#v, want: %#v", tt.args.path, got, tt.want)
			}
			t.Logf("PackageRoot(%#v) got: %#v, want: %#v", tt.args.path, got, tt.want)
		})
	}
}
