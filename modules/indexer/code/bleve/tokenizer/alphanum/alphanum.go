package alphanum

import (
	"unicode"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/character"
	"github.com/blevesearch/bleve/v2/registry"
)

const Name = "alphanum"

func alphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

func TokenizerConstructor(config map[string]any, cache *registry.Cache) (analysis.Tokenizer, error) {
	return character.NewCharacterTokenizer(IsAlphaNumeric), nil
}

func init() {
	registry.RegisterTokenizer(Name, TokenizerConstructor)
}
