package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/action"
	"github.com/asciimoo/filtron/selector"
	"github.com/asciimoo/filtron/types"
)

type Rule struct {
	Interval         uint64               `json:"interval"`
	Limit            uint64               `json:"limit"`
	Name             string               `json:"name"`
	lastTick         uint64               `json:"-"`
	MatchCount       uint64               `json:"match_count"`
	filterMatchCount uint64               `json:"-"`
	Filters          []*selector.Selector `json:"-"`
	RawFilters       []string             `json:"filters"`
	Aggregations     []*Aggregation       `json:"-"`
	RawAggregations  []string             `json:"aggregations"`
	Actions          []action.Action      `json:"-"`
	RawActions       []action.ActionJSON  `json:"actions"`
	SubRules         []*Rule              `json:"subrules"`
	Disabled         bool                 `json:"disabled"`
	Stop             bool                 `json:"stop"`
}

type Aggregation struct {
	sync.RWMutex
	Values   map[string]uint64
	Selector *selector.Selector
}

func Evaluate(rules *[]*Rule, ctx *fasthttp.RequestCtx) types.ResponseState {
	respState := types.UNTOUCHED
	validateRuleList(rules, &respState, ctx)
	return respState
}

func validateRuleList(rules *[]*Rule, state *types.ResponseState, ctx *fasthttp.RequestCtx) {
	for _, rule := range *rules {
		if rule.Disabled {
			continue
		}

		prevMatchCount := rule.MatchCount

		s := rule.Validate(ctx, *state)

		if s > *state {
			*state = s
		}

		if rule.Stop && prevMatchCount < rule.MatchCount {
			break
		}
	}
}

func RulesLength(rules []*Rule) uint64 {
	var len uint64 = 0
	for _, rule := range rules {
		len += 1
		len += RulesLength(rule.SubRules)
	}
	return len
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

func ParseJSONFile(filename string) ([]*Rule, error) {
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseJSON(fileContent)
}

func ParseJSON(jsonData []byte) ([]*Rule, error) {
	var rules []*Rule
	err := json.Unmarshal(jsonData, &rules)
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
	r.filterMatchCount = 0
	r.lastTick = uint64(time.Now().Unix())
	if len(r.RawActions) == 0 && len(r.SubRules) == 0 {
		return errors.New(fmt.Sprintf("At least one subrule or action required in rule: %q", r.Name))
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

func (r *Rule) Validate(ctx *fasthttp.RequestCtx, rs types.ResponseState) types.ResponseState {
	curTime := uint64(time.Now().Unix())
	if r.Limit != 0 && curTime-r.lastTick >= r.Interval {
		r.filterMatchCount = 0
		atomic.StoreUint64(&r.filterMatchCount, 0)
		atomic.StoreUint64(&r.lastTick, curTime)
		for _, a := range r.Aggregations {
			a.Lock()
			a.Values = make(map[string]uint64)
			a.Unlock()
		}
	}
	for _, t := range r.Filters {
		if _, found := t.Match(ctx); !found {
			return types.UNTOUCHED
		}
	}
	matched := false
	state := rs
	if len(r.Aggregations) == 0 {
		atomic.AddUint64(&r.filterMatchCount, 1)
		if r.filterMatchCount > r.Limit {
			matched = true
		}
	} else {
		for _, a := range r.Aggregations {
			if a.Get(ctx) > r.Limit {
				matched = true
			}
		}
	}
	if matched {
		atomic.AddUint64(&r.MatchCount, 1)
		for _, a := range r.Actions {
			// Skip serving actions if we already had one
			s := a.GetResponseState()
			if state == types.SERVED && s == types.SERVED {
				continue
			}
			err := a.Act(r.Name, ctx)
			// TODO error handling
			if err != nil {
				fmt.Println("meh", err)
			}
			if s > state {
				state = s
			}
		}
	}
	if !r.Stop {
		validateRuleList(&r.SubRules, &state, ctx)
	}
	return state
}

func (a *Aggregation) Get(ctx *fasthttp.RequestCtx) uint64 {
	if val, found := a.Selector.Match(ctx); found {
		a.Lock()
		a.Values[val] += 1
		v := a.Values[val]
		a.Unlock()
		return v
	}
	return 0
}
