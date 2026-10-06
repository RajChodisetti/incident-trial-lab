package candidates

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
)

// Embed the actual candidate implementation at build time; no arbitrary path,
// fixture assertions, expected outcomes or credentials are read at runtime.
//
//go:embed service.go
var serviceSource []byte

type Change struct {
	SourceSHA256 string `json:"source_sha256"`
	InitialLogic string `json:"initial_logic"`
	Logic        string `json:"logic"`
	LogicSHA256  string `json:"logic_sha256"`
}

// Changes specializes only the current, restricted quote-arithmetic branch.
// Source drift outside the supported expression/branch shape fails closed.
// Owner-side map keys are semantic IDs and must be replaced before disclosure.
func Changes() (map[string]Change, error) { return changesFromSource(serviceSource) }

func changesFromSource(source []byte) (map[string]Change, error) {
	if len(source) > 16384 {
		return nil, errors.New("candidate source exceeds bound")
	}
	f, err := parser.ParseFile(token.NewFileSet(), "", source, 0)
	if err != nil {
		return nil, errors.New("candidate source is not parsable")
	}
	var initial, condition, replacement ast.Expr
	count := 0
	totalUses, directOutputs := 0, 0
	ast.Inspect(f, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == "total" {
			totalUses++
		}
		if value, ok := node.(*ast.CompositeLit); ok && len(value.Elts) == 1 {
			kind, isSelector := value.Type.(*ast.SelectorExpr)
			if isSelector && kind.Sel.Name == "Total" {
				if field, ok := value.Elts[0].(*ast.KeyValueExpr); ok {
					key, isKey := field.Key.(*ast.Ident)
					id, isID := field.Value.(*ast.Ident)
					if isKey && key.Name == "TotalCents" && isID && id.Name == "total" {
						directOutputs++
					}
				}
			}
		}
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, statement := range block.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || !assignsTotal(assignment, token.DEFINE) || i+1 >= len(block.List) {
				continue
			}
			branch, ok := block.List[i+1].(*ast.IfStmt)
			if !ok || branch.Init != nil || branch.Else != nil || len(branch.Body.List) != 1 {
				continue
			}
			update, ok := branch.Body.List[0].(*ast.AssignStmt)
			if !ok || !assignsTotal(update, token.ASSIGN) {
				continue
			}
			count++
			initial, condition, replacement = assignment.Rhs[0], branch.Cond, update.Rhs[0]
		}
		return true
	})
	if count != 1 || totalUses != 3 || directOutputs != 1 {
		return nil, errors.New("candidate arithmetic shape changed")
	}
	sourceHash := sha256.Sum256(source)
	changes := map[string]Change{}
	for _, id := range IDs() {
		before, err := safeExpression(initial, id)
		if err != nil {
			return nil, err
		}
		after, err := safeExpression(replacement, id)
		if err != nil {
			return nil, err
		}
		predicate, err := safeExpression(condition, id)
		if err != nil {
			return nil, err
		}
		initialCode := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("total")}, Tok: token.DEFINE, Rhs: []ast.Expr{before}}
		code := []ast.Stmt{initialCode}
		if isBool(predicate, "true") {
			code = []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("total")}, Tok: token.DEFINE, Rhs: []ast.Expr{after}}}
		} else if !isBool(predicate, "false") {
			code = append(code, &ast.IfStmt{Cond: predicate, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("total")}, Tok: token.ASSIGN, Rhs: []ast.Expr{after}}}}})
		}
		initialText, err := printCode([]ast.Stmt{initialCode})
		if err != nil {
			return nil, err
		}
		text, err := printCode(code)
		if err != nil {
			return nil, err
		}
		logicHash := sha256.Sum256([]byte(text))
		changes[id] = Change{SourceSHA256: hex.EncodeToString(sourceHash[:]), InitialLogic: initialText, Logic: text, LogicSHA256: hex.EncodeToString(logicHash[:])}
	}
	return changes, nil
}

func assignsTotal(a *ast.AssignStmt, op token.Token) bool {
	if a.Tok != op || len(a.Lhs) != 1 || len(a.Rhs) != 1 {
		return false
	}
	id, ok := a.Lhs[0].(*ast.Ident)
	return ok && id.Name == "total"
}

func printCode(code []ast.Stmt) (string, error) {
	var out bytes.Buffer
	if format.Node(&out, token.NewFileSet(), &ast.BlockStmt{List: code}) != nil || out.Len() > 1024 {
		return "", errors.New("candidate code projection exceeds bound")
	}
	return out.String(), nil
}

func isBool(e ast.Expr, value string) bool { id, ok := e.(*ast.Ident); return ok && id.Name == value }

// No calls, strings, paths, arbitrary identifiers or comments survive projection.
func safeExpression(e ast.Expr, variant string) (ast.Expr, error) {
	switch n := e.(type) {
	case *ast.ParenExpr:
		x, err := safeExpression(n.X, variant)
		if err != nil {
			return nil, err
		}
		if isBool(x, "true") || isBool(x, "false") {
			return x, nil
		}
		return &ast.ParenExpr{X: x}, nil
	case *ast.SelectorExpr:
		q, ok := n.X.(*ast.Ident)
		if !ok || q.Name != "q" || n.Sel.Name != "UnitCents" && n.Sel.Name != "Quantity" && n.Sel.Name != "DiscountPercent" {
			break
		}
		return &ast.SelectorExpr{X: ast.NewIdent("q"), Sel: ast.NewIdent(n.Sel.Name)}, nil
	case *ast.BasicLit:
		if n.Kind != token.INT || len(n.Value) > 20 {
			break
		}
		if _, err := strconv.ParseInt(n.Value, 10, 64); err != nil {
			break
		}
		return &ast.BasicLit{Kind: token.INT, Value: n.Value}, nil
	case *ast.BinaryExpr:
		if n.Op == token.EQL {
			if id, ok := n.X.(*ast.Ident); ok && id.Name == "variant" {
				value, ok := n.Y.(*ast.BasicLit)
				if !ok || value.Kind != token.STRING {
					break
				}
				label, err := strconv.Unquote(value.Value)
				if err != nil || !ValidID(label) {
					break
				}
				return ast.NewIdent(strconv.FormatBool(variant == label)), nil
			}
		}
		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.EQL, token.LAND, token.LOR:
		default:
			return nil, errors.New("unsupported candidate operator")
		}
		x, err := safeExpression(n.X, variant)
		if err != nil {
			return nil, err
		}
		y, err := safeExpression(n.Y, variant)
		if err != nil {
			return nil, err
		}
		if n.Op == token.LOR {
			if isBool(x, "true") || isBool(y, "true") {
				return ast.NewIdent("true"), nil
			}
			if isBool(x, "false") {
				return y, nil
			}
			if isBool(y, "false") {
				return x, nil
			}
		}
		if n.Op == token.LAND {
			if isBool(x, "false") || isBool(y, "false") {
				return ast.NewIdent("false"), nil
			}
			if isBool(x, "true") {
				return y, nil
			}
			if isBool(y, "true") {
				return x, nil
			}
		}
		return &ast.BinaryExpr{X: x, Op: n.Op, Y: y}, nil
	}
	return nil, errors.New("unsupported candidate expression")
}
