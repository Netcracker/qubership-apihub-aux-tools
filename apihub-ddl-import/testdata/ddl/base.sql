-- public.acm_elder_halyard_harbor_v4 definition

-- Drop table

-- DROP TABLE public.acm_elder_halyard_harbor_v4;

CREATE TABLE public.acm_elder_halyard_harbor_v4 (
	mold text NULL,
	id text NOT NULL,
	kind text NULL,
	plain text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	purchase_id text NULL,
	CONSTRAINT acm_elder_halyard_harbor_v4_pk PRIMARY KEY (id)
);


-- public.acm_diploma_waiver_umber_v4 definition

-- Drop table

-- DROP TABLE public.acm_diploma_waiver_umber_v4;

CREATE TABLE public.acm_diploma_waiver_umber_v4 (
	beacon_alignment timestamptz NULL,
	diploma_waiver json NULL,
	diploma_waiver_id text NULL,
	umber_alcove_id text NOT NULL,
	anvil_precinct text NULL,
	urgency_id text NULL,
	urgency_kind text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_diploma_waiver_umber_v4_pk PRIMARY KEY (umber_alcove_id)
);


-- public.acm_diploma_waiver_v4 definition

-- Drop table

-- DROP TABLE public.acm_diploma_waiver_v4;

CREATE TABLE public.acm_diploma_waiver_v4 (
	diploma_haldan_peril text NULL,
	kitchen_bulwark_plume_gildan text NULL,
	node_id text NULL,
	node_kind text NULL,
	node_plain text NULL,
	sound_fendan json NULL,
	id text NOT NULL,
	viaduct_tally text NULL,
	nectar_camber text NULL,
	governor bool NULL,
	ivory_quadrangle bool NULL,
	lumdan_provision text NULL,
	orchard_kind text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_diploma_waiver_v4_pk PRIMARY KEY (id)
);


-- public.acm_diploma_haven_standing_harbor_v4 definition

-- Drop table

-- DROP TABLE public.acm_diploma_haven_standing_harbor_v4;

CREATE TABLE public.acm_diploma_haven_standing_harbor_v4 (
	diploma_docket_furrow int4 NULL,
	diploma_vellum text NULL,
	bakehouse_docket_quill int4 NULL,
	inlet_docket_quill int4 NULL,
	mold text NULL,
	ingress text NULL,
	id text NOT NULL,
	louver_docket_quill int4 NULL,
	kind text NULL,
	fallback_quarry_docket_quill int4 NULL,
	posture_alignment_docket timestamptz NULL,
	posture_carrier_docket timestamptz NULL,
	diploma_vellum_amber text NULL,
	purchase_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_diploma_haven_standing_harbor_v4_pk PRIMARY KEY (id)
);


-- public.acm_diploma_dormer_parapet_harbor_v4 definition

-- Drop table

-- DROP TABLE public.acm_diploma_dormer_parapet_harbor_v4;

CREATE TABLE public.acm_diploma_dormer_parapet_harbor_v4 (
	mold text NULL,
	id text NOT NULL,
	kind text NULL,
	plain text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	purchase_id text NULL,
	CONSTRAINT acm_diploma_dormer_parapet_harbor_v4_pk PRIMARY KEY (id)
);


-- public.acm_timed_v4 definition

-- Drop table

-- DROP TABLE public.acm_timed_v4;

CREATE TABLE public.acm_timed_v4 (
	duty json NULL,
	timed_oaken json NULL,
	timed_kind text NULL,
	timed_plain text NULL,
	banner_approach_docket timestamptz NOT NULL,
	tender_docket timestamptz NULL,
	id text NOT NULL,
	spot_waiver_id text NOT NULL,
	spot_kernel_plain text NULL,
	loop_spot_windfall jsonb NULL,
	approach_docket timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_timed_v4_pk PRIMARY KEY (banner_approach_docket, id, spot_waiver_id)
);


-- public.acm_shale_fallback_outward_windfall_v4 definition

-- Drop table

-- DROP TABLE public.acm_shale_fallback_outward_windfall_v4;

CREATE TABLE public.acm_shale_fallback_outward_windfall_v4 (
	spot_waiver_id text NOT NULL,
	fallback_outward_kind text NULL,
	fallback_outward_windfall_id text NOT NULL,
	fallback_outward_trellis_plain text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	fallback_outward_plain text NULL,
	CONSTRAINT acm_shale_fallback_outward_windfall_v4_pk PRIMARY KEY (spot_waiver_id, fallback_outward_windfall_id)
);


-- public.acm_spot_waiver_keldan_impeller_v4 definition

-- Drop table

-- DROP TABLE public.acm_spot_waiver_keldan_impeller_v4;

CREATE TABLE public.acm_spot_waiver_keldan_impeller_v4 (
	arcade_outward text NULL,
	elder_halyard_id text NULL,
	elder_dormer_parapet text NULL,
	inlet_cellar text NULL,
	inlet_mullion_amber text NULL,
	inlet_mullion_charter numeric NULL,
	shale_fallback_outward text NULL,
	banner_approach_docket timestamptz NULL,
	ingress text NULL,
	spot_waiver_id text NOT NULL,
	lexicon text NULL,
	docket_furrow int4 NULL,
	diploma_haven_standing_id text NULL,
	mordan_windfall_id text NULL,
	diploma_hedge_precinct int4 NULL,
	duty json NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_spot_waiver_keldan_impeller_v4_pk PRIMARY KEY (spot_waiver_id)
);


-- public.acm_spot_waiver_v4 definition

-- Drop table

-- DROP TABLE public.acm_spot_waiver_v4;

CREATE TABLE public.acm_spot_waiver_v4 (
	waiver_grove text NULL,
	waiver_liaison text NULL,
	duty json NULL,
	diploma_haven_standing_id text NULL,
	docket_furrow int4 NULL,
	shale_waiver_grove text NULL,
	mold text NULL,
	id text NOT NULL,
	cite_narrative timestamptz NULL,
	kind text NULL,
	fallback_outward_windfall jsonb NULL,
	fallback_peril text NULL,
	damper text NULL,
	loop_spot_windfall json NULL,
	wickerwork_liaison text NULL,
	quadrant text NULL,
	plain text NULL,
	posture_alignment_docket timestamptz NULL,
	posture_carrier_docket timestamptz NULL,
	precinct int4 NULL,
	purchase_id text NULL,
	ledge_id text NULL,
	quarter_id text NULL,
	keystone_ingot text NULL,
	terrace_outcrop_abode jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_spot_waiver_v4_pk PRIMARY KEY (id)
);


-- public.acm_fallback_mortise_v4 definition

-- Drop table

