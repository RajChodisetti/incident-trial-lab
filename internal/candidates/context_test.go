package candidates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

func TestChangesAreBoundedSourceFactsWithoutLabels(t *testing.T) {
	changes, err := Changes()
	if err != nil {
		t.Fatal(err)
	}
	sourceHash := sha256.Sum256(serviceSource)
	for _, id := range IDs() {
		c := changes[id]
		hash := sha256.Sum256([]byte(c.Logic))
		if c.SourceSHA256 != hex.EncodeToString(sourceHash[:]) || c.LogicSHA256 != hex.EncodeToString(hash[:]) || len(c.Logic) > 1024 || len(c.InitialLogic) > 1024 || !strings.Contains(c.Logic, "q.Quantity") {
			t.Fatal("missing bounded, hashed arithmetic context")
		}
		data, _ := json.Marshal(c)
		for _, label := range append(IDs(), "expected", "verdict", "RawResponse", "variant", "//") {
			if strings.Contains(string(data), label) {
				t.Fatalf("privileged label/comment leaked: %s", label)
			}
		}
	}
	if changes["ineffective"].Logic != changes["ineffective"].InitialLogic || changes["fixed"].Logic == changes["fixed"].InitialLogic || !strings.Contains(changes["misleading"].Logic, "q.UnitCents == 105") {
		t.Fatal("projection omits actual arithmetic differences")
	}
}

func TestChangesRejectUnsupportedSourceDrift(t *testing.T) {
	for _, source := range [][]byte{
		[]byte("package broken {"),
		[]byte(strings.Repeat(" ", 16385)),
		[]byte(strings.Replace(string(serviceSource), "q.UnitCents * (100 - q.DiscountPercent)", "unsafeCall() * (100 - q.DiscountPercent)", 1)),
		[]byte(strings.Replace(string(serviceSource), `variant == "fixed"`, `variant == "unknown"`, 1)),
		[]byte(strings.Replace(string(serviceSource), "w.Header().Set", "total += 1\n w.Header().Set", 1)),
		[]byte(strings.Replace(string(serviceSource), "TotalCents: total", "TotalCents: total + 1", 1)),
	} {
		if _, err := changesFromSource(source); err == nil {
			t.Fatal("unsupported source accepted")
		}
	}
}

// Compare the constrained projection against the actual HTTP handler, rather than
// an expected-verdict oracle. Expected selection answers are never provider input.
func TestProjectedLogicMatchesActualCandidateHandler(t *testing.T) {
	changes, err := Changes()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range IDs() {
		handler, err := Handler(id)
		if err != nil {
			t.Fatal(err)
		}
		for _, unit := range []int64{1, 105, 109, 1000000} {
			for _, quantity := range []int64{1, 3, 4, 100} {
				for _, discount := range []int64{0, 10, 15, 100} {
					q := synthetic.Quote{UnitCents: unit, Quantity: quantity, DiscountPercent: discount}
					body, _ := json.Marshal(q)
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, httptest.NewRequest("POST", "/quote", strings.NewReader(string(body))))
					var response synthetic.Total
					if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &response) != nil {
						t.Fatal("actual candidate request failed")
					}
					if got := runProjection(t, changes[id].Logic, q); got != response.TotalCents {
						t.Fatalf("projection differs from handler: candidate=%s input=%+v", id, q)
					}
				}
			}
		}
	}
}

func runProjection(t *testing.T, logic string, q synthetic.Quote) int64 {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "", "package projected\nfunc quote() "+logic, 0)
	if err != nil {
		t.Fatal(err)
	}
	var value func(ast.Expr) int64
	value = func(e ast.Expr) int64 {
		switch n := e.(type) {
		case *ast.ParenExpr:
			return value(n.X)
		case *ast.BasicLit:
			var number int64
			if json.Unmarshal([]byte(n.Value), &number) != nil {
				t.Fatal("invalid projected integer")
			}
			return number
		case *ast.SelectorExpr:
			switch n.Sel.Name {
			case "UnitCents":
				return q.UnitCents
			case "Quantity":
				return q.Quantity
			case "DiscountPercent":
				return q.DiscountPercent
			}
		case *ast.BinaryExpr:
			x, y := value(n.X), value(n.Y)
			switch n.Op {
			case token.ADD:
				return x + y
			case token.SUB:
				return x - y
			case token.MUL:
				return x * y
			case token.QUO:
				return x / y
			case token.EQL:
				if x == y {
					return 1
				}
				return 0
			case token.LAND:
				if x != 0 && y != 0 {
					return 1
				}
				return 0
			case token.LOR:
				if x != 0 || y != 0 {
					return 1
				}
				return 0
			}
		}
		t.Fatal("unsupported projected expression")
		return 0
	}
	total := int64(0)
	var statements func([]ast.Stmt)
	statements = func(code []ast.Stmt) {
		for _, s := range code {
			switch n := s.(type) {
			case *ast.AssignStmt:
				total = value(n.Rhs[0])
			case *ast.IfStmt:
				if value(n.Cond) != 0 {
					statements(n.Body.List)
				}
			default:
				t.Fatal("unsupported projected statement")
			}
		}
	}
	statements(f.Decls[0].(*ast.FuncDecl).Body.List)
	return total
}
