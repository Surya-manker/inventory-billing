-- ============================================================
-- Migration: 001_init_schema
-- Database:  PostgreSQL 14+
-- ============================================================

-- Enable UUID generation (PostgreSQL 13+ has gen_random_uuid() built-in)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: users
-- System users who operate the application (admin / staff).
-- Separate from customers who are being billed.
-- ============================================================
CREATE TABLE users (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(100)  NOT NULL,
    email         VARCHAR(255)  NOT NULL,
    password_hash VARCHAR(255)  NOT NULL,
    role          VARCHAR(20)   NOT NULL DEFAULT 'staff'
                                CHECK (role IN ('admin', 'staff')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_email UNIQUE (email)
);

-- ============================================================
-- TABLE: customers
-- The individuals or businesses that are invoiced.
-- GSTIN is used for GST-compliant invoicing.
-- ============================================================
CREATE TABLE customers (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    phone      VARCHAR(20),
    address    TEXT,
    company    VARCHAR(100),
    gstin      VARCHAR(20),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_customers_email UNIQUE (email)
);

-- ============================================================
-- TABLE: products
-- Inventory items that can appear on invoice line items.
-- Stock is the live on-hand count; every change is also
-- recorded in stock_logs for a full audit trail.
-- ============================================================
CREATE TABLE products (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100)   NOT NULL,
    description TEXT,
    sku         VARCHAR(50)    NOT NULL,
    price       NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    stock       INTEGER        NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_products_sku UNIQUE (sku)
);

-- ============================================================
-- TABLE: invoices
-- A billing document raised for one customer by one staff user.
-- total_price = sub_total - discount + tax_amount
-- paid_at is NULL until status transitions to 'paid'.
-- ============================================================
CREATE TABLE invoices (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50)    NOT NULL,
    customer_id    UUID           NOT NULL,
    created_by     UUID           NOT NULL,
    sub_total      NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (sub_total >= 0),
    discount       NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    tax_amount     NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    total_price    NUMERIC(12, 2) NOT NULL             CHECK (total_price >= 0),
    status         VARCHAR(20)    NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending', 'paid', 'canceled')),
    notes          TEXT,
    issued_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    due_at         TIMESTAMPTZ    NOT NULL,
    paid_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_invoices_number     UNIQUE (invoice_number),
    CONSTRAINT fk_invoices_customer   FOREIGN KEY (customer_id) REFERENCES customers (id),
    CONSTRAINT fk_invoices_created_by FOREIGN KEY (created_by)  REFERENCES users     (id)
);

CREATE INDEX idx_invoices_customer_id ON invoices (customer_id);
CREATE INDEX idx_invoices_status      ON invoices (status);
CREATE INDEX idx_invoices_issued_at   ON invoices (issued_at DESC);

-- ============================================================
-- TABLE: invoice_items
-- One row per product line on an invoice.
-- unit_price is a snapshot of the price at time of invoicing
-- so future product price changes don't affect old invoices.
-- sub_total = quantity * unit_price (stored for fast reads).
-- ============================================================
CREATE TABLE invoice_items (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID           NOT NULL,
    product_id UUID           NOT NULL,
    quantity   INTEGER        NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
    sub_total  NUMERIC(12, 2) NOT NULL CHECK (sub_total >= 0),
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invoice_items_invoice FOREIGN KEY (invoice_id) REFERENCES invoices  (id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_items_product FOREIGN KEY (product_id) REFERENCES products  (id)
);

CREATE INDEX idx_invoice_items_invoice_id ON invoice_items (invoice_id);
CREATE INDEX idx_invoice_items_product_id ON invoice_items (product_id);

-- ============================================================
-- TABLE: stock_logs
-- Append-only audit trail of every stock movement.
-- quantity_change is signed: positive = stock in, negative = stock out.
-- invoice_id is nullable — not every adjustment comes from a sale.
-- ============================================================
CREATE TABLE stock_logs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id       UUID         NOT NULL,
    user_id          UUID         NOT NULL,
    invoice_id       UUID,
    change_type      VARCHAR(30)  NOT NULL
                                  CHECK (change_type IN ('sale', 'purchase', 'adjustment', 'return')),
    quantity_before  INTEGER      NOT NULL,
    quantity_change  INTEGER      NOT NULL,
    quantity_after   INTEGER      NOT NULL,
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_stock_logs_product FOREIGN KEY (product_id) REFERENCES products (id),
    CONSTRAINT fk_stock_logs_user    FOREIGN KEY (user_id)    REFERENCES users     (id),
    CONSTRAINT fk_stock_logs_invoice FOREIGN KEY (invoice_id) REFERENCES invoices  (id) ON DELETE SET NULL
);

CREATE INDEX idx_stock_logs_product_id ON stock_logs (product_id);
CREATE INDEX idx_stock_logs_created_at ON stock_logs (created_at DESC);