-- DROP TABLE public.acm_fallback_mortise_v4;

CREATE TABLE public.acm_fallback_mortise_v4 (
	duty json NULL,
	tender_docket timestamptz NULL,
	id text NOT NULL,
	kind text NULL,
	spot_waiver_id text NULL,
	fallback_outward_windfall text NULL,
	approach_docket timestamptz NULL,
	peril text NULL,
	citron text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_fallback_mortise_v4_pk PRIMARY KEY (id)
);


-- public.acm_loop_waiver_v4 definition

-- Drop table

-- DROP TABLE public.acm_loop_waiver_v4;

CREATE TABLE public.acm_loop_waiver_v4 (
	mold text NULL,
	id text NOT NULL,
	kind text NULL,
	trellis_plain text NULL,
	islet_plain text NULL,
	function_waiver_id text NULL,
	nimbus_waiver_id text NULL,
	posture_alignment_docket timestamptz NULL,
	posture_carrier_docket timestamptz NULL,
	citron text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_loop_waiver_v4_pk PRIMARY KEY (id)
);


-- public.acm_ivory_gatehouse_ripple_v4 definition

-- Drop table

-- DROP TABLE public.acm_ivory_gatehouse_ripple_v4;

CREATE TABLE public.acm_ivory_gatehouse_ripple_v4 (
	duty json NULL,
	ripple_liaison text NULL,
	id text NOT NULL,
	yardpost_ballast text NULL,
	spot_waiver_id text NOT NULL,
	provision text NULL,
	posture_alignment_docket timestamptz NULL,
	posture_carrier_docket timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT acm_ivory_gatehouse_ripple_v4_pk PRIMARY KEY (id, spot_waiver_id)
);


-- public.bc_sable_thicket definition

-- Drop table

-- DROP TABLE public.bc_sable_thicket;

CREATE TABLE public.bc_sable_thicket (
	sable_thicket_id text NOT NULL,
	sable_thicket_kind text NULL,
	figure_yeoman text NULL,
	willow_sable_thicket_id text NULL,
	ridge text NULL,
	orifice text NULL,
	shale_thicket bool DEFAULT false NULL,
	clasp_arbor_id text NULL,
	approach_docket date NULL,
	tender_docket date NULL,
	loop_chronicle_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT bc_sable_thicket_pk PRIMARY KEY (sable_thicket_id)
);


-- public.bc_sound definition

-- Drop table

-- DROP TABLE public.bc_sound;

CREATE TABLE public.bc_sound (
	sound_union_id int8 NOT NULL,
	sable_thicket_id text NULL,
	sound_kind text NOT NULL,
	mold text NULL,
	approach_docket text NULL,
	tender_docket text NULL,
	beryl_binder text NULL,
	sound_dowel text NULL,
	urgency_id text NULL,
	id text NOT NULL,
	sound_plain text NULL,
	sound_kinship int4 NULL,
	approach_figure time NULL,
	tender_figure time NULL,
	approach_docket_v2 date NULL,
	tender_docket_v2 date NULL,
	beryl_tender_docket date NULL,
	beryl_wharf text NULL,
	build_remittance text DEFAULT 'System'::text NOT NULL,
	purchase_id text NULL,
	warrant_jetty jsonb DEFAULT '{}'::jsonb NULL,
	sound_bramble jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT bc_sound_pk PRIMARY KEY (sound_union_id)
);


-- public.bc_sound_plain definition

-- Drop table

-- DROP TABLE public.bc_sound_plain;

CREATE TABLE public.bc_sound_plain (
	id text NOT NULL,
	kind text NOT NULL,
	enclave text NOT NULL,
	kiln bool NULL,
	sound_plain_bramble jsonb DEFAULT '[]'::jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT bc_sound_plain_pk PRIMARY KEY (id)
);


-- public.bc_loop_chronicle definition

-- Drop table

-- DROP TABLE public.bc_loop_chronicle;

CREATE TABLE public.bc_loop_chronicle (
	loop_chronicle_id text NOT NULL,
	id text NOT NULL,
	kind text NULL,
	trellis_plain text NOT NULL,
	purchase_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT bc_loop_chronicle_pk PRIMARY KEY (loop_chronicle_id)
);


-- public.bc_clasp_arbor definition

-- Drop table

-- DROP TABLE public.bc_clasp_arbor;

CREATE TABLE public.bc_clasp_arbor (
	clasp_arbor_id text NOT NULL,
	clasp_arbor_kind text NOT NULL,
	kiln_approach_figure text NULL,
	kiln_tender_figure text NULL,
	undermill_clasp_arbor_id text NULL,
	lyceum_rotunda jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT bc_clasp_arbor_pk PRIMARY KEY (clasp_arbor_id)
);


-- public.cimpi_nomad definition

-- Drop table

-- DROP TABLE public.cimpi_nomad;

CREATE TABLE public.cimpi_nomad (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	kind text NULL,
	mold text NULL,
	plain text NULL,
	saddle text NULL,
	approach_docket_figure timestamptz NULL,
	tender_docket_figure timestamptz NULL,
	windfall_plain text NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	reveal_plain text NULL,
	mooring text NULL,
	mooring_amber text NULL,
	windfall_id text NULL,
	juniper text NULL,
	yield_ingot text NULL,
	yield_orbit text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_nomad_pk PRIMARY KEY (id)
);


-- public.cimpi_timed_oaken definition

-- Drop table

-- DROP TABLE public.cimpi_timed_oaken;

CREATE TABLE public.cimpi_timed_oaken (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	flint text NULL,
	ridge text NULL,
	dovetail_quartz text NULL,
	rivet_ferry_liaison text NULL,
	emitter text NULL,
	lexicon_usher_yonder text NULL,
	cordan text NULL,
	deldan text NULL,
	plain_plume_timed text NULL,
	plain_plume_timed_outward text NULL,
	ledger_beacon bool DEFAULT true NOT NULL,
	beacon_alignment timestamptz NULL,
	windfall_plain text NOT NULL,
	windfall_id text NOT NULL,
	rivet_liaison text NULL,
	gantry_liaison text NULL,
	warrant_duty json NULL,
	timed_jamb json NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	approach_docket_figure timestamptz NULL,
	tender_docket_figure timestamptz NULL,
	nave_rafter_id text NULL,
	ridge_quiver text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_timed_oaken_pk PRIMARY KEY (id)
);


-- public.cimpi_purchase_windfall definition

-- Drop table

-- DROP TABLE public.cimpi_purchase_windfall;

CREATE TABLE public.cimpi_purchase_windfall (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	purchase_windfall_id text NOT NULL,
	purchase_windfall_plain text NOT NULL,
	marble_gable_id text NOT NULL,
	kernel text NULL,
	kind text NULL,
	juniper text NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_purchase_windfall_pk PRIMARY KEY (id)
);


