--
-- PostgreSQL database dump
--

-- Dumped from database version 16.6
-- Dumped by pg_dump version 16.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: ApplicationState; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."ApplicationState" AS ENUM (
    'ENABLED',
    'DISABLED',
    'INOPERABLE'
);


ALTER TYPE public."ApplicationState" OWNER TO postgres;

--
-- Name: DefaultBlock; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."DefaultBlock" AS ENUM (
    'FINALIZED',
    'LATEST',
    'PENDING',
    'SAFE'
);


ALTER TYPE public."DefaultBlock" OWNER TO postgres;

--
-- Name: EpochStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."EpochStatus" AS ENUM (
    'OPEN',
    'CLOSED',
    'INPUTS_PROCESSED',
    'CLAIM_COMPUTED',
    'CLAIM_SUBMITTED',
    'CLAIM_ACCEPTED',
    'CLAIM_REJECTED'
);


ALTER TYPE public."EpochStatus" OWNER TO postgres;

--
-- Name: InputCompletionStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."InputCompletionStatus" AS ENUM (
    'NONE',
    'ACCEPTED',
    'REJECTED',
    'EXCEPTION',
    'MACHINE_HALTED',
    'OUTPUTS_LIMIT_EXCEEDED',
    'CYCLE_LIMIT_EXCEEDED',
    'TIME_LIMIT_EXCEEDED',
    'PAYLOAD_LENGTH_LIMIT_EXCEEDED'
);


ALTER TYPE public."InputCompletionStatus" OWNER TO postgres;

--
-- Name: SnapshotPolicy; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."SnapshotPolicy" AS ENUM (
    'NONE',
    'EACH_INPUT',
    'EACH_EPOCH'
);


ALTER TYPE public."SnapshotPolicy" OWNER TO postgres;

--
-- Name: ethereum_address; Type: DOMAIN; Schema: public; Owner: postgres
--

CREATE DOMAIN public.ethereum_address AS character varying(42)
	CONSTRAINT ethereum_address_check CHECK (((VALUE)::text ~ '^0x[a-f0-9]{40}$'::text));


ALTER DOMAIN public.ethereum_address OWNER TO postgres;

--
-- Name: hash; Type: DOMAIN; Schema: public; Owner: postgres
--

CREATE DOMAIN public.hash AS bytea
	CONSTRAINT hash_check CHECK ((octet_length(VALUE) = 32));


ALTER DOMAIN public.hash OWNER TO postgres;

--
-- Name: uint64; Type: DOMAIN; Schema: public; Owner: postgres
--

CREATE DOMAIN public.uint64 AS numeric(20,0)
	CONSTRAINT uint64_check CHECK (((VALUE >= (0)::numeric) AND (VALUE <= '18446744073709551615'::numeric)));


ALTER DOMAIN public.uint64 OWNER TO postgres;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: application; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.application (
    id integer NOT NULL,
    name character varying(4096) NOT NULL,
    iapplication_address public.ethereum_address NOT NULL,
    iconsensus_address public.ethereum_address NOT NULL,
    template_hash public.hash NOT NULL,
    template_uri character varying(4096) NOT NULL,
    state public."ApplicationState" NOT NULL,
    reason character varying(4096),
    last_processed_block public.uint64 NOT NULL,
    last_claim_check_block public.uint64 NOT NULL,
    last_output_check_block public.uint64 NOT NULL,
    processed_inputs public.uint64 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT application_name_check CHECK (((name)::text ~ '^[a-z0-9_-]+$'::text)),
    CONSTRAINT reason_required_for_inoperable CHECK ((NOT ((state = 'INOPERABLE'::public."ApplicationState") AND ((reason IS NULL) OR (length((reason)::text) = 0)))))
);


ALTER TABLE public.application OWNER TO postgres;

--
-- Name: application_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.application_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.application_id_seq OWNER TO postgres;

--
-- Name: application_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.application_id_seq OWNED BY public.application.id;


