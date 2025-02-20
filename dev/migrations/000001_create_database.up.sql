-- DROP SCHEMA public;

CREATE SCHEMA IF NOT EXISTS public AUTHORIZATION pg_database_owner;

-- DROP SEQUENCE public.synchronizer_fetch_id_seq;

CREATE SEQUENCE public.synchronizer_fetch_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	START 1
	CACHE 1
	NO CYCLE;-- public.convenience_application definition

-- Drop table

-- DROP TABLE public.convenience_application;

CREATE TABLE public.convenience_application (
	id int4 NOT NULL,
	"name" text NOT NULL,
	app_contract text NOT NULL
);
CREATE INDEX convenience_application_app_contract ON public.convenience_application USING btree (app_contract);
CREATE INDEX convenience_application_id ON public.convenience_application USING btree (id);
CREATE INDEX convenience_application_name ON public.convenience_application USING btree (name);


-- public.convenience_input_raw_references definition

-- Drop table

-- DROP TABLE public.convenience_input_raw_references;

CREATE TABLE public.convenience_input_raw_references (
	id text NOT NULL,
	app_id int4 NOT NULL,
	input_index int4 NOT NULL,
	app_contract text NOT NULL,
	status text NULL,
	chain_id text NULL,
	created_at timestamp NOT NULL
);
CREATE INDEX idx_app_id_input_index ON public.convenience_input_raw_references USING btree (app_id, input_index);
CREATE INDEX idx_convenience_input_raw_references_status_raw_id ON public.convenience_input_raw_references USING btree (status, app_id);


-- public.convenience_inputs definition

-- Drop table

-- DROP TABLE public.convenience_inputs;

CREATE TABLE public.convenience_inputs (
	id text NOT NULL,
	input_index int4 NULL,
	app_contract text NULL,
	status text NULL,
	msg_sender text NULL,
	payload text NULL,
	block_number int4 NULL,
	block_timestamp numeric NULL,
	prev_randao text NULL,
	"exception" text NULL,
	espresso_block_number int4 NULL,
	espresso_block_timestamp numeric NULL,
	input_box_index int4 NULL,
	avail_block_number int4 NULL,
	avail_block_timestamp numeric NULL,
	"type" text NULL,
	cartesi_transaction_id text NULL,
	chain_id text NULL
);
CREATE INDEX idx_input_id ON public.convenience_inputs USING btree (app_contract, id);
CREATE INDEX idx_input_index ON public.convenience_inputs USING btree (input_index);
CREATE INDEX idx_input_index_app_contract ON public.convenience_inputs USING btree (input_index, app_contract);
CREATE INDEX idx_status ON public.convenience_inputs USING btree (status);
CREATE INDEX idx_status_app_contract ON public.convenience_inputs USING btree (status, app_contract);


-- public.convenience_output_raw_references definition

-- Drop table

-- DROP TABLE public.convenience_output_raw_references;

CREATE TABLE public.convenience_output_raw_references (
	app_id int4 NOT NULL,
	input_index int4 NOT NULL,
	app_contract text NOT NULL,
	output_index int4 NOT NULL,
	has_proof bool NULL,
	"type" text NOT NULL,
	executed bool NULL,
	updated_at timestamp NOT NULL,
	created_at timestamp NOT NULL,
	sync_priority int4 NOT NULL,
	CONSTRAINT convenience_output_raw_references_pkey PRIMARY KEY (input_index, output_index, app_contract),
	CONSTRAINT convenience_output_raw_references_type_check CHECK ((type = ANY (ARRAY['voucher'::text, 'notice'::text])))
);
CREATE INDEX idx_convenience_output_raw_references_app_id ON public.convenience_output_raw_references USING btree (app_id);
CREATE INDEX idx_convenience_output_raw_references_has_proof_app_id ON public.convenience_output_raw_references USING btree (has_proof, app_id);


-- public.convenience_reports definition

-- Drop table

-- DROP TABLE public.convenience_reports;

CREATE TABLE public.convenience_reports (
	output_index int4 NOT NULL,
	payload text NULL,
	input_index int4 NOT NULL,
	app_contract text NOT NULL,
	app_id int4 NULL,
	CONSTRAINT convenience_reports_pkey PRIMARY KEY (input_index, output_index, app_contract)
);
CREATE INDEX idx_output_index_app_contract ON public.convenience_reports USING btree (output_index, app_contract);


-- public.convenience_notices definition

-- Drop table

-- DROP TABLE public.convenience_notices;

CREATE TABLE public.convenience_notices (
	payload text NULL,
	input_index int4 NOT NULL,
	output_index int4 NOT NULL,
	app_contract text NOT NULL,
	output_hashes_siblings text NULL,
	proof_output_index int4 NULL DEFAULT 0,
	CONSTRAINT notices_pkey PRIMARY KEY (input_index, output_index, app_contract)
);


-- public.synchronizer_fetch definition

-- Drop table

-- DROP TABLE public.synchronizer_fetch;

CREATE TABLE public.synchronizer_fetch (
	id serial4 NOT NULL,
	timestamp_after int8 NULL,
	ini_cursor_after text NULL,
	log_vouchers_ids text NULL,
	end_cursor_after text NULL,
	ini_input_cursor_after text NULL,
	end_input_cursor_after text NULL,
	ini_report_cursor_after text NULL,
	end_report_cursor_after text NULL,
	CONSTRAINT synchronizer_fetch_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_last_fetched_id ON public.synchronizer_fetch USING btree (id DESC);


-- public.convenience_vouchers definition

-- Drop table

-- DROP TABLE public.convenience_vouchers;

CREATE TABLE public.convenience_vouchers (
	destination text NULL,
	payload text NULL,
	executed bool NULL,
	input_index int4 NOT NULL,
	output_index int4 NOT NULL,
	value text NULL,
	output_hashes_siblings text NULL,
	app_contract text NOT NULL,
	transaction_hash text NOT NULL DEFAULT ''::text,
	proof_output_index int4 NULL DEFAULT 0,
	is_delegated_call bool NULL,
	CONSTRAINT vouchers_pkey PRIMARY KEY (input_index, output_index, app_contract)
);
CREATE INDEX idx_app_contract_input_index ON public.convenience_vouchers USING btree (app_contract, input_index);
CREATE INDEX idx_app_contract_output_index ON public.convenience_vouchers USING btree (app_contract, output_index);
CREATE INDEX idx_input_index_output_index ON public.convenience_vouchers USING btree (input_index, output_index);
