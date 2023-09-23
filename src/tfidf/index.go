package tfidf

// adapted from wilcosheh/tfidf
import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"math"

	"github.com/wilcosheh/tfidf/seg"
	"github.com/wilcosheh/tfidf/util"
)

// TFIDF tfidf model
type TFIDF struct {
	docIndex  map[string]int         // train document index in TermFreqs
	termFreqs []map[string]int       // term frequency for each train document
	termDocs  map[string]int         // documents number for each term in train data
	n         int                    // number of documents in train data
	stopWords map[string]interface{} // words to be filtered
	tokenizer seg.Tokenizer          // tokenizer, space is used as default
}

// New new model with default
func New() *TFIDF {
	return &TFIDF{
		docIndex:  make(map[string]int),
		termFreqs: make([]map[string]int, 0),
		termDocs:  make(map[string]int),
		n:         0,
		tokenizer: &seg.EnTokenizer{},
	}
}

// NewTokenizer new with specified tokenizer
func NewTokenizer(tokenizer seg.Tokenizer) *TFIDF {
	return &TFIDF{
		docIndex:  make(map[string]int),
		termFreqs: make([]map[string]int, 0),
		termDocs:  make(map[string]int),
		n:         0,
		tokenizer: tokenizer,
	}
}

func (f *TFIDF) InitStopWords() {
	if f.stopWords == nil {
		f.stopWords = make(map[string]interface{})
	}

	lines, err := util.ReadLines("../data/stopword")
	if err != nil {
		log.Printf("init stop words with error: %s", err)
	}

	for _, w := range lines {
		f.stopWords[w] = nil
	}
}

// AddStopWords add stop words to be filtered
func (f *TFIDF) AddStopWords(words ...string) {
	if f.stopWords == nil {
		f.stopWords = make(map[string]interface{})
	}

	for _, word := range words {
		f.stopWords[word] = nil
	}
}

// AddStopWordsFile add stop words file to be filtered, with one word a line
func (f *TFIDF) AddStopWordsFile(file string) (err error) {
	lines, err := util.ReadLines(file)
	if err != nil {
		return
	}

	f.AddStopWords(lines...)
	return
}

type tokenizerDef func(string) []string

// AddDocs add train documents
func (f *TFIDF) AddDocs(docs []string, tokenizerArgs ...tokenizerDef) {
	tokenizer := f.tokenizer.Seg
	if len(tokenizerArgs) > 0 {
		tokenizer = tokenizerArgs[0]
	}
	for _, doc := range docs {
		h := hash(doc)
		if f.docHashPos(h) >= 0 {
			return
		}

		termFreq := f.termFreq(doc, tokenizer)
		if len(termFreq) == 0 {
			return
		}

		f.docIndex[h] = f.n
		f.n++

		f.termFreqs = append(f.termFreqs, termFreq)

		for term := range termFreq {
			f.termDocs[term]++
		}
	}
}

// Cal calculate tf-idf weight for specified document
func (f *TFIDF) Cal(doc string, tokenizerArgs ...tokenizerDef) (weight map[string]float64) {
	tokenizer := f.tokenizer.Seg
	if len(tokenizerArgs) > 0 {
		tokenizer = tokenizerArgs[0]
	}
	weight = make(map[string]float64)

	var termFreq map[string]int

	docPos := f.docPos(doc)
	if docPos < 0 {
		termFreq = f.termFreq(doc, tokenizer)
	} else {
		termFreq = f.termFreqs[docPos]
	}

	docTerms := 0
	for _, freq := range termFreq {
		docTerms += freq
	}
	for term, freq := range termFreq {
		weight[term] = tfidf(freq, docTerms, f.termDocs[term], f.n)
	}

	return weight
}

func (f *TFIDF) termFreq(doc string, tokenizer func(string) []string) (m map[string]int) {
	m = make(map[string]int)

	tokens := tokenizer(doc)
	if len(tokens) == 0 {
		return
	}

	for _, term := range tokens {
		if _, ok := f.stopWords[term]; ok {
			continue
		}

		m[term]++
	}

	return
}

func (f *TFIDF) docHashPos(hash string) int {
	if pos, ok := f.docIndex[hash]; ok {
		return pos
	}

	return -1
}

func (f *TFIDF) docPos(doc string) int {
	return f.docHashPos(hash(doc))
}

func hash(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func tfidf(termFreq, docTerms, termDocs, N int) float64 {
	tf := float64(termFreq) / float64(docTerms)
	idf := math.Log(float64(1+N) / (1 + float64(termDocs)))
	return tf * idf
}
