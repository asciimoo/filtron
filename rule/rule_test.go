package rule

import (
	"testing"
)

func TestNew(t *testing.T) {
	r, err := New("test rule", 10, 2, []string{"GET:a=b", "IP"})
	if err != nil {
		t.Error("Cannot create rule:", err)
		return
	}
	if len(r.Triggers) != 2 {
		t.Error("Invalid length of triggers:", len(r.Triggers))
	}
}
