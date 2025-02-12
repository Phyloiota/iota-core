package permanent

import (
	"context"
	"encoding/binary"
	"io"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/core/storable"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/runtime/module"
	"github.com/iotaledger/hive.go/serializer/v2/serix"
	"github.com/iotaledger/hive.go/stringify"
	"github.com/iotaledger/iota-core/pkg/model"
	iotago "github.com/iotaledger/iota.go/v4"
)

// region Settings /////////////////////////////////////////////////////////////////////////////////////////////////////

type Settings struct {
	*settingsModel
	mutex sync.RWMutex

	api iotago.API

	latestCommitment *model.Commitment

	module.Module
}

func NewSettings(path string) (settings *Settings) {
	s := &Settings{
		settingsModel: storable.InitStruct(&settingsModel{
			SnapshotImported:        false,
			ProtocolParameters:      iotago.ProtocolParameters{},
			LatestCommitment:        iotago.NewEmptyCommitment(),
			LatestStateMutationSlot: 0,
			LatestFinalizedSlot:     0,
		}, path),
	}

	s.UpdateAPI()

	return s
}

func (s *Settings) API() iotago.API {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.api
}

func (s *Settings) SnapshotImported() (initialized bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.settingsModel.SnapshotImported
}

func (s *Settings) SetSnapshotImported(initialized bool) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settingsModel.SnapshotImported = initialized

	if err = s.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist initialized flag")
	}

	return nil
}

func (s *Settings) ProtocolParameters() *iotago.ProtocolParameters {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &s.settingsModel.ProtocolParameters
}

func (s *Settings) SetProtocolParameters(params iotago.ProtocolParameters) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settingsModel.ProtocolParameters = params
	s.UpdateAPI()

	if err = s.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist initialized flag")
	}

	return nil
}

func (s *Settings) LatestCommitment() *model.Commitment {
	s.mutex.RLock()
	if s.latestCommitment == nil {
		s.mutex.RUnlock()
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if s.api.SlotTimeProvider().Duration() == 0 {
			panic("accessing the LatestCommitment before the settings are initialized")
		}
		s.latestCommitment = lo.PanicOnErr(model.CommitmentFromCommitment(s.settingsModel.LatestCommitment, s.api))

		return s.latestCommitment
	}
	defer s.mutex.RUnlock()

	return s.latestCommitment
}

func (s *Settings) SetLatestCommitment(latestCommitment *model.Commitment) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settingsModel.LatestCommitment = latestCommitment.Commitment()

	if err = s.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist latest commitment")
	}

	s.latestCommitment = latestCommitment

	return nil
}

func (s *Settings) LatestStateMutationSlot() iotago.SlotIndex {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.settingsModel.LatestStateMutationSlot
}

func (s *Settings) SetLatestStateMutationSlot(index iotago.SlotIndex) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settingsModel.LatestStateMutationSlot = index

	if err = s.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist latest state mutation slot")
	}

	return nil
}

func (s *Settings) LatestFinalizedSlot() iotago.SlotIndex {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.settingsModel.LatestFinalizedSlot
}

func (s *Settings) SetLatestFinalizedSlot(index iotago.SlotIndex) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settingsModel.LatestFinalizedSlot = index

	if err = s.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist latest confirmed slot")
	}

	return nil
}

func (s *Settings) Export(writer io.WriteSeeker) (err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	settingsBytes, err := s.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to convert settings to bytes")
	}

	if err = binary.Write(writer, binary.LittleEndian, uint32(len(settingsBytes))); err != nil {
		return errors.Wrap(err, "failed to write settings length")
	}

	if err = binary.Write(writer, binary.LittleEndian, settingsBytes); err != nil {
		return errors.Wrap(err, "failed to write settings")
	}

	return nil
}

func (s *Settings) Import(reader io.ReadSeeker) (err error) {
	if err = s.tryImport(reader); err != nil {
		return errors.Wrap(err, "failed to import settings")
	}

	s.TriggerInitialized()

	return
}

func (s *Settings) String() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	builder := stringify.NewStructBuilder("Settings", stringify.NewStructField("path", s.FilePath()))
	builder.AddField(stringify.NewStructField("SnapshotImported", s.settingsModel.SnapshotImported))
	builder.AddField(stringify.NewStructField("ProtocolParameters", s.settingsModel.ProtocolParameters))
	builder.AddField(stringify.NewStructField("LatestCommitment", s.settingsModel.LatestCommitment))
	builder.AddField(stringify.NewStructField("LatestStateMutationSlot", s.settingsModel.LatestStateMutationSlot))
	builder.AddField(stringify.NewStructField("LatestFinalizedSlot", s.settingsModel.LatestFinalizedSlot))

	return builder.String()
}

func (s *Settings) tryImport(reader io.ReadSeeker) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var settingsSize uint32
	if err = binary.Read(reader, binary.LittleEndian, &settingsSize); err != nil {
		return errors.Wrap(err, "failed to read settings length")
	}

	settingsBytes := make([]byte, settingsSize)
	if err = binary.Read(reader, binary.LittleEndian, settingsBytes); err != nil {
		return errors.Wrap(err, "failed to read settings bytes")
	}

	if consumedBytes, fromBytesErr := s.FromBytes(settingsBytes); fromBytesErr != nil {
		return errors.Wrapf(fromBytesErr, "failed to read settings")
	} else if consumedBytes != len(settingsBytes) {
		return errors.Errorf("failed to read settings: consumed bytes (%d) != expected bytes (%d)", consumedBytes, len(settingsBytes))
	}

	s.settingsModel.SnapshotImported = true

	s.UpdateAPI()

	if err = s.settingsModel.ToFile(); err != nil {
		return errors.Wrap(err, "failed to persist chain ID")
	}

	return
}

func (s *Settings) UpdateAPI() {
	s.api = iotago.LatestAPI(&s.settingsModel.ProtocolParameters)
	iotago.SwapInternalAPI(s.api)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region settingsModel ////////////////////////////////////////////////////////////////////////////////////////////////

type settingsModel struct {
	SnapshotImported        bool                      `serix:"0"`
	ProtocolParameters      iotago.ProtocolParameters `serix:"1"`
	LatestCommitment        *iotago.Commitment        `serix:"2"`
	LatestStateMutationSlot iotago.SlotIndex          `serix:"3"`
	LatestFinalizedSlot     iotago.SlotIndex          `serix:"4"`

	storable.Struct[settingsModel, *settingsModel]
}

func (s *settingsModel) FromBytes(bytes []byte) (int, error) {
	return serix.DefaultAPI.Decode(context.Background(), bytes, s)
}

func (s settingsModel) Bytes() ([]byte, error) {
	return serix.DefaultAPI.Encode(context.Background(), s)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
