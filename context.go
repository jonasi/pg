package pg

import (
	"golang.org/x/net/context"
	"sync"
	"time"
)

type key int

var contextKey key = 0

func NewContext(ctxt context.Context, q *QuerySet) context.Context {
	return context.WithValue(ctxt, contextKey, q)
}

func FromContext(ctxt context.Context) (*QuerySet, bool) {
	q, ok := ctxt.Value(contextKey).(*QuerySet)
	return q, ok
}

type QueryEvent struct {
	Query    string
	Args     []interface{}
	Duration time.Duration
	Error    error
}

func NewQuerySet() *QuerySet {
	return &QuerySet{
		q: make([]QueryEvent, 0),
	}
}

type QuerySet struct {
	q []QueryEvent
	l sync.Mutex
}

func (q *QuerySet) Add(query string, args []interface{}, start time.Time, err error) {
	q.l.Lock()
	defer q.l.Unlock()

	q.q = append(q.q, QueryEvent{
		Query:    query,
		Args:     args,
		Duration: time.Since(start),
		Error:    err,
	})
}
