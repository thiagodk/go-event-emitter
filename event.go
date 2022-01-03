package goeventemitter

import (
	"container/list"
	"reflect"
	"sync"
)

type EventCallback func(args... interface{})

type EventCallable struct {
	CallableFunc EventCallback
	OnceFlag bool
}

type EventHandler struct {
	sync.Mutex
	eventCount uint
	handlers list.List
}

func (handler *EventHandler) Append(callable *EventCallable) *EventHandler {
	handler.Lock()
	defer handler.Unlock()
	handler.handlers.PushBack(callable)
	return handler
}

func (handler *EventHandler) Prepend(callable *EventCallable) *EventHandler {
	handler.Lock()
	defer handler.Unlock()
	handler.handlers.PushFront(callable)
	return handler
}

func (handler *EventHandler) Iterate(cb func(callable *EventCallable)) {
	var next *list.Element

	for e := handler.handlers.Front(); e != nil; e = next {
		next = e.Next()
		cb(e.Value.(*EventCallable))
	}
}

func (handler *EventHandler) Remove(callable *EventCallable) bool {
	var next *list.Element

	handler.Lock()
	defer handler.Unlock()
	for e := handler.handlers.Front(); e != nil; e = next {
		next = e.Next()
		if e.Value == callable {
			handler.handlers.Remove(e)
			return true
		}
	}
	return false
}

func (handler *EventHandler) FindCallback(cb EventCallback) *EventCallable {
	handler.Lock()
	defer handler.Unlock()
	cbval := reflect.ValueOf(cb)
	for e := handler.handlers.Front(); e != nil; e = e.Next() {
		callable := e.Value.(*EventCallable)
		ecbval := reflect.ValueOf(callable.CallableFunc)
		if ecbval.Pointer() == cbval.Pointer() {
			return callable
		}
	}
	return nil
}

func (handler *EventHandler) GetEventCount() uint {
	return handler.eventCount
} 

func (handler *EventHandler) GetHandlersCount() uint {
	return uint(handler.handlers.Len())
}

func (handler *EventHandler) Call(args... interface{}) {
	var next *list.Element

	handler.Lock()
	defer handler.Unlock()
	for e := handler.handlers.Front(); e != nil; e = next {
		callable := e.Value.(*EventCallable)
		next = e.Next()
		go callable.CallableFunc(args...)
		if callable.OnceFlag {
			handler.handlers.Remove(e)
		}
	}
}

type EventEmitter struct {
	sync.Mutex
	MaxListener uint
	handlers map[string]*EventHandler
}

func NewEventEmitter(maxlistener uint) *EventEmitter {
	emitter := new(EventEmitter)
	emitter.MaxListener = maxlistener
	emitter.handlers = make(map[string]*EventHandler)
	return emitter
}

func (emitter *EventEmitter) Emit(name string, args... interface{}) {
	emitter.Lock()
	defer emitter.Unlock()
	if handler, hasevt := emitter.handlers[name]; hasevt {
		handler.Call(args...)
	}
}

func (emitter *EventEmitter) ListenerCount(name string) uint {
	emitter.Lock()
	defer emitter.Unlock()
	if handler, hasevt := emitter.handlers[name]; hasevt {
		return handler.GetHandlersCount()
	}
	return 0
}

func (emitter *EventEmitter) getOrCreateHandler(name string) *EventHandler {
	handler, hasevt := emitter.handlers[name]
	if !hasevt {
		handler = new(EventHandler)
		emitter.handlers[name] = handler
	}
	return handler
}

func (emitter *EventEmitter) addHandler(name string, cb EventCallback, once bool, end bool) bool {
	handler := emitter.getOrCreateHandler(name)
	if handler.GetHandlersCount() >= emitter.MaxListener {
		return false
	}
	callable := EventCallable{cb, once}
	if end {
		handler.Append(&callable)
	} else {
		handler.Prepend(&callable)
	}
	go emitter.Emit("newListener", name, cb)
	return true
}

func (emitter *EventEmitter) On(name string, cb EventCallback) bool {
	emitter.Lock()
	defer emitter.Unlock()
	return emitter.addHandler(name, cb, false, true)
}

func (emitter *EventEmitter) AddListener(name string, cb EventCallback) bool {
	return emitter.On(name, cb)
}

func (emitter *EventEmitter) Once(name string, cb EventCallback) bool {
	emitter.Lock()
	defer emitter.Unlock()
	return emitter.addHandler(name, cb, true, true)
}

func (emitter *EventEmitter) PrependListener(name string, cb EventCallback) bool {
	emitter.Lock()
	defer emitter.Unlock()
	return emitter.addHandler(name, cb, false, false)
}

func (emitter *EventEmitter) PrependOnceListener(name string, cb EventCallback) bool {
	emitter.Lock()
	defer emitter.Unlock()
	return emitter.addHandler(name, cb, true, false)
}

func (emitter *EventEmitter) RemoveAllListeners(name string) *EventEmitter {
	emitter.Lock()
	defer emitter.Unlock()
	if handler, hasevt := emitter.handlers[name]; hasevt {
		handler.Iterate(func(callable *EventCallable) {
			if handler.Remove(callable) {
				go emitter.Emit("removeListener", name, callable.CallableFunc)
			}
		})
	}
	return emitter
}

func (emitter *EventEmitter) RemoveListener(name string, cb EventCallback) bool {
	emitter.Lock()
	defer emitter.Unlock()
	if handler, hasevt := emitter.handlers[name]; hasevt {
		callable := handler.FindCallback(cb)
		if callable == nil {
			return false
		}
		removed := handler.Remove(callable)
		if removed {
			go emitter.Emit("removeListener", name, callable.CallableFunc)
		}
		return removed
	}
	return false
}

func (emitter *EventEmitter) EventNames() []string {
	var names []string
	emitter.Lock()
	defer emitter.Unlock()
	for name := range emitter.handlers {
		names = append(names, name)
	}
	return names
}

func (emitter *EventEmitter) RawListeners(name string) []EventCallback {
	var cbs []EventCallback
	emitter.Lock()
	defer emitter.Unlock()
	if handler, hasevt := emitter.handlers[name]; hasevt {
		handler.Iterate(func(callable *EventCallable) {
			cbs = append(cbs, callable.CallableFunc)
		})
	}
	return cbs
}
