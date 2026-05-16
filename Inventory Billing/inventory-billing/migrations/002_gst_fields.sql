-- ============================================================
-- Migration: 002_gst_fields
-- Adds GST-related columns and the invoice number sequence.
-- Run after 001_init_schema.sql.
-- ============================================================

-- Invoice number generator (gaps are acceptable in accounting)
CREATE SEQUENCE IF NOT EXISTS invoice_number_seq START 1;

-- ── products ─────────────────────────────────────────────────────────────────
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS tax_rate NUMERIC(5, 2) NOT NULL DEFAULT 0
        CHECK (tax_rate >= 0 AND tax_rate <= 100);

COMMENT ON COLUMN products.tax_rate IS 'GST rate in percent, e.g. 18.0 for 18%';

-- ── invoice_items ─────────────────────────────────────────────────────────────
ALTER TABLE invoice_items
    ADD COLUMN IF NOT EXISTS tax_rate   NUMERIC(5,  2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total      NUMERIC(12, 2) NOT NULL DEFAULT 0;

COMMENT ON COLUMN invoice_items.tax_rate   IS 'GST rate snapshot at time of invoice';
COMMENT ON COLUMN invoice_items.tax_amount IS 'sub_total * tax_rate / 100';
COMMENT ON COLUMN invoice_items.total      IS 'sub_total + tax_amount';

-- ── invoices ──────────────────────────────────────────────────────────────────
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS taxable_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cgst_amount    NUMERIC(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sgst_amount    NUMERIC(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS igst_amount    NUMERIC(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS intra_state    BOOLEAN        NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN invoices.taxable_amount IS 'sub_total - discount; the GST base';
COMMENT ON COLUMN invoices.cgst_amount    IS 'Central GST — set for intra-state transactions';
COMMENT ON COLUMN invoices.sgst_amount    IS 'State GST  — set for intra-state transactions';
COMMENT ON COLUMN invoices.igst_amount    IS 'Integrated GST — set for inter-state transactions';
COMMENT ON COLUMN invoices.intra_state    IS 'true = CGST+SGST applies; false = IGST applies';

-- Enforce: total_price = taxable_amount + tax_amount
ALTER TABLE invoices
    DROP CONSTRAINT IF EXISTS chk_invoice_total,
    ADD  CONSTRAINT chk_invoice_total
         CHECK (ABS(total_price - (taxable_amount + tax_amount)) < 0.01);
