package synchronizernode

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RawRepository struct {
	connectionURL string
	Db            *sqlx.DB
}

type RawInputAppAddress struct {
	Address []byte `db:"iapplication_address"`
}

type RawInput struct {
	Index              uint64    `db:"index"` // numeric(20,0)
	RawData            []byte    `db:"raw_data"`
	BlockNumber        uint64    `db:"block_number"` // numeric(20,0)
	Status             string    `db:"status"`
	MachineHash        []byte    `db:"machine_hash,omitempty"`
	OutputsHash        []byte    `db:"outputs_hash,omitempty"`
	EpochIndex         uint64    `db:"epoch_index"`
	ApplicationId      int       `db:"epoch_application_id"`
	TransactionRef     []byte    `db:"transaction_reference,omitempty"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
	SnapshotURI        []byte    `db:"snapshot_uri,omitempty"`
	ApplicationAddress []byte    `db:"application_address"`
}

type Report struct {
	Index         uint64 `db:"index"`
	InputIndex    uint64 `db:"input_index"`
	ApplicationId int    `db:"input_epoch_application_id"`
	RawData       []byte `db:"raw_data"`
	AppContract   []byte `db:"app_contract"`
}

type Output struct {
	Index                uint64    `db:"index"`
	InputIndex           uint64    `db:"input_index"`
	RawData              []byte    `db:"raw_data"`
	Hash                 []byte    `db:"hash,omitempty"`
	OutputHashesSiblings []byte    `db:"output_hashes_siblings,omitempty"`
	TransactionHash      []byte    `db:"execution_transaction_hash,omitempty"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
	ApplicationId        uint64    `db:"input_epoch_application_id"`
	AppContract          []byte    `db:"app_contract"`
}

type FilterOutput struct {
	IDgt                uint64
	HaveTransactionHash bool
}

type Pagination struct {
	Limit uint64
	// Offset uint64
}

type FilterInput struct {
	AppID         uint64
	InputIndexGte uint64
	IDgt          uint64
	IsStatusNone  bool
	Status        string
}

const LIMIT = uint64(50)

const OUTPUT_ORDER_BY = `
	ORDER BY o.created_at ASC, o.index ASC, o.input_epoch_application_id ASC
`

type FilterID struct {
	IDgt uint64
}

func NewRawRepository(connectionURL string, db *sqlx.DB) *RawRepository {
	return &RawRepository{connectionURL, db}
}

func (s *RawRepository) First50RawInputsGteRefWithStatus(ctx context.Context, inputRef repository.RawInputRef, status string) ([]RawInput, error) {
	query := `
		SELECT
			i.index,
			i.raw_data,
			i.block_number,
			i.status,
			i.machine_hash,
			i.outputs_hash,
			i.epoch_index,
			i.epoch_application_id,
			i.transaction_reference,
			i.created_at,
			i.updated_at,
			i.snapshot_uri,
			a.iapplication_address as application_address
		FROM
			input i
		INNER JOIN
			application a
		ON
			a.id = i.epoch_application_id
		WHERE
			i.created_at >= $1 and i.epoch_application_id >= $2 and i.index >= $3 and status = $4
		ORDER BY
		    i.created_at ASC, i.epoch_application_id ASC, i.index ASC
		LIMIT $5`
	result, err := s.Db.QueryxContext(ctx, query, inputRef.CreatedAt, inputRef.AppID, inputRef.InputIndex, status, LIMIT)
	if err != nil {
		slog.Error("RollupsGraphql: Failed to execute query in First50RawInputsGteRefWithStatus",
			"query", query, "error", err)
		return nil, err
	}
	defer result.Close()
	inputs := []RawInput{}
	for result.Next() {
		var input RawInput
		err := result.StructScan(&input)
		if err != nil {
			slog.Error("Failed to scan row into RawInput struct", "error", err)
			return nil, err
		}
		input.ApplicationAddress = common.Hex2Bytes(string(input.ApplicationAddress[2:]))
		inputs = append(inputs, input)
	}
	slog.Debug("First50RawInputsGteRefWithStatus", "status", status, "results", len(inputs))
	if len(inputs) > 0 {
		slog.Debug("First50RawInputsGteRefWithStatus first result", "appID", inputs[0].ApplicationId,
			"InputIndex", inputs[0].Index,
		)
	}
	return inputs, nil
}

