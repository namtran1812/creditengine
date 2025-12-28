-- deposits, accounts, audits
CREATE TABLE IF NOT EXISTS accounts (
  id bigserial primary key,
  address text unique not null,
  balance bigint not null default 0
);

CREATE TABLE IF NOT EXISTS deposits (
  id bigserial primary key,
  tx_hash text unique not null,
  address text not null,
  amount bigint not null,
  confirmations bigint not null default 0,
  tx_block bigint,
  block_hash text,
  status text not null default 'pending',
  received_at timestamptz not null default now(),
  credited_at timestamptz
);

CREATE TABLE IF NOT EXISTS audits (
  id bigserial primary key,
  deposit_id bigint references deposits(id),
  action text not null,
  created_at timestamptz not null default now()
);
