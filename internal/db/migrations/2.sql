alter table conversations
add column end_time text;

alter table conversation_fragments
add column fragment_time text;

alter table documents
add column insert_time text;

alter table document_fragments 
add column insert_time text;