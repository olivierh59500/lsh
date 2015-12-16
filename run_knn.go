package main

import (
	"flag"
	"lsh"
	"time"
)

var (
	datafile string
	nWorker  int
	nQuery   int
	output   string
	k        int
)

func init() {
	flag.IntVar(&k, "k", 20, "K")
	flag.StringVar(&datafile, "d", "./data/tiny_images_1M.bin", "tiny image data file")
	flag.StringVar(&output, "o", "_knn.json", "output file for query results")
	flag.IntVar(&nWorker, "w", 200, "Number of threads for query tests")
	flag.IntVar(&nQuery, "q", 1000, "Number of queries")
}

func main() {
	flag.Parse()

	// Load data
	parser := lsh.NewTinyImagePointParser()
	nData := lsh.CountPoint(datafile, parser.ByteLen)
	iter := lsh.NewDataPointIterator(datafile, parser)
	data := make([]lsh.Point, nData)
	ids := make([]int, nData)
	for i := 0; i < nData; i++ {
		p, err := iter.Next()
		if err != nil {
			panic(err.Error())
		}
		data[i] = p.Point
		ids[i] = p.Id
	}

	// Run Knn
	knn := lsh.NewKnn(data, ids)
	queryFunc := func(q lsh.DataPoint) lsh.QueryResult {
		start := time.Now()
		out := make(chan int)
		go func() {
			knn.Query(q.Point, k, out)
			close(out)
		}()
		r := make([]int, 0)
		for i := range out {
			r = append(r, i)
		}
		dur := time.Since(start)
		ns := make(lsh.Neighbours, len(r))
		for i := range r {
			ns[i] = lsh.Neighbour{
				Id:       r[i],
				Distance: q.Point.L2(data[i]),
			}
		}
		return lsh.QueryResult{
			QueryId:    q.Id,
			Neighbours: ns,
			Time:       float64(dur) / float64(time.Millisecond),
		}
	}
	// Select queries
	queryIds := lsh.SelectQueries(nData, nQuery)
	iter = lsh.NewQueryPointIterator(datafile, parser, queryIds)
	// Run queries in parallel
	results := lsh.ParallelQueryIndex(iter, queryFunc, nWorker)
	// results := lsh.QueryIndex(iter, queryFunc)
	lsh.DumpJson(output, results)
}
