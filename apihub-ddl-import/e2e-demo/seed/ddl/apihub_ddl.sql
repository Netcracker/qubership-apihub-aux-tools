create sequence activity_tracking_transition_completed_seq
    as integer
    minvalue 0;

alter sequence activity_tracking_transition_completed_seq owner to apihub;

create table schema_migrations
(
    version integer not null
        primary key,
    dirty   boolean not null
);

alter table schema_migrations
    owner to apihub;

create table stored_schema_migration
(
    num       integer not null
        primary key,
    up_hash   varchar not null,
    sql_up    varchar not null,
    down_hash varchar,
    sql_down  varchar
);

alter table stored_schema_migration
    owner to apihub;

create table package_group
(
    id                       varchar                                     not null
        constraint "PK_project_group"
            primary key,
    kind                     varchar,
    name                     varchar,
    alias                    varchar,
    parent_id                varchar
        constraint "FK_parent_package_group"
            references package_group
            on update cascade on delete cascade,
    description              text,
    deleted_at               timestamp,
    created_at               timestamp,
    created_by               varchar,
    deleted_by               varchar,
    default_role             varchar default 'Viewer'::character varying not null,
    default_released_version varchar,
    service_name             varchar,
    release_version_pattern  varchar,
    exclude_from_search      boolean default false,
    rest_grouping_prefix     varchar
);

alter table package_group
    owner to apihub;

create index package_group_name_idx
    on package_group (name, id)
    where (deleted_at IS NULL);

create index package_group_id_pattern_idx
    on package_group (id varchar_pattern_ops);

create table activity_tracking
(
    id         varchar not null
        primary key,
    e_type     varchar not null,
    data       jsonb,
    package_id varchar
        constraint activity_tracking_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    date       timestamp,
    user_id    varchar
);

alter table activity_tracking
    owner to apihub;

create index ix_activity_tracking_package_e_type
    on activity_tracking (package_id, e_type);

create index activity_tracking_date_idx
    on activity_tracking (date desc, id desc);

create index activity_tracking_package_id_date_idx
    on activity_tracking (package_id varchar_pattern_ops asc, date desc, id desc);

create table activity_tracking_transition
(
    id                      varchar   not null
        constraint activity_tracking_transition_pk
            primary key,
    tr_type                 varchar   not null,
    from_id                 varchar   not null,
    to_id                   varchar   not null,
    status                  varchar   not null,
    details                 varchar,
    started_by              varchar   not null,
    started_at              timestamp not null,
    finished_at             timestamp,
    progress_percent        integer,
    affected_objects        integer,
    completed_serial_number integer
);

alter table activity_tracking_transition
    owner to apihub;

alter sequence activity_tracking_transition_completed_seq owned by activity_tracking_transition.completed_serial_number;

create unique index activity_tracking_transition_id_uindex
    on activity_tracking_transition (id);

create table apihub_api_keys
(
    id          varchar   not null
        primary key,
    package_id  varchar   not null,
    name        varchar   not null,
    created_by  varchar   not null,
    created_at  timestamp not null,
    deleted_by  varchar,
    deleted_at  timestamp,
    api_key     varchar   not null,
    roles       character varying[] default '{}'::character varying[],
    created_for varchar
);

alter table apihub_api_keys
    owner to apihub;

create table build
(
    build_id      varchar                 not null
        constraint "PK_build"
            primary key,
    status        varchar                 not null,
    details       varchar,
    package_id    varchar                 not null
        constraint build_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    version       varchar                 not null,
    created_at    timestamp default now() not null,
    last_active   timestamp default now() not null,
    created_by    varchar                 not null,
    restart_count integer,
    client_build  boolean,
    started_at    timestamp,
    builder_id    varchar,
    priority      integer   default 0     not null,
    metadata      jsonb
);

alter table build
    owner to apihub;

create index build_status_index
    on build (status);

create index ix_build_id_text
    on build ((build_id::text));

create index idx_build_migration_id_build_type
    on build ((metadata ->> 'migration_id'::text), (metadata ->> 'build_type'::text));

create table build_cleanup_run
(
    run_id                   integer not null
        primary key,
    scheduled_at             timestamp,
    deleted_rows             integer,
    build_result             integer default 0,
    build_src                integer default 0,
    expired_s3_files_count   integer default 0,
    expired_s3_files_details text    default ''::text
);