--
-- Name: epoch; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.epoch (
    application_id integer NOT NULL,
    index public.uint64 NOT NULL,
    first_block public.uint64 NOT NULL,
    last_block public.uint64 NOT NULL,
    claim_hash public.hash,
    claim_transaction_hash public.hash,
    status public."EpochStatus" NOT NULL,
    virtual_index public.uint64 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.epoch OWNER TO postgres;

--
-- Name: execution_parameters; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.execution_parameters (
    application_id integer NOT NULL,
    snapshot_policy public."SnapshotPolicy" DEFAULT 'NONE'::public."SnapshotPolicy" NOT NULL,
    snapshot_retention bigint DEFAULT 0 NOT NULL,
    advance_inc_cycles bigint DEFAULT 4194304 NOT NULL,
    advance_max_cycles bigint DEFAULT '4611686018427387903'::bigint NOT NULL,
    inspect_inc_cycles bigint DEFAULT 4194304 NOT NULL,
    inspect_max_cycles bigint DEFAULT '4611686018427387903'::bigint NOT NULL,
    advance_inc_deadline bigint DEFAULT '10000000000'::bigint NOT NULL,
    advance_max_deadline bigint DEFAULT '180000000000'::bigint NOT NULL,
    inspect_inc_deadline bigint DEFAULT '10000000000'::bigint NOT NULL,
    inspect_max_deadline bigint DEFAULT '180000000000'::bigint NOT NULL,
    load_deadline bigint DEFAULT '300000000000'::bigint NOT NULL,
    store_deadline bigint DEFAULT '180000000000'::bigint NOT NULL,
    fast_deadline bigint DEFAULT '5000000000'::bigint NOT NULL,
    max_concurrent_inspects integer DEFAULT 10 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT execution_parameters_advance_inc_cycles_check CHECK ((advance_inc_cycles > 0)),
    CONSTRAINT execution_parameters_advance_inc_deadline_check CHECK ((advance_inc_deadline > 0)),
    CONSTRAINT execution_parameters_advance_max_cycles_check CHECK ((advance_max_cycles > 0)),
    CONSTRAINT execution_parameters_advance_max_deadline_check CHECK ((advance_max_deadline > 0)),
    CONSTRAINT execution_parameters_fast_deadline_check CHECK ((fast_deadline > 0)),
    CONSTRAINT execution_parameters_inspect_inc_cycles_check CHECK ((inspect_inc_cycles > 0)),
    CONSTRAINT execution_parameters_inspect_inc_deadline_check CHECK ((inspect_inc_deadline > 0)),
    CONSTRAINT execution_parameters_inspect_max_cycles_check CHECK ((inspect_max_cycles > 0)),
    CONSTRAINT execution_parameters_inspect_max_deadline_check CHECK ((inspect_max_deadline > 0)),
    CONSTRAINT execution_parameters_load_deadline_check CHECK ((load_deadline > 0)),
    CONSTRAINT execution_parameters_max_concurrent_inspects_check CHECK ((max_concurrent_inspects > 0)),
    CONSTRAINT execution_parameters_snapshot_retention_check CHECK ((snapshot_retention >= 0)),
    CONSTRAINT execution_parameters_store_deadline_check CHECK ((store_deadline > 0))
);


ALTER TABLE public.execution_parameters OWNER TO postgres;

--
-- Name: input; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.input (
    epoch_application_id integer NOT NULL,
    epoch_index public.uint64 NOT NULL,
    index public.uint64 NOT NULL,
    block_number public.uint64 NOT NULL,
    raw_data bytea NOT NULL,
    status public."InputCompletionStatus" NOT NULL,
    machine_hash public.hash,
    outputs_hash public.hash,
    transaction_reference public.hash,
    snapshot_uri character varying(4096),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.input OWNER TO postgres;

--
-- Name: node_config; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.node_config (
    default_block public."DefaultBlock" NOT NULL,
    input_box_deployment_block integer NOT NULL,
    input_box_address public.ethereum_address NOT NULL,
    chain_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.node_config OWNER TO postgres;

--
-- Name: output; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.output (
    input_epoch_application_id integer NOT NULL,
    input_index public.uint64 NOT NULL,
    index public.uint64 NOT NULL,
    raw_data bytea NOT NULL,
    hash public.hash,
    output_hashes_siblings bytea[],
    execution_transaction_hash public.hash,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.output OWNER TO postgres;

--
-- Name: report; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.report (
    input_epoch_application_id integer NOT NULL,
    input_index public.uint64 NOT NULL,
    index public.uint64 NOT NULL,
    raw_data bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.report OWNER TO postgres;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: application id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application ALTER COLUMN id SET DEFAULT nextval('public.application_id_seq'::regclass);


--
-- Data for Name: application; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.application (id, name, iapplication_address, iconsensus_address, template_hash, template_uri, state, reason, last_processed_block, last_claim_check_block, last_output_check_block, processed_inputs, created_at, updated_at) FROM stdin;
1	echo-dapp	0x36b9e60acb181da458aa8870646395cd27cd0e6e	0xd121f8ae5ab0d5f472687af19e393d18fd3e140c	\\x84c8181abd120e0281f5032d22422b890f79880ae90d9a1416be1afccb8182a0	applications/echo-dapp/	ENABLED	\N	1018	656	1018	1	2025-01-23 12:52:06.974531+00	2025-01-23 13:06:38.36767+00
\.


--
-- Data for Name: epoch; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.epoch (application_id, index, first_block, last_block, claim_hash, claim_transaction_hash, status, virtual_index, created_at, updated_at) FROM stdin;
1	55	550	559	\\x742f1a860492737713d5cf8fe292ac01483a78f3ce50ac3d67dd5c5f5410f8b5	\\xeb411c916eecd8589b684db040d304a6d7f58d46c0af5452902c0c3f3cf53a85	CLAIM_ACCEPTED	0	2025-01-23 12:59:01.913049+00	2025-01-23 13:00:26.357831+00
1	56	560	569	\\x55376f845589f9a0a682d30a96dd8749477effd73994448a67537800c8bba33e	\\x4ef8636da3c6fe28e41a146c542bf3cbd460920576245e9d41366c1bf9f823b4	CLAIM_ACCEPTED	1	2025-01-23 12:59:13.566917+00	2025-01-23 13:00:37.917187+00
\.


--
-- Data for Name: execution_parameters; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.execution_parameters (application_id, snapshot_policy, snapshot_retention, advance_inc_cycles, advance_max_cycles, inspect_inc_cycles, inspect_max_cycles, advance_inc_deadline, advance_max_deadline, inspect_inc_deadline, inspect_max_deadline, load_deadline, store_deadline, fast_deadline, max_concurrent_inspects, created_at, updated_at) FROM stdin;
1	NONE	0	4194304	4611686018427387903	4194304	4611686018427387903	10000000000	180000000000	10000000000	180000000000	300000000000	180000000000	5000000000	10	2025-01-23 12:52:06.974531+00	2025-01-23 12:52:06.974531+00
\.


--
-- Data for Name: input; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.input (epoch_application_id, epoch_index, index, block_number, raw_data, status, machine_hash, outputs_hash, transaction_reference, snapshot_uri, created_at, updated_at) FROM stdin;
1	55	0	550	\\x415bf3630000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000036b9e60acb181da458aa8870646395cd27cd0e6e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000000002260000000000000000000000000000000000000000000000000000000067923cc94819f8d8df6763310ed667422a3bcdc7291b636dfa4598428e0fdc3b95fab429000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	ACCEPTED	\\x3eb550270b70e88610f5fb6a36949ec189ae8325328dedf8c4d9fc63f51e6bb5	\\x742f1a860492737713d5cf8fe292ac01483a78f3ce50ac3d67dd5c5f5410f8b5	\\x3030303030303030303030303030303030303030303030303030303030303030	\N	2025-01-23 12:59:01.913049+00	2025-01-23 12:59:03.176148+00
1	56	1	568	\\x415bf3630000000000000000000000000000000000000000000000000000000000007a6900000000000000000000000036b9e60acb181da458aa8870646395cd27cd0e6e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000000002380000000000000000000000000000000000000000000000000000000067923cdb32a6967a2138bfa8eeacfb4e2479de56464158b03483b502fb14b487ae7107d6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	ACCEPTED	\\x0925bcbb5ff3af6087ac01bbb7011173849a926e582a20c9ccbbe6dbb16f447e	\\x55376f845589f9a0a682d30a96dd8749477effd73994448a67537800c8bba33e	\\x3030303030303030303030303030303030303030303030303030303030303031	\N	2025-01-23 12:59:13.566917+00	2025-01-23 12:59:17.174646+00
\.


--
-- Data for Name: node_config; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.node_config (default_block, input_box_deployment_block, input_box_address, chain_id, created_at, updated_at) FROM stdin;
FINALIZED	10	0x593e5bcf894d6829dd26d0810da7f064406aebb6	31337	2025-01-23 12:49:13.457466+00	2025-01-23 12:49:13.457466+00
\.


--
-- Data for Name: output; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.output (input_epoch_application_id, input_index, index, raw_data, hash, output_hashes_siblings, execution_transaction_hash, created_at, updated_at) FROM stdin;
1	0	0	\\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	\\x5bcf2df28bd4535c167056d6dcd29391e79a11c85a5cb1f46a02eff40fd04741	{"\\\\x1c2326f86b0ebc1871dc8dcebd896e70f5a55422970d730909001a6b4ec8fad4","\\\\xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5","\\\\xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30","\\\\x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85","\\\\xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344","\\\\x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d","\\\\x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968","\\\\xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83","\\\\x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af","\\\\xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0","\\\\xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5","\\\\xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892","\\\\x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c","\\\\xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb","\\\\x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc","\\\\xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2","\\\\x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f","\\\\xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a","\\\\x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0","\\\\xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0","\\\\xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2","\\\\xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9","\\\\x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377","\\\\x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652","\\\\xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef","\\\\x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d","\\\\xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0","\\\\x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e","\\\\x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e","\\\\x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322","\\\\x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735","\\\\x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9","\\\\x27ae5ba08d7291c96c8cbddcc148bf48a6d68c7974b94356f53754ef6171d757","\\\\xbf558bebd2ceec7f3c5dce04a4782f88c2c6036ae78ee206d0bc5289d20461a2","\\\\xe21908c2968c0699040a6fd866a577a99a9d2ec88745c815fd4a472c789244da","\\\\xae824d72ddc272aab68a8c3022e36f10454437c1886f3ff9927b64f232df414f","\\\\x27e429a4bef3083bc31a671d046ea5c1f5b8c3094d72868d9dfdc12c7334ac5f","\\\\x743cc5c365a9a6a15c1f240ac25880c7a9d1de290696cb766074a1d83d927816","\\\\x4adcf616c3bfabf63999a01966c998b7bb572774035a63ead49da73b5987f347","\\\\x75786645d0c5dd7c04a2f8a75dcae085213652f5bce3ea8b9b9bedd1cab3c5e9","\\\\xb88b152c9b8a7b79637d35911848b0c41e7cc7cca2ab4fe9a15f9c38bb4bb939","\\\\x0c4e2d8ce834ffd7a6cd85d7113d4521abb857774845c4291e6f6d010d97e318","\\\\x5bc799d83e3bb31501b3da786680df30fbc18eb41cbce611e8c0e9c72f69571c","\\\\xa10d3ef857d04d9c03ead7c6317d797a090fa1271ad9c7addfbcb412e9643d4f","\\\\xb33b1809c42623f474055fa9400a2027a7a885c8dfa4efe20666b4ee27d7529c","\\\\x134d7f28d53f175f6bf4b62faa2110d5b76f0f770c15e628181c1fcc18f970a9","\\\\xc34d24b2fc8c50ca9c07a7156ef4e5ff4bdf002eda0b11c1d359d0b59a546807","\\\\x04dbb9db631457879b27e0dfdbe50158fd9cf9b4cf77605c4ac4c95bd65fc9f6","\\\\xf9295a686647cb999090819cda700820c282c613cedcd218540bbc6f37b01c65","\\\\x67c4a1ea624f092a3a5cca2d6f0f0db231972fce627f0ecca0dee60f17551c5f","\\\\x8fdaeb5ab560b2ceb781cdb339361a0fbee1b9dffad59115138c8d6a70dda9cc","\\\\xc1bf0bbdd7fee15764845db875f6432559ff8dbc9055324431bc34e5b93d15da","\\\\x307317849eccd90c0c7b98870b9317c15a5959dcfb84c76dcc908c4fe6ba9212","\\\\x6339bf06e458f6646df5e83ba7c3d35bc263b3222c8e9040068847749ca8e8f9","\\\\x5045e4342aeb521eb3a5587ec268ed3aa6faf32b62b0bc41a9d549521f406fc3","\\\\x08601d83cdd34b5f7b8df63e7b9a16519d35473d0b89c317beed3d3d9424b253","\\\\x84e35c5d92171376cae5c86300822d729cd3a8479583bef09527027dba5f1126","\\\\x3c5cbbeb3834b7a5c1cba9aa5fee0c95ec3f17a33ec3d8047fff799187f5ae20","\\\\x40bbe913c226c34c9fbe4389dd728984257a816892b3cae3e43191dd291f0eb5","\\\\x14af5385bcbb1e4738bbae8106046e6e2fca42875aa5c000c582587742bcc748","\\\\x72f29656803c2f4be177b1b8dd2a5137892b080b022100fde4e96d93ef8c96ff","\\\\xd06f27061c734d7825b46865d00aa900e5cc3a3672080e527171e1171aa5038a","\\\\x28203985b5f2d87709171678169739f957d2745f4bfa5cc91e2b4bd9bf483b40"}	\N	2025-01-23 12:59:03.176148+00	2025-01-23 12:59:10.13474+00
1	0	1	\\xc258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	\\x1c2326f86b0ebc1871dc8dcebd896e70f5a55422970d730909001a6b4ec8fad4	{"\\\\x5bcf2df28bd4535c167056d6dcd29391e79a11c85a5cb1f46a02eff40fd04741","\\\\xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5","\\\\xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30","\\\\x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85","\\\\xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344","\\\\x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d","\\\\x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968","\\\\xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83","\\\\x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af","\\\\xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0","\\\\xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5","\\\\xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892","\\\\x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c","\\\\xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb","\\\\x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc","\\\\xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2","\\\\x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f","\\\\xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a","\\\\x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0","\\\\xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0","\\\\xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2","\\\\xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9","\\\\x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377","\\\\x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652","\\\\xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef","\\\\x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d","\\\\xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0","\\\\x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e","\\\\x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e","\\\\x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322","\\\\x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735","\\\\x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9","\\\\x27ae5ba08d7291c96c8cbddcc148bf48a6d68c7974b94356f53754ef6171d757","\\\\xbf558bebd2ceec7f3c5dce04a4782f88c2c6036ae78ee206d0bc5289d20461a2","\\\\xe21908c2968c0699040a6fd866a577a99a9d2ec88745c815fd4a472c789244da","\\\\xae824d72ddc272aab68a8c3022e36f10454437c1886f3ff9927b64f232df414f","\\\\x27e429a4bef3083bc31a671d046ea5c1f5b8c3094d72868d9dfdc12c7334ac5f","\\\\x743cc5c365a9a6a15c1f240ac25880c7a9d1de290696cb766074a1d83d927816","\\\\x4adcf616c3bfabf63999a01966c998b7bb572774035a63ead49da73b5987f347","\\\\x75786645d0c5dd7c04a2f8a75dcae085213652f5bce3ea8b9b9bedd1cab3c5e9","\\\\xb88b152c9b8a7b79637d35911848b0c41e7cc7cca2ab4fe9a15f9c38bb4bb939","\\\\x0c4e2d8ce834ffd7a6cd85d7113d4521abb857774845c4291e6f6d010d97e318","\\\\x5bc799d83e3bb31501b3da786680df30fbc18eb41cbce611e8c0e9c72f69571c","\\\\xa10d3ef857d04d9c03ead7c6317d797a090fa1271ad9c7addfbcb412e9643d4f","\\\\xb33b1809c42623f474055fa9400a2027a7a885c8dfa4efe20666b4ee27d7529c","\\\\x134d7f28d53f175f6bf4b62faa2110d5b76f0f770c15e628181c1fcc18f970a9","\\\\xc34d24b2fc8c50ca9c07a7156ef4e5ff4bdf002eda0b11c1d359d0b59a546807","\\\\x04dbb9db631457879b27e0dfdbe50158fd9cf9b4cf77605c4ac4c95bd65fc9f6","\\\\xf9295a686647cb999090819cda700820c282c613cedcd218540bbc6f37b01c65","\\\\x67c4a1ea624f092a3a5cca2d6f0f0db231972fce627f0ecca0dee60f17551c5f","\\\\x8fdaeb5ab560b2ceb781cdb339361a0fbee1b9dffad59115138c8d6a70dda9cc","\\\\xc1bf0bbdd7fee15764845db875f6432559ff8dbc9055324431bc34e5b93d15da","\\\\x307317849eccd90c0c7b98870b9317c15a5959dcfb84c76dcc908c4fe6ba9212","\\\\x6339bf06e458f6646df5e83ba7c3d35bc263b3222c8e9040068847749ca8e8f9","\\\\x5045e4342aeb521eb3a5587ec268ed3aa6faf32b62b0bc41a9d549521f406fc3","\\\\x08601d83cdd34b5f7b8df63e7b9a16519d35473d0b89c317beed3d3d9424b253","\\\\x84e35c5d92171376cae5c86300822d729cd3a8479583bef09527027dba5f1126","\\\\x3c5cbbeb3834b7a5c1cba9aa5fee0c95ec3f17a33ec3d8047fff799187f5ae20","\\\\x40bbe913c226c34c9fbe4389dd728984257a816892b3cae3e43191dd291f0eb5","\\\\x14af5385bcbb1e4738bbae8106046e6e2fca42875aa5c000c582587742bcc748","\\\\x72f29656803c2f4be177b1b8dd2a5137892b080b022100fde4e96d93ef8c96ff","\\\\xd06f27061c734d7825b46865d00aa900e5cc3a3672080e527171e1171aa5038a","\\\\x28203985b5f2d87709171678169739f957d2745f4bfa5cc91e2b4bd9bf483b40"}	\N	2025-01-23 12:59:03.176148+00	2025-01-23 12:59:10.13474+00
1	1	2	\\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	\\x5bcf2df28bd4535c167056d6dcd29391e79a11c85a5cb1f46a02eff40fd04741	{"\\\\x1c2326f86b0ebc1871dc8dcebd896e70f5a55422970d730909001a6b4ec8fad4","\\\\xedc8d087f40b6369b8742cf3bb3e7ab1788f29d6c48829d9d1d9728dbde59aa4","\\\\xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30","\\\\x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85","\\\\xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344","\\\\x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d","\\\\x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968","\\\\xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83","\\\\x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af","\\\\xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0","\\\\xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5","\\\\xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892","\\\\x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c","\\\\xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb","\\\\x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc","\\\\xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2","\\\\x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f","\\\\xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a","\\\\x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0","\\\\xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0","\\\\xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2","\\\\xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9","\\\\x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377","\\\\x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652","\\\\xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef","\\\\x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d","\\\\xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0","\\\\x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e","\\\\x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e","\\\\x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322","\\\\x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735","\\\\x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9","\\\\x27ae5ba08d7291c96c8cbddcc148bf48a6d68c7974b94356f53754ef6171d757","\\\\xbf558bebd2ceec7f3c5dce04a4782f88c2c6036ae78ee206d0bc5289d20461a2","\\\\xe21908c2968c0699040a6fd866a577a99a9d2ec88745c815fd4a472c789244da","\\\\xae824d72ddc272aab68a8c3022e36f10454437c1886f3ff9927b64f232df414f","\\\\x27e429a4bef3083bc31a671d046ea5c1f5b8c3094d72868d9dfdc12c7334ac5f","\\\\x743cc5c365a9a6a15c1f240ac25880c7a9d1de290696cb766074a1d83d927816","\\\\x4adcf616c3bfabf63999a01966c998b7bb572774035a63ead49da73b5987f347","\\\\x75786645d0c5dd7c04a2f8a75dcae085213652f5bce3ea8b9b9bedd1cab3c5e9","\\\\xb88b152c9b8a7b79637d35911848b0c41e7cc7cca2ab4fe9a15f9c38bb4bb939","\\\\x0c4e2d8ce834ffd7a6cd85d7113d4521abb857774845c4291e6f6d010d97e318","\\\\x5bc799d83e3bb31501b3da786680df30fbc18eb41cbce611e8c0e9c72f69571c","\\\\xa10d3ef857d04d9c03ead7c6317d797a090fa1271ad9c7addfbcb412e9643d4f","\\\\xb33b1809c42623f474055fa9400a2027a7a885c8dfa4efe20666b4ee27d7529c","\\\\x134d7f28d53f175f6bf4b62faa2110d5b76f0f770c15e628181c1fcc18f970a9","\\\\xc34d24b2fc8c50ca9c07a7156ef4e5ff4bdf002eda0b11c1d359d0b59a546807","\\\\x04dbb9db631457879b27e0dfdbe50158fd9cf9b4cf77605c4ac4c95bd65fc9f6","\\\\xf9295a686647cb999090819cda700820c282c613cedcd218540bbc6f37b01c65","\\\\x67c4a1ea624f092a3a5cca2d6f0f0db231972fce627f0ecca0dee60f17551c5f","\\\\x8fdaeb5ab560b2ceb781cdb339361a0fbee1b9dffad59115138c8d6a70dda9cc","\\\\xc1bf0bbdd7fee15764845db875f6432559ff8dbc9055324431bc34e5b93d15da","\\\\x307317849eccd90c0c7b98870b9317c15a5959dcfb84c76dcc908c4fe6ba9212","\\\\x6339bf06e458f6646df5e83ba7c3d35bc263b3222c8e9040068847749ca8e8f9","\\\\x5045e4342aeb521eb3a5587ec268ed3aa6faf32b62b0bc41a9d549521f406fc3","\\\\x08601d83cdd34b5f7b8df63e7b9a16519d35473d0b89c317beed3d3d9424b253","\\\\x84e35c5d92171376cae5c86300822d729cd3a8479583bef09527027dba5f1126","\\\\x3c5cbbeb3834b7a5c1cba9aa5fee0c95ec3f17a33ec3d8047fff799187f5ae20","\\\\x40bbe913c226c34c9fbe4389dd728984257a816892b3cae3e43191dd291f0eb5","\\\\x14af5385bcbb1e4738bbae8106046e6e2fca42875aa5c000c582587742bcc748","\\\\x72f29656803c2f4be177b1b8dd2a5137892b080b022100fde4e96d93ef8c96ff","\\\\xd06f27061c734d7825b46865d00aa900e5cc3a3672080e527171e1171aa5038a","\\\\x28203985b5f2d87709171678169739f957d2745f4bfa5cc91e2b4bd9bf483b40"}	\N	2025-01-23 12:59:17.174646+00	2025-01-23 12:59:24.137465+00
1	1	3	\\xc258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000	\\x1c2326f86b0ebc1871dc8dcebd896e70f5a55422970d730909001a6b4ec8fad4	{"\\\\x5bcf2df28bd4535c167056d6dcd29391e79a11c85a5cb1f46a02eff40fd04741","\\\\xedc8d087f40b6369b8742cf3bb3e7ab1788f29d6c48829d9d1d9728dbde59aa4","\\\\xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30","\\\\x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85","\\\\xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344","\\\\x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d","\\\\x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968","\\\\xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83","\\\\x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af","\\\\xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0","\\\\xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5","\\\\xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892","\\\\x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c","\\\\xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb","\\\\x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc","\\\\xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2","\\\\x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f","\\\\xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a","\\\\x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0","\\\\xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0","\\\\xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2","\\\\xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9","\\\\x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377","\\\\x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652","\\\\xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef","\\\\x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d","\\\\xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0","\\\\x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e","\\\\x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e","\\\\x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322","\\\\x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735","\\\\x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9","\\\\x27ae5ba08d7291c96c8cbddcc148bf48a6d68c7974b94356f53754ef6171d757","\\\\xbf558bebd2ceec7f3c5dce04a4782f88c2c6036ae78ee206d0bc5289d20461a2","\\\\xe21908c2968c0699040a6fd866a577a99a9d2ec88745c815fd4a472c789244da","\\\\xae824d72ddc272aab68a8c3022e36f10454437c1886f3ff9927b64f232df414f","\\\\x27e429a4bef3083bc31a671d046ea5c1f5b8c3094d72868d9dfdc12c7334ac5f","\\\\x743cc5c365a9a6a15c1f240ac25880c7a9d1de290696cb766074a1d83d927816","\\\\x4adcf616c3bfabf63999a01966c998b7bb572774035a63ead49da73b5987f347","\\\\x75786645d0c5dd7c04a2f8a75dcae085213652f5bce3ea8b9b9bedd1cab3c5e9","\\\\xb88b152c9b8a7b79637d35911848b0c41e7cc7cca2ab4fe9a15f9c38bb4bb939","\\\\x0c4e2d8ce834ffd7a6cd85d7113d4521abb857774845c4291e6f6d010d97e318","\\\\x5bc799d83e3bb31501b3da786680df30fbc18eb41cbce611e8c0e9c72f69571c","\\\\xa10d3ef857d04d9c03ead7c6317d797a090fa1271ad9c7addfbcb412e9643d4f","\\\\xb33b1809c42623f474055fa9400a2027a7a885c8dfa4efe20666b4ee27d7529c","\\\\x134d7f28d53f175f6bf4b62faa2110d5b76f0f770c15e628181c1fcc18f970a9","\\\\xc34d24b2fc8c50ca9c07a7156ef4e5ff4bdf002eda0b11c1d359d0b59a546807","\\\\x04dbb9db631457879b27e0dfdbe50158fd9cf9b4cf77605c4ac4c95bd65fc9f6","\\\\xf9295a686647cb999090819cda700820c282c613cedcd218540bbc6f37b01c65","\\\\x67c4a1ea624f092a3a5cca2d6f0f0db231972fce627f0ecca0dee60f17551c5f","\\\\x8fdaeb5ab560b2ceb781cdb339361a0fbee1b9dffad59115138c8d6a70dda9cc","\\\\xc1bf0bbdd7fee15764845db875f6432559ff8dbc9055324431bc34e5b93d15da","\\\\x307317849eccd90c0c7b98870b9317c15a5959dcfb84c76dcc908c4fe6ba9212","\\\\x6339bf06e458f6646df5e83ba7c3d35bc263b3222c8e9040068847749ca8e8f9","\\\\x5045e4342aeb521eb3a5587ec268ed3aa6faf32b62b0bc41a9d549521f406fc3","\\\\x08601d83cdd34b5f7b8df63e7b9a16519d35473d0b89c317beed3d3d9424b253","\\\\x84e35c5d92171376cae5c86300822d729cd3a8479583bef09527027dba5f1126","\\\\x3c5cbbeb3834b7a5c1cba9aa5fee0c95ec3f17a33ec3d8047fff799187f5ae20","\\\\x40bbe913c226c34c9fbe4389dd728984257a816892b3cae3e43191dd291f0eb5","\\\\x14af5385bcbb1e4738bbae8106046e6e2fca42875aa5c000c582587742bcc748","\\\\x72f29656803c2f4be177b1b8dd2a5137892b080b022100fde4e96d93ef8c96ff","\\\\xd06f27061c734d7825b46865d00aa900e5cc3a3672080e527171e1171aa5038a","\\\\x28203985b5f2d87709171678169739f957d2745f4bfa5cc91e2b4bd9bf483b40"}	\N	2025-01-23 12:59:17.174646+00	2025-01-23 12:59:24.137465+00
\.


--
-- Data for Name: report; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.report (input_epoch_application_id, input_index, index, raw_data, created_at, updated_at) FROM stdin;
1	0	0	\\xdeadbeef	2025-01-23 12:59:03.176148+00	2025-01-23 12:59:03.176148+00
1	1	1	\\xdeadbeef	2025-01-23 12:59:17.174646+00	2025-01-23 12:59:17.174646+00
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.schema_migrations (version, dirty) FROM stdin;
1	f
\.


--
-- Name: application_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.application_id_seq', 1, true);


--
-- Name: application application_iapplication_address_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application
    ADD CONSTRAINT application_iapplication_address_key UNIQUE (iapplication_address);


--
-- Name: application application_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application
    ADD CONSTRAINT application_name_key UNIQUE (name);


--
-- Name: application application_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application
    ADD CONSTRAINT application_pkey PRIMARY KEY (id);


--
-- Name: epoch epoch_application_id_virtual_index_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_application_id_virtual_index_unique UNIQUE (application_id, virtual_index);


--
-- Name: epoch epoch_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_pkey PRIMARY KEY (application_id, index);


--
-- Name: execution_parameters execution_parameters_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.execution_parameters
    ADD CONSTRAINT execution_parameters_pkey PRIMARY KEY (application_id);


--
-- Name: input input_application_id_tx_reference_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_application_id_tx_reference_unique UNIQUE (epoch_application_id, transaction_reference);


--
-- Name: input input_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_pkey PRIMARY KEY (epoch_application_id, index);


--
-- Name: output output_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.output
    ADD CONSTRAINT output_pkey PRIMARY KEY (input_epoch_application_id, index);


--
-- Name: report report_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.report
    ADD CONSTRAINT report_pkey PRIMARY KEY (input_epoch_application_id, index);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: epoch_last_block_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX epoch_last_block_idx ON public.epoch USING btree (application_id, last_block);


--
-- Name: epoch_status_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX epoch_status_idx ON public.epoch USING btree (application_id, status);


--
-- Name: input_block_number_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX input_block_number_idx ON public.input USING btree (epoch_application_id, block_number);


--
-- Name: input_sender_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX input_sender_idx ON public.input USING btree (epoch_application_id, SUBSTRING(raw_data FROM 81 FOR 20));


--
-- Name: input_status_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX input_status_idx ON public.input USING btree (epoch_application_id, status);


--
-- Name: output_raw_data_address_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX output_raw_data_address_idx ON public.output USING btree (input_epoch_application_id, SUBSTRING(raw_data FROM 17 FOR 20)) WHERE (SUBSTRING(raw_data FROM 1 FOR 4) = ANY (ARRAY['\x10321e8b'::bytea, '\x237a816f'::bytea]));


--
-- Name: output_raw_data_type_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX output_raw_data_type_idx ON public.output USING btree (input_epoch_application_id, SUBSTRING(raw_data FROM 1 FOR 4));


--
-- Name: application application_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER application_set_updated_at BEFORE UPDATE ON public.application FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: epoch epoch_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER epoch_set_updated_at BEFORE UPDATE ON public.epoch FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: execution_parameters execution_parameters_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER execution_parameters_set_updated_at BEFORE UPDATE ON public.execution_parameters FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: input input_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER input_set_updated_at BEFORE UPDATE ON public.input FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: node_config node_config_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER node_config_set_updated_at BEFORE UPDATE ON public.node_config FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: output output_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER output_set_updated_at BEFORE UPDATE ON public.output FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: report report_set_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER report_set_updated_at BEFORE UPDATE ON public.report FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: epoch epoch_application_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_application_id_fkey FOREIGN KEY (application_id) REFERENCES public.application(id) ON DELETE CASCADE;


--
-- Name: execution_parameters execution_parameters_application_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.execution_parameters
    ADD CONSTRAINT execution_parameters_application_id_fkey FOREIGN KEY (application_id) REFERENCES public.application(id) ON DELETE CASCADE;


--
-- Name: input input_epoch_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_epoch_id_fkey FOREIGN KEY (epoch_application_id, epoch_index) REFERENCES public.epoch(application_id, index) ON DELETE CASCADE;


--
-- Name: output output_input_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.output
    ADD CONSTRAINT output_input_id_fkey FOREIGN KEY (input_epoch_application_id, input_index) REFERENCES public.input(epoch_application_id, index) ON DELETE CASCADE;


--
-- Name: report report_input_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.report
    ADD CONSTRAINT report_input_id_fkey FOREIGN KEY (input_epoch_application_id, input_index) REFERENCES public.input(epoch_application_id, index) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

