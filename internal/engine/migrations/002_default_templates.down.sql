DELETE FROM templates WHERE creator_id = (SELECT id FROM users WHERE reference_id = 'system');
DELETE FROM users WHERE reference_id = 'system';
