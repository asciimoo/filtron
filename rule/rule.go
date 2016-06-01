package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/asciimoo/filtron/action"
	"github.com/asciimoo/filtron/selector"
	"github.com/asciimoo/filtron/types"
)

type Rule struct {
	Interval        uint   `json:"interval"`
	Limit           uint   `json:"limit"`
	Name            string `json:"name"`
	lastTick        uint
	matchedRequests uint
	Filters         []*selector.Selector `json:-`
	RawFilters      []string             `json:"filters"`
	Aggregations    []*Aggregation       `json:-`
	RawAggregations []string             `json:"aggregations"`
	Actions         []action.Action      `json:-`
	RawActions      []action.ActionJSON  `json:"actions"`
}

type Aggregation struct {
	sync.RWMutex
	Values   map[string]uint
	Selector *selector.Selector
}

func New(name string, interval, limit uint, filters []string) (*Rule, error) {
	r := &Rule{
		Interval: interval,
		Limit:    limit,
		Name:     name,
	}
	r.Init()
	if err := r.ParseFilters(filters); err != nil {
		return nil, err
	}
	return r, nil
}

func ParseJSON(filename string) ([]*Rule, error) {
	var rules []*Rule
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fileContent, &rules)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		r.Init()
		if err := r.ParseFilters(r.RawFilters); err != nil {
			return nil, err
		}
		if err := r.ParseAggregations(r.RawAggregations); err != nil {
			return nil, err
		}
		r.Actions = make([]action.Action, 0, len(r.RawActions))
		for _, actionJSON := range r.RawActions {
			a, err := action.Create(actionJSON)
			if err != nil {
				return nil, err
			}
			r.Actions = append(r.Actions, a)
		}
	}
	return rules, nil
}

func (r *Rule) Init() {
	r.matchedRequests = 0
	r.lastTick = uint(time.Now().Unix())
}

func (r *Rule) ParseAggregations(aggregations []string) error {
	r.Aggregations = make([]*Aggregation, 0, len(aggregations))
	for _, t := range aggregations {
		s, err := selector.Parse(t)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot parse selector '%v': %v", t, err))
		}
		a := &Aggregation{
			Values:   make(map[string]uint),
			Selector: s,
		}
		r.Aggregations = append(r.Aggregations, a)
	}
	return nil
}

func (r *Rule) ParseFilters(filters []string) error {
	r.Filters = make([]*selector.Selector, 0, len(filters))
	for _, t := range filters {
		s, err := selector.Parse(t)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot parse selector '%v': %v", t, err))
		}
		r.Filters = append(r.Filters, s)
	}
	return nil
}

func (r *Rule) Validate(req *http.Request, resp http.ResponseWriter) types.ResponseState {
	curTime := uint(time.Now().Unix())
	if curTime-r.lastTick >= r.Interval {
		r.matchedRequests = 0
		r.lastTick = curTime
		for _, a := range r.Aggregations {
			a.Lock()
			a.Values = make(map[string]uint)
			a.Unlock()
		}
	}
	for _, t := range r.Filters {
		if _, found := t.Match(req); !found {
			return types.UNTOUCHED
		}
	}
	matched := false
	if len(r.Aggregations) == 0 {
		r.matchedRequests += 1
		if len(r.Filters) == 0 || r.matchedRequests > r.Limit {
			matched = true
		}
		return types.UNTOUCHED
	} else {
		for _, a := range r.Aggregations {
			if a.Get(req) > r.Limit {
				matched = true
			}
		}
	}
	state := types.UNTOUCHED
	if matched {
		for _, a := range r.Actions {
			s, err := a.Act(req, resp)
			// TODO error handling
			if err != nil {
				fmt.Println("meh", err)
			}
			if s == types.SERVED {
				state = types.SERVED
			}
		}
	}
	return state
}

func (a *Aggregation) Get(req *http.Request) uint {
	if val, found := a.Selector.Match(req); found {
		a.Lock()
		a.Values[val] += 1
		v := a.Values[val]
		a.Unlock()
		return v
	}
	return 0
}
