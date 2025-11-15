CREATE TABLE fish (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  count INTEGER NOT NULL
);

INSERT INTO fish (name, color, count) VALUES
  ('one fish', 'yellow', 1),
  ('two fish', 'green', 2),
  ('red fish', 'red', 1),
  ('blue fish', 'blue', 1),
  ('black fish', 'black', 3),
  ('star fish', 'gold', 5),
  ('zebra fish', 'striped', 4);
