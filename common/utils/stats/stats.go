package stats

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/tevjef/go-runtime-metrics/collector"
)

// consider adding a read synchronizer channel
type StatsCollectorT struct {
	internal *internal
	collector.Collector
	Mailbox  chan func()
	stopPoll chan struct{}
}

type internal struct {
	sync.Map
}

type StatsItem struct {
	Key   string
	Value interface{}
}

const StartTime = "StartTime"

// singleton
var StatsCollector StatsCollectorT

func init() {
	StatsCollector = New()
}

func New() (sc StatsCollectorT) {
	sc.internal = new(internal)
	sc.SetStartTime(time.Now().UTC())
	sc.Collector = collector.Collector{
		EnableCPU: true,
		EnableMem: true,
		EnableGC:  true,
	}
	// initialize the mailbox with a buffer
	// reads are not guaranteed to capture the last write
	sc.Mailbox = make(chan func(), 10)
	sc.stopPoll = make(chan struct{})
	// begin loop
	sc.Loop()

	return sc
}

// helper func for getting getting nested values
func get(m *internal, path ...string) interface{} {
	var ok bool
	var interf interface{}
	var length = len(path)

	for i, p := range path {
		var last = i == length-1
		if last {
			if interf, ok = m.Load(p); !ok {
				return nil
			}
		}
		if interf, ok = m.Load(p); !ok {
			return nil
		} else if m, ok = interf.(*internal); !ok {
			continue
		}
	}
	return interf
}

// helper func for setting getting nested values
func set(m *internal, value interface{}, path ...string) {
	var length = len(path)
	var ok bool
	var interf interface{}
	var newMap *internal

	for i, p := range path {
		var last = i == length-1
		if last {
			m.Store(p, value)
		} else {
			if interf, ok = m.Load(p); !ok {
				newMap = new(internal)
				m.Store(p, newMap)
				m = newMap
			} else {
				if newMap, ok = interf.(*internal); !ok {
					newMap = new(internal)
					m.Store(p, newMap)
				}
				m = newMap
			}
		}
	}
}

func (sc *StatsCollectorT) Loop() {
	go func() {
		defer close(sc.Mailbox)
		for {
			select {
			// do the work passed to the mailbox
			case action := <-sc.Mailbox:
				action()
			}
		}
	}()
}

func (sc *StatsCollectorT) GetUptime() int64 {
	var startTime = get(sc.internal, StartTime)
	return time.Now().UTC().Sub(startTime.(time.Time)).Nanoseconds()
}

func (sc *StatsCollectorT) SetStartTime(startTime time.Time) {
	sc.internal.Store(StartTime, startTime)
}

func (sc *StatsCollectorT) IncrementUInt64AtFieldPath(fields ...string) {
	sc.Mailbox <- func() { sc.incrementUInt64AtFieldPath(fields...) }
}

func (sc *StatsCollectorT) incrementUInt64AtFieldPath(fields ...string) {
	var valI interface{}

	if valI = get(sc.internal, fields...); valI == nil {
		set(sc.internal, uint64(1), fields...)
		return
	}

	if val, ok := valI.(uint64); ok {
		val++
		set(sc.internal, val, fields...)
	}
}

func (sc *StatsCollectorT) SetValueAtFieldPath(value interface{}, fields ...string) {
	sc.Mailbox <- func() { sc.setValueAtFieldPath(value, fields...) }
}

func (sc *StatsCollectorT) setValueAtFieldPath(value interface{}, fields ...string) {
	set(sc.internal, value, fields...)
}

func (sc *StatsCollectorT) AddUInt64AtFieldPath(value uint64, fields ...string) {
	sc.Mailbox <- func() { sc.addUInt64AtFieldPath(value, fields...) }
}

func (sc *StatsCollectorT) addUInt64AtFieldPath(value uint64, fields ...string) {
	var originalValue = get(sc.internal, fields...)
	var val uint64
	var ok bool

	if val, ok = originalValue.(uint64); !ok {
		val = 0
	}
	val += value
	set(sc.internal, val, fields...)
}

func (sc *StatsCollectorT) GetValueAtField(fields ...string) interface{} {
	return get(sc.internal, fields...)
}

func (sc *StatsCollectorT) GetRuntimeStats() interface{} {
	return sc.OneOff()
}

func (sc *StatsCollectorT) MarshalJSON() ([]byte, error) {
	return json.Marshal(sc.internal.getState())
}

func (sc *StatsCollectorT) PollState(d time.Duration) chan map[string]interface{} {
	var ticker = time.NewTicker(d)
	var channel = make(chan map[string]interface{})

	go func() {
		for {
			select {
			case <-ticker.C:
				channel <- sc.internal.getState()
			case <-sc.stopPoll:
				break
			}
		}
	}()

	return channel
}

func (sc *StatsCollectorT) StopPolling() {
	sc.stopPoll <- struct{}{}
}

func (i *internal) getState() map[string]interface{} {
	var stateCopy = make(map[string]interface{})

	i.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			if val, ok := value.(*internal); ok {
				// recursively add inner map values
				internalState := val.getState()
				stateCopy[keyStr] = internalState
			} else {
				stateCopy[keyStr] = value
			}
		}
		// continue iterating
		return true
	})

	return stateCopy
}
