create virtual table conversation_fragment_embeddings using vss0(
    embedding(1536) factory="L2norm,Flat,IDMap2" metric_type=INNER_PRODUCT,
);

create table conversations (
    id integer primary key autoincrement,
    conversation text
);

create table conversation_fragments (
    id integer primary key autoincrement,
    conversation_id integer,
    conversation_fragment text
);

create virtual table document_fragment_embeddings using vss0(
    embedding(1536) factory="L2norm,Flat,IDMap2" metric_type=INNER_PRODUCT,
);

create table documents (
    id integer primary key autoincrement,
    document text
);

create table document_fragments (
    id integer primary key autoincrement,
    document_id integer,
    document_fragment text
);