func (s *RawRepository) FindAllRawInputs(ctx context.Context) ([]RawInput, error) {
	inputs := []RawInput{}
	query := `
	SELECT
		i.index,
		i.raw_data,
		i.block_number,
		i.status,
		i.machine_hash,
		i.outputs_hash,
		i.epoch_index,
		i.epoch_application_id,
		i.transaction_reference,
		i.created_at,
		i.updated_at,
		i.snapshot_uri,
		a.iapplication_address as application_address
	FROM
		input i
	INNER JOIN
		application a
	ON
		a.id = i.epoch_application_id
	ORDER BY
		i.created_at ASC, i.index ASC, i.epoch_application_id ASC
	LIMIT $1
	`
	result, err := s.Db.QueryxContext(ctx, query, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllInputs",
			"query", query, "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var input RawInput
		err := result.StructScan(&input)
		if err != nil {
			slog.Error("Failed to scan row into RawInput struct", "error", err)
			return nil, err
		}
		input.ApplicationAddress = common.Hex2Bytes(string(input.ApplicationAddress[2:]))
		inputs = append(inputs, input)
	}
	slog.Debug("FindAllRawInputs", "results", len(inputs))
	return inputs, nil
}

func (s *RawRepository) FindAllInputsGtRef(ctx context.Context, inputRef *repository.RawInputRef) ([]RawInput, error) {
	if inputRef == nil {
		return s.FindAllRawInputs(ctx)
	}
	inputs := []RawInput{}
	result, err := s.Db.QueryxContext(ctx, `
	SELECT
		i.index,
		i.raw_data,
		i.block_number,
		i.status,
		i.machine_hash,
		i.outputs_hash,
		i.epoch_index,
		i.epoch_application_id,
		i.transaction_reference,
		i.created_at,
		i.updated_at,
		i.snapshot_uri,
		a.iapplication_address as application_address
	FROM
		input i
	INNER JOIN
		application a
	ON
		a.id = i.epoch_application_id
	WHERE 
		(i.epoch_application_id = $1 AND i.index > $2)
		OR
		(i.epoch_application_id <> $1 AND i.created_at >= $3)
	ORDER BY
		i.created_at ASC, i.index ASC, i.epoch_application_id ASC
	LIMIT $4
	`, inputRef.AppID, inputRef.InputIndex, inputRef.CreatedAt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllInputsGtRef", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var input RawInput
		err := result.StructScan(&input)
		if err != nil {
			slog.Error("Failed to scan row into RawInput struct", "error", err)
			return nil, err
		}
		input.ApplicationAddress = common.Hex2Bytes(string(input.ApplicationAddress[2:]))
		inputs = append(inputs, input)
	}
	slog.Debug("FindAllInputsGtRef", "results", len(inputs))
	return inputs, nil
}

func (s *RawRepository) FindAllReportsByFilter(ctx context.Context, filter FilterID) ([]Report, error) {
	reports := []Report{}

	query := `
	SELECT
		r.index,
		i.index as input_index,
		r.input_epoch_application_id,
		r.raw_data,
		a.iapplication_address as app_contract
	FROM
		report r
	INNER JOIN
		input i
	ON
		i.index = r.input_index
	INNER JOIN
		application a
	ON
		a.id = i.epoch_application_id
	WHERE r.index >= $1
	ORDER BY r.index ASC
	LIMIT $2
	`

	result, err := s.Db.QueryxContext(ctx, query, filter.IDgt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllReportsByFilter", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Report
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Report struct", "error", err)
			return nil, err
		}
		report.AppContract = common.Hex2Bytes(string(report.AppContract[2:]))
		reports = append(reports, report)
	}

	return reports, nil
}

func (s *RawRepository) FindInputByOutput(ctx context.Context, filter FilterID) (*RawInput, error) {
	query := `
		SELECT
			i.index,
			i.raw_data,
			i.block_number,
			i.status,
			i.machine_hash,
			i.outputs_hash,
			i.epoch_index,
			i.epoch_application_id,
			i.transaction_reference,
			i.created_at,
			i.updated_at,
			i.snapshot_uri,
			a.iapplication_address as application_address
		FROM input i
		INNER JOIN application a
		ON a.id = i.epoch_application_id
		WHERE i.index = $1
		LIMIT 1`
	stmt, err := s.Db.Preparex(query)
	if err != nil {
		slog.Error("Failed to prepare statement in FindInputByOutput", "query", query, "error", err)
		return nil, err
	}
	defer stmt.Close()

	var input RawInput
	err = stmt.GetContext(ctx, &input, filter.IDgt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("No input found for given output", "input_id", filter.IDgt)
			return nil, nil
		}
		slog.Error("Failed to get context for input in FindInputByOutput", "error", err)
		return nil, err
	}
	input.ApplicationAddress = common.Hex2Bytes(string(input.ApplicationAddress))

	return &input, nil
}

