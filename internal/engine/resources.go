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
			FloatField("resources_cpu_cores"),
			IntField("resources_memory_mb"),
			IntField("resources_disk_mb"),
			IntField("price_monthly_cents").WithMin(0),
			BoolField("published").WithDefault(false),
			RefField("creator_id", "users").WithInternal(),
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
			StringField("template_version"),
			RefField("customer_id", "users").WithInternal(),
			SoftRefField("node_id", "nodes"),
			StringField("status").WithDefault("pending"),
			JSONField("variables"),
			JSONField("domains"),
			JSONField("containers"),
			FloatField("resources_cpu_cores"),
			IntField("resources_memory_mb"),
			IntField("resources_disk_mb"),
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
	}
}

func NodeResource() Resource {
	return Resource{
		Name:      "nodes",
		Owner:     "creator_id",
		RefPrefix: "node_",
		Fields: []Field{
			StringField("name").WithRequired().WithMinLen(3).WithMaxLen(100),
			RefField("creator_id", "users").WithInternal(),
			StringField("ssh_host").WithRequired(),
			IntField("ssh_port").WithDefault(22),
			StringField("ssh_user").WithRequired(),
			IntField("ssh_key_id").WithNullable(),
			StringField("docker_socket").WithDefault("/var/run/docker.sock"),
			StringField("status").WithDefault("offline"),
			JSONField("capabilities"),
			FloatField("capacity_cpu_cores"),
			IntField("capacity_memory_mb"),
			IntField("capacity_disk_mb"),
			FloatField("capacity_cpu_used"),
			IntField("capacity_memory_used_mb"),
			IntField("capacity_disk_used_mb"),
			StringField("location").WithNullable(),
			TimestampField("last_health_check"),
			StringField("error_message").WithNullable(),
			StringField("provider_type").WithDefault("manual"),
			SoftRefField("provision_id", "cloud_provisions"),
			StringField("base_domain").WithNullable(),
		},
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
			BoolField("private_key_encrypted").WithWriteOnly().WithEncrypted(), // stored as BLOB, write-only
			StringField("fingerprint"),
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
			BoolField("credentials_encrypted").WithWriteOnly().WithEncrypted(), // stored as BLOB, write-only
			StringField("default_region").WithNullable(),
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
				"pending":     {"creating", "failed"},
				"creating":    {"configuring", "failed"},
				"configuring": {"ready", "failed"},
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
