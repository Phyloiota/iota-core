package storage

import (
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/iota-core/pkg/storage/prunable"
)

func WithDBEngine(optsDBEngine hivedb.Engine) options.Option[Storage] {
	return func(s *Storage) {
		s.optsDBEngine = optsDBEngine
	}
}

func WithAllowedDBEngines(optsAllowedDBEngines []hivedb.Engine) options.Option[Storage] {
	return func(s *Storage) {
		s.optsAllowedDBEngines = optsAllowedDBEngines
	}
}

func WithPrunableManagerOptions(opts ...options.Option[prunable.Manager]) options.Option[Storage] {
	return func(s *Storage) {
		s.optsPrunableManagerOptions = append(s.optsPrunableManagerOptions, opts...)
	}
}
