package egen

import (
	"reflect"
	"strconv"
	"testing"
)

func TestGenerateAlternateLinks(t *testing.T) {
	ptBRNonDefault := &Lang{
		Name: "Português do Brasil",
		Tag:  "pt-BR",
	}
	enDefault := &Lang{
		Name:    "English",
		Tag:     "en",
		Default: true,
	}
	ptBRDefault := &Lang{
		Name:    "Português do Brasil",
		Tag:     "pt-BR",
		Default: true,
	}
	enNonDefault := &Lang{
		Name: "English",
		Tag:  "en",
	}

	tests := []struct {
		langs                             []*Lang
		preLangSegments, postLangSegments []string
		res                               []*AlternateLink
	}{
		{
			[]*Lang{
				ptBRNonDefault,
				enDefault,
			},
			nil,
			nil,
			[]*AlternateLink{
				{
					Lang: enDefault,
					URL:  "/",
				},
				{
					Lang: ptBRNonDefault,
					URL:  "/" + ptBRNonDefault.Tag,
				},
			},
		},
		{
			[]*Lang{
				ptBRDefault,
				enNonDefault,
			},
			[]string{"test"},
			[]string{"foo"},
			[]*AlternateLink{
				{
					Lang: ptBRDefault,
					URL:  "/test/foo",
				},
				{
					Lang: enNonDefault,
					URL:  "/test/" + enNonDefault.Tag + "/foo",
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := generateAlternateLinks(test.preLangSegments, test.postLangSegments, test.langs)

			if !reflect.DeepEqual(res, test.res) {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}
