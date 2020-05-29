create table entity_edit
(
    id          int auto_increment          primary key,
    entity_id   int                         not null,
    content     varchar(32) default ''      not null,
    modified    datetime    default now()   not null,
    user_id     int                         not null,
    origin      datetime    default now()   not null
);