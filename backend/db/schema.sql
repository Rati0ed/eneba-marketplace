-- Database schema. Runs on every startup on a fresh, empty database file.
-- Money is always stored as INTEGER cents (1999 = €19.99) to avoid float rounding errors.
-- Foreign keys are switched on in the connection string (see db.go), because
-- SQLite ignores them unless asked.

CREATE TABLE products (
    id           INTEGER PRIMARY KEY,           -- SQLite auto-assigns 1, 2, 3...
    slug         TEXT    NOT NULL UNIQUE,       -- URL-friendly id: "elden-ring-pc-steam-global"
    title        TEXT    NOT NULL,              -- "ELDEN RING (PC) Steam Key GLOBAL"
    platform     TEXT    NOT NULL,              -- Steam, PlayStation, Xbox, Nintendo
    region       TEXT    NOT NULL,              -- Global, Europe, United States
    genre        TEXT    NOT NULL,              -- RPG, Shooter, ... or "Gift Card"
    description  TEXT    NOT NULL,
    image_url    TEXT    NOT NULL,
    msrp_cents   INTEGER NOT NULL CHECK (msrp_cents > 0),   -- official price, used for discount %
    release_date TEXT    NOT NULL,              -- ISO date "2022-02-25" (SQLite has no DATE type)
    sales_count  INTEGER NOT NULL DEFAULT 0     -- popularity, used for "best sellers" and search ties
);

CREATE TABLE sellers (
    id             INTEGER PRIMARY KEY,
    name           TEXT    NOT NULL UNIQUE,
    rating_percent INTEGER NOT NULL CHECK (rating_percent BETWEEN 0 AND 100),
    sales_count    INTEGER NOT NULL DEFAULT 0
);

-- One seller's price for one product. A seller can only have one offer per product.
CREATE TABLE offers (
    id          INTEGER PRIMARY KEY,
    product_id  INTEGER NOT NULL REFERENCES products(id),
    seller_id   INTEGER NOT NULL REFERENCES sellers(id),
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    UNIQUE (product_id, seller_id)
);

CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,             -- bcrypt hash, never the real password
    display_name  TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE orders (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    status      TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed')),
    total_cents INTEGER NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    paid_at     TEXT,                           -- NULL until paid
    card_last4  TEXT                            -- last 4 digits of the card, for "Paid with •••• 4242"
                                                -- (the full card number is never stored)
);

-- Title and price are COPIED at purchase time, so order history stays correct
-- even if the seller changes the price later.
CREATE TABLE order_items (
    id               INTEGER PRIMARY KEY,
    order_id         INTEGER NOT NULL REFERENCES orders(id),
    offer_id         INTEGER NOT NULL REFERENCES offers(id),
    quantity         INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL,
    product_title    TEXT    NOT NULL
);

-- Each row is one digital key. Stock of an offer = number of its 'available' keys.
CREATE TABLE game_keys (
    id            INTEGER PRIMARY KEY,
    offer_id      INTEGER NOT NULL REFERENCES offers(id),
    code          TEXT    NOT NULL UNIQUE,
    status        TEXT    NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'sold')),
    order_item_id INTEGER REFERENCES order_items(id)   -- NULL until sold
);

CREATE TABLE banners (
    id        INTEGER PRIMARY KEY,
    title     TEXT    NOT NULL,
    subtitle  TEXT    NOT NULL,
    image_url TEXT    NOT NULL,
    link_url  TEXT    NOT NULL,
    position  INTEGER NOT NULL                  -- display order: 1 first
);

-- Which blocks the homepage shows, in which order, and which endpoint feeds each one.
CREATE TABLE homepage_blocks (
    id        INTEGER PRIMARY KEY,
    type      TEXT    NOT NULL CHECK (type IN ('banner', 'product_row')),
    title     TEXT    NOT NULL,
    endpoint  TEXT    NOT NULL,                 -- e.g. "/api/home/deals"
    link_url  TEXT,                             -- "Show all" link, e.g. "/store?sort=discount" (NULL = none)
    position  INTEGER NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1        -- SQLite has no BOOLEAN: 1 = true, 0 = false
);

-- Indexes make lookups by these columns fast (like a book's index).
CREATE INDEX idx_offers_product   ON offers(product_id);
CREATE INDEX idx_keys_offer_state ON game_keys(offer_id, status);
CREATE INDEX idx_orders_user      ON orders(user_id);
CREATE INDEX idx_items_order      ON order_items(order_id);
