package engine

import (
	"context"
	"strings"
)

// Schema returns all resource definitions for the Hoster application.
// This is the single source of truth — migrations, API, store, and types are all derived from this.
func Schema() []Resource {
	return []Resource{
		TemplateResource(),
		DeploymentResource(),
		NodeResource(),
		SSHKeyResource(),
		CloudCredentialResource(),
		CloudProvisionResource(),
		InvoiceResource(),
	}
}

func TemplateResource() Resource {
	return Resource{
		Name:      "templates",
		Owner:     "creator_id",
		RefPrefix: "tmpl_",
		PublicRead: true, // Published templates visible to all
		Fields: []Field{
			StringField("name").WithRequired().WithMinLen(3).WithMaxLen(100).WithPattern(`^[a-zA-Z0-9\s\-]+$`),
			StringField("slug").WithUnique().WithComputed(func(row map[string]any) any {
				if name, ok := row["name"].(string); ok {
					return slugify(name)
				}
				return ""
			}),
			StringField("description").WithNullable(),
			StringField("version").WithRequired().WithPattern(`^\d+\.\d+\.\d+$`),
			TextField("compose_spec").WithRequired(),
			JSONField("variables"),
			JSONField("config_files"),
			JSONField("tags"),
			JSONField("required_capabilities"),
			StringField("category").WithNullable(),
			FloatField("resources_cpu_cores").WithDefault(0),
			IntField("resources_memory_mb").WithDefault(0),
			IntField("resources_disk_mb").WithDefault(0),
			IntField("price_monthly_cents").WithMin(0).WithDefault(0),
			BoolField("published").WithDefault(false),
			RefField("creator_id", "users").WithInternal(),
		},
		Actions: []CustomAction{
			{Name: "publish", Method: "POST"},
		},
		Visibility: templateVisibility,
	}
}

func DeploymentResource() Resource {
	return Resource{
		Name:      "deployments",
		Owner:     "customer_id",
		RefPrefix: "", // full UUID
		Fields: []Field{
			StringField("name").WithRequired(),
			RefField("template_id", "templates"),
			StringField("template_version").WithNullable(),
			RefField("customer_id", "users").WithInternal(),
			SoftRefField("node_id", "nodes"),
			StringField("status").WithDefault("pending"),
			JSONField("variables"),
			JSONField("domains"),
			JSONField("containers"),
			FloatField("resources_cpu_cores").WithDefault(0),
			IntField("resources_memory_mb").WithDefault(0),
			IntField("resources_disk_mb").WithDefault(0),
			IntField("proxy_port").WithNullable(),
			StringField("error_message").WithNullable(),
			TimestampField("started_at"),
			TimestampField("stopped_at"),
		},
		StateMachine: &StateMachine{
			Field:   "status",
			Initial: "pending",
			Transitions: map[string][]string{
				"pending":   {"scheduled"},
				"scheduled": {"starting"},
				"starting":  {"running", "failed"},
				"running":   {"stopping", "failed"},
				"stopping":  {"stopped"},
				"stopped":   {"starting", "deleting"},
				"deleting":  {"deleted"},
				"failed":    {"starting", "deleting"},
				"deleted":   {},
			},
			Guards: map[string]GuardFunc{
				"starting": RequireField("node_id"),
			},
			OnEnter: map[string]string{
				"scheduled": "ScheduleDeployment",
				"starting":  "StartDeployment",
				"stopping":  "StopDeployment",
				"deleting":  "DeleteDeployment",
				"running":   "DeploymentRunning",
				"failed":    "DeploymentFailed",
			},
		},
		Actions: []CustomAction{
			{Name: "start", Method: "POST"},
			{Name: "stop", Method: "POST"},
			{Name: "monitoring/health", Method: "GET"},
			{Name: "monitoring/stats", Method: "GET"},
			{Name: "monitoring/logs", Method: "GET"},
			{Name: "monitoring/events", Method: "GET"},
			{Name: "domains", Method: "GET"},
			{Name: "domains", Method: "POST"},
		},
	}
}

func NodeResource() Resource {
	return Resource{
		Name:       "nodes",
		Owner:      "creator_id",
		RefPrefix:  "node_",
		PublicRead: true,
		Fields: []Field{
			StringField("name").WithRequired().WithMinLen(3).WithMaxLen(100),
			RefField("creator_id", "users").WithInternal(),
			StringField("ssh_host").WithRequired().WithOwnerOnly(),
			IntField("ssh_port").WithDefault(22).WithOwnerOnly(),
			StringField("ssh_user").WithRequired().WithOwnerOnly(),
			RefField("ssh_key_id", "ssh_keys").WithNullable().WithOwnerOnly(),
			StringField("docker_socket").WithDefault("/var/run/docker.sock").WithOwnerOnly(),
			StringField("status").WithDefault("offline"),
			BoolField("public").WithDefault(false),
			JSONField("capabilities"),
			FloatField("capacity_cpu_cores").WithDefault(0),
			IntField("capacity_memory_mb").WithDefault(0),
			IntField("capacity_disk_mb").WithDefault(0),
			FloatField("capacity_cpu_used").WithDefault(0),
			IntField("capacity_memory_used_mb").WithDefault(0),
			IntField("capacity_disk_used_mb").WithDefault(0),
			StringField("location").WithNullable(),
			TimestampField("last_health_check"),
			StringField("error_message").WithNullable(),
			StringField("provider_type").WithDefault("manual"),
			SoftRefField("provision_id", "cloud_provisions"),
			StringField("base_domain").WithNullable(),
		},
		Actions: []CustomAction{
			{Name: "maintenance", Method: "POST"},
			{Name: "maintenance", Method: "DELETE"},
		},
		Visibility: nodeVisibility,
	}
}

