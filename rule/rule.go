package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/action"
	"github.com/asciimoo/filtron/selector"
	"github.com/asciimoo/filtron/types"
)

type Rule struct {
	sync.RWMutex

	Name                    string                       `json:"name"`
	Interval                uint64                       `json:"interval"`
	Limit                   uint64                       `json:"limit"`
	RequestCount            uint64                       `json:"requestCount"`
	MatchCount              uint64                       `json:"matchCount"`
	Stop                    bool                         `json:"stop"`
	Disabled                bool                         `json:"disabled"`
	Filters                 []*selector.Selector         `json:"-"`
	RawFilters              []string                     `json:"filters"`
	AggregationSelectors    []*selector.Selector         `json:"-"`
	AggregationValues       map[string]*AggregationValue `json:"values"`
	DefaultAggregationValue *AggregationValue            `json:"-"`
	RawAggregations         []string                     `json:"aggregations"`
	Actions                 []action.Action              `json:"-"`
	RawActions              []action.ActionJSON          `json:"actions"`
	SubRules                []*Rule                      `json:"subrules"`
}

type AggregationValue struct {
	LastTick uint64 `json:"lastTick"`
	Count    uint64 `json:"count"`
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
		log.Println("rule ", rule.Name, s)
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
	selectors := make([]*selector.Selector, 0, len(aggregations))
	for _, t := range aggregations {
		s, err := selector.Parse(t)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot parse selector '%v': %v", t, err))
		}
		selectors = append(selectors, s)
	}
	r.AggregationSelectors = selectors
	r.AggregationValues = make(map[string]*AggregationValue)
	if len(aggregations) == 0 {
		r.DefaultAggregationValue = NewAggreationValue()
		r.AggregationValues["*"] = r.DefaultAggregationValue
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
	// Does it pass all the filters ?
	for _, t := range r.Filters {
		if _, found := t.Match(ctx); !found {
			return types.UNTOUCHED
		}
	}

	//
	requestCount := atomic.AddUint64(&r.RequestCount, 1)
	if requestCount%10 == 0 {
		r.EraseOldAggregationValues()
	}

	// Does it hit the limit ?
	state := rs
	if r.Match(ctx) {
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

func (r *Rule) Match(ctx *fasthttp.RequestCtx) bool {
	// Match the aggregation: increment & check if it is above the limit

	// Get the AggregationValue for the context
	var av *AggregationValue
	if len(r.AggregationSelectors) == 0 {
		// No aggregations: default value
		av = r.DefaultAggregationValue
	} else {
		// Aggregation: get the key
		key := ""
		for _, s := range r.AggregationSelectors {
			// Check
			value, _ := s.Match(ctx)
			// Concat
			key = key + "|" + value
		}
		log.Println("Aggregation key", key)

		// Check if value exists : no --> call NewAggreationValue
		var ok bool
		var newAv *AggregationValue = nil
		if av, ok = r.AggregationValues[key]; !ok {
			// memory allocation outside the Lock/Unlock block
			newAv = NewAggreationValue()
		}

		//
		r.Lock()
		av, ok = r.AggregationValues[key]
		if !ok {
			if newAv == nil {
				// Should not happen
				newAv = NewAggreationValue()
			}
			av = newAv
			r.AggregationValues[key] = av
		}
		r.Unlock()
	}
	// Increment, and return true is the limit has been reached
	return r.IncAndMatch(av)
}

func (r *Rule) IncAndMatch(av *AggregationValue) bool {
	if r.Limit > 0 {
		curTime := uint64(time.Now().Unix())

		if curTime-atomic.LoadUint64(&av.LastTick) >= r.Interval {
			atomic.StoreUint64(&av.Count, 0)
			atomic.StoreUint64(&av.LastTick, curTime)
		}
		return atomic.AddUint64(&av.Count, 1) > r.Limit
	} else {
		atomic.AddUint64(&av.Count, 1)
		return true
	}
}

func (r *Rule) EraseOldAggregationValues() {
	if len(r.AggregationValues) > 1 {
		curTime := uint64(time.Now().Unix())

		r.Lock()
		for k, av := range r.AggregationValues {
			if curTime-av.LastTick >= r.Interval {
				delete(r.AggregationValues, k)
			}
		}
		r.Unlock()
	}
}

func NewAggreationValue() *AggregationValue {
	return &AggregationValue{
		Count:    0,
		LastTick: uint64(time.Now().Unix()),
	}
}
