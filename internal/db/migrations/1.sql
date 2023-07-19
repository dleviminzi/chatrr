create virtual table conversation_fragment_embeddings using vss0(
    embedding(1536) factory="L2norm,Flat,IDMap2" metric_type=INNER_PRODUCT,
);

create table conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation TEXT
);

create table conversation_fragments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER,
    conversation_fragment TEXT
);

create virtual table document_fragment_embeddings using vss0(
    embedding(1536) factory="L2norm,Flat,IDMap2" metric_type=INNER_PRODUCT,
);

create table documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document TEXT
);

create table document_fragments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER,
    document_fragment TEXT
);
