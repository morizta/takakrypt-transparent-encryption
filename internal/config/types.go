package config

type Config struct {
	UserSets     []UserSet     `json:"user_sets"`
	ProcessSets  []ProcessSet  `json:"process_sets"`
	ResourceSets []ResourceSet `json:"resource_sets"`
	GuardPoints  []GuardPoint  `json:"guard_points"`
	Policies     []Policy      `json:"policies"`
}

type UserSet struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	CreatedAt   int64     `json:"created_at"`
	ModifiedAt  int64     `json:"modified_at"`
	Description string    `json:"description"`
	Users       []User    `json:"users"`
}

type User struct {
	Index      int    `json:"index"`
	ID         string `json:"id"`
	UID        int    `json:"uid"`
	UName      string `json:"uname"`
	FName      string `json:"fname"`
	GName      string `json:"gname"`
	GID        int    `json:"gid"`
	OS         string `json:"os"`
	Type       string `json:"type"`
	Email      string `json:"email"`
	OSDomain   string `json:"os_domain"`
	OSUser     string `json:"os_user"`
	CreatedAt  int64  `json:"created_at"`
	ModifiedAt int64  `json:"modified_at"`
}

type ProcessSet struct {
	ID                string                `json:"id"`
	Code              string                `json:"code"`
	Name              string                `json:"name"`
	CreatedAt         int64                 `json:"created_at"`
	ModifiedAt        int64                 `json:"modified_at"`
	Description       string                `json:"description"`
	ResourceSetList   []ProcessSetResource  `json:"resource_set_list"`
}

type ProcessSetResource struct {
	Index                  int      `json:"index"`
	ID                     string   `json:"id"`
	Directory              string   `json:"directory"`
	File                   string   `json:"file"`
	Signature              []string `json:"signature"`
	RWPExemptedResources   []string `json:"rwp_exempted_resources"`
	CreatedAt              int64    `json:"created_at"`
	ModifiedAt             int64    `json:"modified_at"`
}

type ResourceSet struct {
	ID               string     `json:"id"`
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	ResourceSetType  string     `json:"resource_set_type"`
	OS               string     `json:"os"`
	CreatedAt        int64      `json:"created_at"`
	ModifiedAt       int64      `json:"modified_at"`
	Description      string     `json:"description"`
	ResourceList     []Resource `json:"resource_list"`
}

type Resource struct {
	Index      int    `json:"index"`
	ID         string `json:"id"`
	Directory  string `json:"directory"`
	File       string `json:"file"`
	Subfolder  bool   `json:"subfolder"`
	HSDFS      bool   `json:"hsdfs"`
	CreatedAt  int64  `json:"created_at"`
	ModifiedAt int64  `json:"modified_at"`
}

type GuardPoint struct {
	ID                string `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	GuardPointType    string `json:"guard_point_type"`
	ProtectedPath     string `json:"protected_path"`
	SecureStoragePath string `json:"secure_storage_path"`
	Policy            string `json:"policy"`
	PolicyID          string `json:"policy_id"`
	KeyID             string `json:"key_id"`
	Type              string `json:"type"`
	Enabled           bool   `json:"enabled"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

type Policy struct {
	ID            string         `json:"id"`
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	PolicyType    string         `json:"policy_type"`
	Description   string         `json:"description"`
	SecurityRules []SecurityRule `json:"security_rules"`
}

type SecurityRule struct {
	ID          string      `json:"id"`
	Order       int         `json:"order"`
	ResourceSet []string    `json:"resource_set"`
	UserSet     []string    `json:"user_set"`
	ProcessSet  []string    `json:"process_set"`
	Action      []string    `json:"action"`
	Browsing    bool        `json:"browsing"`
	Effect      RuleEffect  `json:"effect"`
}

type RuleEffect struct {
	Permission string       `json:"permission"`
	Option     EffectOption `json:"option"`
}

type EffectOption struct {
	ApplyKey bool `json:"apply_key"`
	Audit    bool `json:"audit"`
}