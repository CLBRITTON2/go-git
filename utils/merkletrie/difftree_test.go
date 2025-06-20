package merkletrie_test

import (
	"bytes"
	ctx "context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/go-git/go-git/v6/utils/merkletrie"
	"github.com/go-git/go-git/v6/utils/merkletrie/internal/fsnoder"
	"github.com/stretchr/testify/suite"
)

type DiffTreeSuite struct {
	suite.Suite
}

func TestDiffTreeSuite(t *testing.T) {
	suite.Run(t, new(DiffTreeSuite))
}

type diffTreeTest struct {
	from     string
	to       string
	expected string
}

func (t diffTreeTest) innerRun(s *DiffTreeSuite, context string, reverse bool) {
	comment := fmt.Sprintf("\n%s", context)
	if reverse {
		comment = fmt.Sprintf("%s [REVERSED]", comment)
	}

	a, err := fsnoder.New(t.from)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t    from = %s", comment, a)

	b, err := fsnoder.New(t.to)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t      to = %s", comment, b)

	expected, err := newChangesFromString(t.expected)
	s.NoError(err, comment)

	if reverse {
		a, b = b, a
		expected = expected.reverse()
	}
	comment = fmt.Sprintf("%s\n\texpected = %s", comment, expected)

	results, err := merkletrie.DiffTree(a, b, fsnoder.HashEqual)
	s.NoError(err, comment)

	obtained, err := newChanges(results)
	s.NoError(err, comment)

	comment = fmt.Sprintf("%s\n\tobtained = %s", comment, obtained)

	sort.Sort(obtained)
	sort.Sort(expected)
	s.Equal(expected, obtained, comment)
}

func (t diffTreeTest) innerRunCtx(s *DiffTreeSuite, context string, reverse bool) {
	comment := fmt.Sprintf("\n%s", context)
	if reverse {
		comment = fmt.Sprintf("%s [REVERSED]", comment)
	}

	a, err := fsnoder.New(t.from)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t    from = %s", comment, a)

	b, err := fsnoder.New(t.to)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t      to = %s", comment, b)

	expected, err := newChangesFromString(t.expected)
	s.NoError(err, comment)

	if reverse {
		a, b = b, a
		expected = expected.reverse()
	}
	comment = fmt.Sprintf("%s\n\texpected = %s", comment, expected)

	results, err := merkletrie.DiffTreeContext(ctx.Background(), a, b, fsnoder.HashEqual)
	s.NoError(err, comment)

	obtained, err := newChanges(results)
	s.NoError(err, comment)

	comment = fmt.Sprintf("%s\n\tobtained = %s", comment, obtained)

	sort.Sort(obtained)
	sort.Sort(expected)
	s.Equal(expected, obtained, comment)
}

func (t diffTreeTest) run(s *DiffTreeSuite, context string) {
	t.innerRun(s, context, false)
	t.innerRun(s, context, true)
	t.innerRunCtx(s, context, false)
	t.innerRunCtx(s, context, true)
}

type change struct {
	merkletrie.Action
	path string
}

func (c change) String() string {
	return fmt.Sprintf("<%s %s>", c.Action, c.path)
}

func (c change) reverse() change {
	ret := change{
		path: c.path,
	}

	switch c.Action {
	case merkletrie.Insert:
		ret.Action = merkletrie.Delete
	case merkletrie.Delete:
		ret.Action = merkletrie.Insert
	case merkletrie.Modify:
		ret.Action = merkletrie.Modify
	default:
		panic(fmt.Sprintf("unknown action type: %d", c.Action))
	}

	return ret
}

type changes []change

func newChanges(original merkletrie.Changes) (changes, error) {
	ret := make(changes, len(original))
	for i, c := range original {
		action, err := c.Action()
		if err != nil {
			return nil, err
		}
		switch action {
		case merkletrie.Insert:
			ret[i] = change{
				Action: merkletrie.Insert,
				path:   c.To.String(),
			}
		case merkletrie.Delete:
			ret[i] = change{
				Action: merkletrie.Delete,
				path:   c.From.String(),
			}
		case merkletrie.Modify:
			ret[i] = change{
				Action: merkletrie.Modify,
				path:   c.From.String(),
			}
		default:
			panic(fmt.Sprintf("unsupported action %d", action))
		}
	}

	return ret, nil
}

