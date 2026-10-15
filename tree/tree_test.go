package tree

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func randomTree(r *rand.Rand, n int) *Node[int] {
	if n == 0 {
		return nil
	}
	k := r.IntN(n) // nodes in left subtree
	return &Node[int]{
		Val:   r.IntN(100) - 50,
		Left:  randomTree(r, k),
		Right: randomTree(r, n-1-k),
	}
}

func recPre(n *Node[int], out *[]int) {
	if n != nil {
		*out = append(*out, n.Val)
		recPre(n.Left, out)
		recPre(n.Right, out)
	}
}

func recIn(n *Node[int], out *[]int) {
	if n != nil {
		recIn(n.Left, out)
		*out = append(*out, n.Val)
		recIn(n.Right, out)
	}
}

func recPost(n *Node[int], out *[]int) {
	if n != nil {
		recPost(n.Left, out)
		recPost(n.Right, out)
		*out = append(*out, n.Val)
	}
}

func recHeight(n *Node[int]) int {
	if n == nil {
		return 0
	}
	return 1 + max(recHeight(n.Left), recHeight(n.Right))
}

func equal(a, b *Node[int]) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Val == b.Val && equal(a.Left, b.Left) && equal(a.Right, b.Right)
}

func TestTraversalsRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 500 {
		n := r.IntN(30)
		root := randomTree(r, n)
		var pre, in, post []int
		recPre(root, &pre)
		recIn(root, &in)
		recPost(root, &post)
		if got := slices.Collect(PreOrder(root)); !slices.Equal(got, pre) {
			t.Fatalf("PreOrder = %v, want %v", got, pre)
		}
		if got := slices.Collect(InOrder(root)); !slices.Equal(got, in) {
			t.Fatalf("InOrder = %v, want %v", got, in)
		}
		if got := slices.Collect(PostOrder(root)); !slices.Equal(got, post) {
			t.Fatalf("PostOrder = %v, want %v", got, post)
		}
		if got := Size(root); got != n {
			t.Fatalf("Size = %d, want %d", got, n)
		}
		if got, want := Height(root), recHeight(root); got != want {
			t.Fatalf("Height = %d, want %d", got, want)
		}
		total := 0
		for level := range LevelOrder(root) {
			total += len(level)
		}
		if total != n {
			t.Fatalf("LevelOrder total = %d, want %d", total, n)
		}
		// Format/Parse round trip.
		s := Format(root)
		if back := Parse(s); !equal(back, root) {
			t.Fatalf("Parse(Format(t)) != t for %s", s)
		}
		// String must not panic and must have Height-related line count.
		if root != nil {
			lines := strings.Split(root.String(), "\n")
			if want := 2*recHeight(root) - 1; len(lines) != want {
				t.Fatalf("String has %d lines, want %d:\n%s", len(lines), want, root)
			}
		}
	}
}

func TestEarlyBreak(t *testing.T) {
	root := Parse("[1,2,3,4,5,6,7]")
	take := func(seq func(func(int) bool), k int) []int {
		var out []int
		for v := range seq {
			if len(out) == k {
				break
			}
			out = append(out, v)
		}
		return out
	}
	if got := take(PreOrder(root), 3); !slices.Equal(got, []int{1, 2, 4}) {
		t.Errorf("PreOrder take 3 = %v", got)
	}
	if got := take(InOrder(root), 3); !slices.Equal(got, []int{4, 2, 5}) {
		t.Errorf("InOrder take 3 = %v", got)
	}
	if got := take(PostOrder(root), 3); !slices.Equal(got, []int{4, 5, 2}) {
		t.Errorf("PostOrder take 3 = %v", got)
	}
	for level := range LevelOrder(root) {
		if !slices.Equal(level, []int{1}) {
			t.Errorf("first level = %v", level)
		}
		break
	}
}

func TestDeepTree(t *testing.T) {
	const n = 100000
	var root *Node[int]
	for i := range n {
		root = &Node[int]{Val: i, Left: root}
	}
	if Size(root) != n || Height(root) != n {
		t.Fatalf("Size/Height of deep tree wrong")
	}
	count := 0
	for range PostOrder(root) {
		count++
	}
	for range InOrder(root) {
		count++
	}
	if count != 2*n {
		t.Fatalf("deep traversal count = %d", count)
	}
}

