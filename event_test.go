package goeventemitter_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	eventemitter "github.com/thiagodk/go-event-emitter"
)

type EventCount map[string]int

func (evcount *EventCount) Add(evname string, evemitter *eventemitter.EventEmitter, t *testing.T) {
	if count, hasev := (*evcount)[evname]; hasev {
		(*evcount)[evname] = count + 1
	} else {
		(*evcount)[evname] = 1
	}
	evcount.Check(evemitter, t)
}

func (evcount *EventCount) Remove(evname string, evemitter *eventemitter.EventEmitter, t *testing.T) {
	if count, hasev := (*evcount)[evname]; hasev && count > 0 {
		(*evcount)[evname] = count - 1
	} else {
		t.Errorf("Removing non-existant handler for event '%s'", evname)
	}
	evcount.Check(evemitter, t)
}

func (evcount *EventCount) RemoveAll(evname string, evemitter *eventemitter.EventEmitter, t *testing.T) {
	if count, hasev := (*evcount)[evname]; hasev && count > 0 {
		(*evcount)[evname] = 0
	} else {
		t.Errorf("Removing non-existant handler for event '%s'", evname)
	}
	evcount.Check(evemitter, t)
}

func (evcount *EventCount) Check(evemitter *eventemitter.EventEmitter, t *testing.T) {
	for _, evname := range evemitter.EventNames() {
		count := evemitter.ListenerCount(evname)
		expectcount, hasev := (*evcount)[evname]
		if !hasev {
			t.Errorf("Unexpected event '%s'", evname)
			continue
		}
		if uint(expectcount) != count {
			t.Errorf("Mismatch event '%s' count. Expected %d but have %d", evname, expectcount, count)
		}
	}
}

func getIncrFunc(incrval int, wg *sync.WaitGroup) eventemitter.EventCallback {
	return func(args ...interface{}) {
		i := args[0].(*int)
		*i += incrval
		wg.Done()
	}
}

func waitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	sig := make(chan struct{})
	go func() {
		defer close(sig)
		wg.Wait()
	}()
	select {
	case <-sig:
		return false

	case <-time.After(timeout):
		return true
	}
}

func TestEvent(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	evtest := eventemitter.NewEventEmitter(3)
	var wg, wgincr sync.WaitGroup
	var cb eventemitter.EventCallback

	evcounts := make(EventCount)

	wg.Add(4)
	incrint := 1
	randint := rand.Int()
	t.Run("onInt", func(t *testing.T) {
		expectval := randint
		evtest.On("testInt", func(args ...interface{}) {
			i := args[0].(int)
			if expectval != i {
				t.Errorf("Event got different integer. Expected %d but have %d", expectval, i)
			}
		})
		evcounts.Add("testInt", evtest, t)
		wg.Done()
	})
	t.Run("onString", func(t *testing.T) {
		expectval := "test"
		evtest.On("testString", func(args ...interface{}) {
			s := args[0].(string)
			if expectval != s {
				t.Errorf("Event got different string. Expected '%s' but have '%s'", expectval, s)
			}
		})
		evcounts.Add("testString", evtest, t)
		wg.Done()
	})
	t.Run("onBool", func(t *testing.T) {
		evbool := false
		evtest.Once("testBool", func(args ...interface{}) {
			b := args[0].(bool)
			if evbool == b {
				t.Error("Once event was called twice. Expected once")
			} else {
				evbool = b
			}
			evcounts.Remove("testBool", evtest, t)
		})
		evcounts.Add("testBool", evtest, t)
		wg.Done()
	})
	t.Run("onIncr", func(t *testing.T) {
		cb = getIncrFunc(1, &wgincr)
		evtest.On("onIncr", cb)
		evcounts.Add("onIncr", evtest, t)
		evtest.On("onIncr", getIncrFunc(2, &wgincr))
		evcounts.Add("onIncr", evtest, t)
		evtest.On("onIncr", getIncrFunc(3, &wgincr))
		evcounts.Add("onIncr", evtest, t)
		evtest.On("onIncr", getIncrFunc(4, &wgincr))
		evcounts.Check(evtest, t)
		wg.Done()
	})
	if waitGroupTimeout(&wg, 30 * time.Second) {
		t.Fatal("Timeout for setting events task")
	}
	evtest.Emit("testInt", randint)
	evtest.Emit("testString", "test")
	evtest.Emit("testBool", true)
	evtest.Emit("testBool", true)
	if incrint != 1 {
		t.Errorf("Invalid initial onIncr value. Expected 1 but have %d", incrint)
	}
	wgincr.Add(3)
	evtest.Emit("onIncr", &incrint)
	if waitGroupTimeout(&wgincr, 5 * time.Second) {
		t.Fatal("Timeout for first emitting onIncr")
	}
	if incrint != 7 {
		t.Errorf("Invalid event onIncr value. Expected 7 but have %d", incrint)
	}
	wgincr.Add(3)
	evtest.Emit("onIncr", &incrint)
	if waitGroupTimeout(&wgincr, 5 * time.Second) {
		t.Fatal("Timeout for second emitting onIncr")
	}
	if incrint != 13 {
		t.Errorf("Invalid event onIncr value. Expected 13 but have %d", incrint)
	}
	evtest.RemoveListener("onIncr", cb)
	evcounts.Remove("onIncr", evtest, t)
	wgincr.Add(2)
	evtest.Emit("onIncr", &incrint)
	if waitGroupTimeout(&wgincr, 5 * time.Second) {
		t.Fatal("Timeout for third emitting onIncr")
	}
	if incrint != 18 {
		t.Errorf("Invalid event onIncr value. Expected 18 but have %d", incrint)
	}
	evtest.RemoveAllListeners("onIncr")
	evcounts.RemoveAll("onIncr", evtest, t)
	evtest.Emit("onIncr", &incrint)
	time.Sleep(1 * time.Second)
	if incrint != 18 {
		t.Errorf("Invalid unchanged onIncr value. Expected 18 but have %d", incrint)
	}
}
