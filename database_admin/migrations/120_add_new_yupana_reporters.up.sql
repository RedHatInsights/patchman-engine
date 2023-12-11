INSERT INTO reporter (id, name) VALUES 
      (5, 'satellite'),
      (6, 'discovery')
ON CONFLICT DO NOTHING;
