package commands

import (
	"bytes"
	"fmt"
	"go/build"
	"os"
	"runtime"
	"strconv"

	"nvim-go/gb"
	"nvim-go/nvim"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/neovim-go/vim"
	"github.com/garyburd/neovim-go/vim/plugin"
	"github.com/rogpeppe/godef/go/ast"
	"github.com/rogpeppe/godef/go/parser"
	"github.com/rogpeppe/godef/go/printer"
	"github.com/rogpeppe/godef/go/token"
	"github.com/rogpeppe/godef/go/types"
)

func init() {
	plugin.HandleCommand("Godef", &plugin.CommandOptions{Range: "%", Eval: "expand('%:p:h:h:h:h')"}, Def)
}

func Def(v *vim.Vim, r [2]int, file string) error {
	defer gb.WithGoBuildForPath(file)()

	b, err := v.CurrentBuffer()
	if err != nil {
		return v.WriteErr("cannot get current buffer")
	}

	buf, err := v.BufferLineSlice(b, 0, -1, true, true)
	if err != nil {
		return err
	}
	src := bytes.Join(buf, []byte{'\n'})

	filename, err := v.BufferName(b)
	if err != nil {
		return v.WriteErr("cannot get current buffer name")
	}

	searchpos, err := nvim.ByteOffset(v)
	if err != nil {
		return v.WriteErr("cannot get current buffer byte offset")
	}

	pkgScope := ast.NewScope(parser.Universe)
	f, err := parser.ParseFile(types.FileSet, filename, src, 0, pkgScope)
	if f == nil {
		nvim.Echomsg(v, "cannot parse %s: %v", filename, err)
	}

	o := findIdentifier(v, f, searchpos)

	switch e := o.(type) {
	case *ast.ImportSpec:
		path := importPath(e)
		pkg, err := build.Default.Import(path, "", build.FindOnly)
		if err != nil {
			fail("error finding import path for %s: %s", path, err)
		}
		fmt.Println(pkg.Dir)
	case ast.Expr:
		if obj, _ := types.ExprType(e, types.DefaultImporter); obj != nil {
			out := types.FileSet.Position(types.DeclPos(obj))
			log.Debugln("def out.String():", out.String())
			log.Debugln("def out.Filename:", out.Filename)
			log.Debugln("def out.Line:", out.Line)
			log.Debugln("def out.Column:", out.Column)
			log.Debugln("def out.Offset:", out.Offset)
			log.Debugln("def out.IsValid():", out.IsValid())

			v.Command("silent lexpr '" + fmt.Sprintf("%v", out) + "'")
			w, err := v.CurrentWindow()
			if err != nil {
				log.Debugln(err)
			}
			v.SetWindowCursor(w, [2]int{out.Line, out.Column - 1})
			// v.Command("silent lgetexpr '" + fmt.Sprintf("%v", out) + "'")
			// v.Command("sil ll 1")
			// v.Feedkeys("zz", "normal", false)
		}
		// fail("no declaration found for %v", pretty{e})
	}
	return nil
}

func fail(s string, a ...interface{}) {
	fmt.Fprint(os.Stderr, "godef: "+fmt.Sprintf(s, a...)+"\n")
	os.Exit(2)
}

func importPath(n *ast.ImportSpec) string {
	p, err := strconv.Unquote(n.Path.Value)
	if err != nil {
		fail("invalid string literal %q in ast.ImportSpec", n.Path.Value)
	}
	return p
}

