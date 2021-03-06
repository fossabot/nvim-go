package command

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/neovim/go-client/nvim"
	"github.com/zchee/nvim-go/src/buildctx"
	"github.com/zchee/nvim-go/src/nvimutil"
	"github.com/zchee/nvim-go/src/testutil"
	"golang.org/x/sync/syncmap"
)

func TestCommand_Build(t *testing.T) {
	ctx := testutil.TestContext(context.Background())

	type fields struct {
		ctx       context.Context
		Nvim      *nvim.Nvim
		buildctxt *buildctx.Context
		errs      *syncmap.Map
	}
	type args struct {
		args []string
		bang bool
		eval *CmdBuildEval
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "go/nvim-go",
			fields: fields{
				ctx:  ctx,
				Nvim: nvimutil.TestNvim(t, "."),
				buildctxt: &buildctx.Context{
					Build: buildctx.Build{
						Tool:        "go",
						ProjectRoot: "",
					},
				},
			},
			args: args{
				args: nil,
				eval: &CmdBuildEval{
					Cwd:  testdataPath,
					File: testdataPath,
				},
			},
			wantErr: false,
		},
		{
			name: "gsftp",
			fields: fields{
				ctx:  ctx,
				Nvim: nvimutil.TestNvim(t, gsftpRoot),
				buildctxt: &buildctx.Context{
					Build: buildctx.Build{
						Tool:        "gb",
						ProjectRoot: gsftpRoot,
					},
				},
			},
			args: args{
				args: nil,
				eval: &CmdBuildEval{
					Cwd:  gsftpRoot,
					File: gsftpRoot,
				},
			},
			wantErr: false,
		},
		{
			name: "correct (astdump)",
			fields: fields{
				ctx:  ctx,
				Nvim: nvimutil.TestNvim(t, filepath.Join(astdump, "astdump.go")), // correct file
				buildctxt: &buildctx.Context{
					Build: buildctx.Build{
						Tool:        "go",
						ProjectRoot: astdump,
					},
				},
			},
			args: args{
				args: nil,
				eval: &CmdBuildEval{
					Cwd:  astdump,
					File: filepath.Join(astdump, "astdump.go"),
				},
			},
			wantErr: false,
		},
		{
			name: "broken (broken)",
			fields: fields{
				ctx:  ctx,
				Nvim: nvimutil.TestNvim(t, brokenMain), // broken file
				buildctxt: &buildctx.Context{
					Build: buildctx.Build{
						Tool:        "gb",
						ProjectRoot: broken,
					},
				},
			},
			args: args{
				args: nil,
				eval: &CmdBuildEval{
					Cwd:  broken,
					File: brokenMain,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCommand(tt.fields.ctx, tt.fields.Nvim, tt.fields.buildctxt)
			err := c.Build(tt.args.args, tt.args.bang, tt.args.eval)
			switch e := err.(type) {
			case error:
				if (err != nil) != tt.wantErr {
					t.Errorf("err: %v, wantErr %v", e, tt.wantErr)
				}
			case []*nvim.QuickfixError:
				if (len(e) != 0) != tt.wantErr {
					t.Errorf("%q. Commands.Build(%v, %v)", tt.name, tt.args.bang, tt.args.eval)
				}
			}
		})
	}
}

func BenchmarkBuildGo(b *testing.B) {
	ctx := testutil.TestContext(context.Background())
	buildctxt := buildctx.NewContext()
	c := NewCommand(ctx, benchVim(b, astdumpMain), buildctxt)

	for i := 0; i < b.N; i++ {
		c.Build(nil, false, &CmdBuildEval{
			Cwd:  astdump,
			File: astdump,
		})
		if len(c.buildContext.Errlist) != 0 {
			b.Errorf("BenchmarkBuildGo: %v", c.buildContext.Errlist)
		}
	}
}

func BenchmarkBuildGb(b *testing.B) {
	ctx := testutil.TestContext(context.Background())
	buildctxt := buildctx.NewContext()
	c := NewCommand(ctx, benchVim(b, gsftpMain), buildctxt)

	for i := 0; i < b.N; i++ {
		c.Build(nil, false, &CmdBuildEval{
			Cwd:  gsftpRoot,
			File: gsftpRoot,
		})
		if len(c.buildContext.Errlist) != 0 {
			b.Errorf("BenchmarkBuildGb: %v", c.buildContext.Errlist)
		}
	}
}

// func TestCommand_compileCmd(t *testing.T) {
// 	gobinary, err := exec.LookPath("go")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	gbbinary, err := exec.LookPath("gb")
// 	if err != nil {
// 		t.Error(err)
// 	}
//
// 	type fields struct {
// 		Nvim  *nvim.Nvim
// 		ctx  *ctx.Context
// 	}
// 	type args struct {
// 		bang bool
// 		dir  string
// 	}
// 	tests := []struct {
// 		name     string
// 		fields   fields
// 		args     args
// 		want     string
// 		wantErr  bool
// 		testfile bool
// 	}{
// 		{
// 			name: "astdump (go build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: astdump,
// 			},
// 			want:    gobinary,
// 			wantErr: false,
// 		},
// 		{
// 			name: "nvim-go (gb build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: "testdata",
// 			},
// 			want:    gbbinary,
// 			wantErr: false,
// 		},
// 		{
// 			name: "gsftp (gb build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: gsftpRoot,
// 			},
// 			want:    gbbinary,
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			c := NewCommand(tt.fields.Nvim, tt.fields.ctx)
// 			c.ctx.SetContext(filepath.Dir(tt.args.dir))
//
// 			got, err := c.compileCmd(tt.args.bang, tt.args.dir, tt.testfile)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Commands.compileCmd(%v, %v) error = %v, wantErr %v", tt.args.bang, tt.args.dir, err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got.Args[0], tt.want) {
// 				t.Errorf("Commands.compileCmd(%v, %v) = %v, want %v", tt.args.bang, tt.args.dir, got, tt.want)
// 			}
// 		})
// 	}
// }