func SSHKeyResource() Resource {
	return Resource{
		Name:      "ssh_keys",
		Owner:     "creator_id",
		RefPrefix: "sshkey_",
		Fields: []Field{
			RefField("creator_id", "users").WithInternal(),
			StringField("name").WithRequired(),
			TextField("private_key").WithWriteOnly().WithEncrypted(),
			TextField("public_key").WithNullable(),
			StringField("fingerprint").WithNullable(),
		},
	}
}

func CloudCredentialResource() Resource {
	return Resource{
		Name:      "cloud_credentials",
		Owner:     "creator_id",
		RefPrefix: "cred_",
		Fields: []Field{
			RefField("creator_id", "users").WithInternal(),
			StringField("name").WithRequired().WithMinLen(3).WithMaxLen(100),
			StringField("provider").WithRequired(),
			TextField("credentials").WithWriteOnly().WithEncrypted(),
			StringField("default_region").WithNullable(),
		},
		Actions: []CustomAction{
			{Name: "regions", Method: "GET"},
			{Name: "sizes", Method: "GET"},
		},
	}
}

func CloudProvisionResource() Resource {
	return Resource{
		Name:      "cloud_provisions",
		Owner:     "creator_id",
		RefPrefix: "prov_",
		Fields: []Field{
			RefField("creator_id", "users").WithInternal(),
			RefField("credential_id", "cloud_credentials"),
			StringField("provider").WithRequired(),
			StringField("status").WithDefault("pending"),
			StringField("instance_name").WithRequired(),
			StringField("region").WithRequired(),
			StringField("size").WithRequired(),
			StringField("provider_instance_id").WithNullable(),
			StringField("public_ip").WithNullable(),
			SoftRefField("node_id", "nodes"),
			SoftRefField("ssh_key_id", "ssh_keys"),
			StringField("current_step").WithNullable(),
			StringField("error_message").WithNullable(),
			TimestampField("completed_at"),
		},
		StateMachine: &StateMachine{
			Field:   "status",
			Initial: "pending",
			Transitions: map[string][]string{
				"pending":     {"creating", "failed", "destroying"},
				"creating":    {"configuring", "failed", "destroying"},
				"configuring": {"ready", "failed", "destroying"},
				"ready":       {"destroying"},
				"failed":      {"pending", "destroying"},
				"destroying":  {"destroyed", "failed"},
				"destroyed":   {},
			},
			OnEnter: map[string]string{
				"creating":    "ProvisionInstance",
				"configuring": "ConfigureInstance",
				"ready":       "ProvisionReady",
				"destroying":  "DestroyInstance",
			},
		},
		Actions: []CustomAction{
			{Name: "retry", Method: "POST"},
		},
	}
}

func InvoiceResource() Resource {
	return Resource{
		Name:      "invoices",
		Owner:     "user_id",
		RefPrefix: "inv_",
		Fields: []Field{
			RefField("user_id", "users").WithInternal(),
			TimestampField("period_start").WithRequired(),
			TimestampField("period_end").WithRequired(),
			JSONField("items"),
			IntField("subtotal_cents").WithDefault(0),
			IntField("tax_cents").WithDefault(0),
			IntField("total_cents").WithDefault(0),
			StringField("currency").WithDefault("USD"),
			StringField("status").WithDefault("draft"),
			StringField("stripe_session_id").WithNullable(),
			StringField("stripe_payment_url").WithNullable(),
			TimestampField("paid_at"),
		},
		StateMachine: &StateMachine{
			Field:   "status",
			Initial: "draft",
			Transitions: map[string][]string{
				"draft":   {"pending"},
				"pending": {"paid", "failed"},
				"failed":  {"pending"},
			},
		},
		Actions: []CustomAction{
			{Name: "pay", Method: "POST"},
		},
	}
}

// =============================================================================
// Visibility functions
// =============================================================================

// templateVisibility allows published templates to be seen by anyone,
// but unpublished ones only by their creator.
func templateVisibility(ctx context.Context, authCtx AuthContext, row map[string]any) bool {
	if pub, ok := row["published"]; ok {
		switch v := pub.(type) {
		case bool:
			if v {
				return true
			}
		case int64:
			if v != 0 {
				return true
			}
		case int:
			if v != 0 {
				return true
			}
		}
	}
	// Unpublished — only creator can see
	if !authCtx.Authenticated {
		return false
	}
	if ownerID, ok := row["creator_id"]; ok {
		switch v := ownerID.(type) {
		case int:
			return v == authCtx.UserID
		case int64:
			return int(v) == authCtx.UserID
		}
	}
	return false
}

// nodeVisibility allows public nodes to be seen by anyone,
// but private nodes only by their creator.
func nodeVisibility(ctx context.Context, authCtx AuthContext, row map[string]any) bool {
	// Owner always sees their own nodes
	if authCtx.Authenticated {
		if ownerID, ok := row["creator_id"]; ok {
			switch v := ownerID.(type) {
			case int:
				if v == authCtx.UserID {
					return true
				}
			case int64:
				if int(v) == authCtx.UserID {
					return true
				}
			}
		}
	}
	// Others only see public nodes
	if pub, ok := row["public"]; ok {
		switch v := pub.(type) {
		case bool:
			return v
		case int64:
			return v != 0
		case int:
			return v != 0
		}
	}
	return false
}

// slugify converts a name to a URL-safe slug.
func slugify(name string) string {
	slug := ""
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			slug += string(r)
		} else if r == ' ' {
			slug += "-"
		}
	}
	return slug
}
