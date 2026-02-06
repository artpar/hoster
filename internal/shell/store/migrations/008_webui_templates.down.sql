DELETE FROM templates WHERE id IN (
    'tmpl_wordpress', 'tmpl_uptime_kuma', 'tmpl_gitea',
    'tmpl_n8n', 'tmpl_it_tools', 'tmpl_metabase'
);
