create table entity
(
    id       int auto_increment     primary key,
    name     varchar(255)           not null,
    folder   int                    not null,
    content  varchar(32) default '' not null,
    type     tinyint                not null,
    modified datetime    default now()    not null,
    size     int         default 0  not null,
    tree     int                    not null,
    path     varchar(2048)          not null
);

create index entity_path_index
    on entity (path);

create table comment
(
    id        int auto_increment
        primary key,
    entity_id int                                not null,
    user_id   int                                not null,
    content   varchar(255)                       null,
    modified  datetime default CURRENT_TIMESTAMP null
);

create table entity_tag
(
    entity_id int not null,
    tag_id    int not null
);

create table entity_user
(
    entity_id int not null,
    user_id   int not null
);

create table favorite
(
    id        int auto_increment
        primary key,
    entity_id int not null,
    user_id   int not null
);

create table tag
(
    id    int auto_increment
        primary key,
    name  varchar(32) not null,
    color varchar(16) not null
);

create table user
(
    id    int auto_increment
        primary key,
    email varchar(64) not null,
    name  varchar(64) not null
);



insert into entity(id, name, folder, type, tree, path) values(1, "", 0, 2, 1, "/");


INSERT INTO tag (id, name, color) VALUES (1, 'Review', '#ddaaff');
INSERT INTO tag (id, name, color) VALUES (2, 'Accepted', '#00ffbb');
INSERT INTO tag (id, name, color) VALUES (3, 'Denied', '#bb00ff');
INSERT INTO tag (id, name, color) VALUES (4, 'Personal', '#aa00aa');

INSERT INTO user (id, email, name) VALUES (1, 'alastor@ya.ru', 'Alastor Moody');
INSERT INTO user (id, email, name) VALUES (2, 'johndawlish@gmail.com', 'John Dawlish');
INSERT INTO user (id, email, name) VALUES (3, 'sirius@gmail.com', 'Sirius Black');
INSERT INTO user (id, email, name) VALUES (4, 'nymphadora@gmail.com', 'Nymphadora Tonks ');