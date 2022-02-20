--drop table if exists users, orders, accruals, withdrawals, balance cascade;

create table if not exists users
(
    user_id serial primary key,
    login varchar(32) unique,
    pwd char(64),
    pwd_salt char(64),
    registered_at timestamp default now()
    );

create table if not exists tokens
(
    id serial,
    iser_id integer,
    key_salt char(64),
    foreign key (user_id) references users (user_id) on delete cascade
)

create table if not exists orders
(
    id serial,
    order_num integer primary key,
    user_id integer,
    uploaded_at timestamp default now(),
    foreign key (user_id) references users (user_id) on delete cascade
    );

create table if not exists accruals
(
    id serial ,
    order_num integer primary key,
    status varchar(10),
    accrual numeric(12,2),
    uploaded_at timestamp default now(),
    foreign key (order_num) references orders (order_num) on delete cascade
    );

create table if not exists withdrawals
(
    id serial,
    order_num integer primary key,
    withdrawal numeric(12,2),
    uploaded_at timestamp default now(),
    foreign key (order_num) references orders (order_num) on delete cascade
    );

create table if not exists balance
(
    id serial primary key,
    user_id integer,
    available numeric(12,2),
    withdrawals numeric(12,2),
    foreign key (user_id) references users (user_id) on delete cascade
    );

create table if not exists cache
(
    id serial primary key,
    user_id integer,
    order_id integer unique,
    uploaded_at timestamp default now(),
    last_checked_at timestamp default now()
);