func newChangesFromString(s string) (changes, error) {
	ret := make([]change, 0)

	s = strings.TrimSpace(s)
	s = removeDuplicatedSpace(s)
	s = turnSpaceIntoLiteralSpace(s)

	if s == "" {
		return ret, nil
	}

	for _, chunk := range strings.Split(s, " ") {
		change := change{
			path: chunk[1:],
		}

		switch chunk[0] {
		case '+':
			change.Action = merkletrie.Insert
		case '-':
			change.Action = merkletrie.Delete
		case '*':
			change.Action = merkletrie.Modify
		default:
			panic(fmt.Sprintf("unsupported action descriptor %q", chunk[0]))
		}

		ret = append(ret, change)
	}

	return ret, nil
}

func removeDuplicatedSpace(s string) string {
	var buf bytes.Buffer

	var lastWasSpace, currentIsSpace bool
	for _, r := range s {
		currentIsSpace = unicode.IsSpace(r)

		if lastWasSpace && currentIsSpace {
			continue
		}
		lastWasSpace = currentIsSpace

		buf.WriteRune(r)
	}

	return buf.String()
}

func turnSpaceIntoLiteralSpace(s string) string {
	return strings.Map(
		func(r rune) rune {
			if unicode.IsSpace(r) {
				return ' '
			}
			return r
		}, s)
}

func (cc changes) Len() int           { return len(cc) }
func (cc changes) Swap(i, j int)      { cc[i], cc[j] = cc[j], cc[i] }
func (cc changes) Less(i, j int) bool { return strings.Compare(cc[i].String(), cc[j].String()) < 0 }

func (cc changes) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "len(%d) [", len(cc))
	sep := ""
	for _, c := range cc {
		fmt.Fprintf(&buf, "%s%s", sep, c)
		sep = ", "
	}
	buf.WriteByte(']')
	return buf.String()
}

func (cc changes) reverse() changes {
	ret := make(changes, len(cc))
	for i, c := range cc {
		ret[i] = c.reverse()
	}

	return ret
}

func do(s *DiffTreeSuite, list []diffTreeTest) {
	for i, t := range list {
		t.run(s, fmt.Sprintf("test #%d:", i))
	}
}

func (s *DiffTreeSuite) TestEmptyVsEmpty() {
	do(s, []diffTreeTest{
		{"()", "()", ""},
		{"A()", "A()", ""},
		{"A()", "()", ""},
		{"A()", "B()", ""},
	})
}

func (s *DiffTreeSuite) TestBasicCases() {
	do(s, []diffTreeTest{
		{"()", "()", ""},
		{"()", "(a<>)", "+a"},
		{"()", "(a<1>)", "+a"},
		{"()", "(a())", ""},
		{"()", "(a(b()))", ""},
		{"()", "(a(b<>))", "+a/b"},
		{"()", "(a(b<1>))", "+a/b"},
		{"(a<>)", "(a<>)", ""},
		{"(a<>)", "(a<1>)", "*a"},
		{"(a<>)", "(a())", "-a"},
		{"(a<>)", "(a(b()))", "-a"},
		{"(a<>)", "(a(b<>))", "-a +a/b"},
		{"(a<>)", "(a(b<1>))", "-a +a/b"},
		{"(a<>)", "(c())", "-a"},
		{"(a<>)", "(c(b()))", "-a"},
		{"(a<>)", "(c(b<>))", "-a +c/b"},
		{"(a<>)", "(c(b<1>))", "-a +c/b"},
		{"(a<>)", "(c(a()))", "-a"},
		{"(a<>)", "(c(a<>))", "-a +c/a"},
		{"(a<>)", "(c(a<1>))", "-a +c/a"},
		{"(a<1>)", "(a<1>)", ""},
		{"(a<1>)", "(a<2>)", "*a"},
		{"(a<1>)", "(b<1>)", "-a +b"},
		{"(a<1>)", "(b<2>)", "-a +b"},
		{"(a<1>)", "(a())", "-a"},
		{"(a<1>)", "(a(b()))", "-a"},
		{"(a<1>)", "(a(b<>))", "-a +a/b"},
		{"(a<1>)", "(a(b<1>))", "-a +a/b"},
		{"(a<1>)", "(a(b<2>))", "-a +a/b"},
		{"(a<1>)", "(c())", "-a"},
		{"(a<1>)", "(c(b()))", "-a"},
		{"(a<1>)", "(c(b<>))", "-a +c/b"},
		{"(a<1>)", "(c(b<1>))", "-a +c/b"},
		{"(a<1>)", "(c(b<2>))", "-a +c/b"},
		{"(a<1>)", "(c())", "-a"},
		{"(a<1>)", "(c(a()))", "-a"},
		{"(a<1>)", "(c(a<>))", "-a +c/a"},
		{"(a<1>)", "(c(a<1>))", "-a +c/a"},
		{"(a<1>)", "(c(a<2>))", "-a +c/a"},
		{"(a())", "(a())", ""},
		{"(a())", "(b())", ""},
		{"(a())", "(a(b()))", ""},
		{"(a())", "(b(a()))", ""},
		{"(a())", "(a(b<>))", "+a/b"},
		{"(a())", "(a(b<1>))", "+a/b"},
		{"(a())", "(b(a<>))", "+b/a"},
		{"(a())", "(b(a<1>))", "+b/a"},
	})
}