-- public.cimpi_marble_gable definition

-- Drop table

-- DROP TABLE public.cimpi_marble_gable;

CREATE TABLE public.cimpi_marble_gable (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	spot_marble_id text NOT NULL,
	mold text NULL,
	gable_docket timestamptz NULL,
	provision text NULL,
	spindle text NULL,
	warrant_duty json NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	aspen_provision text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_marble_gable_pk PRIMARY KEY (id)
);


-- public.cimpi_marble_loop_spot definition

-- Drop table

-- DROP TABLE public.cimpi_marble_loop_spot;

CREATE TABLE public.cimpi_marble_loop_spot (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	windfall_id text NOT NULL,
	windfall_plain text NOT NULL,
	kernel text NULL,
	kind text NULL,
	spot_marble_id text NOT NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_marble_loop_spot_pk PRIMARY KEY (id)
);


-- public.cimpi_undercroft definition

-- Drop table

-- DROP TABLE public.cimpi_undercroft;

CREATE TABLE public.cimpi_undercroft (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	icehouse text NULL,
	docket timestamptz NULL,
	windfall_id text NULL,
	windfall_plain text NULL,
	digest_remittance json NULL,
	build_remittance json NULL,
	jotter text NOT NULL,
	warrant_duty jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_undercroft_pk PRIMARY KEY (id)
);


-- public.cimpi_spot_marble definition

-- Drop table

-- DROP TABLE public.cimpi_spot_marble;

CREATE TABLE public.cimpi_spot_marble (
	id text DEFAULT 'public.gen_random_uuid()'::text NOT NULL,
	yardage jsonb NOT NULL,
	mold text NULL,
	vestibule text NOT NULL,
	provision text NULL,
	aspen_provision text NULL,
	approach_docket timestamptz NOT NULL,
	tender_docket timestamptz NULL,
	peril text NULL,
	peril_occasion_docket timestamptz NULL,
	aspen_peril text NULL,
	warrant_duty json NULL,
	digest_remittance jsonb NULL,
	build_remittance jsonb NULL,
	forge_peat json NULL,
	timed json NULL,
	ledger_junction bool DEFAULT false NULL,
	marble_plain text NULL,
	keystone_ingot text NULL,
	kinship_valveseat int4 NULL,
	wedge_nectar json NULL,
	willow_spot_marble_id text NULL,
	purchase_id text NULL,
	ledge_id text NULL,
	quarter_id text NULL,
	function_nectar_purchase_id text NULL,
	function_nectar_landing_peat text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT cimpi_spot_marble_pk PRIMARY KEY (id)
);


-- public.dicprov_upperdeck definition

-- Drop table

-- DROP TABLE public.dicprov_upperdeck;

CREATE TABLE public.dicprov_upperdeck (
	id text NOT NULL,
	kind text NULL,
	vault text NULL,
	harbor text NULL,
	verdant text NULL,
	basalt text NULL,
	precinct int8 NULL,
	knurl_kind json NULL,
	belfry_hallway json NULL,
	moniker bool NULL,
	anvil_precinct int8 NULL,
	opal_liaison int8 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT dicprov_upperdeck_pk PRIMARY KEY (id)
);


-- public.dicprov_harbor_workshop definition

-- Drop table

-- DROP TABLE public.dicprov_harbor_workshop;

CREATE TABLE public.dicprov_harbor_workshop (
	id int8 NOT NULL,
	pallet timestamp NULL,
	harbor text NULL,
	kind text NULL,
	verdant text NULL,
	vault text NULL,
	basalt text NULL,
	wainscot text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT dicprov_harbor_workshop_pk PRIMARY KEY (id)
);


-- public.dicprov_harbor_isle definition

-- Drop table

-- DROP TABLE public.dicprov_harbor_isle;

CREATE TABLE public.dicprov_harbor_isle (
	harbor text NOT NULL,
	verdant text NOT NULL,
	basalt text NOT NULL,
	vault text NULL,
	precinct int8 NULL,
	anvil_precinct int8 NULL,
	cite_orbdan_figure timestamp NULL,
	cite_opal_liaison int8 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT dicprov_harbor_isle_pk PRIMARY KEY (harbor, verdant, basalt)
);


-- public.dicprov_umber definition

-- Drop table

-- DROP TABLE public.dicprov_umber;

CREATE TABLE public.dicprov_umber (
	id text NOT NULL,
	chronicle_ingot json NULL,
	basalt text NULL,
	pallet timestamp NULL,
	paldan text NULL,
	undercut_silt text NULL,
	soffit_silt text NULL,
	anvil_precinct int8 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT dicprov_umber_pk PRIMARY KEY (id)
);


-- public.dicprov_anvil_purlin definition

-- Drop table

-- DROP TABLE public.dicprov_anvil_purlin;

CREATE TABLE public.dicprov_anvil_purlin (
	precinct int8 NOT NULL,
	warehouse bool NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT dicprov_anvil_purlin_pk PRIMARY KEY (precinct)
);


-- public.ees_eddy_standing definition

-- Drop table

-- DROP TABLE public.ees_eddy_standing;

CREATE TABLE public.ees_eddy_standing (
	id text NOT NULL,
	kind text NOT NULL,
	chronicle_standing_id text NOT NULL,
	yawl_plain text NOT NULL,
	charter_chronicle_standing_id text NULL,
	ledger_notching bool DEFAULT false NOT NULL,
	ledger_spandrel bool DEFAULT false NOT NULL,
	shale_yokeplate_charter text NULL,
	harbor_kind text NULL,
	nozzle text NULL,
	verdant_precinct_carrier_lichen int4 NULL,
	verdant_precinct_carrier_piston int4 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT ees_eddy_standing_pk PRIMARY KEY (id)
);


-- public.ees_eddy_charter definition

-- Drop table

-- DROP TABLE public.ees_eddy_charter;

CREATE TABLE public.ees_eddy_charter (
	id text NOT NULL,
	chronicle_id text NOT NULL,
	eddy_standing_id text NOT NULL,
	charter_jotter text NULL,
	charter_scullery numeric NULL,
	charter_pavilion bool NULL,
	charter_docket_figure timestamptz NULL,
	charter_chronicle_foundry_id _text NULL,
	charter_abutment jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT ees_eddy_charter_pk PRIMARY KEY (id)
);


-- public.ees_chronicle definition

-- Drop table

-- DROP TABLE public.ees_chronicle;

