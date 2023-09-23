package main

import (
	"fmt"

	"github.com/wilcosheh/tfidf"
	"github.com/wilcosheh/tfidf/seg"
	"github.com/wilcosheh/tfidf/similarity"
)

func main() {

	f := tfidf.New()
	f.AddDocs("how are you", "are you fine", "how old are you", "are you ok", "i am ok", "i am file")

	t1 := "it is so cool"
	w1 := f.Cal(t1)
	fmt.Printf("weight of %s is %+v.\n", t1, w1)

	t2 := "you are so beautiful"
	w2 := f.Cal(t2)
	fmt.Printf("weight of %s is %+v.\n", t2, w2)

	sim := similarity.Cosine(w1, w2)
	fmt.Printf("cosine between %s and %s is %f .\n", t1, t2, sim)

	tokenizer := seg.NewJieba()
	defer tokenizer.Free()

	f = tfidf.NewTokenizer(tokenizer)

	f.AddDocs("重庆大学", "上海市复旦大学", "上海交通大学", "重庆理工大学")

	t1 = "重庆市西南大学"
	w1 = f.Cal(t1)
	fmt.Printf("weight of %s is %+v.\n", t1, w1)

	t2 = "重庆市重庆大学"
	w2 = f.Cal(t2)
	fmt.Printf("weight of %s is %+v.\n", t2, w2)

	sim = similarity.Cosine(w1, w2)
	fmt.Printf("cosine between %s and %s is %f .\n", t1, t2, sim)
}