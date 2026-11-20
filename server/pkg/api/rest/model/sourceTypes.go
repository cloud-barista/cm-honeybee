package model

// How a source group type takes the resource_type of its connections.
const (
	// ResourceTypeFieldRequired: every connection names one of the listed
	// resource_type values.
	ResourceTypeFieldRequired = "required"
	// ResourceTypeFieldOptional: a connection may name one, and one that does
	// not is DefaultResourceType.
	ResourceTypeFieldOptional = "optional"
	// ResourceTypeFieldNone: the connections take no resource_type; a value
	// sent is ignored. Such a type lists one connection type, whose
	// resource_type is null.
	ResourceTypeFieldNone = "none"
)

// SourceFieldSpec describes one request field of a source group or of a
// connection, so a client can build its form without knowing the type.
type SourceFieldSpec struct {
	// Name is the JSON field name in the request.
	Name     string `json:"name" example:"ip_address"`
	Required bool   `json:"required"`
	// Secret marks a value honeybee keeps in OpenBao and a form should mask.
	Secret      bool   `json:"secret"`
	Description string `json:"description,omitempty"`
	// Enum lists the values accepted, when they are fixed.
	Enum []string `json:"enum,omitempty"`
	// Default is the value taken when the field is left empty.
	Default string `json:"default,omitempty"`
	// RequiredWhen names the condition under which an optional field becomes
	// required: another field being set ("ip_address") or having a value
	// ("db_access_type=ssh-tunnel").
	RequiredWhen string `json:"required_when,omitempty"`
	// OneOf groups fields of which at least one must be set. The fields with
	// the same OneOf value form one group.
	OneOf string `json:"one_of,omitempty"`
	// ValuesSource is the API that lists the values of the field.
	ValuesSource string `json:"values_source,omitempty"`
	// CredentialSource is the API that lists the keys of a credential field.
	CredentialSource string `json:"credential_source,omitempty"`
}

// ConnectionTypeSpec describes one kind of connection a source group type
// holds.
type ConnectionTypeSpec struct {
	// Type names the connection type, as connection_count_by_type of a source
	// group counts it: the resource_type when there is one, the source group
	// type otherwise.
	Type string `json:"type" example:"vm"`
	// ResourceType is the resource_type value to send, or null when the
	// connection takes none.
	ResourceType *string `json:"resource_type"`
	Description  string  `json:"description,omitempty"`
	// AuthFields are the fields that reach and authenticate to the source.
	AuthFields []SourceFieldSpec `json:"auth_fields"`
	// Fields are the other fields: which resource or what part of it to collect.
	Fields []SourceFieldSpec `json:"fields"`
}

// SourceGroupTypeSpec describes one source group type.
type SourceGroupTypeSpec struct {
	Type string `json:"type" example:"onprem"`
	// Aliases are other values the type is accepted and stored as.
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description,omitempty"`
	// GroupFields are the fields the source group itself takes besides name
	// and description: provider_name, region_name and credential.
	GroupFields []SourceFieldSpec `json:"group_fields"`
	// ResourceTypeField is "required", "optional" or "none".
	ResourceTypeField   string               `json:"resource_type_field"`
	DefaultResourceType string               `json:"default_resource_type,omitempty"`
	ConnectionTypes     []ConnectionTypeSpec `json:"connection_types"`
}

// ListSourceGroupTypesRes is the response payload for GET /source_group/types.
type ListSourceGroupTypesRes struct {
	Types []SourceGroupTypeSpec `json:"types"`
}
