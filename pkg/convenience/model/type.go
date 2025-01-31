package model

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const STATUS_PROPERTY = "Status"
const EXECUTED = "Executed"
const FALSE = "false"
const DESTINATION = "Destination"
const VOUCHER_SELECTOR = "237a816f"
const DELEGATED_CALL_VOUCHER_SELECTOR = "10321e8b"
const NOTICE_SELECTOR = "c258d6e5"
const INPUT_INDEX = "InputIndex"
const APP_CONTRACT = "AppContract"
const DELEGATED_CALL_VOUCHER = "DelegatedCallVoucher"
const STATE = "State"

// Completion status for inputs.
type CompletionStatus int

const (
	CompletionStatusUnprocessed CompletionStatus = iota
	CompletionStatusAccepted
	CompletionStatusRejected
	CompletionStatusException
	CompletionStatusMachineHalted
	CompletionStatusCycleLimitExceeded
	CompletionStatusTimeLimitExceeded
	CompletionStatusPayloadLengthLimitExceeded
)

type ApplicationState uint8

const (
	ApplicationStateEnabled ApplicationState = iota
	ApplicationStateDisabled
	ApplicationStateInoperable
)

func (s ApplicationState) String() *string {
	var output string

	switch s {
	case ApplicationStateEnabled:
		output = "ENABLED"
	case ApplicationStateDisabled:
		output = "DISABLED"
	case ApplicationStateInoperable:
		output = "INOPERABLE"
	}

	return &output
}

type ConvenienceApplication struct {
	ID                   uint64         `db:"id"`
	Name                 string         `db:"name"`
	ApplicationAddress   common.Address `db:"application_address"`
	ConsensusAddress     common.Address `db:"consensus_address"`
	TemplateHash         []byte         `db:"template_hash"`
	TemplateURI          string         `db:"template_uri"`
	EpochLength          uint64         `db:"epoch_length"`
	State                string         `db:"state"`
	Reason               sql.NullString `db:"reason,omitempty"`
	LastProcessedBlock   uint64         `db:"last_processed_block"`
	LastClaimCheckBlock  uint64         `db:"last_claim_check_block"`
	LastOutputCheckBlock uint64         `db:"last_output_check_block"`
	ProcessedInputs      uint64         `db:"processed_inputs"`
	CreatedAt            time.Time      `db:"created_at"`
	UpdatedAt            time.Time      `db:"updated_at"`
}

type ConvenienceNotice struct {
	AppContract          string `db:"app_contract"`
	Payload              string `db:"payload"`
	InputIndex           uint64 `db:"input_index"`
	OutputIndex          uint64 `db:"output_index"`
	OutputHashesSiblings string `db:"output_hashes_siblings"`
	ProofOutputIndex     uint64 `db:"proof_output_index"`
}

// Voucher metadata type
type ConvenienceVoucher struct {
	Destination          common.Address `db:"destination"`
	Payload              string         `db:"payload"`
	InputIndex           uint64         `db:"input_index"`
	OutputIndex          uint64         `db:"output_index"`
	Executed             bool           `db:"executed"`
	Value                string         `db:"value"`
	AppContract          common.Address `db:"app_contract"`
	OutputHashesSiblings string         `db:"output_hashes_siblings"`
	TransactionHash      string         `db:"transaction_hash"`
	ProofOutputIndex     uint64         `db:"proof_output_index"`
	IsDelegatedCall      bool           `db:"is_delegated_call"`
	// future improvements
	// Contract        common.Address
	// Beneficiary     common.Address
	// Label           string
	// Amount          uint64
	// ExecutedAt      uint64
	// ExecutedBlock   uint64
	// InputIndex      int
	// OutputIndex     int
	// MethodSignature string
	// ERCX            string
}

type ConvenienceFilter struct {
	Field *string              `json:"field,omitempty"`
	Eq    *string              `json:"eq,omitempty"`
	Ne    *string              `json:"ne,omitempty"`
	Gt    *string              `json:"gt,omitempty"`
	Gte   *string              `json:"gte,omitempty"`
	Lt    *string              `json:"lt,omitempty"`
	Lte   *string              `json:"lte,omitempty"`
	In    []*string            `json:"in,omitempty"`
	Nin   []*string            `json:"nin,omitempty"`
	And   []*ConvenienceFilter `json:"and,omitempty"`
	Or    []*ConvenienceFilter `json:"or,omitempty"`
}

