CREATE DATABASE computantis
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

\c computantis

CREATE TABLE IF NOT EXISTS addresses (
   id serial PRIMARY KEY,
   public_key VARCHAR ( 64 ) UNIQUE NOT NULL
);

CREATE INDEX address_public_key ON addresses USING HASH (public_key);

CREATE TABLE IF NOT EXISTS transactionsPermanent (
   id serial PRIMARY KEY,
   created_at BIGINT NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   issuer_address VARCHAR ( 64 ) NOT NULL,
   receiver_address VARCHAR ( 64 ) NOT NULL,
   subject VARCHAR ( 100 ) NOT NULL,
   data BYTEA,
   issuer_signature BYTEA NOT NULL,
   receiver_signature BYTEA NOT NULL
);

CREATE INDEX transaction_permanent_hash ON transactionsPermanent USING HASH (hash);
CREATE INDEX transaction_permanent_issuer_address ON transactionsPermanent USING HASH (issuer_address);
CREATE INDEX transaction_permanent_receiver_address ON transactionsPermanent USING HASH (receiver_address);
CREATE INDEX transaction_permanent_created_at ON transactionsPermanent USING BTREE (created_at);

CREATE TABLE IF NOT EXISTS transactionsTemporary (
   id serial PRIMARY KEY,
   created_at BIGINT NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   issuer_address VARCHAR ( 64 ) NOT NULL,
   receiver_address VARCHAR ( 64 ) NOT NULL,
   subject VARCHAR ( 100 ) NOT NULL,
   data BYTEA,
   issuer_signature BYTEA NOT NULL,
   receiver_signature BYTEA
);

CREATE INDEX transaction_temporary_hash ON transactionsTemporary USING HASH (hash);
CREATE INDEX transaction_temporary_created_at ON transactionsTemporary USING BTREE (created_at);

CREATE TABLE IF NOT EXISTS transactionsAwaitingReceiver (
   id serial PRIMARY KEY,
   created_at BIGINT NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   issuer_address VARCHAR ( 64 ) NOT NULL,
   receiver_address VARCHAR ( 64 ) NOT NULL,
   subject VARCHAR ( 100 ) NOT NULL,
   data BYTEA,
   issuer_signature BYTEA NOT NULL,
   receiver_signature BYTEA
);

CREATE INDEX transaction_awaiting_issuer_address ON transactionsAwaitingReceiver USING HASH (issuer_address);
CREATE INDEX transaction_awaiting_receiver_address ON transactionsAwaitingReceiver USING HASH (receiver_address);

CREATE TABLE IF NOT EXISTS blocks (
   id serial PRIMARY KEY,
   index BIGINT UNIQUE NOT NULL,
   timestamp BIGINT NOT NULL,
   nonce INTEGER NOT NULL,
   difficulty INTEGER NOT NULL,
   hash BYTEA UNIQUE NOT NULL,
   prev_hash BYTEA NOT NULL,
   trx_hashes BYTEA[] NOT NULL
);

CREATE INDEX block_index ON blocks USING HASH (index);
CREATE INDEX block_hash ON blocks USING HASH (hash);
CREATE INDEX block_prev_hash ON blocks USING HASH (prev_hash);
CREATE INDEX block_created_at ON blocks USING BTREE (timestamp);

CREATE TABLE IF NOT EXISTS transactionsInBlock (
   id serial PRIMARY KEY,
   block_hash BYTEA NOT NULL,
   transaction_hash BYTEA UNIQUE NOT NULL,
   FOREIGN KEY (block_hash) REFERENCES blocks (hash)
);

CREATE INDEX transaction_hash_in_block ON transactionsInBlock USING HASH (transaction_hash);

CREATE TABLE IF NOT EXISTS tokens (
   id serial PRIMARY KEY,
   token VARCHAR (100) UNIQUE NOT NULL,
   valid BOOLEAN NOT NULL,
   expiration_date BIGINT NOT NULL
);

CREATE INDEX token_token ON tokens USING HASH (token);
CREATE INDEX token_expiration_date ON tokens USING BTREE (expiration_date);

CREATE TABLE IF NOT EXISTS logs (
   id serial PRIMARY KEY,
   level VARCHAR ( 10 ) NOT NULL,
   msg VARCHAR ( 256 ) NOT NULL,
   created_at BIGINT NOT NULL
);

CREATE INDEX logs_created_at ON logs USING BTREE (created_at);

CREATE TABLE IF NOT EXISTS validatorStatus (
   id serial PRIMARY KEY,
   index INTEGER UNIQUE NOT NULL,
   valid BOOLEAN NOT NULL,
   created_at BIGINT NOT NULL,
   FOREIGN KEY (index) REFERENCES blocks (index)
);

CREATE INDEX validator_index ON validatorStatus USING HASH (index);
CREATE INDEX validator_created_at ON validatorStatus USING BTREE (created_at);

CREATE TABLE IF NOT EXISTS nodes (
   id serial PRIMARY KEY,
   node VARCHAR ( 64 ) UNIQUE NOT NULL,
);

CREATE INDEX nodes_index ON nodes USING HASH (node);

CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $$

    DECLARE 
        data json;
        notification json;
    
    BEGIN
    
        IF (TG_OP = 'DELETE') THEN
            data = row_to_json(OLD);
        ELSE
            data = row_to_json(NEW);
        END IF;

        notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'data', data);
        
                        
        PERFORM pg_notify('events',notification::text);
        
        RETURN NULL; 
    END;
    
$$ LANGUAGE plpgsql;

CREATE TABLE blockchainLocks (
    id serial PRIMARY KEY,
    timestamp BIGINT NOT NULL,
    node VARCHAR ( 100 ) NOT NULL
);

CREATE INDEX blockchainLocks_timestamp ON blockchainLocks USING BTREE (timestamp);
CREATE INDEX blockchainLocks_node ON blockchainLocks USING HASH (node);

CREATE TRIGGER blockchainLocks_notify_event
AFTER INSERT OR UPDATE OR DELETE ON blockchainLocks
    FOR EACH ROW EXECUTE PROCEDURE notify_event();


CREATE USER computantis WITH ENCRYPTED PASSWORD 'computantis';

GRANT ALL PRIVILEGES ON DATABASE computantis TO computantis;
GRANT ALL PRIVILEGES ON SCHEMA public TO computantis;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO computantis;
GRANT ALL ON ALL TABLES IN SCHEMA public TO computantis;
GRANT All PRIVILEGES ON FUNCTION notify_event() TO computantis;
