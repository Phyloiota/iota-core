package coreapi

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota-core/components/metricstracker"
	"github.com/iotaledger/iota-core/components/restapi"
	"github.com/iotaledger/iota-core/pkg/blockissuer"
	"github.com/iotaledger/iota-core/pkg/protocol"
	restapipkg "github.com/iotaledger/iota-core/pkg/restapi"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	// RouteInfo is the route for getting the node info.
	// GET returns the node info.
	RouteInfo = "/info"

	// RouteBlockIssuance is the route for getting all needed information for block creation.
	// GET returns the data needed toa attach block.
	RouteBlockIssuance = "/blocks/issuance"

	// RouteBlock is the route for getting a block by its blockID.
	// GET returns the block based on the given type in the request "Accept" header.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteBlock = "/blocks/:" + restapipkg.ParameterBlockID

	// RouteBlockMetadata is the route for getting block metadata by its blockID.
	// GET returns block metadata.
	RouteBlockMetadata = "/blocks/:" + restapipkg.ParameterBlockID + "/metadata"

	// RouteBlocks is the route for creating new blocks.
	// POST creates a single new block and returns the new block ID.
	// The block is parsed based on the given type in the request "Content-Type" header.
	// By providing only the protocolVersion and payload transaction user can POST a transaction.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteBlocks = "/blocks"

	// RouteOutput is the route for getting an output by its outputID (transactionHash + outputIndex).
	// GET returns the output based on the given type in the request "Accept" header.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteOutput = "/outputs/:" + restapipkg.ParameterOutputID

	// RouteOutputMetadata is the route for getting output metadata by its outputID (transactionHash + outputIndex) without getting the data again.
	// GET returns the output metadata.
	RouteOutputMetadata = "/outputs/:" + restapipkg.ParameterOutputID + "/metadata"

	// RouteTransactionsIncludedBlock is the route for getting the block that was first confirmed for a given transaction ID.
	// GET returns the block based on the given type in the request "Accept" header.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteTransactionsIncludedBlock = "/transactions/:" + restapipkg.ParameterTransactionID + "/included-block"

	// RouteTransactionsIncludedBlockMetadata is the route for getting the block metadata that was first confirmed in the ledger for a given transaction ID.
	// GET returns block metadata (including info about "promotion/reattachment needed").
	RouteTransactionsIncludedBlockMetadata = "/transactions/:" + restapipkg.ParameterTransactionID + "/included-block/metadata"

	// RouteCommitmentByID is the route for getting a slot commitment by its ID.
	// GET returns the commitment.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteCommitmentByID = "/commitments/:" + restapipkg.ParameterCommitmentID

	// RouteCommitmentByIDUTXOChanges is the route for getting all UTXO changes of a commitment by its ID.
	// GET returns the output IDs of all UTXO changes.
	RouteCommitmentByIDUTXOChanges = "/commitments/:" + restapipkg.ParameterCommitmentID + "/utxo-changes"

	// RouteCommitmentByIndex is the route for getting a commitment by its SlotIndex.
	// GET returns the commitment.
	// MIMEApplicationJSON => json.
	// MIMEVendorIOTASerializer => bytes.
	RouteCommitmentByIndex = "/commitments/by-index/:" + restapipkg.ParameterSlotIndex

	// RouteCommitmentByIndexUTXOChanges is the route for getting all UTXO changes of a commitment by its SlotIndex.
	// GET returns the output IDs of all UTXO changes.
	RouteCommitmentByIndexUTXOChanges = "/commitments/by-index/:" + restapipkg.ParameterSlotIndex + "/utxo-changes"

	// RouteAccountsByAcciuntID is the route for getting an account by its accountID.
	// GET returns the account details.
	RouteAccountsByAcciuntID = "/accounts/:" + restapipkg.ParameterAccountID

	// RouteAccountMana is the route for getting an account mana by its accountID.
	// GET returns the account mana details.
	RouteAccountMana = "/accounts/:" + restapipkg.ParameterAccountID + "/mana"

	// RoutePeer is the route for getting peers by their peerID.
	// GET returns the peer
	// DELETE deletes the peer.
	RoutePeer = "/peers/:" + restapipkg.ParameterPeerID

	// RoutePeers is the route for getting all peers of the node.
	// GET returns a list of all peers.
	// POST adds a new peer.
	RoutePeers = "/peers"

	// RouteControlDatabasePrune is the control route to manually prune the database.
	// POST prunes the database.
	RouteControlDatabasePrune = "/control/database/prune"

	// RouteControlSnapshotsCreate is the control route to manually create a snapshot files.
	// POST creates a full snapshot.
	RouteControlSnapshotsCreate = "/control/snapshots/create"
)

func init() {
	Component = &app.Component{
		Name:      "CoreAPIV3",
		DepsFunc:  func(cDeps dependencies) { deps = cDeps },
		Configure: configure,
		IsEnabled: func(c *dig.Container) bool {
			return restapi.ParamsRestAPI.Enabled
		},
	}
}

var (
	Component *app.Component
	deps      dependencies

	features = []string{}
)