func (s *DiffTreeSuite) TestHorizontals() {
	do(s, []diffTreeTest{
		{"()", "(a<> b<>)", "+a +b"},
		{"()", "(a<> b<1>)", "+a +b"},
		{"()", "(a<> b())", "+a"},
		{"()", "(a() b<>)", "+b"},
		{"()", "(a<1> b<>)", "+a +b"},
		{"()", "(a<1> b<1>)", "+a +b"},
		{"()", "(a<1> b<2>)", "+a +b"},
		{"()", "(a<1> b())", "+a"},
		{"()", "(a() b<1>)", "+b"},
		{"()", "(a() b())", ""},
		{"()", "(a<> b<> c<> d<>)", "+a +b +c +d"},
		{"()", "(a<> b<1> c() d<> e<2> f())", "+a +b +d +e"},
	})
}

func (s *DiffTreeSuite) TestVerticals() {
	do(s, []diffTreeTest{
		{"()", "(z<>)", "+z"},
		{"()", "(a(z<>))", "+a/z"},
		{"()", "(a(b(z<>)))", "+a/b/z"},
		{"()", "(a(b(c(z<>))))", "+a/b/c/z"},
		{"()", "(a(b(c(d(z<>)))))", "+a/b/c/d/z"},
		{"()", "(a(b(c(d(z<1>)))))", "+a/b/c/d/z"},
	})
}

func (s *DiffTreeSuite) TestSingleInserts() {
	do(s, []diffTreeTest{
		{"()", "(z<>)", "+z"},
		{"(a())", "(a(z<>))", "+a/z"},
		{"(a())", "(a(b(z<>)))", "+a/b/z"},
		{"(a(b(c())))", "(a(b(c(z<>))))", "+a/b/c/z"},
		{"(a<> b<> c<>)", "(a<> b<> c<> z<>)", "+z"},
		{"(a(b<> c<> d<>))", "(a(b<> c<> d<> z<>))", "+a/z"},
		{"(a(b(c<> d<> e<>)))", "(a(b(c<> d<> e<> z<>)))", "+a/b/z"},
		{"(a(b<>) f<>)", "(a(b<>) f<> z<>)", "+z"},
		{"(a(b<>) f<>)", "(a(b<> z<>) f<>)", "+a/z"},
	})
}

func (s *DiffTreeSuite) TestDebug() {
	do(s, []diffTreeTest{
		{"(a(b<>) f<>)", "(a(b<> z<>) f<>)", "+a/z"},
	})
}

