package rpcruntime

import (
	"sync"
	"sync/atomic"
	"time"
)

type ErrorID uint32

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
	errorSeq atomic.Uint32
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

	next := errorSeq.Add(1)
	if next == 0 {
		next = errorSeq.Add(1)
	}
	id := ErrorID(next)
	store := errorRecords
	record := errorRecord{
		text:      err.Error(),
		expiresAt: time.Now().Add(errorTTL),
	}
	store.store(id, record)
	errorCleanupScheduler.schedule(uint64(id), errorTTL, func() {
		store.delete(id)
	})
	return id
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
	defer s.mu.Unlock()

	record, ok := s.records[id]
	if !ok {
		return preparedErrorText{}, false
	}
	if s.expired(record, time.Now()) {
		delete(s.records, id)
		errorCleanupScheduler.cancel(uint64(id))
		return preparedErrorText{}, false
	}

	prepared, err := prepare(record)
	if err != nil {
		return preparedErrorText{}, false
	}

	delete(s.records, id)
	errorCleanupScheduler.cancel(uint64(id))
	return prepared, true
}

func (s *errorStore) has(id ErrorID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	if !ok {
		return false
	}
	if s.expired(record, time.Now()) {
		delete(s.records, id)
		errorCleanupScheduler.cancel(uint64(id))
		return false
	}
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
