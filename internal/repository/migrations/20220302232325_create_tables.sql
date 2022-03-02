-- +goose Up
-- +goose StatementBegin
create table if not exists users
(
    user_id serial primary key,
    login varchar(64) unique,
    pwd char(64),
    pwd_salt char(64),
    registered_at timestamp default now()
);

create table if not exists tokens
(
    id serial,
    user_id integer primary key,
    key_salt char(64),
    foreign key (user_id) references users (user_id) on delete cascade
);

create table if not exists orders
(
    id serial,
    order_num varchar(32) primary key,
    user_id integer,
    uploaded_at timestamp default now(),
    foreign key (user_id) references users (user_id) on delete cascade
);

create table if not exists accruals
(
    id serial ,
    order_num varchar(32) primary key,
    status varchar(16),
    accrual numeric(12,2) default 0,
    uploaded_at timestamp default now(),
    foreign key (order_num) references orders (order_num) on delete cascade
);

create table if not exists withdrawns
(
    id serial,
    order_num varchar(32) primary key,
    withdrawn numeric(12,2),
    processed_at timestamp default now(),
    foreign key (order_num) references orders (order_num) on delete cascade
);

create table if not exists balance
(
    id serial primary key,
    user_id integer unique,
    available numeric(12,2) default 0 check (available >= 0),
    withdrawn numeric(12,2) default 0 check (withdrawn >= 0),
    foreign key (user_id) references users (user_id) on delete cascade
);

create table if not exists queue
(
    id serial primary key,
    order_num varchar(32) unique,
    user_id integer,
    uploaded_at timestamp default now(),
    last_checked_at timestamp default now(),
    in_handling boolean default false
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
