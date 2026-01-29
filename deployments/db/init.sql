\connect l0

CREATE TABLE IF NOT EXISTS deliveries (
    id BIGSERIAL,
    name TEXT,
    phone TEXT,
    zip TEXT,
    city TEXT,
    address TEXT,
    region TEXT,
    email TEXT,

    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS orders (
    order_uid TEXT,
    track_number TEXT UNIQUE,
    entry TEXT,
    delivery_id BIGINT,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT,
    delivery_service TEXT,
    shardkey TEXT,
    sm_id INT,
    date_created TIMESTAMPTZ,
    oof_shard INT,

    PRIMARY KEY (order_uid),
    FOREIGN KEY (delivery_id) REFERENCES deliveries (id) ON DELETE NO ACTION
);

CREATE TABLE IF NOT EXISTS payments (
    transaction TEXT NOT NULL,
    request_id TEXT,
    currency TEXT,
    provider TEXT,
    amount INT,
    payment_dt BIGINT,
    bank TEXT,
    delivery_cost INT,
    goods_total INT,
    custom_fee INT,

    PRIMARY KEY (transaction),
    FOREIGN KEY (transaction) REFERENCES orders (order_uid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS items (
    chrt_id BIGINT,
    track_number TEXT NOT NULL,
    price INT,
    rid TEXT,
    name TEXT,
    sale INT,
    size TEXT,
    total_price INT,
    nm_id BIGINT,
    brand TEXT,
    status INT,

    PRIMARY KEY (chrt_id),
    FOREIGN KEY (track_number) REFERENCES orders (
        track_number
    ) ON DELETE CASCADE
);


CREATE INDEX idx_orders_delivery ON orders (delivery_id);
CREATE INDEX idx_items_order ON items (track_number);
CREATE INDEX idx_payments_order ON payments (transaction);