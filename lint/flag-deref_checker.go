package lint

import (
	"go/ast"
)

func flagDerefCheck(ctx *context) func(*ast.File) {
	return wrapLocalExprChecker(&flagDerefChecker{
		baseLocalExprChecker: baseLocalExprChecker{ctx: ctx},

		flagPtrFuncs: map[string]bool{
			"flag.Bool":     true,
			"flag.Duration": true,
			"flag.Float64":  true,
			"flag.Int":      true,
			"flag.Int64":    true,
			"flag.String":   true,
			"flag.Uint":     true,
			"flag.Uint64":   true,
		},
	})
}

type flagDerefChecker struct {
	// TODO(quasilyte): should be global expr checker. Refs #124.
	baseLocalExprChecker

	flagPtrFuncs map[string]bool
}

func (c *flagDerefChecker) CheckLocalExpr(expr ast.Expr) {
	if expr, ok := expr.(*ast.StarExpr); ok {
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return
		}
		called := functionName(call)
		if c.flagPtrFuncs[called] {
			c.warn(expr, called+"Var")
		}
	}
}

func (c *flagDerefChecker) warn(x ast.Node, suggestion string) {
	c.ctx.Warn(x, "immediate deref in %s is most likely an error; consider using %s",
		nodeString(c.ctx.FileSet, x), suggestion)
}
