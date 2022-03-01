drop table if exists users, tokens, orders, accruals, withdrawns, balance, queue cascade;

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

-- регистрация пользователя
insert into users (login, pwd, pwd_salt) values('aaa', '123', '987') returning user_id;
insert into balance (user_id) values (1);
insert into users (login, pwd, pwd_salt) values('bbb', '234', '654') returning user_id;
insert into balance (user_id) values (2);
insert into users (login, pwd, pwd_salt) values('ccc', '456', '321') returning user_id;
insert into balance (user_id) values (3);

--insert into balance

-- загрузка номера заказа
insert into orders (order_num, user_id) values ('rrr456', 1);
insert into accruals (order_num, status, accrual) values ('rrr456', 'REGISTERED', 0);
insert into queue (order_num, user_id) values ('rrr456', 1);

insert into orders (order_num, user_id) values ('ttt456', 1);
insert into accruals (order_num, status, accrual) values ('ttt456', 'REGISTERED', 0);
insert into queue (order_num, user_id) values ('ttt456', 1);

insert into orders (order_num, user_id) values ('ppp456', 2);
insert into accruals (order_num, status, accrual) values ('ppp456', 'REGISTERED', 0);
insert into queue (order_num, user_id) values ('ppp456', 2);

select user_id from orders where order_num = 'ppp456';

-- получение списка загруженных номеров заказов пользователя
select order_num, status, accrual, uploaded_at from accruals where order_num in (select order_num from orders where user_id = 1);

-- получение текущего баланса пользователя
select available, withdrawn from balance where user_id = 1;

-- запрос на списание средств
insert into orders (order_num, user_id) values ('lll456', 1);
insert into accruals (order_num, status, accrual) values ('lll456', 'REGISTERED', 0);
insert into queue (order_num, user_id) values ('lll456', 1);

update balance set available = available + 100;
update balance set available = available + 100;
update balance set available = available - 200, withdrawn = withdrawn + 200 where user_id = 1;

select * from balance where user_id = 1;

-- получение информации о выводе средств
insert into withdrawns (order_num, withdrawn) values ('lll456', 644);
select order_num, withdrawn, processed_at from withdrawns where order_num in (select order_num from orders where user_id = 1);

----------- Очередь -------------------
-- выборка самого старого заказа
update queue set last_checked_at = default, in_handling = true
where order_num in (select order_num from queue where in_handling = false order by last_checked_at limit 1)
returning order_num;

-- удаление из очереди
delete from queue where order_num = 'lll456';
update balance set available = available + 300 where user_id = 1;
select * from queue;

-- пароли для токенов
insert into tokens (user_id, key_salt) values (2, 'sfdg');
update tokens set key_salt = 'erty' where user_id = 1;
select key_salt from tokens where user_id = 1;
select * from tokens