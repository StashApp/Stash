CREATE TABLE tags_relations (
  parent_id integer,
  child_id integer,
  primary key (parent_id, child_id),
  foreign key (parent_id) references tags(id) on delete cascade,
  foreign key (child_id) references tags(id) on delete cascade
);
