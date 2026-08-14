package ia

import "testing"

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"gato", "", 4},
		{"gato", "gato", 0},
		{"gato", "pato", 1},
		{"kitten", "sitting", 3},
	}

	for _, c := range cases {
		got := Levenshtein(c.a, c.b)
		if got != c.want {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestJaccardSimilarity(t *testing.T) {
	a := []string{"el", "cielo", "es", "azul"}
	b := []string{"el", "cielo", "es", "gris"}

	got := JaccardSimilarity(a, a)
	if got != 1 {
		t.Errorf("expected identical token sets to score 1, got %v", got)
	}

	got = JaccardSimilarity(a, b)
	if got <= 0 || got >= 1 {
		t.Errorf("expected partial overlap in (0,1), got %v", got)
	}

	got = JaccardSimilarity([]string{"a"}, []string{"b"})
	if got != 0 {
		t.Errorf("expected disjoint sets to score 0, got %v", got)
	}
}

func TestStatementSimilarity(t *testing.T) {
	same := StatementSimilarity("El cielo es azul.", "el CIELO es azul")
	if same < 0.9 {
		t.Errorf("expected near-identical statements to score high, got %v", same)
	}

	different := StatementSimilarity("El cielo es azul.", "Mi carro es rojo.")
	if different > same {
		t.Errorf("expected unrelated statements to score lower than identical ones")
	}
}