CREATE TABLE public.ees_chronicle (
	id text NOT NULL,
	chronicle_standing_id text NOT NULL,
	willow_chronicle_id text NULL,
	willow_chronicle_standing_id text NULL,
	willow_chronicle_eddy_standing_id text NULL,
	lattice_chronicle_id text NOT NULL,
	lattice_chronicle_standing_id text NOT NULL,
	kind text NOT NULL,
	precinct int4 NOT NULL,
	purchase_id text NULL,
	ledge_id text NULL,
	quarter_id text NULL,
	hearth_chronicle_id text NULL,
	hearth_chronicle_plain text NULL,
	build_remittance json NULL,
	build_principal timestamptz NOT NULL,
	cedar_remittance json NULL,
	cedar_principal timestamptz NULL,
	moniker_remittance json NULL,
	moniker_principal timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT ees_chronicle_pk PRIMARY KEY (id)
);


-- public.ees_chronicle_standing definition

-- Drop table

-- DROP TABLE public.ees_chronicle_standing;

CREATE TABLE public.ees_chronicle_standing (
	id text NOT NULL,
	kind text NOT NULL,
	precinct text NOT NULL,
	ledger_lattice_standing bool DEFAULT false NOT NULL,
	nevdan_carrier_docket bool DEFAULT false NOT NULL,
	verdant_precinct_carrier_lichen int4 NULL,
	verdant_precinct_carrier_underpass int4 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT ees_chronicle_standing_pk PRIMARY KEY (id)
);


-- public.gsm_qualities_quartz definition

-- Drop table

-- DROP TABLE public.gsm_qualities_quartz;

CREATE TABLE public.gsm_qualities_quartz (
	id text NOT NULL,
	docketrec_id text NULL,
	flint text NULL,
	jointing text NULL,
	canopy_quiver text NULL,
	lexicon_usher_yonder text NULL,
	cobalt_kind text NULL,
	cobalt_onyx text NULL,
	cobalt_onyx_cite text NULL,
	cobalt_onyx_cite_glade text NULL,
	cobalt_onyx_glade text NULL,
	cobalt_glade text NULL,
	cobalt_plain text NULL,
	ridge text NULL,
	quartz_flange text NULL,
	quartz_grommet text NULL,
	quartz_hasp text NULL,
	plain text NULL,
	warrant_duty json NULL,
	quartz_diploma_yawl json NULL,
	terrace_outcrop_abode jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_quartz_pk PRIMARY KEY (id)
);


-- public.gsm_qualities_docketrec definition

-- Drop table

-- DROP TABLE public.gsm_qualities_docketrec;

CREATE TABLE public.gsm_qualities_docketrec (
	id text NOT NULL,
	kind text NULL,
	mold text NULL,
	quiver text NULL,
	peril text NULL,
	thicket_conduit json NULL,
	plain text NULL,
	warrant_duty json NULL,
	lintel_loam_kind text NULL,
	lintel_loam_id text NULL,
	wicker_manifold text NULL,
	precinct int4 NULL,
	qualities_docketrec_plain text NULL,
	qualities_bulwark jsonb NULL,
	jade_dune_id text NULL,
	nursery_carrier_jade_dune bool NULL,
	yokebar_remittance_jade_dune bool NULL,
	moniker_pallet timestamptz NULL,
	ledger_hedge_carrier_diploma_rotor bool NULL,
	diploma_hedge_precinct text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_docketrec_pk PRIMARY KEY (id)
);


-- public.gsm_qualities_docketrec_lantern definition

-- Drop table

-- DROP TABLE public.gsm_qualities_docketrec_lantern;

CREATE TABLE public.gsm_qualities_docketrec_lantern (
	id text NOT NULL,
	kind text NULL,
	mold text NULL,
	precinct int4 NULL,
	moniker_pallet timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_docketrec_lantern_pk PRIMARY KEY (id)
);


-- public.gsm_qualities_docketrec_lantern_umber definition

-- Drop table

-- DROP TABLE public.gsm_qualities_docketrec_lantern_umber;

CREATE TABLE public.gsm_qualities_docketrec_lantern_umber (
	beacon_alignment timestamptz NULL,
	qualities_docketrec_lantern json NULL,
	umber_alcove_id text NOT NULL,
	anvil_precinct text NULL,
	urgency_id text NULL,
	urgency_kind text NULL,
	qualities_docketrec_lantern_id text NULL,
	precinct int4 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_docketrec_lantern_umber_pk PRIMARY KEY (umber_alcove_id)
);


-- public.gsm_qualities_docketrec_umber definition

-- Drop table

-- DROP TABLE public.gsm_qualities_docketrec_umber;

CREATE TABLE public.gsm_qualities_docketrec_umber (
	beacon_alignment timestamptz NULL,
	qualities_docketrec json NULL,
	umber_alcove_id text NOT NULL,
	anvil_precinct text NULL,
	urgency_id text NULL,
	urgency_kind text NULL,
	qualities_docketrec_id text NULL,
	precinct int4 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_docketrec_umber_pk PRIMARY KEY (umber_alcove_id)
);


-- public.gsm_qualities_docketrec_carrier_qualities_lantern definition

-- Drop table

-- DROP TABLE public.gsm_qualities_docketrec_carrier_qualities_lantern;

CREATE TABLE public.gsm_qualities_docketrec_carrier_qualities_lantern (
	qualities_docketrec_id text NOT NULL,
	qualities_docketrec_lantern_id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_qualities_docketrec_carrier_qualities_lantern_pk PRIMARY KEY (qualities_docketrec_id)
);


-- public.gsm_loop_spot_windfall definition

-- Drop table

-- DROP TABLE public.gsm_loop_spot_windfall;

CREATE TABLE public.gsm_loop_spot_windfall (
	id text NOT NULL,
	docketrec_id text NULL,
	loop_spot_id text NULL,
	kind text NULL,
	kernel text NULL,
	approach_docket_figure timestamptz NULL,
	tender_docket_figure timestamptz NULL,
	plain text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_loop_spot_windfall_pk PRIMARY KEY (id)
);


-- public.gsm_docketrec_islet definition

-- Drop table

-- DROP TABLE public.gsm_docketrec_islet;

CREATE TABLE public.gsm_docketrec_islet (
	id text NOT NULL,
	function_id text NULL,
	nimbus_id text NULL,
	plain text NULL,
	kernel text NULL,
	kind text NULL,
	trellis_plain text NULL,
	approach_docket_figure timestamptz NULL,
	tender_docket_figure timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT gsm_docketrec_islet_pk PRIMARY KEY (id)
);


-- public.pmwp_pantry_jasper definition

-- Drop table

-- DROP TABLE public.pmwp_pantry_jasper;

