package rpcruntime

import (
	"sync"
	"sync/atomic"
	"time"
)

type ErrorID int32

type errorRecord struct {
	text      string
	expiresAt time.Time
}

type preparedErrorText struct {
	data   []byte
	ptr    uintptr
	length int32
}

type errorStore struct {
	mu      sync.RWMutex
	records map[ErrorID]errorRecord
}

var (
	errorSeq atomic.Int32
	errorTTL = 3 * time.Second

	errorRecords                    = newErrorStore()
	errorCleanupScheduler           = newCleanupScheduler(100*time.Millisecond, 256)
	pinErrorText                    = PinString
	errorTextLengthToInt32ForExport = LengthToInt32
)

func StoreError(err error) ErrorID {
	if err == nil {
		return 0
	}

	next := nextErrorID()
	id := ErrorID(next)
	store := errorRecords
	record := errorRecord{
		text:      err.Error(),
		expiresAt: time.Now().Add(errorTTL),
	}
	store.store(id, record)
	errorCleanupScheduler.schedule(int32(id), errorTTL, func() {
		store.delete(id)
	})
	return id
}

func nextErrorID() int32 {
	for {
		current := errorSeq.Load()
		next := current + 1
		if current >= 1<<31-1 || next <= 0 {
			next = 1
		}
		if errorSeq.CompareAndSwap(current, next) {
			return next
		}
	}
}

func TakeErrorText(id ErrorID) ([]byte, uintptr, bool) {
	if id == 0 {
		return nil, 0, false
	}

	prepared, ok := errorRecords.takePrepared(id, func(record errorRecord) (preparedErrorText, error) {
		data, ptr, err := pinErrorText(record.text)
		if err != nil {
			return preparedErrorText{}, err
		}
		return preparedErrorText{data: data, ptr: ptr}, nil
	})
	if !ok {
		return nil, 0, false
	}
	return prepared.data, prepared.ptr, true
}

func takeErrorTextForExport(id ErrorID) (preparedErrorText, bool) {
	if id == 0 {
		return preparedErrorText{}, false
	}

	return errorRecords.takePrepared(id, func(record errorRecord) (preparedErrorText, error) {
		data, ptr, err := pinErrorText(record.text)
		if err != nil {
			return preparedErrorText{}, err
		}
		length, err := errorTextLengthToInt32ForExport(len(data))
		if err != nil {
			Release(ptr)
			return preparedErrorText{}, err
		}
		return preparedErrorText{
			data:   data,
			ptr:    ptr,
			length: length,
		}, nil
	})
}

func newErrorStore() *errorStore {
	return &errorStore{records: make(map[ErrorID]errorRecord)}
}

func (s *errorStore) store(id ErrorID, record errorRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = record
}

func (s *errorStore) takePrepared(id ErrorID, prepare func(errorRecord) (preparedErrorText, error)) (preparedErrorText, bool) {
	s.mu.Lock()
	cancelCleanup := false

	record, ok := s.records[id]
	if !ok {
		s.mu.Unlock()
		return preparedErrorText{}, false
	}
	if s.expired(record, time.Now()) {
		delete(s.records, id)
		cancelCleanup = true
		s.mu.Unlock()
		if cancelCleanup {
			errorCleanupScheduler.cancel(int32(id))
		}
		return preparedErrorText{}, false
	}

	prepared, err := prepare(record)
	if err != nil {
		s.mu.Unlock()
		return preparedErrorText{}, false
	}

	delete(s.records, id)
	cancelCleanup = true
	s.mu.Unlock()
	if cancelCleanup {
		errorCleanupScheduler.cancel(int32(id))
	}
	return prepared, true
}

func (s *errorStore) has(id ErrorID) bool {
	s.mu.Lock()
	cancelCleanup := false
	record, ok := s.records[id]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if s.expired(record, time.Now()) {
		delete(s.records, id)
		cancelCleanup = true
		s.mu.Unlock()
		if cancelCleanup {
			errorCleanupScheduler.cancel(int32(id))
		}
		return false
	}
	s.mu.Unlock()
	return true
}

func (s *errorStore) delete(id ErrorID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
}

func (s *errorStore) expired(record errorRecord, now time.Time) bool {
	return !now.Before(record.expiresAt)
}
