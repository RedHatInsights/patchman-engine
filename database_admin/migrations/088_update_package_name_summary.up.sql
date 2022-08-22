UPDATE package_name pn
   SET summary = latest.summary
  FROM (SELECT DISTINCT ON (p.name_id) p.name_id, str.value as summary
		  FROM package p
		  JOIN strings str ON p.summary_hash = str.id
		 ORDER BY p.name_id, p.id desc) as latest
 WHERE pn.id = latest.name_id
   AND latest.summary IS NOT NULL
   AND (latest.summary != pn.summary OR pn.summary IS NULL);