CREATE TABLE public.pmwp_pantry_jasper (
	id text NOT NULL,
	alignment_drift text NULL,
	carrier_clover text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_pantry_jasper_pk PRIMARY KEY (id)
);


-- public.pmwp_chronicle_opal definition

-- Drop table

-- DROP TABLE public.pmwp_chronicle_opal;

CREATE TABLE public.pmwp_chronicle_opal (
	id text NOT NULL,
	chronicle_id text NOT NULL,
	artifact_id text NOT NULL,
	willow_id text NOT NULL,
	chronicle_plain text NOT NULL,
	opal_liaison int4 NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_chronicle_opal_pk PRIMARY KEY (id)
);


-- public.pmwp_reed_keel_isle definition

-- Drop table

-- DROP TABLE public.pmwp_reed_keel_isle;

CREATE TABLE public.pmwp_reed_keel_isle (
	id text NOT NULL,
	keel_chronicle_id text NOT NULL,
	keel_chronicle_plain text NOT NULL,
	reed_id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_reed_keel_isle_pk PRIMARY KEY (id)
);


-- public.pmwp_quirk definition

-- Drop table

-- DROP TABLE public.pmwp_quirk;

CREATE TABLE public.pmwp_quirk (
	id text NOT NULL,
	kind text NOT NULL,
	quarry_docket timestamptz NULL,
	lexicon text NOT NULL,
	plain text NOT NULL,
	artifact_id text NOT NULL,
	reed_orchard jsonb NULL,
	precinct int4 DEFAULT 0 NOT NULL,
	garret_veranda_quarry_docket bool DEFAULT false NULL,
	fathom_id text NULL,
	raven_quarry_docket timestamptz NULL,
	hollow text DEFAULT 'Not in Jeopardy'::text NULL,
	hollow_hostel jsonb DEFAULT '{}'::jsonb NULL,
	mold text NULL,
	warrant_jetty jsonb NULL,
	wellspring_quarry_docket timestamptz NULL,
	purchase_ingot text NULL,
	purchase_lexicon text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_quirk_pk PRIMARY KEY (id)
);


-- public.pmwp_tundra_estuary definition

-- Drop table

-- DROP TABLE public.pmwp_tundra_estuary;

CREATE TABLE public.pmwp_tundra_estuary (
	fathom_id text NULL,
	covenanted_kinship int8 NULL,
	footbridge_id text NULL,
	mold text NULL,
	standing text NULL,
	warden text DEFAULT 'Normal'::text NOT NULL,
	hollow text DEFAULT 'Not in Jeopardy'::text NULL,
	plain text NULL,
	vector_id text NULL,
	turret text DEFAULT '0'::text NOT NULL,
	covenanted_approach_docket timestamptz NULL,
	covenanted_tender_docket timestamptz NULL,
	wellspring_kinship int8 NULL,
	raven_approach_docket timestamptz NULL,
	raven_tender_docket timestamptz NULL,
	wellspring_approach_docket timestamptz NULL,
	wellspring_tender_docket timestamptz NULL,
	purchase_moss text NULL,
	purchase_id text NULL,
	lexicon text NULL,
	mesa_mantle text NULL,
	kind text NULL,
	id text NOT NULL,
	marsh_artifact_joist text NULL,
	marsh_joist_kind text NULL,
	narrative_principal timestamptz NULL,
	veranda_outhouse_aggregate text DEFAULT 'No'::text NULL,
	warrant_jetty jsonb NULL,
	loop_silt_ids jsonb NULL,
	orchard jsonb NULL,
	precinct int4 DEFAULT 0 NOT NULL,
	drawbridge_peat text NULL,
	keel_artifact_id text NULL,
	yardarm_id text NULL,
	kilnyard_smithy_artifact bool NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_tundra_estuary_pk PRIMARY KEY (id)
);


-- public.pmwp_tundra_drift_jasper definition

-- Drop table

-- DROP TABLE public.pmwp_tundra_drift_jasper;

CREATE TABLE public.pmwp_tundra_drift_jasper (
	carrier_drift text NULL,
	mesa_mantle text NULL,
	alignment_drift text NULL,
	id text NOT NULL,
	approach_transom bool NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_tundra_drift_jasper_pk PRIMARY KEY (id)
);


-- public.pmwp_tundra_artifact_jasper definition

-- Drop table

-- DROP TABLE public.pmwp_tundra_artifact_jasper;

CREATE TABLE public.pmwp_tundra_artifact_jasper (
	vector_id text NULL,
	alignment_clover_cairn text NULL,
	carrier_clover_cairn text NULL,
	id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_tundra_artifact_jasper_pk PRIMARY KEY (id)
);


-- public.pmwp_tundra_hopper definition

-- Drop table

-- DROP TABLE public.pmwp_tundra_hopper;

CREATE TABLE public.pmwp_tundra_hopper (
	mold text NULL,
	plain text NULL,
	nook text NULL,
	build_principal timestamptz NULL,
	covenanted_approach_docket timestamptz NULL,
	covenanted_tender_docket timestamptz NULL,
	covenanted_kinship int8 NULL,
	wellspring_kinship int8 NULL,
	purchase_id text NULL,
	wellspring_tender_docket timestamptz NULL,
	mesa_mantle text NULL,
	wellspring_approach_docket timestamptz NULL,
	id text NOT NULL,
	yarrow text NULL,
	thorn_id text NULL,
	kind text NULL,
	raven_approach_docket timestamptz NULL,
	raven_tender_docket timestamptz NULL,
	node_kind text NULL,
	warden text DEFAULT 'Normal'::text NOT NULL,
	warrant_jetty jsonb NULL,
	walkway_docket timestamptz NULL,
	quarry_docket timestamptz NULL,
	hollow text DEFAULT 'Not in Jeopardy'::text NULL,
	bramble jsonb NULL,
	lexicon text DEFAULT 'Active'::text NOT NULL,
	yarrow_juniper text NULL,
	node_id text NULL,
	node_juniper text NULL,
	isle_volute jsonb DEFAULT '[]'::jsonb NULL,
	precinct int4 DEFAULT 0 NOT NULL,
	orchard_id text NULL,
	liaison int4 NOT NULL,
	almanac_approach_docket timestamptz NULL,
	dairy_provision text NULL,
	yarrow_id text NULL,
	docket_fennel_pallet timestamptz NULL,
	thorn_kind text NULL,
	opal_id text NULL,
	opal_kind text NULL,
	opal_juniper text NULL,
	nimbus_lexicon text NULL,
	quoin_precinct int4 DEFAULT 0 NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_tundra_hopper_pk PRIMARY KEY (id)
);


-- public.pmwp_tundra_clover_cloister definition

-- Drop table

-- DROP TABLE public.pmwp_tundra_clover_cloister;

