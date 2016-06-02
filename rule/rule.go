package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asciimoo/filtron/action"
	"github.com/asciimoo/filtron/selector"
	"github.com/asciimoo/filtron/types"
)

type Rule struct {
	Interval        uint64 `json:"interval"`
	Limit           uint64 `json:"limit"`
	Name            string `json:"name"`
	lastTick        uint64
	matchedRequests uint64
	Filters         []*selector.Selector `json:-`
	RawFilters      []string             `json:"filters"`
	Aggregations    []*Aggregation       `json:-`
	RawAggregations []string             `json:"aggregations"`
	Actions         []action.Action      `json:-`
	RawActions      []action.ActionJSON  `json:"actions"`
	SubRules        []*Rule              `json:"subrules"`
}

type Aggregation struct {
	sync.RWMutex
	Values   map[string]uint64
	Selector *selector.Selector
}

func New(name string, interval, limit uint64, filters []string) (*Rule, error) {
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
		err := r.Init()
		if err != nil {
			return nil, err
		}
	}
	return rules, nil
}

func (r *Rule) Init() error {
	r.matchedRequests = 0
	r.lastTick = uint64(time.Now().Unix())
	if len(r.RawActions) == 0 {
		return errors.New(fmt.Sprintf("Missing actions in rule: %v", r.Name))
	}
	if err := r.ParseFilters(r.RawFilters); err != nil {
		return err
	}
	if err := r.ParseAggregations(r.RawAggregations); err != nil {
		return err
	}
	r.Actions = make([]action.Action, 0, len(r.RawActions))
	for _, actionJSON := range r.RawActions {
		a, err := action.FromJSON(actionJSON)
		if err != nil {
			return err
		}
		r.Actions = append(r.Actions, a)
	}
	for _, sr := range r.SubRules {
		err := sr.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Rule) ParseAggregations(aggregations []string) error {
	r.Aggregations = make([]*Aggregation, 0, len(aggregations))
	for _, t := range aggregations {
		s, err := selector.Parse(t)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot parse selector '%v': %v", t, err))
		}
		a := &Aggregation{
			Values:   make(map[string]uint64),
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

func (r *Rule) Validate(req *http.Request, resp http.ResponseWriter, rs types.ResponseState) types.ResponseState {
	curTime := uint64(time.Now().Unix())
	if r.Limit != 0 && curTime-r.lastTick >= r.Interval {
		r.matchedRequests = 0
		r.lastTick = curTime
		for _, a := range r.Aggregations {
			a.Lock()
			a.Values = make(map[string]uint64)
			a.Unlock()
		}
	}
	for _, t := range r.Filters {
		if _, found := t.Match(req); !found {
			return types.UNTOUCHED
		}
	}
	matched := false
	state := rs
	if len(r.Aggregations) == 0 {
		atomic.AddUint64(&r.matchedRequests, 1)
		if r.matchedRequests > r.Limit {
			matched = true
		}
	} else {
		for _, a := range r.Aggregations {
			if a.Get(req) > r.Limit {
				matched = true
			}
		}
	}
	if matched {
		for _, a := range r.Actions {
			// Skip serving actions if we already had one
			s := a.GetResponseState()
			if state == types.SERVED && s == types.SERVED {
				continue
			}
			err := a.Act(r.Name, req, resp)
			// TODO error handling
			if err != nil {
				fmt.Println("meh", err)
			}
			if s > state {
				state = s
			}
		}
	}
	for _, sr := range r.SubRules {
		s := sr.Validate(req, resp, state)
		if s > state {
			state = s
		}
	}
	return state
}

func (a *Aggregation) Get(req *http.Request) uint64 {
	if val, found := a.Selector.Match(req); found {
		a.Lock()
		a.Values[val] += 1
		v := a.Values[val]
		a.Unlock()
		return v
	}
	return 0
}