//	   root
//	   / | \
//	  /  |  ----
//	 f   d      h --------
//	/\         /  \      |
//
// e   a      j   b/      g
// |  / \     |
// l  n  k    icm
//
//	|
//	o
//	|
//	p/
func (s *DiffTreeSuite) TestCrazy() {
	crazy := "(f(e(l<1>) a(n(o(p())) k<1>)) d<1> h(j(i<1> c<2> m<>) b() g<>))"
	do(s, []diffTreeTest{
		{
			crazy,
			"()",
			"-d -f/e/l -f/a/k -h/j/i -h/j/c -h/j/m -h/g",
		}, {
			crazy,
			crazy,
			"",
		}, {
			crazy,
			"(d<1>)",
			"-f/e/l -f/a/k -h/j/i -h/j/c -h/j/m -h/g",
		}, {
			crazy,
			"(d<1> h(b() g<>))",
			"-f/e/l -f/a/k -h/j/i -h/j/c -h/j/m",
		}, {
			crazy,
			"(d<1> f(e(l()) a()) h(b() g<>))",
			"-f/e/l -f/a/k -h/j/i -h/j/c -h/j/m",
		}, {
			crazy,
			"(d<1> f(e(l<1>) a()) h(b() g<>))",
			"-f/a/k -h/j/i -h/j/c -h/j/m",
		}, {
			crazy,
			"(d<2> f(e(l<2>) a(s(t<1>))) h(b() g<> r<> j(i<> c<3> m<>)))",
			"+f/a/s/t +h/r -f/a/k *d *f/e/l *h/j/c *h/j/i",
		}, {
			crazy,
			"(f(e(l<2>) a(n(o(p<1>)) k<>)) h(j(i<1> c<2> m<>) b() g<>))",
			"*f/e/l +f/a/n/o/p *f/a/k -d",
		}, {
			crazy,
			"(f(e(l<1>) a(n(o(p(r<1>))) k<1>)) d<1> h(j(i<1> c<2> b() m<>) g<1>))",
			"+f/a/n/o/p/r *h/g",
		},
	})
}

func (s *DiffTreeSuite) TestSameNames() {
	do(s, []diffTreeTest{
		{
			"(a(a(a<>)))",
			"(a(a(a<1>)))",
			"*a/a/a",
		}, {
			"(a(b(a<>)))",
			"(a(b(a<>)) b(a<>))",
			"+b/a",
		}, {
			"(a(b(a<>)))",
			"(a(b()) b(a<>))",
			"-a/b/a +b/a",
		},
	})
}

func (s *DiffTreeSuite) TestIssue275() {
	do(s, []diffTreeTest{
		{
			"(a(b(c.go<1>) b.go<2>))",
			"(a(b(c.go<1> d.go<3>) b.go<2>))",
			"+a/b/d.go",
		},
	})
}

func (s *DiffTreeSuite) TestIssue1057() {
	p1 := "TestAppWithUnicodéPath"
	p2 := "TestAppWithUnicodéPath"
	s.False(p1 == p2)
	do(s, []diffTreeTest{
		{
			fmt.Sprintf("(%s(x.go<1>))", p1),
			fmt.Sprintf("(%s(x.go<1>) %s(x.go<1>))", p1, p2),
			fmt.Sprintf("+%s/x.go", p2),
		},
	})
	// swap p1 with p2
	do(s, []diffTreeTest{
		{
			fmt.Sprintf("(%s(x.go<1>))", p2),
			fmt.Sprintf("(%s(x.go<1>) %s(x.go<1>))", p1, p2),
			fmt.Sprintf("+%s/x.go", p1),
		},
	})
}

func (s *DiffTreeSuite) TestCancel() {
	t := diffTreeTest{"()", "(a<> b<1> c() d<> e<2> f())", "+a +b +d +e"}
	comment := fmt.Sprintf("\n%s", "test cancel:")

	a, err := fsnoder.New(t.from)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t    from = %s", comment, a)

	b, err := fsnoder.New(t.to)
	s.NoError(err, comment)
	comment = fmt.Sprintf("%s\n\t      to = %s", comment, b)

	expected, err := newChangesFromString(t.expected)
	s.NoError(err, comment)

	comment = fmt.Sprintf("%s\n\texpected = %s", comment, expected)
	context, cancel := ctx.WithCancel(ctx.Background())
	cancel()
	results, err := merkletrie.DiffTreeContext(context, a, b, fsnoder.HashEqual)
	s.Nil(results, comment)
	s.ErrorContains(err, "operation canceled")

}