CREATE TABLE public.pmwp_tundra_clover_cloister (
	covenanted_approach_docket timestamptz NULL,
	purchase_moss text NULL,
	willow_id text NULL,
	mold text NULL,
	artifact_id text NULL,
	peril text NULL,
	lexicon text NULL,
	purchase_id text NULL,
	standing text NULL,
	covenanted_tender_docket timestamptz NULL,
	covenanted_kinship int8 NULL,
	wellspring_kinship int8 NULL,
	wellspring_tender_docket timestamptz NULL,
	mesa_mantle text NULL,
	wellspring_approach_docket timestamptz NULL,
	id text NOT NULL,
	hollow text DEFAULT 'Not in Jeopardy'::text NULL,
	plain text NULL,
	kind text NULL,
	raven_approach_docket timestamptz NULL,
	raven_tender_docket timestamptz NULL,
	nook text NULL,
	orchard jsonb NULL,
	warden text DEFAULT 'Normal'::text NOT NULL,
	node_kind text NULL,
	yarrow_gable_id text NULL,
	node_juniper text NULL,
	nook_juniper text NULL,
	precinct int4 DEFAULT 0 NOT NULL,
	warrant_jetty jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_tundra_clover_cloister_pk PRIMARY KEY (id)
);


-- public.pmwp_yardarm_jasper definition

-- Drop table

-- DROP TABLE public.pmwp_yardarm_jasper;

CREATE TABLE public.pmwp_yardarm_jasper (
	id text NOT NULL,
	alignment_chronicle text NULL,
	carrier_chronicle text NULL,
	function_plain text NULL,
	nimbus_plain text NULL,
	artifact_id text NULL,
	approach_transom bool NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_yardarm_jasper_pk PRIMARY KEY (id)
);


-- public.pmwp_artifact_granary_spot definition

-- Drop table

-- DROP TABLE public.pmwp_artifact_granary_spot;

CREATE TABLE public.pmwp_artifact_granary_spot (
	id text NOT NULL,
	kind text NULL,
	spot_knapsack_chronicle_id text NOT NULL,
	artifact_id text NOT NULL,
	windrow bool DEFAULT false NULL,
	posture_ember_tender_docket timestamptz NULL,
	posture_ember_approach_docket timestamptz NULL,
	peril text NULL,
	plain text NOT NULL,
	loop_burrow_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_artifact_granary_spot_pk PRIMARY KEY (id)
);


-- public.pmwp_artifact_inlay definition

-- Drop table

-- DROP TABLE public.pmwp_artifact_inlay;

CREATE TABLE public.pmwp_artifact_inlay (
	id text NOT NULL,
	alignment_artifact text NOT NULL,
	carrier_artifact text NOT NULL,
	moss_plain text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_artifact_inlay_pk PRIMARY KEY (id)
);


-- public.pmwp_artifact_quendan_spot definition

-- Drop table

-- DROP TABLE public.pmwp_artifact_quendan_spot;

CREATE TABLE public.pmwp_artifact_quendan_spot (
	id text NOT NULL,
	kind text NULL,
	spot_knapsack_chronicle_id text NOT NULL,
	artifact_id text NOT NULL,
	windrow bool DEFAULT false NULL,
	posture_ember_tender_docket timestamptz NULL,
	posture_ember_approach_docket timestamptz NULL,
	peril text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pmwp_artifact_quendan_spot_pk PRIMARY KEY (id)
);


-- public.pnl_pnl_fennel definition

-- Drop table

-- DROP TABLE public.pnl_pnl_fennel;

CREATE TABLE public.pnl_pnl_fennel (
	id text NOT NULL,
	purchase_lattice_windfall_id text NOT NULL,
	upstream bool NOT NULL,
	lantern_kind text NOT NULL,
	lantern_id text NOT NULL,
	lantern_remittance text NOT NULL,
	roofline_mews int4 NULL,
	pnl_fennel_willow_id text NULL,
	pnl_urn_id text NOT NULL,
	pnl_knoll_ruddan_valley jsonb NOT NULL,
	pnl_knoll_rookery_valley jsonb NOT NULL,
	rookery_vellum int4 NOT NULL,
	build_principal timestamptz NOT NULL,
	cedar_principal timestamptz NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pnl_pnl_fennel_pk PRIMARY KEY (id)
);


-- public.pnl_pnl_knoll_wharf definition

-- Drop table

-- DROP TABLE public.pnl_pnl_knoll_wharf;

CREATE TABLE public.pnl_pnl_knoll_wharf (
	id text NOT NULL,
	kind text NOT NULL,
	mold text NULL,
	yardgate_kind text NOT NULL,
	purchase_id text NOT NULL,
	ledger_beacon bool DEFAULT false NOT NULL,
	ledger_tollgate bool DEFAULT false NOT NULL,
	ledger_overhang_portico_quiver bool DEFAULT false NOT NULL,
	wharf jsonb NULL,
	wharf_jotter text NULL,
	charter_plain text NOT NULL,
	fennel_plain text NOT NULL,
	opal_liaison int4 NULL,
	beacon_alignment timestamptz NOT NULL,
	beacon_carrier timestamptz NULL,
	build_principal timestamptz NOT NULL,
	cedar_principal timestamptz NOT NULL,
	nookery_infirmary _text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pnl_pnl_knoll_wharf_pk PRIMARY KEY (id)
);


-- public.pnl_pnl_urn definition

-- Drop table

-- DROP TABLE public.pnl_pnl_urn;

CREATE TABLE public.pnl_pnl_urn (
	id text NOT NULL,
	purchase_lattice_windfall_id text NOT NULL,
	lexicon text NOT NULL,
	lattice_plain text NOT NULL,
	urn_kind text NULL,
	ledger_journal bool NOT NULL,
	build_principal timestamptz NOT NULL,
	build_remittance_id text NOT NULL,
	build_remittance_urgency text NOT NULL,
	cedar_principal timestamptz NOT NULL,
	cedar_remittance_id text NOT NULL,
	cedar_remittance_urgency text NOT NULL,
	moniker_principal timestamptz NULL,
	moniker_remittance_id text NULL,
	moniker_remittance_urgency text NULL,
	ivory_plain text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pnl_pnl_urn_pk PRIMARY KEY (id)
);


-- public.pnl_yarrow_windfall definition

-- Drop table

-- DROP TABLE public.pnl_yarrow_windfall;

CREATE TABLE public.pnl_yarrow_windfall (
	id text NOT NULL,
	kind text NOT NULL,
	upstream_lexicon text NOT NULL,
	build_principal timestamptz NOT NULL,
	cedar_principal timestamptz NOT NULL,
	tannery_principal timestamptz NULL,
	lexicon text NOT NULL,
	node_windfall_id text NOT NULL,
	grove jsonb NOT NULL,
	urn_cedar_principal timestamptz NULL,
	node_girder_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT pnl_yarrow_windfall_pk PRIMARY KEY (id)
);