alter table build_cleanup_run
    owner to apihub;

create table build_depends
(
    build_id  varchar not null
        constraint "FK_build_depends_id"
            references build
            on update cascade on delete cascade,
    depend_id varchar not null
        constraint "FK_build_depends_depend"
            references build
            on update cascade on delete cascade
);

alter table build_depends
    owner to apihub;

create index build_depends_index
    on build_depends (depend_id);

create table build_result
(
    build_id varchar not null
        constraint "FK_build_result_build_id"
            references build
            on update cascade on delete cascade,
    data     bytea   not null
);

alter table build_result
    owner to apihub;

create table build_src
(
    build_id varchar not null
        constraint "FK_build_src"
            references build
            on update cascade on delete cascade,
    source   bytea,
    config   jsonb   not null
);

alter table build_src
    owner to apihub;

create table builder_notifications
(
    build_id varchar not null
        references build
            on update cascade on delete cascade,
    severity varchar,
    message  varchar,
    file_id  varchar
);

alter table builder_notifications
    owner to apihub;

create table business_metric
(
    year    integer                                      not null,
    month   integer                                      not null,
    day     integer                                      not null,
    metric  varchar                                      not null,
    data    jsonb,
    user_id varchar default 'unknown'::character varying not null,
    primary key (year, month, day, metric, user_id)
);

alter table business_metric
    owner to apihub;

create table endpoint_calls
(
    path    varchar not null,
    hash    varchar not null,
    options jsonb,
    count   integer,
    primary key (path, hash)
);

alter table endpoint_calls
    owner to apihub;

create table user_data
(
    user_id            varchar                               not null
        constraint "PK_user_data"
            primary key,
    email              varchar
        constraint email_unique
            unique,
    name               varchar,
    avatar_url         varchar,
    password           bytea,
    private_package_id varchar default ''::character varying not null
        constraint private_package_id_unique
            unique
);

alter table user_data
    owner to apihub;

create table external_identity
(
    provider    varchar                               not null,
    external_id varchar                               not null,
    internal_id varchar                               not null
        constraint "FK_user_data"
            references user_data
            on update cascade on delete cascade,
    provider_id varchar default ''::character varying not null,
    primary key (provider, provider_id, external_id)
);

alter table external_identity
    owner to apihub;

create table favorite_packages
(
    user_id    varchar not null
        constraint "FK_favorite_packages_user_data"
            references user_data
            on delete cascade,
    package_id varchar not null
        constraint "FK_favorite_packages_package_group"
            references package_group
            on delete cascade,
    constraint "PK_favorite_packages"
        primary key (user_id, package_id)
);

alter table favorite_packages
    owner to apihub;

create table migrated_version_changes
(
    package_id     varchar not null,
    version        varchar not null,
    revision       varchar not null,
    build_id       varchar not null,
    migration_id   varchar not null,
    changes        jsonb,
    unique_changes character varying[]
);

alter table migrated_version_changes
    owner to apihub;

create index migrated_version_changes_build_id
    on migrated_version_changes (build_id);

create table migration_changes
(
    migration_id varchar not null
        primary key,
    changes      jsonb
);

alter table migration_changes
    owner to apihub;

create table migration_run
(
    id                        varchar,
    started_at                timestamp,
    status                    varchar,
    stage                     varchar,
    package_ids               character varying[],
    versions                  character varying[],
    is_rebuild                boolean,
    is_rebuild_changelog_only boolean,
    current_builder_version   varchar,
    error_details             varchar,
    finished_at               timestamp,
    updated_at                timestamp,
    skip_validation           boolean,
    instance_id               varchar,
    sequence_number           serial
        constraint migration_run_seq
            unique,
    post_check_result         jsonb,
    retry_count               integer default 0,
    stages_execution          jsonb
);

alter table migration_run
    owner to apihub;

create table operation_data
(
    data_hash varchar not null
        constraint pk_operation_data
            primary key,
    data      bytea
);

alter table operation_data
    owner to apihub;

create table published_version
(
    package_id                  varchar   not null
        constraint "FK_package_group"
            references package_group
            on update cascade on delete cascade,
    version                     varchar   not null,
    revision                    integer   not null,
    status                      varchar   not null,
    published_at                timestamp not null,
    deleted_at                  timestamp,
    metadata                    jsonb,
    previous_version            varchar,
    previous_version_package_id varchar,
    labels                      character varying[],
    created_by                  varchar,
    deleted_by                  varchar,
    constraint "PK_published_version"
        primary key (package_id, version, revision)
);