func TestFromLevelOrder(t *testing.T) {
	root := FromLevelOrder[int](1, nil, 2, 3)
	if root.Val != 1 || root.Left != nil || root.Right.Val != 2 || root.Right.Left.Val != 3 {
		t.Errorf("FromLevelOrder structure wrong: %s", Format(root))
	}
	if FromLevelOrder[int]() != nil || FromLevelOrder[int](nil) != nil || FromLevelOrder[int](nil, nil) != nil {
		t.Errorf("empty FromLevelOrder should be nil")
	}
	// Trailing nils are fine.
	if got := Format(FromLevelOrder[int](1, nil, nil, nil)); got != "[1]" {
		t.Errorf("trailing nils = %s", got)
	}
	// Strings and numeric conversion.
	s := FromLevelOrder[string]("a", "b", nil, "c")
	if got := Format(s); got != "[a,b,null,c]" {
		t.Errorf("string tree = %s", got)
	}
	f := FromLevelOrder[float64](1, 2.5, -3)
	if got := slices.Collect(PreOrder(f)); !slices.Equal(got, []float64{1, 2.5, -3}) {
		t.Errorf("float tree = %v", got)
	}
	type id int64
	ids := FromLevelOrder[id](1, 2)
	if ids.Left.Val != 2 {
		t.Errorf("named int tree wrong")
	}
	u := FromLevelOrder[uint8](255, 0)
	if u.Val != 255 {
		t.Errorf("uint8 tree wrong")
	}
}

func TestFromLevelOrderPanics(t *testing.T) {
	tests := []struct {
		name string
		f    func()
		msg  string
	}{
		{"wrong type", func() { FromLevelOrder[int](1, "x") }, "index 1 has type string, want int"},
		{"lossy float", func() { FromLevelOrder[int](1, 2.5) }, "does not fit"},
		{"overflow", func() { FromLevelOrder[int8](1, 300) }, "does not fit"},
		{"negative to uint", func() { FromLevelOrder[uint](1, -1) }, "does not fit"},
		{"orphan", func() { FromLevelOrder[int](1, nil, nil, 2) }, "no parent"},
		{"orphan after nil root", func() { FromLevelOrder[int](nil, 1) }, "no parent"},
		{"parse brackets", func() { Parse("1,2") }, "enclosed"},
		{"parse value", func() { Parse("[1,x]") }, `invalid value "x"`},
		{"parse empty value", func() { Parse("[1,,2]") }, `invalid value ""`},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				r := recover()
				msg, _ := r.(string)
				if r == nil || !strings.Contains(msg, tt.msg) || !strings.HasPrefix(msg, "tree: ") {
					t.Errorf("%s: panic = %v, want containing %q", tt.name, r, tt.msg)
				}
			}()
			tt.f()
		}()
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"[]", "[]"},
		{"  [ ] ", "[]"},
		{"[1]", "[1]"},
		{"[1,2,null,3]", "[1,2,null,3]"},
		{"[ 1 , null , 2 , 3 ]", "[1,null,2,3]"},
		{"[1,2,3,null,null,null,null]", "[1,2,3]"},
		{"[-5,-3,null,-1]", "[-5,-3,null,-1]"},
		{"[5,4,8,11,null,13,4,7,2,null,null,null,1]", "[5,4,8,11,null,13,4,7,2,null,null,null,1]"},
	}
	for _, tt := range tests {
		if got := Format(Parse(tt.in)); got != tt.want {
			t.Errorf("Format(Parse(%q)) = %q, want %q", tt.in, got, tt.want)
		}
	}
	// A string value "null" must not be trimmed as a missing node.
	if got := Format(FromLevelOrder[string]("a", nil, "null")); got != "[a,null,null]" {
		t.Errorf("Format with \"null\" value = %s", got)
	}
}

func TestLevelOrder(t *testing.T) {
	root := Parse("[3,9,20,null,null,15,7]")
	var got [][]int
	for level := range LevelOrder(root) {
		got = append(got, level)
	}
	want := [][]int{{3}, {9, 20}, {15, 7}}
	if !slices.EqualFunc(got, want, slices.Equal[[]int]) {
		t.Errorf("LevelOrder = %v, want %v", got, want)
	}
	for range LevelOrder[int](nil) {
		t.Errorf("LevelOrder(nil) yielded")
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		in   *Node[int]
		want string
	}{
		{nil, "<nil>"},
		{Parse("[1]"), "1"},
		{Parse("[1,2,3]"), " 1\n/ \\\n2 3"},
		{Parse("[1,2]"), " 1\n/\n2"},
		{Parse("[1,null,2]"), "1\n \\\n 2"},
		{
			Parse("[10,2,30,null,4]"),
			"" +
				" _10_\n" +
				"/    \\\n" +
				"2   30\n" +
				" \\\n" +
				" 4",
		},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("String(%s) =\n%s\nwant\n%s", Format(tt.in), got, tt.want)
		}
	}
	s := FromLevelOrder[string]("", "é", "x")
	if got := s.String(); !strings.Contains(got, `""`) || !strings.Contains(got, "é") {
		t.Errorf("String with empty/unicode labels =\n%s", got)
	}
}