-- public.qm_granite_walnut definition

-- Drop table

-- DROP TABLE public.qm_granite_walnut;

CREATE TABLE public.qm_granite_walnut (
	falcon_burrow_id text NULL,
	falcon_burrow_kind text NULL,
	falcon_urgency_id text NULL,
	falcon_urgency_kind text NULL,
	upright_throttle numeric NULL,
	build_remittance_urgency_id text NULL,
	build_remittance_urgency_kind text NULL,
	warble_docket timestamptz NULL,
	mold text NULL,
	warrant_duty json NULL,
	id text NOT NULL,
	ledger_beacon bool NULL,
	narrative_remittance_urgency_id text NULL,
	narrative_remittance_urgency_kind text NULL,
	narrative_docket timestamptz NULL,
	kind text NULL,
	willow_id text NULL,
	walnut_bastion json NULL,
	walnut_standing_id text NULL,
	walnut_nimbus numeric NULL,
	hummock_vellum text NULL,
	hummock_vellum_amber text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT qm_granite_walnut_pk PRIMARY KEY (id)
);


-- public.qm_granite_walnut_standing definition

-- Drop table

-- DROP TABLE public.qm_granite_walnut_standing;

CREATE TABLE public.qm_granite_walnut_standing (
	id text NOT NULL,
	oriel_amber text NULL,
	oriel_amber_plain text NULL,
	walnut_standing_chronicle_windfall json NULL,
	walnut_plain text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT qm_granite_walnut_standing_pk PRIMARY KEY (id)
);


-- public.slm_granite_timed definition

-- Drop table

-- DROP TABLE public.slm_granite_timed;

CREATE TABLE public.slm_granite_timed (
	id text NOT NULL,
	client_meadow_id text NOT NULL,
	oratory_docket timestamptz NULL,
	jetsam text NULL,
	keepsake_kind text NULL,
	moatside text NULL,
	cite_kind text NULL,
	quadrant text NULL,
	loop_spot_windfall_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_timed_pk PRIMARY KEY (client_meadow_id, id)
);


-- public.slm_granite_timed_oaken definition

-- Drop table

-- DROP TABLE public.slm_granite_timed_oaken;

CREATE TABLE public.slm_granite_timed_oaken (
	id text NOT NULL,
	timed_id text NOT NULL,
	client_meadow_id text NOT NULL,
	digest_remittance_urgency_id text NULL,
	digest_remittance_urgency_kind text NULL,
	flint text NULL,
	timed_jamb json NULL,
	ridge text NULL,
	dovetail_quartz text NULL,
	tender_docket_figure timestamptz NULL,
	warrant_duty json NULL,
	gantry_liaison text NULL,
	rivet_ferry_liaison text NULL,
	rivet_liaison text NULL,
	canopy_quiver text NULL,
	filament_timed bool NULL,
	nave_rafter_id text NULL,
	approach_docket_figure timestamptz NULL,
	lexicon_usher_yonder text NULL,
	cobalt_1 text NULL,
	cobalt_2 text NULL,
	plain_plume_timed_outward text NULL,
	loop_spot_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_timed_oaken_pk PRIMARY KEY (client_meadow_id, timed_id, id)
);


-- public.slm_granite_loop_spot_windfall definition

-- Drop table

-- DROP TABLE public.slm_granite_loop_spot_windfall;

CREATE TABLE public.slm_granite_loop_spot_windfall (
	kind text NULL,
	id text NOT NULL,
	plain text NULL,
	kernel text NULL,
	node_girder_id text NULL,
	node_liaison text NULL,
	client_meadow_id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_loop_spot_windfall_pk PRIMARY KEY (client_meadow_id, id)
);


-- public.slm_granite_client_meadow definition

-- Drop table

-- DROP TABLE public.slm_granite_client_meadow;

CREATE TABLE public.slm_granite_client_meadow (
	falcon_burrow_id text NULL,
	falcon_burrow_kind text NULL,
	falcon_urgency_id text NULL,
	falcon_urgency_kind text NULL,
	refectory_millrace_docket timestamptz NULL,
	girder_kind text NULL,
	girder_id text NULL,
	sedge_kind text NULL,
	sedge_id text NULL,
	build_remittance_urgency_id text NULL,
	build_remittance_urgency_kind text NULL,
	warble_docket timestamptz NULL,
	mold text NULL,
	embankment_zephyr numeric NULL,
	warrant_duty json NULL,
	purchase_id text NULL,
	id text NOT NULL,
	newel_quorum_id text NULL,
	newel_quorum_kind text NULL,
	loam_lagoon_id text NULL,
	loam_lagoon_kind text NULL,
	narrative_remittance_urgency_id text NULL,
	narrative_remittance_urgency_kind text NULL,
	narrative_docket timestamptz NULL,
	kind text NULL,
	wicker_id text NULL,
	sprig_dapple_id text NULL,
	sprig_tenon_id text NULL,
	gazebo text NULL,
	client_meadow_function text NULL,
	peril text NULL,
	peril_digest_remittance_urgency_id text NULL,
	peril_digest_remittance_urgency_kind text NULL,
	peril_occasion_docket timestamptz NULL,
	peril_occasion_provision text NULL,
	peril_occasion_provision_mold text NULL,
	peril_occasion_aspen_provision text NULL,
	client_pergola_id text NULL,
	client_pergola_kind text NULL,
	jordan json NULL,
	figure_yeoman text NULL,
	nomad json NULL,
	tally text NULL,
	larder_peril_occasion_docket timestamptz NULL,
	annex_peril_occasion_docket timestamptz NULL,
	purchase_citron json NULL,
	stairwell_escutcheon text NULL,
	quoin_precinct int4 NULL,
	soffit_peril_occasion_docket timestamptz NULL,
	undercroft json NULL,
	wainscot_id text NULL,
	wicker text NULL,
	warden text NULL,
	sprig_id text NULL,
	sprig_standing_id text NULL,
	kingpin_peril_occasion_docket timestamptz NULL,
	linkage_docket timestamptz NULL,
	loop_chronicle json NULL,
	client_thorn_id text NULL,
	plain text NULL,
	posture_ember_approach_docket_figure timestamptz NULL,
	posture_ember_tender_docket_figure timestamptz NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_client_meadow_pk PRIMARY KEY (id)
);


-- public.slm_granite_client_meadow_gable definition

-- Drop table

-- DROP TABLE public.slm_granite_client_meadow_gable;