func (s *RawRepository) findAllOutputsLimited(ctx context.Context) ([]Output, error) {
	outputs := []Output{}
	query := `
		SELECT
			o.index,
			i.index as input_index,
			o.raw_data,
			o.hash,
			o.output_hashes_siblings,
			o.execution_transaction_hash,
			o.created_at,
			o.updated_at,
			o.input_epoch_application_id,
			a.iapplication_address as app_contract
		FROM
			output o
		INNER JOIN input i
			ON i.index = o.input_index
		INNER JOIN application a
			ON a.id = o.input_epoch_application_id
		ORDER BY 
			o.created_at ASC, o.index ASC, o.input_epoch_application_id ASC
		LIMIT $1`
	result, err := s.Db.QueryxContext(ctx, query, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in findAllOutputsLimited", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var output Output
		err := result.StructScan(&output)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		output.AppContract = common.Hex2Bytes(string(output.AppContract[2:]))
		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsGtRefLimited(ctx context.Context, outputRef *repository.RawOutputRef) ([]Output, error) {
	outputs := []Output{}

	if outputRef == nil {
		return s.findAllOutputsLimited(ctx)
	}
	result, err := s.Db.QueryxContext(ctx, `
        SELECT
			o.index,
			i.index as input_index,
			o.raw_data,
			o.hash,
			o.output_hashes_siblings,
			o.execution_transaction_hash,
			o.created_at,
			o.updated_at,
			o.input_epoch_application_id,
			a.iapplication_address as app_contract
		FROM
			output o
		INNER JOIN input i
			ON i.index = o.input_index
		INNER JOIN application a
			ON a.id = o.input_epoch_application_id
		WHERE
			(o.input_epoch_application_id = $3 and o.index > $1 and o.created_at >= $2)
			OR
			(o.input_epoch_application_id > $3 and o.created_at >= $2)
		ORDER BY
			o.created_at ASC, o.index ASC, o.input_epoch_application_id ASC
		LIMIT $4`, outputRef.OutputIndex, outputRef.CreatedAt, outputRef.AppID, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsGtRefLimited", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var output Output
		err := result.StructScan(&output)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		output.AppContract = common.Hex2Bytes(string(output.AppContract[2:]))
		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsWithProofGte(ctx context.Context, filter *repository.RawOutputRef) ([]Output, error) {
	outputs := []Output{}
	result, err := s.Db.QueryxContext(ctx, `
		SELECT
			o.index,
			o.input_index,
			o.raw_data,
			o.hash,
			o.output_hashes_siblings,
			o.execution_transaction_hash,
			o.created_at,
			o.updated_at,
			o.input_epoch_application_id,
			a.iapplication_address as app_contract
		FROM
			output o
		INNER JOIN application a
		ON
			a.id = o.input_epoch_application_id
		WHERE
			output_hashes_siblings IS NOT NULL
			AND
			(
				(o.index >= $2 AND o.input_epoch_application_id >= $1 )
			)
		ORDER BY
			o.index ASC, 
			o.input_epoch_application_id ASC
		LIMIT $3
	`, filter.AppID, filter.OutputIndex, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsWithProof", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var output Output
		err := result.StructScan(&output)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsWithProof(ctx context.Context, filter FilterID) ([]Output, error) {
	outputs := []Output{}
	result, err := s.Db.QueryxContext(ctx, `
        SELECT
			o.index,
			o.input_index,
			o.raw_data,
			o.hash,
			o.output_hashes_siblings,
			o.execution_transaction_hash,
			o.created_at,
			o.updated_at,
			o.input_epoch_application_id,
			a.iapplication_address as app_contract
		FROM
			output o
		INNER JOIN application a
		ON
			a.id = o.input_epoch_application_id
		WHERE
			o.index >= $1
			AND output_hashes_siblings IS NOT NULL
		ORDER BY
			o.index
		LIMIT $2
    `, filter.IDgt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsWithProof", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Output
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		outputs = append(outputs, report)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsExecutedAfter(ctx context.Context, outputRef *repository.RawOutputRef) ([]Output, error) {
	outputs := []Output{}
	result, err := s.Db.QueryxContext(ctx, `
        SELECT
			o.index,
			o.input_index,
			o.raw_data,
			o.hash,
			o.output_hashes_siblings,
			o.execution_transaction_hash,
			o.created_at,
			o.updated_at,
			o.input_epoch_application_id,
			a.iapplication_address as app_contract
		FROM
			output o
		INNER JOIN application a
		ON
			a.id = o.input_epoch_application_id
		WHERE
			(
				o.execution_transaction_hash IS NOT NULL
			)
				AND 
			(
				(o.updated_at > $1)
					OR
				(o.updated_at = $1 AND o.input_epoch_application_id = $2 AND o.index > $3)
			)
		ORDER BY
			o.updated_at ASC,
			o.index ASC,
			o.input_epoch_application_id ASC
		LIMIT $4
    `, outputRef.UpdatedAt, outputRef.AppID, outputRef.OutputIndex, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsExecuted", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var output Output
		err := result.StructScan(&output)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		output.AppContract = common.Hex2Bytes(string(output.AppContract[2:]))
		outputs = append(outputs, output)
	}

	return outputs, nil
}
