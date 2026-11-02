package handlers

import "testing"

func TestSlugifyKeepsAccentedLetters(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"El pipeline pasó pero nadie sabe por qué": "el-pipeline-paso-pero-nadie-sabe-por-que",
		"Fui despedido — ¡Ñandú!":                  "fui-despedido-nandu",
		"Demissão em massa: não acredito":          "demissao-em-massa-nao-acredito",
		"Plain ASCII title":                        "plain-ascii-title",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