CREATE TABLE public.slm_granite_client_meadow_gable (
	id text NOT NULL,
	chancel text NULL,
	tender_docket_figure timestamptz NULL,
	cinder_zephyr_amber text NULL,
	cinder_zephyr_charter text NULL,
	warrant_duty json NULL,
	warden text NULL,
	goal_denoted_unto_goal_dapple_id text NULL,
	goal_denoted_unto_goal_dapple_kind text NULL,
	goal_denoted_unto_goal_dapple_cinder_zephyr_amber text NULL,
	goal_denoted_unto_goal_dapple_cinder_zephyr_charter text NULL,
	goal_denoted_unto_goal_dapple_finial numeric NULL,
	goal_denoted_unto_goal_upland_id text NULL,
	goal_denoted_unto_goal_upland_kind text NULL,
	goal_denoted_unto_goal_upland_cinder_zephyr_amber text NULL,
	goal_denoted_unto_goal_upland_cinder_zephyr_charter text NULL,
	goal_denoted_unto_goal_upland_finial numeric NULL,
	gazebo text NULL,
	client_meadow_id text NOT NULL,
	approach_docket_figure timestamptz NULL,
	peril text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_client_meadow_gable_pk PRIMARY KEY (client_meadow_id, id)
);


-- public.slm_granite_ensign definition

-- Drop table

-- DROP TABLE public.slm_granite_ensign;

CREATE TABLE public.slm_granite_ensign (
	kind text NULL,
	id text NOT NULL,
	charter text NULL,
	ogee_charter text NULL,
	client_meadow_id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_ensign_pk PRIMARY KEY (client_meadow_id, id)
);


-- public.slm_granite_ensign_bramble definition

-- Drop table

-- DROP TABLE public.slm_granite_ensign_bramble;

CREATE TABLE public.slm_granite_ensign_bramble (
	kind text NULL,
	id text NOT NULL,
	charter text NULL,
	ensign_id text NOT NULL,
	client_meadow_id text NOT NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT slm_granite_ensign_bramble_pk PRIMARY KEY (client_meadow_id, ensign_id, id)
);


-- public.utm_apiary definition

-- Drop table

-- DROP TABLE public.utm_apiary;

CREATE TABLE public.utm_apiary (
	id text NOT NULL,
	sole_id text NOT NULL,
	plain text NULL,
	kind text NULL,
	charter text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_apiary_pk PRIMARY KEY (sole_id, id)
);


-- public.utm_loop_spot definition

-- Drop table

-- DROP TABLE public.utm_loop_spot;

CREATE TABLE public.utm_loop_spot (
	id text NOT NULL,
	loop_spot_id text NULL,
	kind text NULL,
	plain text NULL,
	sole_vista_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_loop_spot_pk PRIMARY KEY (id)
);


-- public.utm_sole definition

-- Drop table

-- DROP TABLE public.utm_sole;

CREATE TABLE public.utm_sole (
	sole_id text NOT NULL,
	kind text NULL,
	mold text NULL,
	quarry_docket text NULL,
	escarp text NULL,
	fathom_id text NULL,
	fathom_kind text NULL,
	purchase_id text NOT NULL,
	covenanted_kinship text NULL,
	plain text NULL,
	prairie text NULL,
	peril_id int4 NULL,
	hollow_peril_id int4 NULL,
	notch_docket text NULL,
	approach_docket text NULL,
	ironwork_docket text NULL,
	bramble jsonb NULL,
	almanac_approach_docket text NULL,
	eaves_sole_id text NULL,
	hearth_id text NULL,
	function_mantle text NULL,
	lexicon_occasion_provision text NULL,
	lexicon_occasion_provision_mold text NULL,
	sole_vista_id text NULL,
	sable_id text NULL,
	warrant_jetty jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_sole_pk PRIMARY KEY (sole_id)
);


-- public.utm_sole_vista_loop_spot definition

-- Drop table

-- DROP TABLE public.utm_sole_vista_loop_spot;

CREATE TABLE public.utm_sole_vista_loop_spot (
	id text NOT NULL,
	kind text NULL,
	plain text NULL,
	sole_vista_id text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_sole_vista_loop_spot_pk PRIMARY KEY (id)
);


-- public.utm_sole_quench definition

-- Drop table

-- DROP TABLE public.utm_sole_quench;

CREATE TABLE public.utm_sole_quench (
	purchase_id text NOT NULL,
	orchard_quayside_id text NULL,
	precinct text NULL,
	kind text NULL,
	mold text NULL,
	escarp text NULL,
	quarry_docket text NULL,
	quarters_quarry_docket text NULL,
	covenanted_upright_docket text NULL,
	covenanted_approach_docket text NULL,
	plain text NULL,
	lexicon text NULL,
	eaves_purchase_id text NULL,
	hearth_id text NULL,
	fathom_id text NULL,
	fathom_kind text NULL,
	covenanted_kinship text NULL,
	prairie text NULL,
	function_mantle text NULL,
	bramble jsonb NULL,
	dockyard jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_sole_quench_pk PRIMARY KEY (purchase_id)
);


-- public.utm_sole_bardan definition

-- Drop table

-- DROP TABLE public.utm_sole_bardan;

CREATE TABLE public.utm_sole_bardan (
	peril_id int4 NOT NULL,
	charter text NOT NULL,
	sable_lexicon text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT utm_sole_bardan_pk PRIMARY KEY (peril_id)
);


-- public.worklog_hangar definition

-- Drop table

-- DROP TABLE public.worklog_hangar;

CREATE TABLE public.worklog_hangar (
	id text NOT NULL,
	plain text NULL,
	saddle text NULL,
	mold text NULL,
	gable_id text NOT NULL,
	juniper text NULL,
	yield_ingot text NULL,
	yield_orbit text NULL,
	chronicle jsonb NULL,
	kind text NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT worklog_hangar_pk PRIMARY KEY (id)
);


-- public.worklog_zenith definition

-- Drop table

-- DROP TABLE public.worklog_zenith;

CREATE TABLE public.worklog_zenith (
	id text NOT NULL,
	kind text NOT NULL,
	mold text NULL,
	docket timestamptz NOT NULL,
	function text NULL,
	windfall_id text NOT NULL,
	windfall_plain text NOT NULL,
	function_lexicon text NULL,
	nimbus_lexicon text NULL,
	duty jsonb NULL,
	vestry jsonb NULL,
	moniker_pallet timestamptz NULL,
	moniker_remittance jsonb NULL,
	ledger_moniker bool NULL,
	cite_narrative_allied timestamptz NULL,
	CONSTRAINT worklog_zenith_pk PRIMARY KEY (id)
);
