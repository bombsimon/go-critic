package checkers

import (
	"go/ast"
	"go/types"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
	"github.com/go-toolsmith/astcast"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "deferUnlambda"
	info.Tags = []string{"style", "experimental"}
	info.Summary = "Detects deferred function literals that can be simplified"
	info.Before = `defer func() { f() }()`
	info.After = `f()`

	collection.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		return astwalk.WalkerForStmt(&deferUnlambdaChecker{ctx: ctx})
	})
}

type deferUnlambdaChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext
}

func (c *deferUnlambdaChecker) VisitStmt(x ast.Stmt) {
	def, ok := x.(*ast.DeferStmt)
	if !ok {
		return
	}

	// We don't analyze deferred function args.
	// Most deferred calls don't have them, so it's not a big deal to skip them.
	if len(def.Call.Args) != 0 {
		return
	}

	fn, ok := def.Call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}

	if len(fn.Body.List) != 1 {
		return
	}

	call, ok := astcast.ToExprStmt(fn.Body.List[0]).X.(*ast.CallExpr)
	if !ok || !c.isFunctionCall(call) {
		return
	}

	// Skip recover() as it can't be moved outside of the lambda.
	// Skip panic() to avoid affecting the stack trace.
	switch qualifiedName(call.Fun) {
	case "recover", "panic":
		return
	}

	for _, arg := range call.Args {
		if !c.isConstExpr(arg) {
			return
		}
	}

	c.warn(def, call)
}

func (c *deferUnlambdaChecker) isFunctionCall(e *ast.CallExpr) bool {
	switch fnExpr := e.Fun.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		x, ok := fnExpr.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = c.ctx.TypesInfo.ObjectOf(x).(*types.PkgName)
		return ok
	default:
		return false
	}
}

func (c *deferUnlambdaChecker) isConstExpr(e ast.Expr) bool {
	return c.ctx.TypesInfo.Types[e].Value != nil
}

func (c *deferUnlambdaChecker) warn(cause, suggestion ast.Node) {
	c.ctx.Warn(cause, "can rewrite as `defer %s`", suggestion)
}