type SynchronizerFetch struct {
	Id                   int64  `db:"id"`
	TimestampAfter       uint64 `db:"timestamp_after"`
	IniCursorAfter       string `db:"ini_cursor_after"`
	LogVouchersIds       string `db:"log_vouchers_ids"`
	EndCursorAfter       string `db:"end_cursor_after"`
	IniInputCursorAfter  string `db:"ini_input_cursor_after"`
	EndInputCursorAfter  string `db:"end_input_cursor_after"`
	IniReportCursorAfter string `db:"ini_report_cursor_after"`
	EndReportCursorAfter string `db:"end_report_cursor_after"`
	IniAppCursorAfter    string `db:"ini_app_cursor_after"`
	EndAppCursorAfter    string `db:"end_app_cursor_after"`
}

// Rollups input, which can be advance or inspect.
type Input interface{}

// Rollups report type.
type Report struct {
	Index       int `db:"output_index"`
	InputIndex  int `db:"input_index"`
	Payload     string
	AppContract common.Address `json:"app_contract" db:"app_contract"`
	AppID       uint64         `db:"app_id"`
}

type FastReport struct {
	Index       int    `db:"output_index"`
	InputIndex  int    `db:"input_index"`
	Payload     string `db:"payload"`
	AppContract []byte `json:"app_contract" db:"app_contract"`
	AppID       uint64 `db:"app_id"`
}

// Rollups advance input type.
type AdvanceInput struct {
	ID                     string           `db:"id"`
	Index                  int              `db:"input_index"`
	Status                 CompletionStatus `db:"status"`
	MsgSender              common.Address   `db:"msg_sender"`
	Payload                string           `db:"payload"`
	BlockNumber            uint64           `db:"block_number"`
	BlockTimestamp         time.Time        `db:"block_timestamp"`
	PrevRandao             string           `db:"prev_randao"`
	ChainId                string           `db:"chain_id"`
	AppContract            common.Address   `db:"app_contract"`
	Vouchers               []ConvenienceVoucher
	Notices                []ConvenienceNotice
	Reports                []Report
	Exception              []byte
	EspressoBlockNumber    int       `db:"espresso_block_number"`
	EspressoBlockTimestamp time.Time `db:"espresso_block_timestamp"`
	InputBoxIndex          int       `db:"input_box_index"`
	AvailBlockNumber       int       `db:"avail_block_number"`
	AvailBlockTimestamp    time.Time `db:"avail_block_timestamp"`
	Type                   string    `db:"type"`
	CartesiTransactionId   string    `db:"cartesi_transaction_id"`
}

type ConvertedInput struct {
	ChainId        *big.Int       `json:"chainId"`
	MsgSender      common.Address `json:"msgSender"`
	AppContract    common.Address `json:"app_contract"`
	BlockNumber    *big.Int       `json:"blockNumber"`
	BlockTimestamp int64          `json:"blockTimestamp"`
	PrevRandao     string         `json:"prevRandao"`
	Payload        string         `json:"payload"`
	InputBoxIndex  int64          `json:"input_box_index"`
}

type InputEdge struct {
	Cursor string `json:"cursor"`
	Node   struct {
		Index int    `json:"index"`
		Blob  string `json:"blob"`
	} `json:"node"`
}

type OutputEdge struct {
	Cursor string `json:"cursor"`
	Node   struct {
		Index      int    `json:"index"`
		Blob       string `json:"blob"`
		InputIndex int    `json:"inputIndex"`
	} `json:"node"`
}

type DecoderInterface interface {
	HandleOutputV2(
		ctx context.Context,
		processOutputData ProcessOutputData,
	) error

	HandleInput(
		ctx context.Context,
		input InputEdge,
		status CompletionStatus,
	) error

	HandleReport(
		ctx context.Context,
		index int,
		outputIndex int,
		payload string,
	) error

	GetConvertedInput(output InputEdge) (ConvertedInput, error)

	RetrieveDestination(payload string) (common.Address, error)
}

type ProcessOutputData struct {
	OutputIndex uint64 `json:"outputIndex"`
	InputIndex  uint64 `json:"inputIndex"`
	Payload     string `json:"payload"`
	Destination string `json:"destination"`
}

type RepoSynchronizer interface {
	GetDB() *sqlx.DB
	CreateTables() error
	Create(ctx context.Context, data *SynchronizerFetch) (*SynchronizerFetch, error)
	Count(ctx context.Context) (uint64, error)
	GetLastFetched(ctx context.Context) (*SynchronizerFetch, error)
}

type contextKey string

const AppContractKey contextKey = "appContract"
