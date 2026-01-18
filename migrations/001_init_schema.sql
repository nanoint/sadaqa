-- Enable UUID extension if not already enabled (for ID generation)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table to store Telegram User context
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY, -- Telegram User ID
    first_name TEXT NOT NULL,
    username TEXT,
    language_code TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Subscriptions table for recurring payments
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id),
    
    -- Donation Details
    mosque_id TEXT NOT NULL,
    mosque_name TEXT NOT NULL,
    amount INTEGER NOT NULL, -- Amount in minor units (e.g., Tiyin) or major depending on logic. Usually integer is safer.
    currency TEXT DEFAULT 'KZT',
    frequency TEXT NOT NULL, -- 'Daily', 'Weekly', 'Monthly'
    
    -- Freedom Pay Specifics
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'active', 'paused', 'cancelled', 'payment_failed'
    recurring_token TEXT, -- The pg_recurring_profile_id or pg_card_id
    card_pan TEXT, -- Masked card number for UI display
    
    -- Scheduling
    next_payment_date DATE NOT NULL,
    last_payment_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for the Worker Pool
-- This allows ProcessDailyPayments to efficiently find due active subscriptions
CREATE INDEX idx_worker_query ON subscriptions(status, next_payment_date);

-- Index for API queries (e.g., fetching a user's profile)
CREATE INDEX idx_user_subscriptions ON subscriptions(user_id);