comment on column published_version.status is 'DRAFT / APPROVED / RELEASED / ARCHIVE';

alter table published_version
    owner to apihub;

create table operation
(
    package_id                   varchar not null,
    version                      varchar not null,
    revision                     integer not null,
    operation_id                 varchar not null,
    data_hash                    varchar
        constraint "FK_operation_data"
            references operation_data
            on update cascade on delete cascade,
    deprecated                   boolean not null,
    kind                         varchar,
    title                        varchar,
    metadata                     jsonb,
    type                         varchar not null,
    deprecated_info              varchar,
    deprecated_items             jsonb,
    previous_release_versions    character varying[],
    models                       jsonb,
    custom_tags                  jsonb,
    api_audience                 varchar default 'external'::character varying,
    document_id                  varchar,
    version_internal_document_id varchar,
    constraint pk_operation
        primary key (package_id, version, revision, operation_id),
    constraint "FK_published_version"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table operation
    owner to apihub;

create index operation_data_hash_idx
    on operation (data_hash);

create index ix_operation_pvrt
    on operation (package_id, version, revision, type);

create table version_comparison
(
    package_id          varchar   not null,
    version             varchar   not null,
    revision            integer   not null,
    previous_package_id varchar   not null,
    previous_version    varchar   not null,
    previous_revision   integer   not null,
    comparison_id       varchar   not null
        unique,
    operation_types     jsonb,
    refs                character varying[],
    open_count          bigint    not null,
    last_active         timestamp not null,
    no_content          boolean   not null,
    builder_version     varchar,
    metadata            jsonb,
    contract_types      jsonb,
    primary key (package_id, version, revision, previous_package_id, previous_version, previous_revision)
);

alter table version_comparison
    owner to apihub;

create table operation_comparison
(
    package_id                      varchar not null,
    version                         varchar not null,
    revision                        integer not null,
    previous_package_id             varchar not null,
    previous_version                varchar not null,
    previous_revision               integer not null,
    operation_id                    varchar,
    data_hash                       varchar,
    previous_data_hash              varchar,
    changes_summary                 jsonb,
    changes                         jsonb,
    comparison_id                   varchar
        constraint "FK_version_comparison"
            references version_comparison (comparison_id)
            on update cascade on delete cascade,
    previous_operation_id           varchar,
    comparison_internal_document_id varchar
);

alter table operation_comparison
    owner to apihub;

create index operation_comparison_comparison_id_index
    on operation_comparison (comparison_id);

create table operation_group
(
    package_id        varchar not null,
    version           varchar not null,
    revision          integer not null,
    api_type          varchar not null,
    group_name        varchar not null,
    autogenerated     boolean not null,
    group_id          varchar not null
        unique,
    description       varchar,
    template_checksum varchar,
    template_filename varchar,
    primary key (package_id, version, revision, api_type, group_name),
    constraint "FK_published_version"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table operation_group
    owner to apihub;

create table grouped_operation
(
    group_id     varchar not null
        constraint "FK_operation_group"
            references operation_group (group_id)
            on update cascade on delete cascade,
    package_id   varchar not null,
    version      varchar not null,
    revision     integer not null,
    operation_id varchar not null,
    constraint "FK_operation"
        foreign key (package_id, version, revision, operation_id) references operation
            on update cascade on delete cascade
);

alter table grouped_operation
    owner to apihub;

create index grouped_operation_idx
    on grouped_operation (package_id, version, revision, operation_id);

create index grouped_operation_group_id
    on grouped_operation (group_id);

create table operation_group_history
(
    group_id  varchar,
    action    varchar,
    data      jsonb,
    user_id   varchar,
    date      timestamp,
    automatic boolean
);

alter table operation_group_history
    owner to apihub;

create table operation_group_publication
(
    publish_id varchar not null
        primary key,
    status     varchar,
    details    varchar
);

alter table operation_group_publication
    owner to apihub;

create table operation_group_template
(
    checksum varchar not null
        primary key,
    template bytea
);

alter table operation_group_template
    owner to apihub;

create table operation_open_count
(
    package_id   varchar not null
        constraint operation_open_count_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    version      varchar not null,
    operation_id varchar not null,
    open_count   bigint,
    primary key (package_id, version, operation_id)
);

alter table operation_open_count
    owner to apihub;

create table package_member_role
(
    user_id    varchar   not null,
    package_id varchar   not null
        constraint "FK_package_group"
            references package_group
            on update cascade on delete cascade,
    created_by varchar   not null,
    created_at timestamp not null,
    updated_by varchar,
    updated_at timestamp,
    roles      character varying[] default '{}'::character varying[],
    constraint "PK_package_member_role"
        primary key (package_id, user_id)
);

alter table package_member_role
    owner to apihub;

create index package_member_role_user_id_idx
    on package_member_role (user_id, package_id);

create table package_service
(
    package_id   varchar not null
        constraint "FK_package_group"
            references package_group
            on update cascade on delete cascade,
    service_name varchar not null,
    workspace_id varchar not null
        constraint "FK_package_group_workspace"
            references package_group
            on update cascade on delete cascade,
    constraint "PK_package_service"
        primary key (workspace_id, package_id, service_name),
    unique (workspace_id, service_name)
);

alter table package_service
    owner to apihub;

create table package_transition
(
    old_package_id varchar not null,
    new_package_id varchar not null
);

alter table package_transition
    owner to apihub;

create index package_transition_old_package_id_index
    on package_transition (old_package_id);

create table published_data
(
    package_id varchar not null
        constraint published_data_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    checksum   varchar not null,
    media_type varchar not null,
    data       bytea   not null,
    constraint "PK_published_data"
        primary key (checksum, package_id)
);

comment on column published_data.media_type is 'HTTP media-type';

alter table published_data
    owner to apihub;

create table published_document_open_count
(
    package_id varchar not null
        constraint published_document_open_count_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    version    varchar not null,
    slug       varchar not null,
    open_count bigint,
    primary key (package_id, version, slug)
);

alter table published_document_open_count
    owner to apihub;

create table published_sources
(
    package_id       varchar not null,
    version          varchar not null,
    revision         integer not null,
    config           bytea,
    metadata         bytea,
    archive_checksum varchar,
    constraint "FK_published_sources_version_revision"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table published_sources
    owner to apihub;

create unique index published_sources_package_id_version_revision_uindex
    on published_sources (package_id, version, revision);

create table published_sources_archives
(
    checksum varchar not null
        constraint published_sources_archives_pk
            primary key,
    data     bytea
);

alter table published_sources_archives
    owner to apihub;

create table published_version_open_count
(
    package_id varchar not null
        constraint published_version_open_count_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    version    varchar not null,
    open_count bigint,
    primary key (package_id, version)
);

alter table published_version_open_count
    owner to apihub;

create table published_version_reference
(
    package_id                varchar                               not null,
    version                   varchar                               not null,
    revision                  integer                               not null,
    reference_id              varchar                               not null,
    reference_version         varchar                               not null,
    reference_revision        integer default 0                     not null,
    parent_reference_id       varchar default ''::character varying not null,
    parent_reference_version  varchar default ''::character varying not null,
    parent_reference_revision integer default 0                     not null,
    excluded                  boolean default false,
    constraint "PK_published_version_reference"
        primary key (package_id, version, revision, reference_id, reference_version, reference_revision,
                     parent_reference_id, parent_reference_version, parent_reference_revision),
    constraint "FK_published_version"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table published_version_reference
    owner to apihub;

create table published_version_revision_content
(
    package_id          varchar                                      not null,
    version             varchar                                      not null,
    revision            integer                                      not null,
    checksum            varchar                                      not null,
    index               integer default 0                            not null,
    file_id             varchar                                      not null,
    path                varchar,
    slug                varchar                                      not null,
    data_type           varchar                                      not null,
    name                varchar                                      not null,
    metadata            jsonb,
    title               varchar,
    format              varchar,
    operation_ids       character varying[],
    filename            varchar,
    shareability_status varchar default 'unknown'::character varying not null,
    api_kind            varchar,
    constraint published_version_revision_content_pk
        primary key (package_id, version, revision, file_id),
    constraint "FK_published_version_revision"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade,
    constraint published_version_revision_content_published_data_fk
        foreign key (checksum, package_id) references published_data
            on update cascade
);

comment on column published_version_revision_content.data_type is 'OpenAPI / Swagger / MD';

alter table published_version_revision_content
    owner to apihub;

create index pvrc_checksum_idx
    on published_version_revision_content (checksum);

create table published_version_validation
(
    package_id varchar not null,
    version    varchar not null,
    revision   integer not null,
    changelog  jsonb,
    spectral   jsonb,
    bwc        jsonb,
    constraint "PK_published_version_validation"
        primary key (package_id, version, revision),
    constraint "FK_published_version_validation"
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table published_version_validation
    owner to apihub;

create table role
(
    id          varchar not null
        primary key,
    role        varchar not null,
    rank        integer not null,
    permissions character varying[],
    read_only   boolean
);

alter table role
    owner to apihub;

create table shared_url_info
(
    package_id varchar not null
        constraint shared_url_info_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    version    varchar not null,
    file_id    varchar not null,
    shared_id  varchar not null
        constraint "PK_shared_url_info"
            primary key,
    constraint shared_url_info__file_info
        unique (package_id, version, file_id)
);

alter table shared_url_info
    owner to apihub;

create table system_role
(
    user_id varchar not null
        constraint "PK_system_role"
            primary key,
    role    varchar not null
);

alter table system_role
    owner to apihub;

create table transformed_content_data
(
    package_id     varchar                                            not null,
    version        varchar                                            not null,
    revision       integer                                            not null,
    api_type       varchar                                            not null,
    group_id       varchar                                            not null
        constraint "FK_transformed_content_data_operation_group"
            references operation_group (group_id)
            on update cascade on delete cascade,
    data           bytea,
    documents_info jsonb,
    build_type     varchar default 'documentGroup'::character varying not null,
    format         varchar default 'json'::character varying          not null,
    primary key (package_id, version, revision, api_type, group_id, build_type, format)
);

alter table transformed_content_data
    owner to apihub;

create table user_avatar_data
(
    user_id  varchar not null
        constraint "PK_user_avatar_data"
            primary key,
    avatar   bytea,
    checksum bytea
);

alter table user_avatar_data
    owner to apihub;

create table versions_cleanup_run
(
    run_id        uuid                    not null
        constraint pk_versions_cleanup_run
            primary key,
    started_at    timestamp default now() not null,
    package_id    varchar,
    delete_before timestamp               not null,
    status        varchar                 not null,
    details       varchar,
    deleted_items integer,
    instance_id   uuid,
    finished_at   timestamp
);

alter table versions_cleanup_run
    owner to apihub;

create table csv_dashboard_publication
(
    publish_id varchar not null
        primary key,
    status     varchar,
    message    varchar,
    csv_report bytea
);

alter table csv_dashboard_publication
    owner to apihub;

create table personal_access_tokens
(
    id         varchar                 not null
        constraint personal_access_tokens_pk
            primary key,
    user_id    varchar
        constraint personal_access_tokens_user_fk
            references user_data,
    token_hash varchar,
    name       varchar                 not null,
    created_at timestamp default now() not null,
    expires_at timestamp,
    deleted_at timestamp
);

alter table personal_access_tokens
    owner to apihub;

create unique index personal_access_tokens_hash_index
    on personal_access_tokens (token_hash);

create table package_export_config
(
    package_id             varchar
        constraint package_export_config_package_group_id_fk
            references package_group
            on update cascade on delete cascade,
    allowed_oas_extensions character varying[] not null
);

alter table package_export_config
    owner to apihub;

create unique index package_export_config_package_id_uindex
    on package_export_config (package_id);

create table export_result
(
    export_id  varchar   not null
        constraint export_result_pk
            primary key,
    created_at timestamp not null,
    created_by varchar   not null,
    config     json      not null,
    filename   varchar   not null,
    data       bytea     not null
);

alter table export_result
    owner to apihub;

create table locks
(
    name        varchar          not null
        primary key,
    instance_id varchar          not null,
    acquired_at timestamp        not null,
    expires_at  timestamp        not null,
    version     bigint default 1 not null
);

alter table locks
    owner to apihub;

create table comparisons_cleanup_run
(
    run_id        uuid      not null
        primary key,
    instance_id   uuid      not null,
    started_at    timestamp not null,
    status        varchar   not null,
    details       varchar,
    delete_before timestamp not null,
    deleted_items integer,
    finished_at   timestamp
);

alter table comparisons_cleanup_run
    owner to apihub;

create table soft_deleted_data_cleanup_run
(
    run_id        uuid                    not null
        primary key,
    instance_id   uuid                    not null,
    started_at    timestamp default now() not null,
    finished_at   timestamp,
    status        varchar                 not null,
    details       varchar,
    delete_before timestamp               not null,
    deleted_items jsonb
);

alter table soft_deleted_data_cleanup_run
    owner to apihub;

create table unreferenced_data_cleanup_run
(
    run_id        uuid                    not null
        primary key,
    instance_id   uuid                    not null,
    started_at    timestamp default now() not null,
    finished_at   timestamp,
    status        varchar                 not null,
    details       varchar,
    deleted_items jsonb
);

alter table unreferenced_data_cleanup_run
    owner to apihub;

create table version_internal_document_data
(
    hash varchar not null
        primary key,
    data bytea
);

alter table version_internal_document_data
    owner to apihub;

create table version_internal_document
(
    package_id  varchar not null,
    version     varchar not null,
    revision    integer not null,
    document_id varchar not null,
    filename    varchar,
    hash        varchar
        constraint version_internal_document_data_fk
            references version_internal_document_data,
    primary key (package_id, version, revision, document_id),
    constraint version_internal_document_published_version_fk
        foreign key (package_id, version, revision) references published_version
            on update cascade on delete cascade
);

alter table version_internal_document
    owner to apihub;

create index version_internal_document_hash_idx
    on version_internal_document (hash);

create table comparison_internal_document_data
(
    hash varchar not null
        primary key,
    data bytea
);

alter table comparison_internal_document_data
    owner to apihub;

create table comparison_internal_document
(
    package_id          varchar not null,
    version             varchar not null,
    revision            integer not null,
    previous_package_id varchar not null,
    previous_version    varchar not null,
    previous_revision   integer not null,
    document_id         varchar not null,
    filename            varchar,
    hash                varchar
        constraint comparison_internal_document_data_fk
            references comparison_internal_document_data,
    primary key (package_id, version, revision, previous_package_id, previous_version, previous_revision, document_id),
    constraint comparison_internal_document_version_comparison_fk
        foreign key (package_id, version, revision, previous_package_id, previous_version,
                     previous_revision) references version_comparison
            on update cascade on delete cascade
);

alter table comparison_internal_document
    owner to apihub;

create index comparison_internal_document_hash_idx
    on comparison_internal_document (hash);

create table sources_update_tracking
(
    id           varchar   not null
        constraint sources_update_tracking_pk
            primary key,
    package_id   varchar   not null,
    version      varchar   not null,
    revision     integer   not null,
    old_checksum varchar   not null,
    new_checksum varchar   not null,
    performed_by varchar   not null,
    performed_at timestamp not null
);

alter table sources_update_tracking
    owner to apihub;

create table fts_operation_search_text
(
    package_id       varchar not null,
    version          varchar not null,
    revision         integer not null,
    operation_id     varchar not null,
    status           varchar not null,
    api_type         varchar not null,
    search_data_hash varchar,
    data_vector      tsvector,
    constraint pk_fts_operation_search_text
        primary key (package_id, version, revision, operation_id)
);

alter table fts_operation_search_text
    owner to apihub;

create index fts_operation_search_text_data_vector_idx
    on fts_operation_search_text using gin (data_vector);

create index fts_operation_search_text_scope_idx
    on fts_operation_search_text (status, api_type, package_id varchar_pattern_ops);

create table ai_chat
(
    id                         uuid                     not null
        primary key,
    user_id                    varchar                  not null
        constraint ai_chat_user_fk
            references user_data
            on delete cascade,
    title                      text    default ''::text not null,
    pinned                     boolean default false    not null,
    created_at                 timestamp                not null,
    last_message_at            timestamp                not null,
    messages_count             integer default 0        not null,
    compacted_up_to_created_at timestamp,
    compaction_summary         text,
    last_turn_tokens           integer
);

alter table ai_chat
    owner to apihub;

create index ai_chat_user_sort_idx
    on ai_chat (user_id asc, pinned desc, last_message_at desc);

create index ai_chat_retention_idx
    on ai_chat (user_id, pinned, last_message_at);

create table ai_chat_message
(
    id                uuid      not null
        primary key,
    chat_id           uuid      not null
        constraint ai_chat_message_chat_fk
            references ai_chat
            on delete cascade,
    role              varchar   not null,
    content           text      not null,
    client_message_id uuid,
    tool_invocations  jsonb,
    created_at        timestamp not null
);

alter table ai_chat_message
    owner to apihub;

create index ai_chat_message_chat_time_idx
    on ai_chat_message (chat_id asc, created_at desc);

create unique index ai_chat_message_client_id_idx
    on ai_chat_message (chat_id, client_message_id)
    where (client_message_id IS NOT NULL);

create table ephemeral_file
(
    id           uuid      not null
        primary key,
    user_id      varchar   not null,
    filename     text      not null,
    storage_path text      not null,
    mime_type    varchar,
    size_bytes   bigint,
    created_at   timestamp not null,
    expires_at   timestamp not null
);

alter table ephemeral_file
    owner to apihub;

create index ephemeral_file_expires_idx
    on ephemeral_file (expires_at);

create table ddl_tables
(
    package_id                   varchar not null,
    version                      varchar not null,
    revision                     integer not null,
    ddl_entity_id                varchar not null,
    kind                         varchar not null
        constraint ddl_tables_kind_check
            check ((kind)::text = ANY ((ARRAY ['table'::character varying, 'view'::character varying])::text[])),
    schema_name                  varchar,
    name                         varchar,
    description                  varchar,
    metadata                     jsonb,
    data_hash                    varchar,
    document_id                  varchar,
    version_internal_document_id varchar,
    constraint pk_ddl_tables
        primary key (package_id, version, revision, ddl_entity_id)
);

alter table ddl_tables
    owner to apihub;

create index ddl_tables_kind_idx
    on ddl_tables (package_id, version, revision, kind);

create index ddl_tables_document_id_idx
    on ddl_tables (document_id);

create table ddl_table_data
(
    data_hash varchar not null
        primary key,
    data      bytea
);

alter table ddl_table_data
    owner to apihub;

create table ddl_comparison
(
    package_id                      varchar not null,
    version                         varchar not null,
    revision                        integer not null,
    previous_package_id             varchar not null,
    previous_version                varchar not null,
    previous_revision               integer not null,
    ddl_entity_id                   varchar not null,
    previous_ddl_entity_id          varchar not null,
    comparison_id                   varchar
        constraint ddl_comparison_version_comparison_comparison_id_fk
            references version_comparison (comparison_id)
            on update cascade on delete cascade,
    data_hash                       varchar,
    previous_data_hash              varchar,
    kind                            varchar,
    previous_kind                   varchar,
    name                            varchar,
    previous_name                   varchar,
    schema_name                     varchar,
    previous_schema_name            varchar,
    description                     varchar,
    previous_description            varchar,
    changes_summary                 jsonb,
    changes                         jsonb,
    comparison_internal_document_id varchar,
    constraint pk_ddl_comparison
        primary key (package_id, version, revision, previous_package_id, previous_version, previous_revision,
                     ddl_entity_id, previous_ddl_entity_id)
);

alter table ddl_comparison
    owner to apihub;

create index ddl_comparison_comparison_id_idx
    on ddl_comparison (comparison_id);

create table fts_ddl_search_text
(
    package_id       varchar not null,
    version          varchar not null,
    revision         integer not null,
    ddl_entity_id    varchar not null,
    status           varchar not null,
    kind             varchar not null,
    search_data_hash varchar,
    data_vector      tsvector,
    constraint pk_fts_ddl_search_text
        primary key (package_id, version, revision, ddl_entity_id)
);

alter table fts_ddl_search_text
    owner to apihub;

create index fts_ddl_search_text_data_vector_idx
    on fts_ddl_search_text using gin (data_vector);

create table mcp_entities
(
    package_id                   varchar not null,
    version                      varchar not null,
    revision                     integer not null,
    mcp_entity_id                varchar not null,
    kind                         varchar not null
        constraint mcp_entities_kind_check
            check ((kind)::text = ANY
                   ((ARRAY ['init'::character varying, 'tool'::character varying, 'prompt'::character varying, 'resource'::character varying])::text[])),
    title                        varchar,
    description                  varchar,
    mcp_endpoint                 varchar not null,
    metadata                     jsonb,
    data_hash                    varchar,
    document_id                  varchar,
    version_internal_document_id varchar,
    constraint pk_mcp_entities
        primary key (package_id, version, revision, mcp_entity_id)
);

alter table mcp_entities
    owner to apihub;

create index mcp_entities_kind_idx
    on mcp_entities (package_id, version, revision, kind);

create index mcp_entities_document_id_idx
    on mcp_entities (document_id);

create table mcp_entity_data
(
    data_hash varchar not null
        primary key,
    data      bytea
);

alter table mcp_entity_data
    owner to apihub;

create table fts_mcp_search_text
(
    package_id       varchar not null,
    version          varchar not null,
    revision         integer not null,
    mcp_entity_id    varchar not null,
    status           varchar not null,
    kind             varchar not null,
    search_data_hash varchar,
    data_vector      tsvector,
    constraint pk_fts_mcp_search_text
        primary key (package_id, version, revision, mcp_entity_id)
);

alter table fts_mcp_search_text
    owner to apihub;

create index fts_mcp_search_text_data_vector_idx
    on fts_mcp_search_text using gin (data_vector);

create view operation_ws1_test
            (package_id, version, revision, operation_id, data_hash, deprecated, kind, title, metadata, type,
             deprecated_info, deprecated_items, previous_release_versions, models, custom_tags, api_audience,
             document_id, version_internal_document_id)
as
SELECT package_id,
    version,
    revision,
    operation_id,
    data_hash,
    deprecated,
    kind,
    title,
    metadata,
    type,
    deprecated_info,
    deprecated_items,
    previous_release_versions,
    models,
    custom_tags,
    api_audience,
    document_id,
    version_internal_document_id
   FROM operation
  WHERE package_id::text = 'ws1.test'::text;

alter table operation_ws1_test
    owner to apihub;

create function get_latest_revision(package_id character varying, version character varying) returns integer
    language plpgsql
as
$$
declare
    latest_revision integer;
begin
    execute '
     select max(revision)
     from published_version
     where package_id = $1 and version = $2;'
     into latest_revision
     using package_id,version;
    if latest_revision is null then return 0;
    end if;
    return latest_revision;
end;$$;

alter function get_latest_revision(varchar, varchar) owner to apihub;

create function merge_json_path(jsonb[]) returns jsonb[]
    strict
    language plpgsql
as
$$
declare
    items alias for $1;
    jsonpath text;
    ret jsonb[];
begin
    for i in array_lower(items, 1)..array_upper(items, 1) loop
    select string_agg(el, '/') into jsonpath from jsonb_array_elements_text(items[i]->'jsonPath') el;
    ret[i] := jsonb_set(items[i], '{jsonPath}', to_jsonb(jsonpath), false);
     end loop;
    return ret;
end;
$$;

alter function merge_json_path(jsonb[]) owner to apihub;

create function parent_package_names(character varying) returns character varying[]
    language plpgsql
as
$$
declare
    split varchar[] := string_to_array($1, '.')::varchar[];
    parent_ids varchar[];
    parent_names varchar[];
begin

    if coalesce(array_length(split, 1), 0) <= 1 then
     return ARRAY[]::varchar[];
    end if;

    parent_ids = parent_ids || split[1];

    for i in 2..(array_length(split, 1) - 1)
     loop
    parent_ids = parent_ids || (parent_ids[i-1] ||'.'|| split[i])::character varying;
     end loop;

    execute '
select array_agg(name) from (
  select name from package_group
  join unnest($1) with ordinality t(id, ord) using (id) --sort by parent_ids array
  order by t.ord) n'
     into parent_names
     using parent_ids;

    return parent_names;

end;
$$;

alter function parent_package_names(varchar) owner to apihub;

create function split_json_path(jsonb[]) returns jsonb[]
    strict
    language plpgsql
as
$$
declare
    items alias for $1;
    ret jsonb[];
begin
    for i in array_lower(items, 1)..array_upper(items, 1)
     loop
    ret[i] := jsonb_set(items[i], '{jsonPath}',
    (array_to_json(string_to_array(trim(both '"' from (items[i] -> 'jsonPath')::text),
    '/')))::jsonb, false);
     end loop;
    return ret;
end;
$$;

alter function split_json_path(jsonb[]) owner to apihub;

create function package_ancestor_ids(package_id character varying) returns character varying[]
    immutable
    parallel safe
    language sql
as
$$
    select array_agg(array_to_string(parts[1:depth], '.')::character varying order by depth)
    from (select string_to_array(package_id, '.') as parts) segments,
         generate_subscripts(segments.parts, 1) as depth;
$$;

alter function package_ancestor_ids(varchar) owner to apihub;