type dependencies struct {
	dig.In

	Protocol         *protocol.Protocol
	AppInfo          *app.Info
	RestRouteManager *restapi.RestRouteManager
	BlockIssuer      *blockissuer.BlockIssuer
	MetricsTracker   *metricstracker.MetricsTracker
}

func configure() error {
	// check if RestAPI plugin is disabled
	if !Component.App().IsComponentEnabled(restapi.Component.Identifier()) {
		Component.LogPanic("RestAPI plugin needs to be enabled to use the CoreAPIV3 plugin")
	}

	routeGroup := deps.RestRouteManager.AddRoute("core/v3")

	// Check for features
	if restapi.ParamsRestAPI.PoW.Enabled {
		AddFeature("pow")
	}

	if restapi.ParamsRestAPI.AllowIncompleteBlock {
		AddFeature("allowIncompleteBlock")
	}

	routeGroup.GET(RouteInfo, func(c echo.Context) error {
		resp, err := info()
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteBlock, func(c echo.Context) error {
		block, err := blockByID(c)
		if err != nil {
			return err
		}

		return responseByHeader(c, block.Block())
	})

	routeGroup.GET(RouteBlockMetadata, func(c echo.Context) error {
		// TODO: fill in blockReason, TxState, TxReason.
		resp, err := blockMetadataResponseByID(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.POST(RouteBlocks, func(c echo.Context) error {
		resp, err := sendBlock(c)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderLocation, resp.BlockID)

		return httpserver.JSONResponse(c, http.StatusCreated, resp)
	}, checkNodeSynced(), checkUpcomingUnsupportedProtocolVersion())

	routeGroup.GET(RouteBlockIssuance, func(c echo.Context) error {
		resp, err := blockIssuance(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteCommitmentByID, func(c echo.Context) error {
		index, err := indexByCommitmentID(c)
		if err != nil {
			return err
		}

		resp, err := getCommitment(index)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.GET(RouteCommitmentByIDUTXOChanges, func(c echo.Context) error {
		index, err := indexByCommitmentID(c)
		if err != nil {
			return err
		}

		resp, err := getSlotUTXOChanges(index)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.GET(RouteCommitmentByIndex, func(c echo.Context) error {
		indexUint64, err := httpserver.ParseUint64Param(c, restapipkg.ParameterSlotIndex)
		if err != nil {
			return err
		}

		resp, err := getCommitment(iotago.SlotIndex(indexUint64))
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.GET(RouteCommitmentByIndexUTXOChanges, func(c echo.Context) error {
		index, err := httpserver.ParseUint64Param(c, restapipkg.ParameterSlotIndex)
		if err != nil {
			return err
		}

		resp, err := getSlotUTXOChanges(iotago.SlotIndex(index))
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.GET(RouteOutput, func(c echo.Context) error {
		output, err := getOutput(c)
		if err != nil {
			return err
		}

		return responseByHeader(c, output.Output())
	}, checkNodeSynced())

	routeGroup.GET(RouteOutputMetadata, func(c echo.Context) error {
		// TODO: add CommitmentIDSpent, TransactionIDSpent
		resp, err := getOutputMetadata(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	routeGroup.GET(RouteTransactionsIncludedBlock, func(c echo.Context) error {
		block, err := blockByTransactionID(c)
		if err != nil {
			return err
		}

		return responseByHeader(c, block.Block())
	}, checkNodeSynced())

	routeGroup.GET(RouteTransactionsIncludedBlockMetadata, func(c echo.Context) error {
		resp, err := blockMetadataFromTransactionID(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}, checkNodeSynced())

	return nil
}

// AddFeature adds a feature to the RouteInfo endpoint.
func AddFeature(feature string) {
	features = append(features, strings.ToLower(feature))
}

func checkNodeSynced() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !deps.Protocol.SyncManager.IsNodeSynced() {
				return errors.WithMessage(echo.ErrServiceUnavailable, "node is not synced")
			}

			return next(c)
		}
	}
}

func checkUpcomingUnsupportedProtocolVersion() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// todo update with protocol upgrades support
			// if !deps.ProtocolManager.NextPendingSupported() {
			//	return errors.WithMessage(echo.ErrServiceUnavailable, "node does not support the upcoming protocol upgrade")
			// }

			return next(c)
		}
	}
}

func responseByHeader(c echo.Context, obj any) error {
	mimeType, err := httpserver.GetAcceptHeaderContentType(c, httpserver.MIMEApplicationVendorIOTASerializerV1, echo.MIMEApplicationJSON)
	if err != nil && err != httpserver.ErrNotAcceptable {
		return err
	}

	// default to echo.MIMEApplicationJSON
	switch mimeType {
	case httpserver.MIMEApplicationVendorIOTASerializerV1:
		b, err := deps.Protocol.API().Encode(obj)
		if err != nil {
			return err
		}

		return c.Blob(http.StatusOK, httpserver.MIMEApplicationVendorIOTASerializerV1, b)

	default:
		j, err := deps.Protocol.API().JSONEncode(obj)
		if err != nil {
			return err
		}

		return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, j)
	}
}