// findIdentifier looks for an identifier at byte-offset searchpos
// inside the parsed source represented by node.
// If it is part of a selector expression, it returns
// that expression rather than the identifier itself.
//
// As a special case, if it finds an import spec, it returns ImportSpec.
func findIdentifier(v *vim.Vim, f *ast.File, searchpos int) ast.Node {
	ec := make(chan ast.Node)
	found := func(startPos, endPos token.Pos) bool {
		start := types.FileSet.Position(startPos).Offset
		end := start + int(endPos-startPos)
		return start <= searchpos && searchpos <= end
	}
	go func() {
		var visit func(ast.Node) bool
		visit = func(n ast.Node) bool {
			var startPos token.Pos
			switch n := n.(type) {
			default:
				return true
			case *ast.Ident:
				startPos = n.NamePos
			case *ast.SelectorExpr:
				startPos = n.Sel.NamePos
			case *ast.ImportSpec:
				startPos = n.Pos()
			case *ast.StructType:
				// If we find an anonymous bare field in a
				// struct type, its definition points to itself,
				// but we actually want to go elsewhere,
				// so assume (dubiously) that the expression
				// works globally and return a new node for it.
				for _, field := range n.Fields.List {
					if field.Names != nil {
						continue
					}
					t := field.Type
					if pt, ok := field.Type.(*ast.StarExpr); ok {
						t = pt.X
					}
					if id, ok := t.(*ast.Ident); ok {
						if found(id.NamePos, id.End()) {
							ec <- parseExpr(f.Scope, id.Name)
							runtime.Goexit()
						}
					}
				}
				return true
			}
			if found(startPos, n.End()) {
				ec <- n
				runtime.Goexit()
			}
			return true
		}
		ast.Walk(FVisitor(visit), f)
		ec <- nil
	}()
	ev := <-ec
	if ev == nil {
		log.Debugln("def: no identifier found")
		nvim.Echomsg(v, "def: no identifier found")
		// fail("no identifier found")
	}
	return ev
}

func typeStr(obj *ast.Object, typ types.Type) string {
	switch obj.Kind {
	case ast.Fun, ast.Var:
		return fmt.Sprintf("%s %v", obj.Name, prettyType{typ})
	case ast.Pkg:
		return fmt.Sprintf("import (%s %s)", obj.Name, typ.Node.(*ast.ImportSpec).Path.Value)
	case ast.Con:
		if decl, ok := obj.Decl.(*ast.ValueSpec); ok {
			return fmt.Sprintf("const %s %v = %s", obj.Name, prettyType{typ}, pretty{decl.Values[0]})
		}
		return fmt.Sprintf("const %s %v", obj.Name, prettyType{typ})
	case ast.Lbl:
		return fmt.Sprintf("label %s", obj.Name)
	case ast.Typ:
		typ = typ.Underlying(false, types.DefaultImporter)
		return fmt.Sprintf("type %s %v", obj.Name, prettyType{typ})
	}
	return fmt.Sprintf("unknown %s %v", obj.Name, typ.Kind)
}

func parseExpr(s *ast.Scope, expr string) ast.Expr {
	n, err := parser.ParseExpr(types.FileSet, "<arg>", expr, s)
	if err != nil {
		fail("cannot parse expression: %v", err)
	}
	switch n := n.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return n
	}
	fail("no identifier found in expression")
	return nil
}

type FVisitor func(n ast.Node) bool

func (f FVisitor) Visit(n ast.Node) ast.Visitor {
	if f(n) {
		return f
	}
	return nil
}

// pkgName returns the package name implemented by the go source filename
func pkgName(filename string) string {
	prog, _ := parser.ParseFile(types.FileSet, filename, nil, parser.PackageClauseOnly, nil)
	if prog != nil {
		return prog.Name.Name
	}
	return ""
}

func hasSuffix(s, suff string) bool {
	return len(s) >= len(suff) && s[len(s)-len(suff):] == suff
}

type pretty struct {
	n interface{}
}

func (p pretty) String() string {
	var b bytes.Buffer
	printer.Fprint(&b, types.FileSet, p.n)
	return b.String()
}

type prettyType struct {
	n types.Type
}

func (p prettyType) String() string {
	// TODO print path package when appropriate.
	// Current issues with using p.n.Pkg:
	//	- we should actually print the local package identifier
	//	rather than the package path when possible.
	//	- p.n.Pkg is non-empty even when
	//	the type is not relative to the package.
	return pretty{p.n.Node}.String()
}
